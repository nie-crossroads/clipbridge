package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
	"github.com/nie-crossroads/clipbridge/internal/security"
	"github.com/nie-crossroads/clipbridge/internal/service"
)

const cookieName = "cb_session"

type server struct {
	parse    *service.ParseService
	sessions *service.Sessions
	store    *mysql.Store
	secure   bool
	limit    service.Limiter
}

type contextKey uint8

const (
	requestKey contextKey = iota
	sessionKey
)

func New(parse *service.ParseService, sessions *service.Sessions, assets fs.FS, secure bool) http.Handler {
	s := &server{parse: parse, sessions: sessions, store: parse.Store, secure: secure}
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := security.RequestID()
			w.Header().Set("X-Request-ID", id)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("X-Frame-Options", "DENY")
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestKey, id)))
		})
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { respond(w, r, map[string]string{"status": "ok"}, nil) })
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if s.store.Ready(ctx) != nil {
			respond(w, r, nil, domain.Unavailable)
			return
		}
		respond(w, r, map[string]string{"status": "ready"}, nil)
	})
	r.Route("/api", func(api chi.Router) {
		api.NotFound(notFound)
		api.MethodNotAllowed(methodNotAllowed)
		api.Get("/web/platforms", s.platforms)
		api.Post("/web/parse", s.parseLink)
		api.Post("/admin/login", s.login)
		api.Group(func(admin chi.Router) {
			admin.Use(s.requireSession)
			admin.Get("/admin/session", func(w http.ResponseWriter, r *http.Request) { respond(w, r, currentSession(r), nil) })
			admin.Post("/admin/logout", s.logout)
			admin.Post("/admin/parse", s.parseLink)
			admin.Get("/admin/platforms", s.platforms)
			admin.Post("/admin/platforms/{code}", s.updatePlatform)
			admin.Get("/admin/dashboard", s.dashboard)
			admin.Get("/admin/requests", s.records)
			admin.Get("/admin/audit", s.audits)
			admin.Get("/admin/audit/actors", s.actors)
			admin.Get("/admin/keys", s.keys)
			admin.Post("/admin/keys", s.createKey)
			admin.Post("/admin/keys/{id}/action", s.changeKey)
			admin.Get("/admin/keys/{id}/usage", s.keyUsage)
			admin.Get("/admin/settings", s.settings)
			admin.Post("/admin/settings", s.saveSettings)
			admin.Post("/admin/settings/clear", s.clearData)
		})
	})
	static := http.FileServerFS(assets)
	r.Handle("/assets/*", static)
	r.Handle("/media/*", static)
	r.Handle("/logo-sky.svg", static)
	index := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w, r)
			return
		}
		page, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			respond(w, r, nil, domain.Unavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if r.Method == http.MethodGet {
			_, _ = w.Write(page)
		}
	}
	for _, path := range []string{"/", "/developers", "/admin", "/admin/login", "/admin/dashboard", "/admin/parse-test", "/admin/platforms", "/admin/keys", "/admin/requests", "/admin/settings", "/admin/audit"} {
		r.Handle(path, http.HandlerFunc(index))
	}
	r.NotFound(notFound)
	r.MethodNotAllowed(methodNotAllowed)
	return r
}

func requestID(r *http.Request) string { return r.Context().Value(requestKey).(string) }
func currentSession(r *http.Request) *service.Session {
	value, _ := r.Context().Value(sessionKey).(*service.Session)
	return value
}

func respond(w http.ResponseWriter, r *http.Request, data any, err error) {
	payload := struct {
		RequestID string          `json:"request_id"`
		Data      any             `json:"data,omitempty"`
		Error     *domain.Problem `json:"error,omitempty"`
	}{RequestID: requestID(r), Data: data}
	status := 200
	if err != nil {
		if !errors.As(err, &payload.Error) {
			payload.Error = domain.Unavailable
		}
		status = payload.Error.Status
		payload.Data = nil
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func notFound(w http.ResponseWriter, r *http.Request) {
	respond(w, r, nil, &domain.Problem{Status: 404, Code: "NOT_FOUND", Message: "请求的地址不存在。"})
}
func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	respond(w, r, nil, &domain.Problem{Status: 405, Code: "METHOD_NOT_ALLOWED", Message: "请求方法不受支持。"})
}

func readJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(new(any)) != io.EOF {
		respond(w, r, nil, domain.InvalidInput)
		return false
	}
	return true
}

func client(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}
	return r.RemoteAddr
}

func (s *server) parseLink(w http.ResponseWriter, r *http.Request) {
	session := currentSession(r)
	channel := "web"
	var adminID *uint64
	if session != nil {
		channel, adminID = "admin", &session.Admin.ID
	}
	limit, window := service.WebRate, 60
	if channel == "web" {
		settings, _ := s.parse.Settings()
		limit, window = settings.WebRate, settings.WebWindowSeconds
	}
	if !s.limit.Allow("parse:"+channel+":"+strconv.Itoa(limit)+":"+strconv.Itoa(window)+":"+client(r), limit, time.Duration(window)*time.Second) {
		respond(w, r, nil, service.RateLimit)
		return
	}
	var input struct {
		URL      string `json:"url"`
		Platform string `json:"platform"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	result, err := s.parse.Parse(r.Context(), requestID(r), channel, adminID, input.URL, input.Platform, maskedSource(r))
	respond(w, r, result, err)
}
func (s *server) platforms(w http.ResponseWriter, r *http.Request) {
	if currentSession(r) == nil && !s.limit.Allow("platforms:"+client(r), 60, time.Minute) {
		respond(w, r, nil, service.RateLimit)
		return
	}
	items, err := s.parse.Platforms(r.Context())
	if currentSession(r) == nil {
		for index := range items {
			items[index].TimeoutSeconds = 0
		}
	}
	respond(w, r, items, err)
}

func (s *server) sessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/api/admin", MaxAge: maxAge,
		HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode})
}
func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		respond(w, r, nil, domain.InvalidInput)
		return
	}
	if !s.limit.Allow("login:"+client(r), 8, time.Minute) {
		respond(w, r, nil, service.RateLimit)
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	result, err := s.sessions.Login(r.Context(), strings.TrimSpace(input.Username), input.Password)
	if err == nil {
		s.sessionCookie(w, result.Token, 0)
	}
	respond(w, r, result, err)
}
func (s *server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			respond(w, r, nil, service.Unauthorized)
			return
		}
		session, err := s.sessions.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		if r.Method != http.MethodGet && !session.ValidateCSRF(r.Header.Get("X-CSRF-Token")) {
			respond(w, r, nil, service.InvalidCSRF)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey, &session)))
	})
}
func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	err := s.store.DeleteSession(ctx, currentSession(r).Hash)
	if err == nil {
		s.sessionCookie(w, "", -1)
	}
	respond(w, r, map[string]bool{"ok": true}, err)
}
func (s *server) updatePlatform(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if s.parse.Registry[code] == nil {
		respond(w, r, nil, &domain.Problem{Status: 400, Code: "PLATFORM_UNAVAILABLE", Message: "该平台适配器尚未实现。"})
		return
	}
	var input mysql.PlatformPatch
	if !readJSON(w, r, &input) {
		return
	}
	if input.Enabled == nil && input.TimeoutSeconds == nil {
		respond(w, r, nil, domain.InvalidInput)
		return
	}
	if input.TimeoutSeconds != nil && !service.ValidPlatformTimeout(*input.TimeoutSeconds) {
		respond(w, r, nil, &domain.Problem{Status: 400, Code: "INVALID_TIMEOUT", Message: "平台超时须为 1–25 秒。"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	err := s.store.SetPlatform(ctx, code, input, currentSession(r).Admin.ID, requestID(r))
	respond(w, r, input, err)
}

func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	duration := 24 * time.Hour
	switch r.URL.Query().Get("range") {
	case "7d":
		duration = 7 * 24 * time.Hour
	case "14d":
		duration = 14 * 24 * time.Hour
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	stats, err := s.store.Stats(ctx, time.Now().UTC().Add(-duration))
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	stats.HeapBytes = memory.HeapAlloc
	stats.RSSBytes = processRSS()
	respond(w, r, stats, err)
}

func pageOffset(r *http.Request) int {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 || page > 1000 {
		return 0
	}
	return (page - 1) * 25
}
func pageData[T any](items []T) any {
	more := len(items) > 25
	if more {
		items = items[:25]
	}
	return map[string]any{"items": items, "has_more": more}
}
