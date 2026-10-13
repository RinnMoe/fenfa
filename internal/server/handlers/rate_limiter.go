package handlers

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const rateLimiterMaxKeys = 10000

// rateLimiter is a small in-process sliding-window limiter. It is used for
// unauthenticated endpoints where keeping state in the database would add
// more cost than protection.
type rateLimiter struct {
	mu          sync.Mutex
	entries     map[string][]time.Time
	lastCleanup time.Time
	limit       int
	window      time.Duration
	maxKeys     int
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{
		entries:     make(map[string][]time.Time),
		lastCleanup: time.Now(),
		limit:       limit,
		window:      window,
		maxKeys:     rateLimiterMaxKeys,
	}
}

func (limiter *rateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	cutoff := now.Add(-limiter.window)
	if now.Sub(limiter.lastCleanup) >= time.Minute || len(limiter.entries) >= limiter.maxKeys {
		for entryKey, timestamps := range limiter.entries {
			hasRecent := false
			for _, timestamp := range timestamps {
				if timestamp.After(cutoff) {
					hasRecent = true
					break
				}
			}
			if !hasRecent {
				delete(limiter.entries, entryKey)
			}
		}
		limiter.lastCleanup = now
	}

	previous := limiter.entries[key]
	wasTracked := len(previous) > 0
	kept := previous[:0]
	for _, timestamp := range previous {
		if timestamp.After(cutoff) {
			kept = append(kept, timestamp)
		}
	}
	if len(kept) == 0 {
		delete(limiter.entries, key)
		wasTracked = false
	} else {
		limiter.entries[key] = kept
	}

	if len(kept) >= limiter.limit {
		oldest := kept[0]
		for _, timestamp := range kept[1:] {
			if timestamp.Before(oldest) {
				oldest = timestamp
			}
		}
		retryAfter := oldest.Add(limiter.window).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter
	}
	if !wasTracked && len(limiter.entries) >= limiter.maxKeys {
		return false, limiter.window
	}

	limiter.entries[key] = append(kept, now)
	return true, 0
}

func (limiter *rateLimiter) reset(key string) {
	if key == "" {
		return
	}
	limiter.mu.Lock()
	delete(limiter.entries, key)
	limiter.mu.Unlock()
}

func writeRateLimitResponse(c *gin.Context, message string, retryAfter time.Duration) {
	seconds := int((retryAfter + time.Second - 1) / time.Second)
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.JSON(http.StatusTooManyRequests, gin.H{
		"ok": false,
		"error": gin.H{
			"code":    "RATE_LIMITED",
			"message": message,
		},
	})
}
