package share

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/nie-crossroads/clipbridge/internal/outbound"
	"github.com/nie-crossroads/clipbridge/internal/parser"
)

type fetchFunc func(string, *url.URL, []byte) ([]byte, *url.URL, error)

func (f fetchFunc) Fetch(_ context.Context, method string, u *url.URL, body []byte, _ http.Header, _ bool, _ string) ([]byte, *url.URL, error) {
	return f(method, u, body)
}

func TestVideoAdapters(t *testing.T) {
	tests := []struct {
		spec                                           profile
		share, endpoint, pageData, apiData, id, source string
	}{
		{douyinProfile, "https://www.douyin.com/video/7693799956506774803", "",
			`<script>window._ROUTER_DATA={"data":{"awemeId":"7693799956506774803","desc":"抖音样例","video":{"playAddr":[{"src":"https://v.douyinvod.com/dy.mp4"}]}}};</script>`, "",
			"7693799956506774803", "https://v.douyinvod.com/dy.mp4"},
		{kuaishouProfile, "https://www.kuaishou.com/short-video/Work123", "",
			`<script>window.INIT_STATE={"photo":{"photoId":"Work123","mainMvUrls":[{"url":"https://v.kwaicdn.com/ks.mp4"}]}};</script>`, "",
			"Work123", "https://v.kwaicdn.com/ks.mp4"},
		{xhsProfile, "https://www.xiaohongshu.com/explore/123456789012345678901234", "",
			`<script>window.__INITIAL_STATE__={"note":{"noteDetailMap":{"123456789012345678901234":{"note":{"unused":undefined,"video":{"media":{"stream":{"h264":[{"masterUrl":"https://v.xhscdn.com/xhs.mp4"}]}}}}}}}};</script>`, "",
			"123456789012345678901234", "https://v.xhscdn.com/xhs.mp4"},
		{weiboProfile, "https://weibo.com/tv/show/1034:5000451810754592", "/tv/api/component", "",
			`{"code":100000,"data":{"Component_Play_Playinfo":{"urls":{"720P":"//v.video.weibo.com/wb.mp4"}}}}`,
			"1034:5000451810754592", "https://v.video.weibo.com/wb.mp4"},
		{toutiaoProfile, "https://www.toutiao.com/video/7000000000000000000/", "",
			`<script id="RENDER_DATA">%7B%22data%22%3A%7B%22initialVideo%22%3A%7B%22videoPlayInfo%22%3A%7B%22video_list%22%3A%5B%7B%22main_url%22%3A%22https%3A%2F%2Fv.ixiguavideo.com%2Ftt.mp4%22%7D%5D%7D%7D%7D%7D</script>`, "",
			"7000000000000000000", "https://v.ixiguavideo.com/tt.mp4"},
		{pipixiaProfile, "https://h5.pipix.com/item/7355350944046913804", "/bds/cell/cell_h5_comment/", "",
			`{"status_code":0,"data":{"cell_comments":[{"comment_info":{"item":{"item_id":7355350944046913803,"video":{"video_high":{"url_list":[{"url":"https://v.ppxvod.com/other.mp4"}]}}}}},{"comment_info":{"item":{"item_id":7355350944046913804,"video":{"video_high":{"url_list":[{"url":"https://v.ppxvod.com/px.mp4"}]}}}}}]}}`,
			"7355350944046913804", "https://v.ppxvod.com/px.mp4"},
		{pipigxProfile, "https://h5.pipigx.com/pp/post/12345", "/ppapi/share/fetch_content", "",
			`{"data":{"post":{"id":12345,"videos":{"100":{"url":"https://video01.szsttkj.com/pg.mp4"},"200":{"url":"https://video01.szsttkj.com/second.mp4"}}}}}`,
			"12345", "https://video01.szsttkj.com/pg.mp4"},
		{zuiyouProfile, "https://share.xiaochuankeji.cn/share/post?pid=12345&vid=879", "/planck/share/post/detail_h5", "",
			`{"data":{"post":{"id":12345,"imgs":[{"id":878},{"id":879}],"videos":{"878":{"url":"https://v.izuiyou.com/other.mp4"},"879":{"url":"https://v.izuiyou.com/zy.mp4"}}}}}`,
			"12345", "https://v.izuiyou.com/zy.mp4"},
		{jimengProfile, "https://jimeng.jianying.com/activities/reflux/mproject?id=7000000000000000000", "/mweb/v1/get_item_info", "",
			`{"ret":"0","data":{"common_attr":{"published_item_id":"7000000000000000000"},"video":{"origin_video":{"video_url":"https://v3-dreamnia.jimeng.com/jm.mp4"}}}}`,
			"7000000000000000000", "https://v3-dreamnia.jimeng.com/jm.mp4"},
		{doubaoProfile, "https://www.doubao.com/video-sharing?share_id=share123&video_id=video123", "/creativity/share/get_video_share_info", "",
			`{"code":0,"data":{"play_info":{"main":"https://v9-videoweb.doubao.com/db.mp4","width":720,"height":1280,"definition":"720p"},"prompt":"样例"}}`,
			"video123", "https://v9-videoweb.doubao.com/db.mp4"},
	}
	registry := parser.NewRegistry(Adapters()...)
	for _, sample := range tests {
		t.Run(sample.spec.code, func(t *testing.T) {
			_, _, code, err := registry.Select("复制分享链接 "+sample.share+" 查看作品", "")
			if err != nil || code != sample.spec.code {
				t.Fatalf("registry selected %s: %v", code, err)
			}
			a := &adapter{profile: sample.spec,
				pages: fetchFunc(func(method string, u *url.URL, _ []byte) ([]byte, *url.URL, error) {
					if method != http.MethodGet || sample.pageData == "" {
						t.Fatalf("unexpected page fetch to %s", u.Path)
					}
					return []byte(sample.pageData), u, nil
				}),
				api: fetchFunc(func(method string, u *url.URL, body []byte) ([]byte, *url.URL, error) {
					if u.Path != sample.endpoint || sample.endpoint == "" {
						t.Fatalf("unexpected API call to %s", u.Path)
					}
					if method == http.MethodPost && len(body) == 0 {
						t.Fatal("missing API payload")
					}
					return []byte(sample.apiData), u, nil
				}),
			}
			u, err := url.Parse(sample.share)
			if err != nil {
				t.Fatal(err)
			}
			media, err := a.Parse(context.Background(), u)
			if err != nil {
				t.Fatal(err)
			}
			if media.Platform != sample.spec.code || media.ItemID != sample.id || media.Type != "video" || media.Video == nil || media.Video.URL != sample.source || media.Video.WatermarkStatus != "unknown" {
				t.Fatalf("unexpected media: %+v", media)
			}
			if len(media.Video.Alternatives) != 0 {
				t.Fatal("unrelated video returned as an alternative")
			}
		})
	}
}

func TestSharedDomainSelectsCorrectPlatform(t *testing.T) {
	registry := parser.NewRegistry(Adapters()...)
	for _, tc := range []struct{ work, platform string }{
		{"https://www.iesdouyin.com/share/video/7693799956506774803/", "douyin"},
		{"https://www.iesdouyin.com/xg/video/7676450021063735414/", "toutiao"},
		{"https://www.ixigua.com/7676450021063735414", "toutiao"},
	} {
		_, _, code, err := registry.Select(tc.work, "")
		if err != nil || code != tc.platform {
			t.Fatalf("%s selected %s: %v", tc.work, code, err)
		}
	}
}

func TestShareRedirectRequiresWork(t *testing.T) {
	u, _ := url.Parse("https://v.kuaishou.com/Test123")
	final, _ := url.Parse("https://www.kuaishou.com/fw/user/abc123")
	a := &adapter{profile: kuaishouProfile, pages: fetchFunc(func(_ string, _ *url.URL, _ []byte) ([]byte, *url.URL, error) {
		return []byte(`<meta property="og:video" content="https://v.kwaicdn.com/unrelated.mp4">`), final, nil
	})}
	if media, err := a.Parse(context.Background(), u); media != nil || err != outbound.NotFound {
		t.Fatalf("profile redirect = %v, %v", media, err)
	}
	final, _ = url.Parse("https://www.kuaishou.com/verify")
	if media, err := a.Parse(context.Background(), u); media != nil || err != outbound.AuthRequired {
		t.Fatalf("gate redirect = %v, %v", media, err)
	}
}

func TestMediaRejectsExternalHostsAndSegmentedStreams(t *testing.T) {
	a := &adapter{profile: pipixiaProfile}
	media := a.video("test", "", "", "", nil, 0, 0,
		candidate{raw: "https://ppxvod.com.example.org/video.mp4"},
		candidate{raw: "https://ppxvod.com@external.example.org/video.mp4"},
		candidate{raw: "https://v.ppxvod.com/index.m3u8"},
		candidate{raw: "http://v.ppxvod.com/back.mp4"},
		candidate{raw: "https://v.ppxvod.com/main.mp4"},
		candidate{raw: "https://v.ppxvod.com/main.mp4"},
	)
	if media == nil || media.Video.URL != "https://v.ppxvod.com/main.mp4" || len(media.Video.Alternatives) != 1 || media.Video.Alternatives[0].URL != "http://v.ppxvod.com/back.mp4" {
		t.Fatalf("unexpected media sources: %+v", media)
	}
}
