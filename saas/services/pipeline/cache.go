package main

import (
"net/http"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ── Redis query cache — L6 Caching layer ─────────────────────────────────────
// Caches expensive Python prediction results in Redis
// Cache keys: "cache:predict:forecast:orgID" TTL: 5 min
//             "cache:predict:churn:orgID"    TTL: 5 min
//             "cache:predict:segments:orgID" TTL: 5 min
//             "cache:predict:fraud:orgID"    TTL: 2 min
//             "cache:insights:report:orgID"  TTL: 10 min

var (
	cacheClient *goredis.Client
	cacheCtx    = context.Background()
)

const (
	cacheTTLForecast = 5 * time.Minute
	cacheTTLChurn    = 5 * time.Minute
	cacheTTLSegments = 5 * time.Minute
	cacheTTLFraud    = 2 * time.Minute
	cacheTTLReport   = 10 * time.Minute
	cacheTTLInsight  = 30 * time.Minute
)

func initCache() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := goredis.ParseURL(redisURL)
	if err != nil {
		log.Printf("WARNING: cache — Redis URL error: %v (cache disabled)", err)
		return
	}

	cacheClient = goredis.NewClient(opt)
	if err := cacheClient.Ping(cacheCtx).Err(); err != nil {
		log.Printf("WARNING: cache — Redis ping failed: %v (cache disabled)", err)
		cacheClient = nil
		return
	}
	log.Println("✓ Query cache connected to Redis")
}

// cacheKey builds a namespaced cache key
func cacheKey(category, orgID string) string {
	return fmt.Sprintf("cache:%s:%s", category, orgID)
}

// cacheGet retrieves a cached result — returns nil if miss or cache disabled
func cacheGet(key string) map[string]interface{} {
	if cacheClient == nil {
		return nil
	}

	val, err := cacheClient.Get(cacheCtx, key).Result()
	if err != nil {
		return nil // cache miss
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil
	}

	log.Printf("cache HIT: %s", key)
	return result
}

// cacheSet stores a result in Redis with TTL
func cacheSet(key string, data map[string]interface{}, ttl time.Duration) {
	if cacheClient == nil {
		return
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	if err := cacheClient.Set(cacheCtx, key, bytes, ttl).Err(); err != nil {
		log.Printf("cache SET error: %v", err)
		return
	}

	log.Printf("cache SET: %s TTL=%s", key, ttl)
}

// cacheInvalidate removes a cached result
func cacheInvalidate(pattern string) {
	if cacheClient == nil {
		return
	}

	keys, err := cacheClient.Keys(cacheCtx, pattern).Result()
	if err != nil || len(keys) == 0 {
		return
	}

	cacheClient.Del(cacheCtx, keys...)
	log.Printf("cache INVALIDATED: %d keys matching %s", len(keys), pattern)
}

// cacheStats returns cache hit/miss stats
func cacheStats() map[string]interface{} {
	if cacheClient == nil {
		return map[string]interface{}{"enabled": false}
	}

	info, err := cacheClient.Info(cacheCtx, "stats").Result()
	if err != nil {
		return map[string]interface{}{"enabled": true, "error": err.Error()}
	}

	// Count cache keys
	keys, _ := cacheClient.Keys(cacheCtx, "cache:*").Result()

	return map[string]interface{}{
		"enabled":    true,
		"cache_keys": len(keys),
		"info":       info[:min(len(info), 200)],
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GET /cache/stats — returns cache statistics
func handleCacheStats(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok":    true,
		"cache": cacheStats(),
	})
}
