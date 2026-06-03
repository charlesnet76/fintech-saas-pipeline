package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ── Config ────────────────────────────────────────────────────────────────────

var (
	dbURL        = getEnv("DATABASE_URL", "postgres://localhost:5432/fintech?sslmode=disable")
	port         = getEnv("PORT", "8083")
	slackWebhook = getEnv("SLACK_WEBHOOK_URL", "")
	checkEvery   = getDuration("CHECK_INTERVAL", 5*time.Minute)
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Printf("invalid %s=%q, using default %s", key, v, fallback)
		return fallback
	}
	return d
}

// ── Prometheus metrics ────────────────────────────────────────────────────────

var (
	staleFeatures = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "feature_store_stale_features_total",
		Help: "Number of features currently violating their freshness SLA.",
	}, []string{"feature", "org"})

	featureAgeSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "feature_store_age_seconds",
		Help: "Age in seconds of the most recent value for each feature.",
	}, []string{"feature", "org"})

	slaViolations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "feature_store_sla_violations_total",
		Help: "Cumulative SLA violations detected per feature.",
	}, []string{"feature", "org"})

	checksTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "feature_store_checks_total",
		Help: "Total freshness check runs.",
	})

	slackAlertsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "feature_store_slack_alerts_total",
		Help: "Slack alerts sent.",
	}, []string{"status"}) // status: sent | failed | skipped
)

func initMetrics() {
	prometheus.MustRegister(
		staleFeatures,
		featureAgeSeconds,
		slaViolations,
		checksTotal,
		slackAlertsTotal,
	)
}

// ── Metrics middleware ────────────────────────────────────────────────────────

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
	})
}

// ── Database ──────────────────────────────────────────────────────────────────

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err = db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("✔ Connected to PostgreSQL")
}

// ── Models ────────────────────────────────────────────────────────────────────

type StalenessResult struct {
	FeatureName         string
	OrgName             string
	OrgID               string
	FreshnessSLAMinutes int
	LastUpdated         *time.Time // nil = never updated
	AgeMinutes          float64
	IsStale             bool
}

// ── Freshness check ───────────────────────────────────────────────────────────

func checkFreshness() ([]StalenessResult, error) {
	rows, err := db.Query(`
		SELECT
			fr.name                    AS feature_name,
			o.name                     AS org_name,
			fr.org_id::TEXT            AS org_id,
			fr.freshness_sla_minutes,
			MAX(fv.created_at)         AS last_updated,
			EXTRACT(EPOCH FROM (NOW() - MAX(fv.created_at))) / 60.0
			                           AS age_minutes
		FROM feature_registry fr
		JOIN organizations o ON o.id = fr.org_id
		LEFT JOIN feature_values fv ON fv.feature_id = fr.id
		WHERE fr.deleted_at IS NULL
		  AND o.deleted_at  IS NULL
		GROUP BY fr.id, fr.name, fr.org_id, fr.freshness_sla_minutes, o.name
		ORDER BY o.name, fr.name
	`)
	if err != nil {
		return nil, fmt.Errorf("freshness query: %w", err)
	}
	defer rows.Close()

	var results []StalenessResult
	for rows.Next() {
		var r StalenessResult
		var ageMinutes sql.NullFloat64
		var lastUpdated sql.NullTime

		if err := rows.Scan(
			&r.FeatureName,
			&r.OrgName,
			&r.OrgID,
			&r.FreshnessSLAMinutes,
			&lastUpdated,
			&ageMinutes,
		); err != nil {
			log.Printf("scan error: %v", err)
			continue
		}

		if lastUpdated.Valid {
			t := lastUpdated.Time
			r.LastUpdated = &t
		}
		if ageMinutes.Valid {
			r.AgeMinutes = ageMinutes.Float64
		} else {
			// Never had a value written — treat as infinitely stale
			r.AgeMinutes = float64(r.FreshnessSLAMinutes) * 10
		}

		r.IsStale = r.AgeMinutes > float64(r.FreshnessSLAMinutes)
		results = append(results, r)
	}

	return results, rows.Err()
}

// ── Prometheus update ─────────────────────────────────────────────────────────

func updateMetrics(results []StalenessResult) {
	// Reset stale gauge each cycle so recovered features go to 0
	staleFeatures.Reset()

	for _, r := range results {
		labels := prometheus.Labels{"feature": r.FeatureName, "org": r.OrgName}

		featureAgeSeconds.With(labels).Set(r.AgeMinutes * 60)

		if r.IsStale {
			staleFeatures.With(labels).Set(1)
			slaViolations.With(labels).Inc()
		}
	}
}

// ── Slack alerting ────────────────────────────────────────────────────────────

type slackPayload struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string       `json:"color"`
	Fields []slackField `json:"fields"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func sendSlackAlert(stale []StalenessResult) {
	if slackWebhook == "" {
		slackAlertsTotal.WithLabelValues("skipped").Inc()
		return
	}
	if len(stale) == 0 {
		return
	}

	fields := make([]slackField, 0, len(stale))
	for _, r := range stale {
		lastSeen := "never"
		if r.LastUpdated != nil {
			lastSeen = r.LastUpdated.UTC().Format(time.RFC3339)
		}
		fields = append(fields, slackField{
			Title: fmt.Sprintf("%s / %s", r.OrgName, r.FeatureName),
			Value: fmt.Sprintf("Age: %.1f min | SLA: %d min | Last seen: %s",
				r.AgeMinutes, r.FreshnessSLAMinutes, lastSeen),
			Short: false,
		})
	}

	payload := slackPayload{
		Text: fmt.Sprintf(":warning: *Feature Store SLA Violation* — %d feature(s) stale", len(stale)),
		Attachments: []slackAttachment{
			{Color: "danger", Fields: fields},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("slack marshal error: %v", err)
		slackAlertsTotal.WithLabelValues("failed").Inc()
		return
	}

	resp, err := http.Post(slackWebhook, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("slack send error: %v", err)
		slackAlertsTotal.WithLabelValues("failed").Inc()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("slack non-200: %d", resp.StatusCode)
		slackAlertsTotal.WithLabelValues("failed").Inc()
		return
	}

	slackAlertsTotal.WithLabelValues("sent").Inc()
	log.Printf("✔ Slack alert sent: %d stale feature(s)", len(stale))
}

// ── Check loop ────────────────────────────────────────────────────────────────

func runCheckLoop() {
	log.Printf("✔ Freshness monitor running — checking every %s", checkEvery)

	ticker := time.NewTicker(checkEvery)
	defer ticker.Stop()

	// Run immediately on startup, then on every tick
	runCheck()
	for range ticker.C {
		runCheck()
	}
}

func runCheck() {
	checksTotal.Inc()

	results, err := checkFreshness()
	if err != nil {
		log.Printf("freshness check error: %v", err)
		return
	}

	updateMetrics(results)

	var stale []StalenessResult
	for _, r := range results {
		if r.IsStale {
			stale = append(stale, r)
			log.Printf("STALE feature=%s org=%s age=%.1fmin sla=%dmin",
				r.FeatureName, r.OrgName, r.AgeMinutes, r.FreshnessSLAMinutes)
		}
	}

	if len(stale) > 0 {
		sendSlackAlert(stale)
	} else {
		log.Printf("✔ All %d feature(s) within SLA", len(results))
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"status": "degraded", "error": err.Error(),
		})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{
		"status": "ok", "service": "freshness-monitor",
	})
}

// GET /status — returns current freshness state for all features
func handleStatus(w http.ResponseWriter, r *http.Request) {
	results, err := checkFreshness()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	staleCount := 0
	for _, r := range results {
		if r.IsStale {
			staleCount++
		}
	}

	type featureStatus struct {
		Feature    string     `json:"feature"`
		Org        string     `json:"org"`
		SLAMinutes int        `json:"sla_minutes"`
		AgeMinutes float64    `json:"age_minutes"`
		LastUpdated *time.Time `json:"last_updated"`
		IsStale    bool       `json:"is_stale"`
	}

	statuses := make([]featureStatus, 0, len(results))
	for _, r := range results {
		statuses = append(statuses, featureStatus{
			Feature:     r.FeatureName,
			Org:         r.OrgName,
			SLAMinutes:  r.FreshnessSLAMinutes,
			AgeMinutes:  r.AgeMinutes,
			LastUpdated: r.LastUpdated,
			IsStale:     r.IsStale,
		})
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"checked_at":   time.Now().UTC(),
		"total":        len(results),
		"stale_count":  staleCount,
		"healthy_count": len(results) - staleCount,
		"features":     statuses,
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	initMetrics()
	initDB()
	defer db.Close()

	// Start the background check loop
	go runCheckLoop()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/status", handleStatus)

	handler := metricsMiddleware(mux)

	log.Printf("✔ Freshness monitor listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
