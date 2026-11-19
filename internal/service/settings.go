package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
)

func DefaultSettings() mysql.RuntimeSettings {
	return mysql.RuntimeSettings{
		WebRate: WebRate, WebWindowSeconds: 60, ParseConcurrency: ParseConcurrency, RequestDays: 7, AuditDays: 90,
		CacheItems: 200, CacheMiB: 16, CacheTTLSeconds: 120,
	}
}

func ValidSettings(s mysql.RuntimeSettings) bool {
	return inRange(s.WebRate, 1, 10000) && inRange(s.WebWindowSeconds, 1, 3600) && inRange(s.ParseConcurrency, 1, 32) &&
		inRange(s.RequestDays, 1, 3650) && inRange(s.AuditDays, 1, 3650) && inRange(s.CacheItems, 1, 2000) &&
		inRange(s.CacheMiB, 1, 128) && inRange(s.CacheTTLSeconds, 1, 120)
}

func inRange(value, min, max int) bool { return value >= min && value <= max }

type settingsSnapshot struct {
	Values mysql.RuntimeSettings
	Saved  bool
}

func (s *ParseService) Settings() (mysql.RuntimeSettings, bool) {
	data := s.settings.Load()
	return data.Values, data.Saved
}

func (s *ParseService) LoadSettings(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	settings := DefaultSettings()
	saved, err := s.Store.RuntimeSettings(ctx, &settings)
	if err != nil || !ValidSettings(settings) {
		slog.Warn("persisted settings could not be loaded")
		return
	}
	s.settings.Store(&settingsSnapshot{Values: settings, Saved: saved})
}

func (s *ParseService) SaveSettings(ctx context.Context, settings mysql.RuntimeSettings, actor uint64, requestID string) error {
	s.settingsWrite.Lock()
	defer s.settingsWrite.Unlock()
	if err := s.Store.SaveRuntimeSettings(ctx, settings, actor, requestID); err != nil {
		return err
	}
	s.settings.Store(&settingsSnapshot{Values: settings, Saved: true})
	s.ClearCache()
	return nil
}

func (s *ParseService) MaintainLogs(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			settings, saved := s.Settings()
			if !saved {
				continue
			}
			work, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := s.Store.ExpireLogs(work, now.UTC().AddDate(0, 0, -settings.RequestDays), now.UTC().AddDate(0, 0, -settings.AuditDays))
			cancel()
			if err != nil {
				slog.Warn("log retention job failed")
			}
		}
	}
}
