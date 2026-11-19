package domain

import "time"

type Source struct {
	URL             string     `json:"url"`
	Quality         string     `json:"quality"`
	WatermarkStatus string     `json:"watermark_status"`
	MIMEType        string     `json:"mime_type"`
	Codec           *string    `json:"codec"`
	Width           *int       `json:"width"`
	Height          *int       `json:"height"`
	Bitrate         *int64     `json:"bitrate"`
	SizeBytes       *int64     `json:"size_bytes"`
	ExpiresAt       *time.Time `json:"expires_at"`
	Alternatives    []Source   `json:"alternatives,omitempty"`
}

type Author struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type Image struct {
	URL string `json:"url"`
}

type Video struct {
	ItemID string  `json:"item_id"`
	Title  string  `json:"title"`
	Video  *Source `json:"video"`
}

type LivePhoto struct {
	Image Image  `json:"image"`
	Video Source `json:"video"`
}

type Media struct {
	Platform        string         `json:"platform"`
	ItemID          string         `json:"item_id"`
	Type            string         `json:"type"`
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	Author          *Author        `json:"author"`
	Cover           string         `json:"cover"`
	DurationSeconds *float64       `json:"duration_seconds"`
	PublishedAt     *time.Time     `json:"published_at"`
	Video           *Source        `json:"video"`
	Images          []Image        `json:"images"`
	LivePhotos      []LivePhoto    `json:"live_photos"`
	Music           *Source        `json:"music"`
	Videos          []Video        `json:"videos"`
	Extra           map[string]any `json:"extra"`
	Warnings        []string       `json:"warnings"`
}

func NewMedia(platform, itemID, kind string) Media {
	return Media{
		Platform: platform, ItemID: itemID, Type: kind,
		Images: []Image{}, LivePhotos: []LivePhoto{}, Videos: []Video{}, Warnings: []string{}, Extra: map[string]any{},
	}
}

type Problem struct {
	Status    int    `json:"-"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (p *Problem) Error() string { return p.Code }

var Unavailable = &Problem{503, "SERVICE_UNAVAILABLE", "服务暂时不可用，请稍后重试。", true}
var InvalidInput = &Problem{400, "INVALID_INPUT", "请粘贴一条有效的作品分享链接。", false}
var Upstream = &Problem{502, "UPSTREAM_ERROR", "平台暂时未返回可用媒体，请稍后重试。", true}
var Timeout = &Problem{504, "UPSTREAM_TIMEOUT", "平台响应超时，请重试。", true}
