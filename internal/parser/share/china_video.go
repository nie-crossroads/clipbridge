package share

import (
	"context"
	"encoding/base64"
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

var douyinPath = regexp.MustCompile(`^/(?:share/)?video/([1-9][0-9]{15,19})(?:\.html)?/?$`)
var douyinProfile = profile{
	code: "douyin", pageHosts: []string{"douyin.com", "www.douyin.com", "v.douyin.com", "www.iesdouyin.com", "iesdouyin.com"},
	apiHosts: []string{"api.amemv.com"}, cdn: []string{"douyinvod.com", "douyin.com", "douyinpic.com", "byteimg.com", "byteicdn.com", "amemv.com", "snssdk.com"},
	id: func(u *url.URL) string { return pathID(u, douyinPath) }, short: func(u *url.URL) bool { return shortOn(u, "v.douyin.com") },
	mobile: true, readWorkPage: true, provider: douyin,
}

func douyin(ctx context.Context, a *adapter, _ *url.URL, doc page, id string) (*domain.Media, error) {
	for _, key := range []string{"_ROUTER_DATA", "RENDER_DATA"} {
		root := doc.state(key)
		node := find(root, func(node obj) bool {
			return first(node.str("aweme_id"), node.str("awemeId")) == id && node.at("video").value != nil
		})
		if media, err := douyinItem(a, node, id); media != nil || err != outbound.NoVideo {
			return media, err
		}
	}
	data, err := a.get(ctx, "https://api.amemv.com/aweme/v1/aweme/detail/", url.Values{
		"aweme_id": {id}, "aid": {"6383"}, "device_platform": {"web"}, "version_code": {"190500"}, "version_name": {"19.5.0"},
	}, nil)
	if err != nil {
		return nil, err
	}
	if data.str("status_code") != "0" {
		return nil, outbound.NoVideo
	}
	node := data.at("aweme_detail")
	if node.str("aweme_id") != id {
		return nil, outbound.NoVideo
	}
	return douyinItem(a, node, id)
}

func douyinItem(a *adapter, node obj, id string) (*domain.Media, error) {
	if len(node.at("images").values()) > 0 || len(node.at("image_post_info", "images").values()) > 0 {
		return nil, noVideoType
	}
	video := node.at("video")
	var items []candidate
	width, height := dimensions(video, "width", "height")
	for _, entry := range append(video.at("play_addr", "url_list").values(), video.at("playAddr").values()...) {
		raw := first(text(entry), asObj(entry).str("src"), asObj(entry).str("url"))
		if raw != "" {
			items = append(items, candidate{raw: raw, width: width, height: height})
		}
	}
	for _, raw := range video.at("bit_rate").values() {
		item := asObj(raw)
		w, h := dimensions(item.at("play_addr"), "width", "height")
		for _, src := range listCandidates(item.at("play_addr", "url_list")) {
			src.width, src.height, src.quality = w, h, resolutions(item.str("gear_name"))
			items = append(items, src)
		}
	}
	author := node.at("author")
	media := a.video(id, first(node.str("desc"), node.str("title")), "",
		first(listString(video.at("cover", "url_list")), listString(video.at("origin_cover", "url_list")), listString(video.at("cover"))),
		a.author(first(author.str("uid"), author.str("sec_uid")), author.str("nickname"), first(listString(author.at("avatar_thumb", "url_list")), listString(author.at("avatarThumb")))),
		video.number("duration")/1000, int64(node.number("create_time")), items...)
	if media == nil {
		return nil, outbound.NoVideo
	}
	return media, nil
}

var kuaishouPath = regexp.MustCompile(`^/(?:short-video|fw/photo|photo|f)/([0-9a-zA-Z_-]{5,48})/?$`)
var kuaishouProfile = profile{
	code: "kuaishou", pageHosts: []string{"kuaishou.com", "www.kuaishou.com", "v.kuaishou.com", "c.kuaishou.com", "m.kuaishou.com", "live.kuaishou.com", "v.m.chenzhongtech.com"},
	cdn: []string{"kwaicdn.com", "kwimgs.com", "yximgs.com", "gifshow.com", "kuaishou.com", "ksapisrv.com", "ksyuncdn.com"},
	id:  func(u *url.URL) string { return pathID(u, kuaishouPath) }, short: func(u *url.URL) bool { return shortOn(u, "v.kuaishou.com") },
	mobile: true, readWorkPage: true, provider: kuaishou,
}

func kuaishou(_ context.Context, a *adapter, address *url.URL, doc page, id string) (*domain.Media, error) {
	numericID := param(address, "photoId", decimalID)
	for _, key := range []string{"INIT_STATE", "__APOLLO_STATE__", "__NEXT_DATA__"} {
		node := find(doc.state(key), func(o obj) bool {
			for _, field := range []string{"photoId", "photo_id", "id", "encryptedPhotoId", "photoIdStr", "encryptedId"} {
				value := o.str(field)
				if value != "" && (value == id || value == numericID) && (o.at("mainMvUrls").value != nil || o.at("photoUrl").value != nil || o.at("manifest").value != nil || o.at("photoType").value != nil) {
					return true
				}
			}
			return false
		})
		if node.value == nil {
			continue
		}
		if strings.EqualFold(node.str("photoType"), "IMAGE") || len(node.at("mainMvUrls").values()) == 0 && node.at("ext_params", "atlas").value != nil {
			return nil, noVideoType
		}
		items := listCandidates(node.at("mainMvUrls"))
		if direct := node.str("photoUrl"); direct != "" {
			items = append(items, candidate{raw: direct})
		}
		for _, set := range node.at("manifest", "adaptationSet").values() {
			for _, raw := range asObj(set).at("representation").values() {
				entry := asObj(raw)
				w, h := dimensions(entry, "width", "height")
				items = append(items, candidate{raw: entry.str("url"), quality: resolutions(entry.str("qualityLabel")), width: w, height: h})
			}
		}
		media := a.video(id, node.str("caption"), "", first(listString(node.at("coverUrls")), node.str("coverUrl")),
			a.author(node.str("userId"), node.str("userName"), node.str("headUrl")),
			node.number("duration")/1000, int64(node.number("timestamp")), items...)
		if media != nil {
			return media, nil
		}
	}
	return nil, outbound.NoVideo
}

var xhsPath = regexp.MustCompile(`^/(?:explore|discovery/item|user/profile/[0-9a-fA-F]{24})/([0-9a-fA-F]{24})/?$`)
var xhsShortPath = regexp.MustCompile(`^/(?:(?:a|o)/)?[a-zA-Z0-9_-]{3,40}/?$`)
var xhsProfile = profile{
	code: "xiaohongshu", pageHosts: []string{"xiaohongshu.com", "www.xiaohongshu.com", "xhslink.com", "www.xhslink.com"},
	cdn: []string{"xhscdn.com", "xiaohongshu.com", "xhsimg.com"},
	id:  func(u *url.URL) string { return pathID(u, xhsPath) }, short: func(u *url.URL) bool {
		return (u.Hostname() == "xhslink.com" || u.Hostname() == "www.xhslink.com") && xhsShortPath.MatchString(u.Path)
	},
	mobile: true, readWorkPage: true, provider: xiaohongshu,
}

func xiaohongshu(_ context.Context, a *adapter, _ *url.URL, doc page, id string) (*domain.Media, error) {
	root := doc.state("__INITIAL_STATE__")
	note := root.at("note", "noteDetailMap", id, "note")
	if note.value == nil {
		note = find(root, func(o obj) bool { return first(o.str("noteId"), o.str("note_id")) == id && o.at("video").value != nil })
	}
	if note.str("type") == "normal" && note.at("video").value == nil {
		return nil, noVideoType
	}
	streams := note.at("video", "media", "stream")
	if streams.value == nil {
		streams = note.at("video", "mediaV2", "stream")
	}
	var sources []candidate
	for _, codec := range []string{"h264", "h265", "av1", "vp9"} {
		for _, raw := range streams.at(codec).values() {
			item := asObj(raw)
			width, height := dimensions(item, "width", "height")
			quality := resolutions(item.str("qualityType"))
			sources = append(sources, candidate{raw: item.str("masterUrl"), quality: quality, codec: codec, width: width, height: height})
			for _, backup := range listCandidates(item.at("backupUrls")) {
				backup.quality, backup.codec, backup.width, backup.height = quality, codec, width, height
				sources = append(sources, backup)
			}
		}
	}
	user := note.at("user")
	cover := first(doc.meta["og:image"], note.str("cover", "urlDefault"), note.str("cover", "url"))
	media := a.video(id, note.str("title"), note.str("desc"), cover,
		a.author(user.str("userId"), user.str("nickname"), user.str("avatar")), note.number("video", "capa", "duration"), int64(note.number("time")), sources...)
	if media != nil {
		return media, nil
	}
	return nil, outbound.NoVideo
}

var weiboTVPath = regexp.MustCompile(`^/tv/show/(1034:[a-zA-Z0-9]{8,40})/?$`)
var weiboTVID = regexp.MustCompile(`^1034:[a-zA-Z0-9]{8,40}$`)
var weiboPostPath = regexp.MustCompile(`^/(?:[0-9]+|status|detail)/([a-zA-Z0-9]{6,24})/?$`)
var weiboProfile = profile{
	code: "weibo", pageHosts: []string{"weibo.com", "www.weibo.com", "video.weibo.com", "m.weibo.cn", "weibo.cn", "t.cn"},
	apiHosts: []string{"weibo.com", "m.weibo.cn"}, cdn: []string{"video.weibo.com", "weibo.cn", "weibocdn.com", "sinaimg.cn", "sinaimg.com", "sina.com.cn"},
	id: func(u *url.URL) string {
		if strings.EqualFold(u.Hostname(), "video.weibo.com") && strings.TrimSuffix(u.Path, "/") == "/show" {
			return param(u, "fid", weiboTVID)
		}
		return first(pathID(u, weiboTVPath), pathID(u, weiboPostPath))
	}, short: func(u *url.URL) bool { return shortOn(u, "t.cn") },
	provider: weibo,
}

func weibo(ctx context.Context, a *adapter, _ *url.URL, _ page, id string) (*domain.Media, error) {
	var node obj
	var items []candidate
	if strings.HasPrefix(id, "1034:") {
		payload, _ := json.Marshal(map[string]any{"Component_Play_Playinfo": map[string]string{"oid": id}})
		form := url.Values{"data": {string(payload)}}.Encode()
		data, err := a.json(ctx, http.MethodPost, "https://weibo.com/tv/api/component", url.Values{"page": {"/tv/show/" + id}}, []byte(form),
			http.Header{"Content-Type": {"application/x-www-form-urlencoded"}, "Referer": {"https://weibo.com/tv/show/" + id}})
		if err != nil {
			return nil, err
		}
		if data.str("code") != "100000" {
			return nil, outbound.AuthRequired
		}
		node = data.at("data", "Component_Play_Playinfo")
		fields := node.at("urls").fields()
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			items = append(items, candidate{raw: text(fields[key]), quality: resolutions(key)})
		}
	} else {
		data, err := a.get(ctx, "https://m.weibo.cn/statuses/show", url.Values{"id": {id}},
			http.Header{"Referer": {"https://m.weibo.cn/"}})
		if err != nil {
			return nil, err
		}
		if data.str("ok") != "1" {
			return nil, outbound.AuthRequired
		}
		post := data.at("data")
		if expected := first(post.str("id"), post.str("mid")); expected != "" && expected != id && post.str("bid") != id {
			return nil, outbound.NoVideo
		}
		node = post.at("page_info")
		if typ := node.str("type"); typ != "" && typ != "video" {
			return nil, noVideoType
		}
		for _, field := range []string{"mp4_1080p_mp4", "mp4_hd_url", "mp4_sd_url", "stream_url_hd", "stream_url"} {
			items = append(items, candidate{raw: node.str("media_info", field), quality: resolutions(field)})
		}
	}
	media := a.video(id, first(node.str("title"), node.str("page_title")), node.str("description"),
		first(node.str("cover_image"), node.str("page_pic", "url")),
		a.author(node.str("author_id"), node.str("author"), node.str("avatar")), node.number("duration_time"), 0, items...)
	if media != nil {
		return media, nil
	}
	return nil, outbound.NoVideo
}

var toutiaoPath = regexp.MustCompile(`^/(?:(?:video|group)/|i)([1-9][0-9]{10,19})/?$`)
var ixiguaPath = regexp.MustCompile(`^/([1-9][0-9]{10,19})/?$`)
var ixiguaSharePath = regexp.MustCompile(`^/xg/video/([1-9][0-9]{10,19})/?$`)
var toutiaoProfile = profile{
	code: "toutiao", pageHosts: []string{"toutiao.com", "www.toutiao.com", "m.toutiao.com", "toutiaoimg.com", "www.toutiaoimg.com", "www.ixigua.com", "ixigua.com", "v.ixigua.com", "m.ixigua.com", "www.iesdouyin.com"},
	cdn: []string{"toutiao.com", "toutiaoimg.com", "ixigua.com", "ixiguavideo.com", "bdxiguaimg.com", "pstatp.com", "byteimg.com", "bytecdn.cn", "bytedance.com", "ixgvideo.com", "snssdk.com", "xycdn.com"},
	id: func(u *url.URL) string {
		host := strings.ToLower(u.Hostname())
		if host == "www.iesdouyin.com" {
			return pathID(u, ixiguaSharePath)
		}
		if host == "ixigua.com" || strings.HasSuffix(host, ".ixigua.com") {
			return first(pathID(u, ixiguaPath), pathID(u, toutiaoPath))
		}
		return pathID(u, toutiaoPath)
	}, short: func(u *url.URL) bool {
		return shortOn(u, "v.ixigua.com") || (u.Hostname() == "m.toutiao.com" || u.Hostname() == "www.toutiao.com") && strings.HasPrefix(u.Path, "/is/") && shareID.MatchString(strings.Trim(strings.TrimPrefix(u.Path, "/is/"), "/"))
	},
	readWorkPage: true, provider: toutiao,
}

func toutiao(_ context.Context, a *adapter, _ *url.URL, doc page, id string) (*domain.Media, error) {
	data := doc.state("RENDER_DATA")
	root := data.at("data")
	if reported := root.str("itemId"); reported != "" && reported != id {
		return nil, outbound.NoVideo
	}
	video := root.at("initialVideo")
	items := video.at("videoPlayInfo", "video_list")
	var rawItems []any
	if list := items.values(); list != nil {
		rawItems = list
	} else {
		for _, key := range slices.Sorted(maps.Keys(items.fields())) {
			rawItems = append(rawItems, items.fields()[key])
		}
	}
	var sources []candidate
	for _, raw := range rawItems {
		item := asObj(raw)
		value := item.str("main_url")
		if !strings.HasPrefix(value, "http") && !strings.HasPrefix(value, "//") {
			decoded, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				decoded, _ = base64.RawStdEncoding.DecodeString(value)
			}
			value = string(decoded)
		}
		sources = append(sources, candidate{raw: value, quality: resolutions(first(item.str("definition"), item.str("quality"))), codec: item.str("codec_type")})
	}
	user := video.at("itemCell", "userInfo")
	media := a.video(id, video.str("title"), video.str("description"), video.str("coverUrl"),
		a.author(user.str("userID"), user.str("name"), user.str("avatarURL")), video.number("duration"), 0, sources...)
	if media != nil {
		return media, nil
	}
	return nil, outbound.NoVideo
}
