package service

import (
	"sync"
	"time"
)

const WebRate = 12
const ParseConcurrency = 4

type counter struct {
	expires time.Time
	hits    int
}

type Limiter struct {
	mutex     sync.Mutex
	entries   map[string]counter
	nextSweep time.Time
}

func (l *Limiter) Allow(key string, max int, window time.Duration) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()
	if l.entries == nil {
		l.entries = make(map[string]counter)
	}
	if now.After(l.nextSweep) {
		for name, value := range l.entries {
			if !now.Before(value.expires) {
				delete(l.entries, name)
			}
		}
		l.nextSweep = now.Add(time.Minute)
	}
	entry, exists := l.entries[key]
	if !exists && len(l.entries) >= 4096 {
		return false
	}
	if !now.Before(entry.expires) {
		entry = counter{expires: now.Add(window)}
	}
	if entry.hits >= max {
		return false
	}
	entry.hits++
	l.entries[key] = entry
	return true
}
