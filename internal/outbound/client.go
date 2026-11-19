package outbound

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/domain"
)

type Client struct {
	http  *http.Client
	hosts []string
}

func New(hosts ...string) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.MaxIdleConns = 32
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || port != "443" || !exact(host, hosts) {
			return nil, errors.New("unexpected upstream destination")
		}
		return dialPublic(ctx, network, address)
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, hosts: hosts}
}

func dialPublic(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	dial := net.Dialer{}
	for _, ip := range ips {
		if !IsPublic(ip) {
			continue
		}
		conn, err := dial.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("no reachable public upstream address")
}

var excluded = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("3fff::/20"),
}
var globalIPv6 = netip.MustParsePrefix("2000::/3")

func IsPublic(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.Is6() && !globalIPv6.Contains(ip) {
		return false
	}
	for _, prefix := range excluded {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}
func exact(host string, hosts []string) bool {
	for _, allowed := range hosts {
		if strings.EqualFold(host, allowed) {
			return true
		}
	}
	return false
}

var AuthRequired = &domain.Problem{Status: 502, Code: "UPSTREAM_ACCESS_RESTRICTED", Message: "平台限制了此分享页的匿名访问，请确认作品公开并重新复制完整分享链接。"}
var NotFound = &domain.Problem{Status: 404, Code: "MEDIA_NOT_FOUND", Message: "平台返回作品不存在或分享链接已失效。"}
var NoVideo = &domain.Problem{Status: 502, Code: "UPSTREAM_VIDEO_UNAVAILABLE", Message: "平台未在分享页中提供可用的视频直链。", Retryable: true}

func (c *Client) Fetch(ctx context.Context, method string, address *url.URL, payload []byte, headers http.Header, follow bool, agent string) ([]byte, *url.URL, error) {
	target := *address
	if target.Scheme == "http" {
		target.Scheme = "https"
	}
	for hop := 0; hop <= 5; hop++ {
		if target.Scheme != "https" || target.User != nil || target.Port() != "" || !exact(target.Hostname(), c.hosts) {
			return nil, nil, AuthRequired
		}
		request, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(payload))
		if err != nil {
			return nil, nil, domain.Upstream
		}
		request.Header.Set("User-Agent", agent)
		request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
		for key, values := range headers {
			request.Header[key] = values
		}
		response, err := c.http.Do(request)
		if err != nil {
			var timeout net.Error
			if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &timeout) && timeout.Timeout() {
				return nil, nil, domain.Timeout
			}
			return nil, nil, domain.Upstream
		}
		if follow && response.StatusCode >= 300 && response.StatusCode < 400 {
			next, err := response.Location()
			response.Body.Close()
			if err != nil {
				return nil, nil, domain.Upstream
			}
			target = *next
			continue
		}
		defer response.Body.Close()
		switch response.StatusCode {
		case 401, 403, 429:
			return nil, nil, AuthRequired
		case 301, 302, 303, 307, 308:
			return nil, nil, AuthRequired
		case 404, 410:
			return nil, nil, NotFound
		case 200:
		default:
			return nil, nil, domain.Upstream
		}
		data, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, nil, domain.Timeout
		}
		if err != nil || len(data) > 8<<20 {
			return nil, nil, domain.Upstream
		}
		return data, &target, nil
	}
	return nil, nil, domain.Upstream
}

func Resource(raw string, suffixes ...string) string {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "//") {
		value = "https:" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || parsed.Port() != "" {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	for _, suffix := range suffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return parsed.String()
		}
	}
	return ""
}
