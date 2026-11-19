package service

import (
	"encoding/json"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
)

type cacheEntry struct {
	media   *domain.Media
	expires time.Time
	size    int
}

func (s *ParseService) ClearCache() {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	clear(s.cache)
	s.cacheBytes = 0
	s.cacheEpoch++
}

func (s *ParseService) cached(key [32]byte) (*domain.Media, uint64) {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	if entry, ok := s.cache[key]; ok {
		if time.Now().Before(entry.expires) {
			return entry.media, s.cacheEpoch
		}
		delete(s.cache, key)
		s.cacheBytes -= entry.size
	}
	return nil, s.cacheEpoch
}

func (s *ParseService) cacheResult(key [32]byte, epoch uint64, media *domain.Media, settings mysql.RuntimeSettings) {
	expires := time.Now().Add(time.Duration(settings.CacheTTLSeconds) * time.Second)
	check := func(source *domain.Source) {
		if source == nil {
			return
		}
		if source.ExpiresAt != nil && source.ExpiresAt.Before(expires) {
			expires = *source.ExpiresAt
		}
		for _, alt := range source.Alternatives {
			if alt.ExpiresAt != nil && alt.ExpiresAt.Before(expires) {
				expires = *alt.ExpiresAt
			}
		}
	}
	check(media.Video)
	check(media.Music)
	for i := range media.Videos {
		check(media.Videos[i].Video)
	}
	for i := range media.LivePhotos {
		check(&media.LivePhotos[i].Video)
	}
	data, err := json.Marshal(media)
	maxBytes := settings.CacheMiB * 1048576
	if err != nil || len(data) > maxBytes || !expires.After(time.Now()) {
		return
	}

	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	if epoch != s.cacheEpoch {
		return
	}
	now := time.Now()
	for key, entry := range s.cache {
		if !now.Before(entry.expires) {
			delete(s.cache, key)
			s.cacheBytes -= entry.size
		}
	}
	if old, ok := s.cache[key]; ok {
		delete(s.cache, key)
		s.cacheBytes -= old.size
	}
	for len(s.cache) >= settings.CacheItems || s.cacheBytes+len(data) > maxBytes {
		var oldest [32]byte
		var until time.Time
		for key, entry := range s.cache {
			if until.IsZero() || entry.expires.Before(until) {
				oldest, until = key, entry.expires
			}
		}
		s.cacheBytes -= s.cache[oldest].size
		delete(s.cache, oldest)
	}
	s.cache[key] = cacheEntry{media: media, expires: expires, size: len(data)}
	s.cacheBytes += len(data)
}
