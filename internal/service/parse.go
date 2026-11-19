package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/parser"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
	"github.com/nie-crossroads/clipbridge/internal/security"
)

type ParseService struct {
	Registry      parser.Registry
	Store         *mysql.Store
	active        atomic.Int64
	settings      atomic.Pointer[settingsSnapshot]
	settingsWrite sync.Mutex
	cacheLock     sync.Mutex
	cache         map[[32]byte]cacheEntry
	cacheBytes    int
	cacheEpoch    uint64
}

func NewParseService(registry parser.Registry, store *mysql.Store) *ParseService {
	service := &ParseService{Registry: registry, Store: store, cache: make(map[[32]byte]cacheEntry)}
	service.settings.Store(&settingsSnapshot{Values: DefaultSettings()})
	return service
}

func (s *ParseService) Platforms(ctx context.Context) ([]parser.Platform, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	items := s.Registry.Platforms()
	if err := s.Store.Platforms(ctx, items); err != nil {
		return nil, domain.Unavailable
	}
	for index := range items {
		if items[index].Implemented && !ValidPlatformTimeout(items[index].TimeoutSeconds) {
			items[index].TimeoutSeconds = DefaultPlatformTimeout
		}
	}
	return items, nil
}

var RateLimit = &domain.Problem{Status: 429, Code: "RATE_LIMITED", Message: "请求过于频繁，请稍后重试。", Retryable: true}

const DefaultPlatformTimeout = 25

func ValidPlatformTimeout(seconds int) bool {
	return seconds >= 1 && seconds <= DefaultPlatformTimeout
}

func (s *ParseService) Parse(ctx context.Context, requestID, channel string, adminID *uint64, text, filter string, sourceIP *string) (result *domain.Media, err error) {
	started := time.Now()
	adapter, address, platform, err := s.Registry.Select(text, filter)
	record := mysql.Record{ID: requestID, Channel: channel, AdminID: adminID, Platform: platform, InputHash: security.Digest(text), SourceIP: sourceIP}
	defer func() {
		record.Status = 200
		if err != nil {
			var problem *domain.Problem
			if !errors.As(err, &problem) {
				problem = domain.Unavailable
			}
			record.Status, record.ErrorCode = problem.Status, &problem.Code
		}
		if result != nil {
			record.ItemID, record.MediaType = result.ItemID, result.Type
		}
		record.DurationMS = time.Since(started).Milliseconds()
		dbctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if writeErr := s.Store.SaveRecord(dbctx, record); writeErr != nil {
			slog.Warn("parse record could not be persisted", "request_id", requestID)
		}
		if channel == "admin" && record.Accepted == 1 && ctx.Err() == nil {
			status := "success"
			if err != nil {
				status = "failed"
			}
			if writeErr := s.Store.SavePlatformTest(dbctx, platform, status, record.ErrorCode); writeErr != nil {
				slog.Warn("platform test could not be persisted", "request_id", requestID)
			}
		}
	}()
	if err != nil {
		return nil, err
	}
	platforms, err := s.Platforms(ctx)
	if err != nil {
		return nil, err
	}
	timeout := DefaultPlatformTimeout
	var configVersion uint64
	for _, item := range platforms {
		if item.Code != platform {
			continue
		}
		if !item.Enabled && channel != "admin" {
			return nil, &domain.Problem{Status: 403, Code: "PLATFORM_DISABLED", Message: "该平台当前未开放解析。"}
		}
		timeout = item.TimeoutSeconds
		configVersion = item.ConfigVersion
		break
	}
	if ctx.Err() != nil {
		return nil, domain.Timeout
	}
	settings, _ := s.Settings()
	active := s.active.Add(1)
	defer s.active.Add(-1)
	if active > int64(settings.ParseConcurrency) {
		return nil, RateLimit
	}
	record.Accepted = 1
	cacheHit := false
	record.CacheHit = &cacheHit
	var key [32]byte
	var epoch uint64
	if channel == "web" {
		address.Fragment, address.RawFragment = "", ""
		key = sha256.Sum256([]byte(platform + ":" + strconv.FormatUint(configVersion, 10) + ":" + address.String()))
		var cached *domain.Media
		cached, epoch = s.cached(key)
		if cached != nil {
			cacheHit = true
			return cached, nil
		}
	}
	upstreamCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	result, err = adapter.Parse(upstreamCtx, address)
	if err == nil && result != nil && channel == "web" {
		settings, _ := s.Settings()
		s.cacheResult(key, epoch, result, settings)
	}
	return result, err
}
