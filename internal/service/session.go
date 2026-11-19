package service

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
	"github.com/nie-crossroads/clipbridge/internal/security"
	"golang.org/x/crypto/bcrypt"
)

var Unauthorized = &domain.Problem{Status: 401, Code: "UNAUTHORIZED", Message: "请登录管理后台。"}
var BadCredentials = &domain.Problem{Status: 401, Code: "BAD_CREDENTIALS", Message: "用户名或密码错误。"}
var InvalidCSRF = &domain.Problem{Status: 403, Code: "INVALID_CSRF", Message: "安全令牌无效，请刷新后重试。"}

type Session struct {
	Admin    mysql.Admin `json:"-"`
	Token    string      `json:"-"`
	Hash     []byte      `json:"-"`
	CSRFHash []byte      `json:"-"`
	Name     string      `json:"name"`
	CSRF     string      `json:"csrf_token"`
}

type Sessions struct {
	Store     *mysql.Store
	dummyHash []byte
}

func NewSessions(store *mysql.Store) (*Sessions, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(security.NewToken()), bcrypt.DefaultCost)
	return &Sessions{Store: store, dummyHash: hash}, err
}

func sessionFor(admin mysql.Admin, token string) Session {
	csrf := base64.RawURLEncoding.EncodeToString(security.Digest("clipbridge-csrf:" + token))
	return Session{Admin: admin, Token: token, Hash: security.Digest(token), CSRFHash: security.Digest(csrf), Name: admin.Name, CSRF: csrf}
}

func (s *Sessions) Login(ctx context.Context, username, password string) (Session, error) {
	if username == "" || len(username) > 256 || password == "" || len(password) > 72 {
		return Session{}, BadCredentials
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	admin, err := s.Store.FindAdmin(ctx, username)
	hash := s.dummyHash
	if err == nil {
		hash = []byte(admin.PasswordHash)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Session{}, domain.Unavailable
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil || err != nil {
		return Session{}, BadCredentials
	}
	result := sessionFor(admin, security.NewToken())
	if err := s.Store.CreateSession(ctx, admin.ID, result.Hash, result.CSRFHash, time.Now().UTC().Add(12*time.Hour)); err != nil {
		return Session{}, domain.Unavailable
	}
	return result, nil
}

func (s *Sessions) Authenticate(ctx context.Context, token string) (Session, error) {
	if len(token) != 43 {
		return Session{}, Unauthorized
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	admin, csrfHash, err := s.Store.Session(ctx, security.Digest(token))
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, Unauthorized
	}
	if err != nil {
		return Session{}, domain.Unavailable
	}
	result := sessionFor(admin, token)
	result.CSRFHash = csrfHash
	return result, nil
}

func (s Session) ValidateCSRF(token string) bool {
	return subtle.ConstantTimeCompare(s.CSRFHash, security.Digest(token)) == 1
}
