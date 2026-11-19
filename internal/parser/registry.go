package parser

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nie-crossroads/clipbridge/internal/domain"
)

type Parser interface {
	Platform() string
	Match(*url.URL) bool
	Capabilities() []string
	Parse(context.Context, *url.URL) (*domain.Media, error)
}

type Platform struct {
	Code           string     `json:"platform"`
	Name           string     `json:"name"`
	Implemented    bool       `json:"implemented"`
	Enabled        bool       `json:"enabled"`
	Capabilities   []string   `json:"capabilities"`
	LastTestStatus *string    `json:"last_test_status"`
	LastTestedAt   *time.Time `json:"last_tested_at"`
	TimeoutSeconds int        `json:"timeout_seconds,omitempty"`
	ConfigVersion  uint64     `json:"-"`
}

var catalog = []struct {
	code, name string
	hosts      []string
}{
	{"douyin", "抖音", []string{"douyin.com", "iesdouyin.com"}},
	{"kuaishou", "快手", []string{"kuaishou.com", "gifshow.com", "chenzhongtech.com"}},
	{"bilibili", "B站", []string{"bilibili.com", "b23.tv"}},
	{"xiaohongshu", "小红书", []string{"xiaohongshu.com", "xhslink.com"}},
	{"weibo", "微博", []string{"weibo.com", "weibo.cn", "t.cn"}},
	{"toutiao", "今日头条", []string{"toutiao.com", "ixigua.com", "toutiaoimg.com", "www.iesdouyin.com"}},
	{"pipixia", "皮皮虾", []string{"pipix.com"}},
	{"pipigx", "皮皮搞笑", []string{"ippzone.com", "pipigx.com"}},
	{"zuiyou", "最右", []string{"izuiyou.com", "xiaochuankeji.cn"}},
	{"jimeng", "即梦", []string{"jimeng.jianying.com"}},
	{"doubao", "豆包", []string{"doubao.com"}},
	{"netease", "网易云音乐", []string{"music.163.com", "163cn.tv"}},
}

type Registry map[string]Parser

func NewRegistry(adapters ...Parser) Registry {
	result := make(Registry, len(adapters))
	for _, adapter := range adapters {
		result[adapter.Platform()] = adapter
	}
	return result
}

func (r Registry) Platforms() []Platform {
	items := make([]Platform, 0, len(catalog))
	for _, entry := range catalog {
		adapter := r[entry.code]
		item := Platform{Code: entry.code, Name: entry.name, Capabilities: []string{}}
		if adapter != nil {
			item.Implemented = true
			item.Capabilities = adapter.Capabilities()
		}
		items = append(items, item)
	}
	return items
}

var linkPattern = regexp.MustCompile(`(?i)https?://[^\s<>"'，。！？；、）】》]+`)

func (r Registry) Select(text, filter string) (Parser, *url.URL, string, error) {
	if utf8.RuneCountInString(text) > 500 {
		return nil, nil, "", domain.InvalidInput
	}
	var foundURL bool
	for _, raw := range linkPattern.FindAllString(text, 10) {
		u, err := url.Parse(strings.TrimRightFunc(raw, func(c rune) bool { return unicode.IsSpace(c) || strings.ContainsRune(".,!?);]}", c) }))
		if err != nil || u.User != nil || u.Port() != "" || u.Hostname() == "" {
			continue
		}
		foundURL = true
		host := strings.ToLower(u.Hostname())
		for _, entry := range catalog {
			for _, suffix := range entry.hosts {
				if host != suffix && !strings.HasSuffix(host, "."+suffix) {
					continue
				}
				if filter != "" && filter != entry.code {
					break
				}
				adapter := r[entry.code]
				if adapter == nil {
					return nil, nil, entry.code, &domain.Problem{Status: 400, Code: "PLATFORM_UNAVAILABLE", Message: "该平台暂未开放解析。"}
				}
				if !adapter.Match(u) {
					break
				}
				return adapter, u, entry.code, nil
			}
		}
	}
	if !foundURL {
		return nil, nil, "", domain.InvalidInput
	}
	return nil, nil, "", &domain.Problem{Status: 400, Code: "UNSUPPORTED_URL", Message: "该分享链接暂不支持，请检查作品地址和所选平台。"}
}
