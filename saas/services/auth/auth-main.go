package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/bcrypt"
)

// ── Config ────────────────────────────────────────────────────────────────────

var (
	dbURL            = getEnv("DATABASE_URL", "postgresql://fintech_user:fintech_pass@localhost:5432/fintech?sslmode=disable")
	jwtAccessSecret  = getEnv("JWT_ACCESS_SECRET", "dev-access-secret-min-32-chars-here")
	jwtRefreshSecret = getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-min-32-chars-here")
	port             = getEnv("PORT", "8081")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ── Prometheus metrics ────────────────────────────────────────────────────────

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_http_requests_total",
			Help: "Total number of HTTP requests by method, path and status",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	activeConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "auth_active_connections",
		Help: "Number of active HTTP connections",
	})

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query"},
	)

	loginTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_login_total",
			Help: "Total login attempts by result",
		},
		[]string{"result"}, // success, failure
	)

	registrationTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "auth_registration_total",
		Help: "Total user registrations",
	})
)

func initMetrics()
	initRateLimit()
	initSentry("auth-service") {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		activeConnections,
		dbQueryDuration,
		loginTotal,
		registrationTotal,
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
		// Skip metrics endpoint itself
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
		log.Printf("WARNING: failed to ping db: %v (continuing without DB)", err); return
	}
	log.Println("✓ Connected to PostgreSQL")
}

// ── Models ────────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	OrgName  string `json:"org_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type Claims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// ── JWT helpers ───────────────────────────────────────────────────────────────

func signAccessToken(userID, orgID, role string) (string, error) {
	claims := Claims{
		UserID: userID, OrgID: orgID, Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtAccessSecret))
}

func signRefreshToken(userID, orgID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtRefreshSecret))
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" || req.OrgName == "" {
		jsonError(w, "email, password and org_name are required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var orgID string
	orgSlug := slugify(req.OrgName)

	timer := prometheus.NewTimer(dbQueryDuration.WithLabelValues("insert_org"))
	err = tx.QueryRow(`INSERT INTO organizations (name, slug) VALUES ($1, $2) RETURNING id`, req.OrgName, orgSlug).Scan(&orgID)
	timer.ObserveDuration()
	if err != nil {
		jsonError(w, "organization name already taken", http.StatusConflict)
		return
	}

	var userID string
	err = tx.QueryRow(`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`, req.Email, string(hash)).Scan(&userID)
	if err != nil {
		jsonError(w, "email already registered", http.StatusConflict)
		return
	}

	_, err = tx.Exec(`INSERT INTO memberships (org_id, user_id, role) VALUES ($1, $2, 'owner')`, orgID, userID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	accessToken, _ := signAccessToken(userID, orgID, "owner")
	refreshToken, _ := signRefreshToken(userID, orgID)

	db.Exec(`INSERT INTO refresh_tokens (user_id, org_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, orgID, refreshToken, time.Now().Add(7*24*time.Hour))

	registrationTotal.Inc()
	log.Printf("registered user=%s org=%s", userID, orgID)

	jsonResponse(w, http.StatusCreated, AuthResponse{
		AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: 900,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration.WithLabelValues("select_user"))
	var userID, passwordHash string
	err := db.QueryRow(`SELECT id, password_hash FROM users WHERE email = $1 AND deleted_at IS NULL`, req.Email).Scan(&userID, &passwordHash)
	timer.ObserveDuration()

	compareHash := passwordHash
	if err != nil {
		compareHash = "$2b$12$invalidhashfortimingnormalization000000000000"
	}
	if bcrypt.CompareHashAndPassword([]byte(compareHash), []byte(req.Password)) != nil || err != nil {
		loginTotal.WithLabelValues("failure").Inc()
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	var orgID, role string
	db.QueryRow(`SELECT org_id, role FROM memberships WHERE user_id = $1 AND deleted_at IS NULL LIMIT 1`, userID).Scan(&orgID, &role)

	accessToken, _ := signAccessToken(userID, orgID, role)
	refreshToken, _ := signRefreshToken(userID, orgID)

	db.Exec(`INSERT INTO refresh_tokens (user_id, org_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, orgID, refreshToken, time.Now().Add(7*24*time.Hour))
	db.Exec(`UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)

	loginTotal.WithLabelValues("success").Inc()
	log.Printf("login user=%s", userID)

	jsonResponse(w, http.StatusOK, AuthResponse{
		AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: 900,
	})
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var userID, orgID string
	err := db.QueryRow(`
		SELECT user_id, org_id FROM refresh_tokens
		WHERE token = $1 AND revoked_at IS NULL AND expires_at > NOW()`,
		req.RefreshToken).Scan(&userID, &orgID)
	if err != nil {
		jsonError(w, "invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	db.Exec(`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token = $1`, req.RefreshToken)

	var role string
	db.QueryRow(`SELECT role FROM memberships WHERE user_id = $1 AND org_id = $2 AND deleted_at IS NULL`, userID, orgID).Scan(&role)

	accessToken, _ := signAccessToken(userID, orgID, role)
	newRefresh, _ := signRefreshToken(userID, orgID)
	db.Exec(`INSERT INTO refresh_tokens (user_id, org_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, orgID, newRefresh, time.Now().Add(7*24*time.Hour))

	jsonResponse(w, http.StatusOK, AuthResponse{
		AccessToken: accessToken, RefreshToken: newRefresh, ExpiresIn: 900,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.RefreshToken != "" {
		db.Exec(`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token = $1`, req.RefreshToken)
	}
	jsonResponse(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "service": "auth"})
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

func slugify(s string) string {
	slug := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			slug += string(c)
		} else if c >= 'A' && c <= 'Z' {
			slug += string(c + 32)
		} else if c == ' ' || c == '-' {
			slug += "-"
		}
	}
	return slug
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	initMetrics()
	initRateLimit()
	initSentry("auth-service")
	initDB()
	defer db.Close()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/auth/register", handleRegister)
	mux.HandleFunc("/auth/login", handleLogin)
	mux.HandleFunc("/auth/refresh", handleRefresh)
	mux.HandleFunc("/auth/logout", handleLogout)

	// Wrap all routes with metrics middleware
	handler := metricsMiddleware(mux)

	log.Printf("✓ Auth service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
