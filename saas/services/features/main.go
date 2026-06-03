package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ── Config ───────────────────────────────────────────────────────────────────

var (
	dbURL  = getEnv("DATABASE_URL", "postgres://localhost:5432/fintech?sslmode=disable")
	port   = getEnv("PORT", "8082")
	jwtKey = getEnv("JWT_ACCESS_SECRET", "dev-secret")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ── Prometheus metrics ────────────────────────────────────────────────────────

var (
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "features_http_requests_total",
		Help: "Total HTTP requests to the features service.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "features_http_request_duration_seconds",
		Help:    "HTTP request duration for features service.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	activeConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "features_active_connections",
		Help: "Active HTTP connections on features service.",
	})

	dbQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "features_db_query_duration_seconds",
		Help:    "DB query duration for features service.",
		Buckets: prometheus.DefBuckets,
	}, []string{"query"})

	featureRegistrations = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "features_registrations_total",
		Help: "Total feature registrations.",
	})

	featureServeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "features_serve_total",
		Help: "Total feature serve requests.",
	}, []string{"feature", "mode"}) // mode: latest | point_in_time

	featureIngestTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "features_ingest_total",
		Help: "Total feature values ingested.",
	})
)

func initMetrics() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		activeConnections,
		dbQueryDuration,
		featureRegistrations,
		featureServeTotal,
		featureIngestTotal,
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
		start := time.Now()
		activeConnections.Inc()
		defer activeConnections.Dec()

		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rec.status)
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
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
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err = db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("✔ Connected to PostgreSQL")
}

// ── Auth middleware ───────────────────────────────────────────────────────────
// Reuses the same JWT claims structure as the auth service.
// Extracts org_id and sets app.org_id for RLS on every request.

type contextKey string

const (
	ctxOrgID  contextKey = "org_id"
	ctxUserID contextKey = "user_id"
	ctxRole   contextKey = "role"
)

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			jsonError(w, "missing or invalid authorization header", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate JWT — same HS256 + Claims shape as auth service
		orgID, userID, role, err := parseJWT(tokenStr)
		if err != nil {
			jsonError(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Set RLS context for this request's DB queries
		r = r.WithContext(withClaims(r.Context(), orgID, userID, role))
		next(w, r)
	}
}

// ── Models ────────────────────────────────────────────────────────────────────

type Feature struct {
	ID                  string    `json:"id"`
	OrgID               string    `json:"org_id"`
	Name                string    `json:"name"`
	Description         string    `json:"description,omitempty"`
	Dtype               string    `json:"dtype"`
	Owner               string    `json:"owner,omitempty"`
	FreshnessSLAMinutes int       `json:"freshness_sla_minutes"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type RegisterFeatureRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	Dtype               string `json:"dtype"`
	Owner               string `json:"owner"`
	FreshnessSLAMinutes int    `json:"freshness_sla_minutes"`
}

type IngestRequest struct {
	EntityID string          `json:"entity_id"`
	Value    json.RawMessage `json:"value"`
	ValidAt  *time.Time      `json:"valid_at"` // optional — defaults to NOW()
}

type ServeResponse struct {
	FeatureName string          `json:"feature_name"`
	EntityID    string          `json:"entity_id"`
	Value       json.RawMessage `json:"value"`
	ValidAt     time.Time       `json:"valid_at"`
	AsOf        *time.Time      `json:"as_of,omitempty"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// POST /features/register
// Registers a new feature definition for the org.
func handleRegisterFeature(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgID(r)

	var req RegisterFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Dtype == "" {
		jsonError(w, "name and dtype are required", http.StatusBadRequest)
		return
	}

	validDtypes := map[string]bool{"float": true, "int": true, "bool": true, "string": true, "json": true}
	if !validDtypes[req.Dtype] {
		jsonError(w, "dtype must be one of: float, int, bool, string, json", http.StatusBadRequest)
		return
	}

	if req.FreshnessSLAMinutes == 0 {
		req.FreshnessSLAMinutes = 60
	}

	timer := prometheus.NewTimer(dbQueryDuration.WithLabelValues("insert_feature"))
	var feature Feature
	err := db.QueryRow(`
		INSERT INTO feature_registry
			(org_id, name, description, dtype, owner, freshness_sla_minutes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, org_id, name, description, dtype, owner, freshness_sla_minutes, created_at, updated_at`,
		orgID, req.Name, req.Description, req.Dtype, req.Owner, req.FreshnessSLAMinutes,
	).Scan(
		&feature.ID, &feature.OrgID, &feature.Name, &feature.Description,
		&feature.Dtype, &feature.Owner, &feature.FreshnessSLAMinutes,
		&feature.CreatedAt, &feature.UpdatedAt,
	)
	timer.ObserveDuration()

	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			jsonError(w, "feature name already exists for this org", http.StatusConflict)
			return
		}
		log.Printf("register feature error: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	featureRegistrations.Inc()
	log.Printf("feature registered org=%s name=%s", orgID, req.Name)
	jsonResponse(w, http.StatusCreated, feature)
}

// GET /features
// Lists all features for the authenticated org.
func handleListFeatures(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgID(r)

	timer := prometheus.NewTimer(dbQueryDuration.WithLabelValues("list_features"))
	rows, err := db.Query(`
		SELECT id, org_id, name, description, dtype, owner, freshness_sla_minutes, created_at, updated_at
		FROM feature_registry
		WHERE org_id = $1 AND deleted_at IS NULL
		ORDER BY name ASC`,
		orgID,
	)
	timer.ObserveDuration()
	if err != nil {
		log.Printf("list features error: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	features := []Feature{}
	for rows.Next() {
		var f Feature
		if err := rows.Scan(
			&f.ID, &f.OrgID, &f.Name, &f.Description,
			&f.Dtype, &f.Owner, &f.FreshnessSLAMinutes,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			log.Printf("scan feature error: %v", err)
			continue
		}
		features = append(features, f)
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"features": features,
		"count":    len(features),
	})
}

// GET /features/{name}/serve?entity_id=X&as_of=2024-01-01T00:00:00Z
// Serves a feature value. If as_of is provided, returns point-in-time value.
// If omitted, returns the latest value.
func handleServeFeature(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgID(r)

	// Extract feature name from path: /features/{name}/serve
	featureName := extractPathSegment(r.URL.Path, "/features/", "/serve")
	if featureName == "" {
		jsonError(w, "feature name is required", http.StatusBadRequest)
		return
	}

	entityID := r.URL.Query().Get("entity_id")
	if entityID == "" {
		jsonError(w, "entity_id query parameter is required", http.StatusBadRequest)
		return
	}

	asOfStr := r.URL.Query().Get("as_of")
	mode := "latest"

	var resp ServeResponse
	resp.FeatureName = featureName
	resp.EntityID = entityID

	if asOfStr != "" {
		// ── Point-in-Time retrieval ──────────────────────────────────────────
		// Returns the feature value as it existed at the given timestamp.
		// This is the core correctness primitive of any ML feature store.
		asOf, err := time.Parse(time.RFC3339, asOfStr)
		if err != nil {
			jsonError(w, "as_of must be RFC3339 format e.g. 2024-01-01T00:00:00Z", http.StatusBadRequest)
			return
		}
		resp.AsOf = &asOf
		mode = "point_in_time"

		timer := prometheus.NewTimer(dbQueryDuration.WithLabelValues("serve_pit"))
		err = db.QueryRow(`
			SELECT fv.value, fv.valid_at
			FROM feature_values fv
			JOIN feature_registry fr ON fr.id = fv.feature_id
			WHERE fr.name    = $1
			  AND fr.org_id  = $2
			  AND fv.entity_id = $3
			  AND fv.valid_at  <= $4
			  AND fr.deleted_at IS NULL
			ORDER BY fv.valid_at DESC
			LIMIT 1`,
			featureName, orgID, entityID, asOf,
		).Scan(&resp.Value, &resp.ValidAt)
		timer.ObserveDuration()

		if err == sql.ErrNoRows {
			jsonError(w, "no feature value found for entity at requested time", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("serve pit error: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		// ── Latest value retrieval ───────────────────────────────────────────
		timer := prometheus.NewTimer(dbQueryDuration.WithLabelValues("serve_latest"))
		err := db.QueryRow(`
			SELECT fv.value, fv.valid_at
			FROM feature_values fv
			JOIN feature_registry fr ON fr.id = fv.feature_id
			WHERE fr.name      = $1
			  AND fr.org_id    = $2
			  AND fv.entity_id = $3
			  AND fr.deleted_at IS NULL
			ORDER BY fv.valid_at DESC
			LIMIT 1`,
			featureName, orgID, entityID,
		).Scan(&resp.Value, &resp.ValidAt)
		timer.ObserveDuration()

		if err == sql.ErrNoRows {
			jsonError(w, "no feature value found for entity", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("serve latest error: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	featureServeTotal.WithLabelValues(featureName, mode).Inc()
	jsonResponse(w, http.StatusOK, resp)
}

// POST /features/{name}/ingest
// Writes a new feature value for an entity.
// Body: { "entity_id": "user_123", "value": 0.87, "valid_at": "2024-01-01T00:00:00Z" }
func handleIngestFeature(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgID(r)

	featureName := extractPathSegment(r.URL.Path, "/features/", "/ingest")
	if featureName == "" {
		jsonError(w, "feature name is required", http.StatusBadRequest)
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.EntityID == "" {
		jsonError(w, "entity_id is required", http.StatusBadRequest)
		return
	}
	if req.Value == nil {
		jsonError(w, "value is required", http.StatusBadRequest)
		return
	}

	validAt := time.Now()
	if req.ValidAt != nil {
		validAt = *req.ValidAt
	}

	// Resolve feature_id from name + org
	timer := prometheus.NewTimer(dbQueryDuration.WithLabelValues("resolve_feature"))
	var featureID string
	err := db.QueryRow(`
		SELECT id FROM feature_registry
		WHERE name = $1 AND org_id = $2 AND deleted_at IS NULL`,
		featureName, orgID,
	).Scan(&featureID)
	timer.ObserveDuration()

	if err == sql.ErrNoRows {
		jsonError(w, "feature not found — register it first via POST /features/register", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("resolve feature error: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Insert the feature value
	timer2 := prometheus.NewTimer(dbQueryDuration.WithLabelValues("insert_feature_value"))
	_, err = db.Exec(`
		INSERT INTO feature_values (org_id, feature_id, entity_id, value, valid_at)
		VALUES ($1, $2, $3, $4, $5)`,
		orgID, featureID, req.EntityID, req.Value, validAt,
	)
	timer2.ObserveDuration()

	if err != nil {
		log.Printf("ingest feature error: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	featureIngestTotal.Inc()
	log.Printf("feature ingested org=%s feature=%s entity=%s valid_at=%s", orgID, featureName, req.EntityID, validAt)
	jsonResponse(w, http.StatusCreated, map[string]any{
		"feature":   featureName,
		"entity_id": req.EntityID,
		"valid_at":  validAt,
		"status":    "ingested",
	})
}

// GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "service": "features"})
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

// extractPathSegment pulls the middle segment from /prefix/{segment}/suffix
func extractPathSegment(path, prefix, suffix string) string {
	path = strings.TrimPrefix(path, prefix)
	path = strings.TrimSuffix(path, suffix)
	return path
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	initMetrics()
	initDB()
	defer db.Close()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", handleHealth)

	// Feature store endpoints — all require JWT auth
	mux.HandleFunc("/features/register", authMiddleware(handleRegisterFeature))
	mux.HandleFunc("/features", authMiddleware(handleListFeatures))

	// Dynamic routes: /features/{name}/serve and /features/{name}/ingest
	// net/http doesn't support path params natively — we route the prefix
	// and extract the segment inside the handler.
	mux.HandleFunc("/features/", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/serve") && r.Method == http.MethodGet:
			handleServeFeature(w, r)
		case strings.HasSuffix(path, "/ingest") && r.Method == http.MethodPost:
			handleIngestFeature(w, r)
		default:
			jsonError(w, "not found", http.StatusNotFound)
		}
	}))

	handler := metricsMiddleware(mux)

	log.Printf("✔ Features service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
