package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

// ── Sentry error tracking — drop this file into any Go service ───────────────
// Captures: panics, errors, slow requests, custom events
// DSN: set SENTRY_DSN env var or falls back to hardcoded

func initSentry(serviceName string) {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		dsn = "https://5ac640ff05c304eaa783f1c32a1f16a5@o4511527292370944.ingest.us.sentry.io/4511527315046400"
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "production"
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          serviceName + "@" + os.Getenv("VERSION"),
		TracesSampleRate: 0.2, // 20% of requests traced
		EnableTracing:    true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Scrub sensitive fields before sending
			if event.Request != nil {
				event.Request.Headers["Authorization"] = "[REDACTED]"
				event.Request.Headers["X-Api-Key"] = "[REDACTED]"
			}
			return event
		},
	})
	if err != nil {
		log.Printf("WARNING: Sentry init failed: %v", err)
		return
	}

	// Set global tags
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("service", serviceName)
		scope.SetTag("region", "canadacentral")
	})

	log.Printf("✓ Sentry initialized — service=%s env=%s", serviceName, env)
}

// sentryMiddleware captures panics and errors for every HTTP request
func sentryMiddleware(serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip health and metrics endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(r)
		hub.Scope().SetTag("service", serviceName)
		hub.Scope().SetTag("method", r.Method)
		hub.Scope().SetTag("path", r.URL.Path)

		orgID := r.Header.Get("X-Org-ID")
		if orgID != "" {
			hub.Scope().SetUser(sentry.User{ID: orgID})
		}

		// Capture panics
		defer func() {
			if err := recover(); err != nil {
				hub.Recover(err)
				hub.Flush(2 * time.Second)
				http.Error(w, `{"ok":false,"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()

		// Wrap response writer to capture status code
		srw := &sentryResponseWriter{ResponseWriter: w, status: 200}
		start := time.Now()

		next.ServeHTTP(srw, r)

		duration := time.Since(start)

		// Capture slow requests (> 2 seconds)
		if duration > 2*time.Second {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("duration_ms", duration.Milliseconds())
				scope.SetExtra("path", r.URL.Path)
				scope.SetLevel(sentry.LevelWarning)
				hub.CaptureMessage("Slow request detected")
			})
		}

		// Capture 5xx errors
		if srw.status >= 500 {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("status_code", srw.status)
				scope.SetExtra("path", r.URL.Path)
				scope.SetExtra("method", r.Method)
				scope.SetLevel(sentry.LevelError)
				hub.CaptureMessage("HTTP 5xx error")
			})
		}
	})
}

// captureError sends an error to Sentry with context
func captureError(err error, context map[string]interface{}) {
	if err == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		for k, v := range context {
			scope.SetExtra(k, v)
		}
		sentry.CaptureException(err)
	})
}

// sentryResponseWriter wraps http.ResponseWriter to capture status codes
type sentryResponseWriter struct {
	http.ResponseWriter
	status int
}

func (srw *sentryResponseWriter) WriteHeader(status int) {
	srw.status = status
	srw.ResponseWriter.WriteHeader(status)
}
