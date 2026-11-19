package share

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/outbound"
)

var appShortPath = regexp.MustCompile(`^/s/([A-Za-z0-9_-]{2,40})/?$`)
var pipixiaPath = regexp.MustCompile(`^/item/([1-9][0-9]{10,19})/?$`)
var pipixiaProfile = profile{
	code: "pipixia", pageHosts: []string{"pipix.com", "www.pipix.com", "h5.pipix.com"},
	apiHosts: []string{"h5.pipix.com"}, cdn: []string{"ppxvod.com", "ppxcdn.com", "pipix.com", "byteimg.com", "pstatp.com", "bytecdn.cn", "snssdk.com"},
	id: func(u *url.URL) string { return pathID(u, pipixiaPath) }, short: func(u *url.URL) bool { return appShortPath.MatchString(u.Path) },
	mobile: true, provider: pipixia,
}

func pipixia(ctx context.Context, a *adapter, _ *url.URL, _ page, id string) (*domain.Media, error) {
	data, err := a.get(ctx, "https://h5.pipix.com/bds/cell/cell_h5_comment/", url.Values{"count": {"5"}, "aid": {"1319"}, "app_name": {"super"}, "cell_id": {id}}, nil)
	if err != nil {
		return nil, err
	}
	if data.str("status_code") != "0" {
		return nil, outbound.NoVideo
	}
	for _, raw := range data.at("data", "cell_comments").values() {
		item := asObj(raw).at("comment_info", "item")
		if first(item.str("item_id_str"), item.str("item_id")) != id {
			continue
		}
		video, author := item.at("video"), item.at("author")
		if video.value == nil {
			return nil, noVideoType
		}
		var sources []candidate
		for _, key := range []string{"video_high", "video_fallback", "video_download"} {
			entry := video.at(key)
			width, height := dimensions(entry, "width", "height")
			for _, source := range listCandidates(entry.at("url_list")) {
				source.width, source.height, source.codec = width, height, entry.str("codec_type")
				source.quality = resolutions(entry.str("definition"))
				sources = append(sources, source)
			}
		}
		return result(a.video(id, item.str("content"), "", first(listString(item.at("cover", "url_list")), listString(video.at("cover_image", "url_list"))),
			a.author(first(author.str("id_str"), author.str("id")), author.str("name"), listString(author.at("avatar", "url_list"))),
			video.number("duration"), int64(item.number("create_time")), sources...))
	}
	return nil, outbound.NoVideo
}

var postSharePath = regexp.MustCompile(`^/(?:[A-Za-z0-9_-]+/)*(?:post|share|detail|index)(?:[._][A-Za-z0-9_-]+)?/?$`)
var pipigxPostPath = regexp.MustCompile(`^/pp/post/([1-9][0-9]{0,19})/?$`)

func postID(u *url.URL) string {
	if u.Path != "/" && !strings.HasPrefix(u.Path, "/share/") && !postSharePath.MatchString(u.Path) {
		return ""
	}
	return param(u, "pid", decimalID)
}

var pipigxProfile = profile{
	code: "pipigx", pageHosts: []string{"pipigx.com", "www.pipigx.com", "h5.pipigx.com", "share.pipigx.com", "ippzone.com", "share.ippzone.com", "h5.ippzone.com"},
	apiHosts: []string{"h5.pipigx.com"}, cdn: []string{"pipigx.com", "ippzone.com", "szsttkj.com", "ksyuncdn.com"},
	id: func(u *url.URL) string {
		return first(pathID(u, pipigxPostPath), postID(u))
	},
	short: func(u *url.URL) bool {
		return appShortPath.MatchString(u.Path) || shortOn(u, "share.pipigx.com", "share.ippzone.com")
	},
	provider: pipigx,
}

func pipigx(ctx context.Context, a *adapter, u *url.URL, _ page, id string) (*domain.Media, error) {
	payload := map[string]any{"pid": json.Number(id), "type": "post"}
	if mid := param(u, "mid", decimalID); mid != "" {
		payload["mid"] = json.Number(mid)
	}
	data, err := a.post(ctx, "https://h5.pipigx.com/ppapi/share/fetch_content", payload)
	if err != nil {
		return nil, err
	}
	post := data.at("data", "post")
	if post.value == nil || first(post.str("pid"), post.str("id"), id) != id {
		return nil, outbound.NoVideo
	}
	videos := post.at("videos")
	entries := videos.values()
	for _, key := range slices.Sorted(maps.Keys(videos.fields())) {
		entries = append(entries, videos.fields()[key])
	}
	for _, raw := range entries {
		video := asObj(raw)
		cover := first(listString(video.at("cover_urls")), video.str("cover"))
		if thumb := video.str("thumb"); cover == "" && shareID.MatchString(thumb) {
			cover = "https://file.ippzone.com/img/frame/id/" + thumb
		}
		if media := a.video(id, post.str("content"), "", cover, nil, 0, int64(post.number("ct")), candidate{raw: video.str("url")}); media != nil {
			return media, nil
		}
	}
	if len(entries) == 0 {
		return nil, noVideoType
	}
	return nil, outbound.NoVideo
}

var zuiyouProfile = profile{
	code: "zuiyou", pageHosts: []string{"izuiyou.com", "www.izuiyou.com", "share.izuiyou.com", "share.xiaochuankeji.cn"},
	apiHosts: []string{"share.xiaochuankeji.cn"}, cdn: []string{"izuiyou.com", "xiaochuankeji.cn", "izuiyou.net", "ksyuncdn.com"},
	id: postID, short: func(u *url.URL) bool { return appShortPath.MatchString(u.Path) || shortOn(u, "share.izuiyou.com") },
	provider: zuiyou,
}

func zuiyou(ctx context.Context, a *adapter, u *url.URL, _ page, id string) (*domain.Media, error) {
	data, err := a.post(ctx, "https://share.xiaochuankeji.cn/planck/share/post/detail_h5", map[string]any{"pid": json.Number(id), "h_av": "5.2.13.011"})
	if err != nil {
		return nil, err
	}
	post := data.at("data", "post")
	if post.value == nil || first(post.str("pid"), post.str("id"), id) != id {
		return nil, outbound.NoVideo
	}
	videos, member := post.at("videos"), post.at("member")
	if len(videos.fields()) == 0 {
		return nil, noVideoType
	}
	requested := param(u, "vid", decimalID)
	for _, raw := range post.at("imgs").values() {
		image := asObj(raw)
		vid := image.str("id")
		if vid == "" || requested != "" && vid != requested {
			continue
		}
		if media := a.video(id, post.str("content"), "", listString(image.at("urls", "540_webp", "urls")),
			a.author(member.str("id"), member.str("name"), listString(member.at("avatar_urls", "origin", "urls"))), 0, 0,
			candidate{raw: videos.str(vid, "url")}); media != nil {
			return media, nil
		}
	}
	return nil, outbound.NoVideo
}

var jimengPath = regexp.MustCompile(`^/(?:ai-tool/)?(?:share|item|video|detail|post)(?:/[A-Za-z0-9_-]+)*/([1-9][0-9]{10,19})/?$`)
var jimengQueryPath = regexp.MustCompile(`^/(?:ai-tool/)?(?:share|item|video|detail|post|community)(?:/[A-Za-z0-9_.-]+)*/?$`)
var bytedanceMedia = []string{"byteimg.com", "byteicdn.com", "bytecdn.cn", "jianying.com", "capcut.com", "volccdn.com", "douyinvod.com", "bytevod.com", "ibytedtos.com", "tos-cn-beijing.volces.com"}
var jimengProfile = profile{
	code: "jimeng", pageHosts: []string{"jimeng.jianying.com"}, apiHosts: []string{"jimeng.jianying.com"},
	cdn: append([]string{"jimeng.com"}, bytedanceMedia...),
	id: func(u *url.URL) string {
		if strings.TrimSuffix(u.Path, "/") == "/activities/reflux/mproject" {
			return param(u, "id", decimalID)
		}
		if jimengQueryPath.MatchString(u.Path) {
			return first(param(u, "item_id", decimalID), param(u, "id", decimalID), pathID(u, jimengPath))
		}
		return pathID(u, jimengPath)
	},
	short: func(u *url.URL) bool { return appShortPath.MatchString(u.Path) }, provider: jimeng,
}

func jimeng(ctx context.Context, a *adapter, _ *url.URL, _ page, id string) (*domain.Media, error) {
	data, err := a.post(ctx, "https://jimeng.jianying.com/mweb/v1/get_item_info", map[string]string{"published_item_id": id})
	if err != nil {
		return nil, err
	}
	if code := data.str("ret"); code != "" && code != "0" {
		return nil, outbound.NoVideo
	}
	common, video, author := data.at("data", "common_attr"), data.at("data", "video"), data.at("data", "author")
	if common.str("published_item_id") != id {
		return nil, outbound.NoVideo
	}
	if video.value == nil {
		return nil, noVideoType
	}
	transcoded := video.at("transcoded_video")
	origin := video.at("origin_video")
	items := []candidate{{raw: transcoded.str("origin", "video_url")}, {raw: origin.str("video_url")}}
	for _, key := range slices.Sorted(maps.Keys(transcoded.fields())) {
		entry := transcoded.at(key)
		width, height := dimensions(entry, "width", "height")
		items = append(items, candidate{raw: entry.str("video_url"), quality: resolutions(key), width: width, height: height})
	}
	duration := video.number("duration_ms") / 1000
	if duration == 0 {
		duration = video.number("duration")
	}
	return result(a.video(id, common.str("description"), "", first(common.str("cover_url_map", "1080"), common.str("cover_url_map", "720"), common.str("cover_url_map", "original"), common.str("cover_url")),
		a.author(author.str("uid"), author.str("name"), author.str("avatar_url")), duration, int64(common.number("create_time")), items...))
}

var doubaoProfile = profile{
	code: "doubao", pageHosts: []string{"doubao.com", "www.doubao.com"}, apiHosts: []string{"www.doubao.com"},
	cdn: append([]string{"doubao.com", "douyinpic.com"}, bytedanceMedia...),
	id: func(u *url.URL) string {
		if strings.TrimSuffix(u.Path, "/") != "/video-sharing" || param(u, "share_id", shareID) == "" {
			return ""
		}
		return param(u, "video_id", shareID)
	},
	short: func(u *url.URL) bool { return appShortPath.MatchString(u.Path) }, provider: doubao,
}

func doubao(ctx context.Context, a *adapter, u *url.URL, _ page, id string) (*domain.Media, error) {
	ref := url.Values{"share_id": {param(u, "share_id", shareID)}, "video_id": {id}}
	payload, err := json.Marshal(map[string]string{"share_id": ref.Get("share_id"), "vid": id, "creation_id": ""})
	if err != nil {
		return nil, domain.Upstream
	}
	data, err := a.json(ctx, http.MethodPost, "https://www.doubao.com/creativity/share/get_video_share_info", url.Values{
		"version_code": {"20800"}, "language": {"zh-CN"}, "device_platform": {"web"}, "aid": {"497858"}, "real_aid": {"497858"}, "samantha_web": {"1"},
	}, payload, http.Header{"Content-Type": {"application/json"}, "Origin": {"https://www.doubao.com"}, "Referer": {"https://www.doubao.com/video-sharing?" + ref.Encode()}})
	if err != nil {
		return nil, err
	}
	if data.str("code") != "0" {
		return nil, outbound.NoVideo
	}
	root := data.at("data")
	play, user := root.at("play_info"), root.at("user_info")
	width, height := dimensions(play, "width", "height")
	quality := resolutions(play.str("definition"))
	if media := a.video(id, root.str("prompt"), "", play.str("poster_url"),
		a.author(user.str("user_id"), first(user.str("nickname"), user.str("user_name")), ""), 0, 0,
		candidate{raw: play.str("main"), width: width, height: height, quality: quality},
		candidate{raw: play.str("backup"), width: width, height: height, quality: quality},
	); media != nil {
		return media, nil
	}
	for _, node := range []obj{root.at("video"), root.at("video_info"), root.at("video_data"), root} {
		if reported := first(node.str("video_id"), node.str("vid")); reported != "" && reported != id {
			continue
		}
		var items []candidate
		for _, key := range []string{"video_url", "play_url", "download_url", "url"} {
			items = append(items, candidate{raw: node.str(key)})
		}
		items = append(items, listCandidates(node.at("url_list"))...)
		if media := a.video(id, first(node.str("title"), root.str("title")), root.str("description"), first(node.str("cover_url"), root.str("cover_url")),
			nil, node.number("duration"), 0, items...); media != nil {
			return media, nil
		}
	}
	return nil, outbound.NoVideo
}

func result(media *domain.Media) (*domain.Media, error) {
	if media == nil {
		return nil, outbound.NoVideo
	}
	return media, nil
}
