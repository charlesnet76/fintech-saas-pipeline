package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
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
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
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

// POST /auth/register
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

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Create org + user in a transaction
	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert org
	var orgID string
	orgSlug := slugify(req.OrgName)
	err = tx.QueryRow(
		`INSERT INTO organizations (name, slug) VALUES ($1, $2) RETURNING id`,
		req.OrgName, orgSlug,
	).Scan(&orgID)
	if err != nil {
		jsonError(w, "organization name already taken", http.StatusConflict)
		return
	}

	// Insert user
	var userID string
	err = tx.QueryRow(
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		req.Email, string(hash),
	).Scan(&userID)
	if err != nil {
		jsonError(w, "email already registered", http.StatusConflict)
		return
	}

	// Insert membership as owner
	_, err = tx.Exec(
		`INSERT INTO memberships (org_id, user_id, role) VALUES ($1, $2, 'owner')`,
		orgID, userID,
	)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Issue tokens
	accessToken, err := signAccessToken(userID, orgID, "owner")
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	refreshToken, err := signRefreshToken(userID, orgID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Store refresh token
	_, err = db.Exec(
		`INSERT INTO refresh_tokens (user_id, org_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, orgID, refreshToken, time.Now().Add(7*24*time.Hour),
	)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("registered user=%s org=%s", userID, orgID)
	jsonResponse(w, http.StatusCreated, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
	})
}

// POST /auth/login
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get user
	var userID, passwordHash string
	err := db.QueryRow(
		`SELECT id, password_hash FROM users WHERE email = $1 AND deleted_at IS NULL`,
		req.Email,
	).Scan(&userID, &passwordHash)

	// Always run bcrypt to prevent timing attacks
	compareHash := passwordHash
	if err != nil {
		compareHash = "$2b$12$invalidhashfortimingnormalization000000000000"
	}
	if bcrypt.CompareHashAndPassword([]byte(compareHash), []byte(req.Password)) != nil || err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Get org + role
	var orgID, role string
	err = db.QueryRow(
		`SELECT org_id, role FROM memberships WHERE user_id = $1 AND deleted_at IS NULL LIMIT 1`,
		userID,
	).Scan(&orgID, &role)
	if err != nil {
		jsonError(w, "no organization found for user", http.StatusUnauthorized)
		return
	}

	// Issue tokens
	accessToken, _ := signAccessToken(userID, orgID, role)
	refreshToken, _ := signRefreshToken(userID, orgID)

	// Store refresh token
	db.Exec(
		`INSERT INTO refresh_tokens (user_id, org_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, orgID, refreshToken, time.Now().Add(7*24*time.Hour),
	)

	// Update last login
	db.Exec(`UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)

	log.Printf("login user=%s org=%s", userID, orgID)
	jsonResponse(w, http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
	})
}

// POST /auth/refresh
func handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Look up refresh token
	var userID, orgID string
	err := db.QueryRow(
		`SELECT user_id, org_id FROM refresh_tokens
		 WHERE token = $1 AND revoked_at IS NULL AND expires_at > NOW()`,
		req.RefreshToken,
	).Scan(&userID, &orgID)
	if err != nil {
		jsonError(w, "invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	// Revoke old token
	db.Exec(`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token = $1`, req.RefreshToken)

	// Get role
	var role string
	db.QueryRow(
		`SELECT role FROM memberships WHERE user_id = $1 AND org_id = $2 AND deleted_at IS NULL`,
		userID, orgID,
	).Scan(&role)

	// Issue new tokens
	accessToken, _ := signAccessToken(userID, orgID, role)
	newRefresh, _ := signRefreshToken(userID, orgID)

	db.Exec(
		`INSERT INTO refresh_tokens (user_id, org_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, orgID, newRefresh, time.Now().Add(7*24*time.Hour),
	)

	jsonResponse(w, http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresIn:    900,
	})
}

// POST /auth/logout
func handleLogout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.RefreshToken != "" {
		db.Exec(`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token = $1`, req.RefreshToken)
	}
	jsonResponse(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "auth",
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
	initDB()
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/auth/register", handleRegister)
	mux.HandleFunc("/auth/login", handleLogin)
	mux.HandleFunc("/auth/refresh", handleRefresh)
	mux.HandleFunc("/auth/logout", handleLogout)

	log.Printf("✓ Auth service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
