package httptransport

import (
	"context"
	"net/http"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
	"github.com/nie-crossroads/clipbridge/internal/service"
)

type settingsResponse struct {
	mysql.RuntimeSettings
	RetentionScheduled bool `json:"retention_scheduled"`
	ProxyEnabled       bool `json:"proxy_enabled"`
	ProxyConcurrency   int  `json:"proxy_concurrency"`
}

func (s *server) settings(w http.ResponseWriter, r *http.Request) {
	values, saved := s.parse.Settings()
	respond(w, r, settingsResponse{RuntimeSettings: values, RetentionScheduled: saved}, nil)
}

func (s *server) saveSettings(w http.ResponseWriter, r *http.Request) {
	var values mysql.RuntimeSettings
	if !readJSON(w, r, &values) {
		return
	}
	if !service.ValidSettings(values) {
		respond(w, r, nil, &domain.Problem{Status: 400, Code: "INVALID_SETTINGS", Message: "存在超出允许范围的设置，请检查各项输入。"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err := s.parse.SaveSettings(ctx, values, currentSession(r).Admin.ID, requestID(r))
	respond(w, r, settingsResponse{RuntimeSettings: values, RetentionScheduled: true}, err)
}

func (s *server) clearData(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Action string `json:"action"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	if input.Action != "clear_requests" && input.Action != "clear_cache" {
		respond(w, r, nil, domain.InvalidInput)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	err := s.store.ClearRuntimeData(ctx, input.Action, currentSession(r).Admin.ID, requestID(r))
	if err == nil && input.Action == "clear_cache" {
		s.parse.ClearCache()
	}
	respond(w, r, map[string]bool{"ok": true}, err)
}
