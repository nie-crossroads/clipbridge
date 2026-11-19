package httptransport

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
	"github.com/nie-crossroads/clipbridge/internal/security"
)

var keyInvalid = &domain.Problem{Status: 400, Code: "INVALID_KEY_SETTINGS", Message: "请检查名称、渠道、权限、平台白名单、额度和有效期。"}
var keyMissing = &domain.Problem{Status: 404, Code: "KEY_NOT_FOUND", Message: "密钥不存在。"}
var keyConflict = &domain.Problem{Status: 409, Code: "KEY_STATE_CONFLICT", Message: "当前密钥状态或确认名称不允许该操作。请刷新列表后重试。"}

func generateKey(channel string) (string, []byte, string) {
	prefix := "cb_api_"
	if channel == "mcp" {
		prefix = "cb_mcp_"
	}
	token := security.NewToken()
	secret := prefix + token
	return secret, security.Digest(secret), prefix + "********" + token[len(token)-4:]
}

func (s *server) keys(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	items, err := s.store.Keys(ctx)
	respond(w, r, items, err)
}

func (s *server) createKey(w http.ResponseWriter, r *http.Request) {
	var input mysql.KeySettings
	if !readJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 40 ||
		(input.Channel != "rest" && input.Channel != "mcp") ||
		input.Rate > 100000 || input.Daily > 1000000000 || input.Concurrency < 1 || input.Concurrency > 100 ||
		input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) ||
		!validValues(input.Scopes, []string{"media:parse", "media:batch", "platforms:read", "media:proxy"}) || len(input.Scopes) == 0 {
		respond(w, r, nil, keyInvalid)
		return
	}
	var platforms []string
	for _, item := range s.parse.Registry.Platforms() {
		platforms = append(platforms, item.Code)
	}
	if !validValues(input.Platforms, platforms) {
		respond(w, r, nil, keyInvalid)
		return
	}
	if input.Platforms == nil {
		input.Platforms = []string{}
	}
	if slices.Contains(input.Scopes, "media:batch") && !slices.Contains(input.Scopes, "media:parse") {
		input.Scopes = append(input.Scopes, "media:parse")
	}
	if input.ExpiresAt != nil {
		utc := input.ExpiresAt.UTC()
		input.ExpiresAt = &utc
	}
	secret, hash, hint := generateKey(input.Channel)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	err := s.store.CreateKey(ctx, input, hash, hint, currentSession(r).Admin.ID, requestID(r))
	respond(w, r, map[string]string{"key": secret}, err)
}

func validValues(values, allowed []string) bool {
	if len(values) > len(allowed) {
		return false
	}
	for index, value := range values {
		if !slices.Contains(allowed, value) || slices.Contains(values[:index], value) {
			return false
		}
	}
	return true
}

func (s *server) changeKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		respond(w, r, nil, keyMissing)
		return
	}
	var input mysql.KeyAction
	if !readJSON(w, r, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	secret, err := s.store.ChangeKey(ctx, id, input, currentSession(r).Admin.ID, requestID(r), generateKey)
	if errors.Is(err, sql.ErrNoRows) {
		err = keyMissing
	}
	if errors.Is(err, mysql.ErrKeyState) {
		err = keyConflict
	}
	data := map[string]any{"ok": true}
	if secret != "" {
		data["key"] = secret
	}
	respond(w, r, data, err)
}

func (s *server) keyUsage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		respond(w, r, nil, keyMissing)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	date := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format(time.DateOnly)
	usage, err := s.store.KeyUsage(ctx, id, date)
	if errors.Is(err, sql.ErrNoRows) {
		err = keyMissing
	}
	respond(w, r, usage, err)
}
