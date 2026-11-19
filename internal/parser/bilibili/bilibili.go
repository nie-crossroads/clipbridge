package bilibili

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/outbound"
)

type Parser struct{ client *http.Client }

var shortPath = regexp.MustCompile(`^/[a-zA-Z0-9]{1,32}/?$`)
var videoPath = regexp.MustCompile(`^/video/(BV[0-9A-Za-z]{10}|av[1-9][0-9]{0,17})/?$`)

func New() *Parser {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.MaxIdleConns = 8
	transport.DialContext = dialPublic
	return &Parser{client: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (*Parser) Platform() string       { return "bilibili" }
func (*Parser) Capabilities() []string { return []string{"video"} }
func (*Parser) Match(u *url.URL) bool {
	switch strings.ToLower(u.Hostname()) {
	case "b23.tv":
		return shortPath.MatchString(u.Path)
	case "bilibili.com", "www.bilibili.com", "m.bilibili.com":
		return videoPath.MatchString(u.Path)
	}
	return false
}

func dialPublic(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if port != "443" || (host != "b23.tv" && host != "api.bilibili.com") {
		return nil, errors.New("unexpected upstream destination")
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: 4 * time.Second}
	for _, ip := range ips {
		if outbound.IsPublic(ip) {
			connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return connection, nil
			}
		}
	}
	return nil, errors.New("no reachable public upstream address")
}

type detail struct {
	BVID     string
	Title    string
	Desc     string
	Pic      string
	Duration float64
	Pubdate  int64
	Owner    struct {
		Mid  int64
		Name string
		Face string
	}
	Pages []struct {
		CID  int64
		Part string
	}
}

type playback struct {
	Format   string
	Segments []struct {
		URL    string
		Backup []string `json:"backup_url"`
		Size   int64
	} `json:"durl"`
}

func (p *Parser) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	res, err := p.client.Do(req)
	if err != nil {
		var timeout net.Error
		if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &timeout) && timeout.Timeout() {
			return nil, domain.Timeout
		}
		return nil, domain.Upstream
	}
	return res, nil
}

func (p *Parser) expand(ctx context.Context, share *url.URL) (*url.URL, error) {
	res, err := p.get(ctx, "https://b23.tv"+share.EscapedPath())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	target, err := res.Location()
	if err != nil || res.StatusCode < 300 || res.StatusCode >= 400 || target.User != nil || target.Port() != "" ||
		(target.Scheme != "https" && target.Scheme != "http") || strings.EqualFold(target.Hostname(), "b23.tv") || !p.Match(target) {
		return nil, domain.Upstream
	}
	return target, nil
}

func readAPI[T any](ctx context.Context, p *Parser, path string, query url.Values) (T, error) {
	var envelope struct {
		Code *int
		Data *T
	}
	var empty T
	res, err := p.get(ctx, "https://api.bilibili.com"+path+"?"+query.Encode())
	if err != nil {
		return empty, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return empty, domain.Upstream
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return empty, domain.Timeout
	}
	if err != nil || len(body) > 8<<20 || json.Unmarshal(body, &envelope) != nil || envelope.Code == nil {
		return empty, domain.Upstream
	}
	if path == "/x/web-interface/view" && *envelope.Code == -404 {
		return empty, &domain.Problem{Status: 404, Code: "MEDIA_NOT_FOUND", Message: "平台返回作品不存在。"}
	}
	if *envelope.Code != 0 || envelope.Data == nil {
		return empty, domain.Upstream
	}
	return *envelope.Data, nil
}

func (p *Parser) Parse(ctx context.Context, share *url.URL) (*domain.Media, error) {
	if strings.EqualFold(share.Hostname(), "b23.tv") {
		longURL, err := p.expand(ctx, share)
		if err != nil {
			return nil, err
		}
		share = longURL
	}
	match := videoPath.FindStringSubmatch(share.Path)
	if match == nil {
		return nil, domain.InvalidInput
	}
	id := match[1]
	query := url.Values{"bvid": {id}}
	if strings.HasPrefix(id, "av") {
		query = url.Values{"aid": {id[2:]}}
	}
	metadata, err := readAPI[detail](ctx, p, "/x/web-interface/view", query)
	if err != nil {
		return nil, err
	}
	if metadata.BVID == "" || len(metadata.Pages) == 0 {
		return nil, domain.Upstream
	}
	part := 1
	if raw := share.Query().Get("p"); raw != "" {
		part, err = strconv.Atoi(raw)
		if err != nil || part < 1 || part > len(metadata.Pages) {
			return nil, domain.InvalidInput
		}
	}
	page := metadata.Pages[part-1]
	if page.CID <= 0 {
		return nil, domain.Upstream
	}
	data, err := readAPI[playback](ctx, p, "/x/player/playurl", url.Values{
		"bvid": {metadata.BVID}, "cid": {strconv.FormatInt(page.CID, 10)}, "fnval": {"1"}, "qn": {"64"}, "platform": {"html5"},
	})
	if err != nil {
		return nil, err
	}
	if len(data.Segments) != 1 || !strings.HasPrefix(data.Format, "mp4") {
		return nil, &domain.Problem{Status: 502, Code: "MEDIA_FORMAT_UNSUPPORTED", Message: "平台没有提供完整的 MP4 直链。"}
	}
	segment := data.Segments[0]
	var sources []domain.Source
	seen := map[string]bool{}
	for _, raw := range append([]string{segment.URL}, segment.Backup...) {
		address := outbound.Resource(raw, "bilivideo.com", "bilivideo.cn", "bilibili.com", "bilibili.net")
		if address == "" || seen[address] {
			continue
		}
		seen[address] = true
		source := domain.Source{URL: address, Quality: "unknown", WatermarkStatus: "unknown", MIMEType: "video/mp4"}
		if segment.Size > 0 {
			source.SizeBytes = &segment.Size
		}
		sources = append(sources, source)
	}
	if len(sources) == 0 {
		return nil, domain.Upstream
	}
	media := domain.NewMedia("bilibili", metadata.BVID, "video")
	media.Title, media.Description = metadata.Title, metadata.Desc
	media.Cover = outbound.Resource(metadata.Pic, "hdslb.com")
	if metadata.Owner.Mid > 0 {
		media.Author = &domain.Author{ID: strconv.FormatInt(metadata.Owner.Mid, 10), Name: metadata.Owner.Name, Avatar: outbound.Resource(metadata.Owner.Face, "hdslb.com")}
	}
	if metadata.Duration > 0 && len(metadata.Pages) == 1 {
		media.DurationSeconds = &metadata.Duration
	}
	if metadata.Pubdate > 0 {
		stamp := time.Unix(metadata.Pubdate, 0).UTC()
		media.PublishedAt = &stamp
	}
	slices.SortStableFunc(sources, func(a, b domain.Source) int {
		if strings.HasPrefix(a.URL, "https:") == strings.HasPrefix(b.URL, "https:") {
			return 0
		}
		if strings.HasPrefix(a.URL, "https:") {
			return -1
		}
		return 1
	})
	media.Video = &sources[0]
	media.Video.Alternatives = sources[1:]
	if len(metadata.Pages) > 1 {
		media.Extra["part"] = part
		media.Extra["part_count"] = len(metadata.Pages)
	}
	return &media, nil
}
