package httptransport

import (
	"context"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
)

var invalidFilters = &domain.Problem{Status: 400, Code: "INVALID_FILTER", Message: "筛选条件无效，请检查时间和筛选参数。"}

func logTimeRange(query url.Values) (*time.Time, *time.Time, bool) {
	now := time.Now().UTC()
	var duration time.Duration
	value := query.Get("range")
	switch value {
	case "", "all":
		return nil, nil, true
	case "1h":
		duration = time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	case "custom":
		start, startErr := time.Parse(time.RFC3339, query.Get("from"))
		end, endErr := time.Parse(time.RFC3339, query.Get("to"))
		if startErr != nil || endErr != nil || !start.Before(end) {
			return nil, nil, false
		}
		start, end = start.UTC(), end.UTC()
		return &start, &end, true
	default:
		return nil, nil, false
	}
	start := now.Add(-duration)
	return &start, nil, true
}

func (s *server) records(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to, valid := logTimeRange(q)
	f := mysql.RecordFilter{Channel: q.Get("channel"), Platform: q.Get("platform"), ErrorCode: q.Get("error_code"),
		Status: q.Get("status"), RequestID: q.Get("request_id"), From: from, To: to}
	if !valid || len(f.Channel) > 16 || len(f.Platform) > 32 || len(f.ErrorCode) > 64 || len(f.RequestID) > 36 ||
		f.Status != "" && f.Status != "success" && f.Status != "failed" {
		respond(w, r, nil, invalidFilters)
		return
	}
	switch key := q.Get("key"); key {
	case "":
	case "web":
		f.Public = true
	default:
		id, err := strconv.ParseUint(key, 10, 64)
		if err != nil || id == 0 {
			respond(w, r, nil, invalidFilters)
			return
		}
		f.KeyID = &id
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	items, err := s.store.Records(ctx, f, pageOffset(r))
	respond(w, r, pageData(items), err)
}

func (s *server) audits(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to, valid := logTimeRange(q)
	f := mysql.AuditFilter{Action: q.Get("action"), Actor: q.Get("actor"), Target: q.Get("target_type"), Object: q.Get("object"), RequestID: q.Get("request_id"), From: from, To: to}
	f.Object = strings.ReplaceAll(f.Object, "•", "*")
	if !valid || len(f.Action) > 64 || len(f.Actor) > 512 || len(f.Target) > 32 || len(f.Object) > 512 || len(f.RequestID) > 36 {
		respond(w, r, nil, invalidFilters)
		return
	}
	// Search by the displayed platform name as well as its code.
	for _, platform := range s.parse.Registry.Platforms() {
		if f.Object == platform.Name {
			f.Object = platform.Code
			break
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	items, err := s.store.Audits(ctx, f, pageOffset(r))
	respond(w, r, pageData(items), err)
}

func (s *server) actors(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	actors, err := s.store.AuditActors(ctx)
	respond(w, r, actors, err)
}

func maskedSource(r *http.Request) *string {
	addr, err := netip.ParseAddr(client(r))
	if err != nil {
		return nil
	}
	addr = addr.Unmap().WithZone("")
	bits := 48
	if addr.Is4() {
		bits = 24
	}
	masked := netip.PrefixFrom(addr, bits).Masked().String()
	return &masked
}
