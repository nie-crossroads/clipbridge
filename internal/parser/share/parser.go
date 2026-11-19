package share

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/outbound"
	"github.com/nie-crossroads/clipbridge/internal/parser"
)

const desktopAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
const mobileAgent = "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36"

var decimalID = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
var shareID = regexp.MustCompile(`^[a-zA-Z0-9_-]{2,128}$`)
var noVideoType = &domain.Problem{Status: 400, Code: "UNSUPPORTED_MEDIA_TYPE", Message: "链接指向的作品没有可解析的视频。"}

type profile struct {
	code                     string
	pageHosts, apiHosts, cdn []string
	id                       func(*url.URL) string
	short                    func(*url.URL) bool
	mobile, readWorkPage     bool
	provider                 func(context.Context, *adapter, *url.URL, page, string) (*domain.Media, error)
}
type adapter struct {
	profile
	pages, api fetcher
}

type fetcher interface {
	Fetch(context.Context, string, *url.URL, []byte, http.Header, bool, string) ([]byte, *url.URL, error)
}

func Adapters() []parser.Parser {
	profiles := []profile{douyinProfile, kuaishouProfile, xhsProfile, weiboProfile, toutiaoProfile, pipixiaProfile, pipigxProfile, zuiyouProfile, jimengProfile, doubaoProfile}
	adapters := make([]parser.Parser, 0, len(profiles))
	for _, spec := range profiles {
		adapters = append(adapters, &adapter{profile: spec, pages: outbound.New(spec.pageHosts...), api: outbound.New(spec.apiHosts...)})
	}
	return adapters
}

func (a *adapter) Platform() string     { return a.code }
func (*adapter) Capabilities() []string { return []string{"video"} }
func (a *adapter) Match(u *url.URL) bool {
	return slices.Contains(a.pageHosts, strings.ToLower(u.Hostname())) && (a.id(u) != "" || a.short != nil && a.short(u))
}

func (a *adapter) Parse(ctx context.Context, address *url.URL) (*domain.Media, error) {
	if !a.Match(address) {
		return nil, domain.InvalidInput
	}
	id := a.id(address)
	var doc page
	var pageErr error
	// Standalone detail interfaces do not need a second page request for a full link.
	if id == "" || a.readWorkPage {
		agent := desktopAgent
		if a.mobile {
			agent = mobileAgent
		}
		body, final, err := a.pages.Fetch(ctx, http.MethodGet, address, nil, nil, true, agent)
		if err != nil {
			if id == "" {
				return nil, err
			}
			pageErr = err
		} else {
			landing := readPage(body)
			if gated(final, landing) {
				return nil, outbound.AuthRequired
			}
			if finalID := a.id(final); finalID != "" {
				if id != "" && finalID != id {
					return nil, outbound.NotFound
				}
				address, id = final, finalID
				doc = landing
			} else if id == "" {
				return nil, outbound.NotFound
			} else {
				pageErr = outbound.NotFound
			}
		}
	}
	if id == "" {
		return nil, domain.InvalidInput
	}
	if pageErr == domain.Timeout {
		return nil, pageErr
	}
	media, err := a.provider(ctx, a, address, doc, id)
	if media != nil || err == noVideoType || err == domain.Timeout {
		return media, err
	}
	if found := a.standard(doc, id); found != nil {
		return found, nil
	}
	if err != nil && err != outbound.NoVideo {
		return nil, err
	}
	if pageErr != nil {
		return nil, pageErr
	}
	return nil, outbound.NoVideo
}

func gated(final *url.URL, doc page) bool {
	path := strings.ToLower(final.Path)
	return strings.HasPrefix(path, "/login") || strings.HasPrefix(path, "/404/sec_") ||
		strings.Contains(path, "/captcha") || strings.Contains(path, "/verify") ||
		strings.Contains(doc.title, "访问受限") || strings.Contains(doc.title, "安全验证")
}
func shortOn(address *url.URL, hosts ...string) bool {
	return slices.Contains(hosts, strings.ToLower(address.Hostname())) && shareID.MatchString(strings.Trim(address.Path, "/"))
}
func pathID(address *url.URL, re *regexp.Regexp) string {
	match := re.FindStringSubmatch(address.Path)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}
func param(address *url.URL, key string, re *regexp.Regexp) string {
	value := address.Query().Get(key)
	if re.MatchString(value) {
		return value
	}
	return ""
}
func (a *adapter) get(ctx context.Context, endpoint string, query url.Values, headers http.Header) (obj, error) {
	return a.json(ctx, http.MethodGet, endpoint, query, nil, headers)
}
func (a *adapter) post(ctx context.Context, endpoint string, payload any) (obj, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return obj{}, domain.Upstream
	}
	return a.json(ctx, http.MethodPost, endpoint, nil, body, http.Header{"Content-Type": {"application/json"}})
}
func (a *adapter) json(ctx context.Context, method, endpoint string, query url.Values, payload []byte, headers http.Header) (obj, error) {
	address, err := url.Parse(endpoint)
	if err != nil {
		return obj{}, domain.Upstream
	}
	if query != nil {
		address.RawQuery = query.Encode()
	}
	if headers == nil {
		headers = http.Header{}
	}
	headers.Set("Accept", "application/json")
	agent := desktopAgent
	if a.mobile {
		agent = mobileAgent
	}
	body, _, err := a.api.Fetch(ctx, method, address, payload, headers, false, agent)
	if err != nil {
		return obj{}, err
	}
	data := decode(body)
	if data.value == nil {
		return obj{}, outbound.NoVideo
	}
	return data, nil
}

type candidate struct {
	raw, quality, codec string
	width, height       int
}

func (a *adapter) video(id, title, description, cover string, author *domain.Author, duration float64, date int64, items ...candidate) *domain.Media {
	var sources []domain.Source
	seen := map[string]bool{}
	for _, item := range items {
		address := outbound.Resource(item.raw, a.cdn...)
		if address == "" || seen[address] {
			continue
		}
		u, _ := url.Parse(address)
		path := strings.ToLower(u.Path)
		if strings.HasSuffix(path, ".m3u8") || strings.HasSuffix(path, ".mpd") || strings.HasSuffix(path, ".m4s") || strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".webp") {
			continue
		}
		seen[address] = true
		source := domain.Source{URL: address, Quality: "unknown", WatermarkStatus: "unknown"}
		if item.quality != "" {
			source.Quality = item.quality
		}
		if item.codec != "" {
			source.Codec = &item.codec
		}
		if item.width > 0 {
			source.Width = &item.width
		}
		if item.height > 0 {
			source.Height = &item.height
		}
		if strings.HasSuffix(path, ".mp4") {
			source.MIMEType = "video/mp4"
		}
		sources = append(sources, source)
	}
	if len(sources) == 0 {
		return nil
	}
	// Preserve platform order within each protocol and put HTTPS sources first.
	slices.SortStableFunc(sources, func(x, y domain.Source) int {
		return strings.Compare(strings.SplitN(y.URL, ":", 2)[0], strings.SplitN(x.URL, ":", 2)[0])
	})
	media := domain.NewMedia(a.code, id, "video")
	media.Title, media.Description = title, description
	media.Cover = outbound.Resource(cover, a.cdn...)
	media.Author = author
	if duration > 0 {
		media.DurationSeconds = &duration
	}
	if date > 0 {
		if date > 9999999999 {
			date /= 1000
		}
		ts := time.Unix(date, 0).UTC()
		media.PublishedAt = &ts
	}
	media.Video = &sources[0]
	media.Video.Alternatives = sources[1:]
	return &media
}

func (a *adapter) author(id, name, avatar string) *domain.Author {
	if id == "" && name == "" {
		return nil
	}
	return &domain.Author{ID: id, Name: name, Avatar: outbound.Resource(avatar, a.cdn...)}
}

func resolutions(raw string) string {
	for _, value := range []string{"2160", "1440", "1080", "720", "540", "480", "360", "240"} {
		if strings.Contains(raw, value) {
			return value + "p"
		}
	}
	if strings.Contains(strings.ToUpper(raw), "2K") {
		return "2K"
	}
	if strings.Contains(strings.ToUpper(raw), "4K") {
		return "4K"
	}
	return ""
}
func dimensions(node obj, widthKey, heightKey string) (int, int) {
	width, _ := strconv.Atoi(node.str(widthKey))
	height, _ := strconv.Atoi(node.str(heightKey))
	return width, height
}
func listCandidates(node obj) []candidate {
	var result []candidate
	for _, value := range node.values() {
		if raw := text(value); raw != "" {
			result = append(result, candidate{raw: raw})
			continue
		}
		if raw := asObj(value).str("url"); raw != "" {
			result = append(result, candidate{raw: raw})
		}
	}
	return result
}
