package share

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"golang.org/x/net/html"
)

type obj struct{ value any }

func asObj(value any) obj { return obj{value} }
func (o obj) at(keys ...string) obj {
	value := o.value
	for _, key := range keys {
		m, _ := value.(map[string]any)
		value = m[key]
	}
	return obj{value}
}
func (o obj) str(keys ...string) string     { return text(o.at(keys...).value) }
func (o obj) number(keys ...string) float64 { n, _ := strconv.ParseFloat(o.str(keys...), 64); return n }
func (o obj) fields() map[string]any        { m, _ := o.value.(map[string]any); return m }
func (o obj) values() []any                 { a, _ := o.value.([]any); return a }
func text(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return string(v)
	}
	return ""
}
func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func decode(raw []byte) obj {
	if len(raw) == 0 {
		return obj{}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil || decoder.Decode(new(any)) != io.EOF {
		return obj{}
	}
	return obj{value}
}
func listString(values obj) string {
	for _, value := range values.values() {
		if s := first(text(value), asObj(value).str("url")); s != "" {
			return s
		}
	}
	return ""
}

type page struct {
	meta       map[string]string
	scripts    map[string][]byte
	structured []obj
	title      string
}

var assignment = regexp.MustCompile(`(?:window\.)?(__INITIAL_STATE__|__APOLLO_STATE__|INIT_STATE|_ROUTER_DATA|__NEXT_DATA__)\s*=\s*`)

func readPage(raw []byte) page {
	result := page{meta: map[string]string{}, scripts: map[string][]byte{}}
	tokenizer := html.NewTokenizer(bytes.NewReader(raw))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return result
		case html.StartTagToken, html.SelfClosingTagToken:
			tag := tokenizer.Token()
			var id, kind, property, content string
			for _, attr := range tag.Attr {
				switch attr.Key {
				case "id":
					id = attr.Val
				case "type":
					kind = attr.Val
				case "name", "property", "itemprop":
					property = attr.Val
				case "content":
					content = attr.Val
				}
			}
			switch tag.Data {
			case "meta":
				if property != "" && result.meta[property] == "" {
					result.meta[property] = content
				}
			case "title":
				if tokenizer.Next() == html.TextToken {
					result.title = strings.TrimSpace(string(tokenizer.Text()))
				}
			case "script":
				if tokenizer.Next() != html.TextToken {
					continue
				}
				data := bytes.TrimSpace(tokenizer.Text())
				if id == "RENDER_DATA" || id == "__NEXT_DATA__" || id == "SIGI_STATE" {
					result.scripts[id] = bytes.Clone(data)
				}
				if kind == "application/ld+json" {
					if data := decode(data); data.value != nil {
						result.structured = append(result.structured, data)
					}
				}
				if found := assignment.FindSubmatchIndex(data); len(found) == 4 {
					name := string(data[found[2]:found[3]])
					result.scripts[name] = bytes.Clone(bytes.TrimSuffix(bytes.TrimSpace(data[found[1]:]), []byte(";")))
				}
			}
		}
	}
}

func undefinedAsNull(input []byte) []byte {
	var output bytes.Buffer
	quoted, escaped := false, false
	for i := 0; i < len(input); i++ {
		c := input[i]
		if quoted {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				quoted = false
			}
		} else if c == '"' {
			quoted = true
		} else if bytes.HasPrefix(input[i:], []byte("undefined")) {
			output.WriteString("null")
			i += 8
			continue
		}
		output.WriteByte(c)
	}
	return output.Bytes()
}
func (p page) state(key string) obj {
	body := p.scripts[key]
	if data := decode(body); data.value != nil {
		return data
	}
	if key == "RENDER_DATA" {
		if decoded, err := url.PathUnescape(string(body)); err == nil && decoded != string(body) {
			body = []byte(decoded)
		}
	}
	return decode(undefinedAsNull(body))
}
func find(node obj, predicate func(obj) bool) obj {
	return findDepth(node, predicate, 0)
}
func findDepth(node obj, predicate func(obj) bool, depth int) obj {
	if depth > 30 {
		return obj{}
	}
	if fields := node.fields(); fields != nil {
		if predicate(node) {
			return node
		}
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			if hit := findDepth(asObj(fields[key]), predicate, depth+1); hit.value != nil {
				return hit
			}
		}
	}
	for _, entry := range node.values() {
		if hit := findDepth(asObj(entry), predicate, depth+1); hit.value != nil {
			return hit
		}
	}
	return obj{}
}

func (a *adapter) standard(p page, id string) *domain.Media {
	title := first(p.meta["og:title"], p.title)
	cover := first(p.meta["og:image:secure_url"], p.meta["og:image"])
	desc := p.meta["og:description"]
	for _, node := range p.structured {
		entry := find(node, func(o obj) bool { return o.str("@type") == "VideoObject" && o.str("contentUrl") != "" })
		if raw := entry.str("contentUrl"); raw != "" {
			media := a.video(id, first(entry.str("name"), title), first(entry.str("description"), desc), first(entry.str("thumbnailUrl"), cover),
				nil, 0, 0, candidate{raw: raw})
			if media != nil {
				return media
			}
		}
	}
	var items []candidate
	for _, key := range []string{"og:video:secure_url", "og:video:url", "og:video", "twitter:player:stream"} {
		raw := p.meta[key]
		format := p.meta["og:video:type"]
		if raw != "" && (strings.HasPrefix(format, "video/") || strings.Contains(strings.ToLower(raw), ".mp4")) {
			items = append(items, candidate{raw: raw})
		}
	}
	return a.video(id, title, desc, cover, nil, 0, 0, items...)
}
