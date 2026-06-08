package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ── Redis rate limiter — drop this file into any Go service ──────────────────
// Limits: per-IP per-minute + per-org per-hour
// Falls back gracefully if Redis is unavailable

var (
	rlClient *goredis.Client
	rlCtx    = context.Background()
)

// Per-endpoint rate limits (requests per minute)
var endpointLimits = map[string]int{
	"/auth/login":    10,  // brute force protection
	"/auth/register": 5,   // very strict
	"/upload/csv":    20,  // file uploads
	"/insights":      30,  // AI calls cost money
	"/health":        300, // generous for health checks
	"/metrics":       300,
}

var (
	globalLimitPerMin  = getEnvInt("RATE_LIMIT_PER_MIN", 60)
	globalLimitPerHour = getEnvInt("RATE_LIMIT_PER_HOUR", 1000)
)

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func initRateLimit() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	opt, err := goredis.ParseURL(redisURL)
	if err != nil {
		log.Printf("WARNING: rate limiter — Redis URL error: %v (disabled)", err)
		return
	}
	rlClient = goredis.NewClient(opt)
	if err := rlClient.Ping(rlCtx).Err(); err != nil {
		log.Printf("WARNING: rate limiter — Redis ping failed: %v (disabled)", err)
		rlClient = nil
		return
	}
	log.Printf("✓ Rate limiter connected to Redis — %d req/min per IP, %d req/hr per org",
		globalLimitPerMin, globalLimitPerHour)
}

// slidingWindow checks a sliding window rate limit using Redis sorted sets
func slidingWindow(key string, limit int, window time.Duration) (allowed bool, remaining int, resetAt time.Time) {
	if rlClient == nil {
		return true, limit, time.Now().Add(window)
	}

	now := time.Now()
	windowStart := now.Add(-window)
	resetAt = now.Add(window)

	pipe := rlClient.Pipeline()
	pipe.ZRemRangeByScore(rlCtx, key, "0", strconv.FormatInt(windowStart.UnixMilli(), 10))
	countCmd := pipe.ZCard(rlCtx, key)
	pipe.ZAdd(rlCtx, key, goredis.Z{
		Score:  float64(now.UnixMilli()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	pipe.Expire(rlCtx, key, window+time.Second)
	pipe.Exec(rlCtx)

	count := int(countCmd.Val())
	remaining = limit - count - 1
	if remaining < 0 {
		remaining = 0
	}
	return count < limit, remaining, resetAt
}

// rateLimitMiddleware wraps an http.Handler with per-IP + per-org rate limiting
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health and metrics
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		orgID := r.Header.Get("X-Org-ID")
		endpoint := r.URL.Path

		// Per-endpoint limit
		limit := globalLimitPerMin
		if epLimit, ok := endpointLimits[endpoint]; ok {
			limit = epLimit
		}

		// Check 1: per-IP per-minute
		ipKey := fmt.Sprintf("rl:ip:%s:%s", ip, endpoint)
		allowed, remaining, resetAt := slidingWindow(ipKey, limit, time.Minute)
		if !allowed {
			rlError(w, "ip_rate_limit_exceeded", limit, 0, resetAt, 60)
			return
		}

		// Set IP rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		// Check 2: per-org per-hour
		if orgID != "" {
			orgKey := fmt.Sprintf("rl:org:%s", orgID)
			orgAllowed, orgRemaining, orgResetAt := slidingWindow(orgKey, globalLimitPerHour, time.Hour)
			if !orgAllowed {
				rlError(w, "org_quota_exceeded", globalLimitPerHour, 0, orgResetAt, 3600)
				return
			}
			w.Header().Set("X-RateLimit-Org-Limit", strconv.Itoa(globalLimitPerHour))
			w.Header().Set("X-RateLimit-Org-Remaining", strconv.Itoa(orgRemaining))
			w.Header().Set("X-RateLimit-Org-Reset", strconv.FormatInt(orgResetAt.Unix(), 10))
		}

		next.ServeHTTP(w, r)
	})
}

func rlError(w http.ResponseWriter, code string, limit, remaining int, resetAt time.Time, retryAfter int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":          false,
		"error":       "rate limit exceeded",
		"code":        code,
		"limit":       limit,
		"remaining":   remaining,
		"reset_at":    resetAt.Unix(),
		"retry_after": retryAfter,
	})
}
