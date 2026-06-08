package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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

// ── Config ────────────────────────────────────────────────────────────────────

var (
	dbURL = getEnv("DATABASE_URL", "postgresql://fintech_user:fintech_pass@localhost:5432/fintech?sslmode=disable")
	port  = getEnv("PORT", "8086")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ── Prometheus metrics ────────────────────────────────────────────────────────

var (
	uploadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_uploads_total",
			Help: "Total CSV uploads by org and result",
		},
		[]string{"org_id", "result"},
	)

	rowsIngested = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_rows_ingested_total",
			Help: "Total rows ingested per org",
		},
		[]string{"org_id"},
	)

	uploadDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipeline_upload_duration_seconds",
			Help:    "CSV upload and ingestion duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"org_id"},
	)
)

func initMetrics() {
	prometheus.MustRegister(uploadsTotal, rowsIngested, uploadDuration)
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

// ── CSV Upload Handler ────────────────────────────────────────────────────────

/*
Expected CSV columns (flexible — maps by header name):
  transaction_id, customer_ref, province, age_group, account_type,
  merchant, category, amount, currency, status, txn_date, txn_month, txn_year

Required: transaction_id, amount, status, txn_date
Optional: all others default to empty/null
*/

type CSVRow struct {
	TransactionID string
	CustomerRef   string
	Province      string
	AgeGroup      string
	AccountType   string
	Merchant      string
	Category      string
	Amount        float64
	Currency      string
	Status        string
	TxnDate       string
	TxnMonth      string
	TxnYear       int
}

type UploadResult struct {
	OK           bool     `json:"ok"`
	OrgID        string   `json:"org_id"`
	RowsReceived int      `json:"rows_received"`
	RowsInserted int      `json:"rows_inserted"`
	RowsSkipped  int      `json:"rows_skipped"`
	Errors       []string `json:"errors,omitempty"`
	DurationMs   int64    `json:"duration_ms"`
}

func parseCSV(r io.Reader) ([]CSVRow, []string, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Normalize headers
	headerIdx := map[string]int{}
	for i, h := range headers {
		headerIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Validate required columns
	required := []string{"transaction_id", "amount", "status", "txn_date"}
	for _, req := range required {
		if _, ok := headerIdx[req]; !ok {
			return nil, nil, fmt.Errorf("missing required column: %s", req)
		}
	}

	get := func(row []string, col string) string {
		if idx, ok := headerIdx[col]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	var rows []CSVRow
	var parseErrors []string
	lineNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		lineNum++
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("line %d: %v", lineNum, err))
			continue
		}

		amount, err := strconv.ParseFloat(get(record, "amount"), 64)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("line %d: invalid amount '%s'", lineNum, get(record, "amount")))
			continue
		}
		if amount <= 0 {
			parseErrors = append(parseErrors, fmt.Sprintf("line %d: amount must be > 0", lineNum))
			continue
		}

		status := strings.ToLower(get(record, "status"))
		if status != "completed" && status != "pending" && status != "failed" {
			parseErrors = append(parseErrors, fmt.Sprintf("line %d: invalid status '%s'", lineNum, status))
			continue
		}

		txnYear := 0
		if y := get(record, "txn_year"); y != "" {
			txnYear, _ = strconv.Atoi(y)
		}

		currency := get(record, "currency")
		if currency == "" {
			currency = "CAD"
		}

		rows = append(rows, CSVRow{
			TransactionID: get(record, "transaction_id"),
			CustomerRef:   get(record, "customer_ref"),
			Province:      get(record, "province"),
			AgeGroup:      get(record, "age_group"),
			AccountType:   get(record, "account_type"),
			Merchant:      get(record, "merchant"),
			Category:      get(record, "category"),
			Amount:        amount,
			Currency:      currency,
			Status:        status,
			TxnDate:       get(record, "txn_date"),
			TxnMonth:      get(record, "txn_month"),
			TxnYear:       txnYear,
		})
	}

	return rows, parseErrors, nil
}

func insertRows(orgID string, rows []CSVRow) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO transactions (
			org_id, transaction_id, customer_ref, province, age_group,
			account_type, merchant, category, amount, currency,
			status, txn_date, txn_month, txn_year
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (org_id, transaction_id) DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, row := range rows {
		var txnMonth string
		if row.TxnMonth != "" {
			txnMonth = row.TxnMonth
		} else if len(row.TxnDate) >= 7 {
			txnMonth = row.TxnDate[:7]
		}

		var txnYear interface{} = nil
		if row.TxnYear > 0 {
			txnYear = row.TxnYear
		}

		_, err := stmt.Exec(
			orgID, row.TransactionID, row.CustomerRef, row.Province, row.AgeGroup,
			row.AccountType, row.Merchant, row.Category, row.Amount, row.Currency,
			row.Status, row.TxnDate, txnMonth, txnYear,
		)
		if err != nil {
			log.Printf("insert error: %v", err)
			continue
		}
		inserted++
	}

	return inserted, tx.Commit()
}

// POST /upload/csv
// Header: X-Org-ID: <org_id>
// Body: multipart/form-data with field "file"
func handleUpload(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		jsonError(w, "X-Org-ID header required", http.StatusBadRequest)
		return
	}

	// Verify org exists
	var exists bool
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM organizations WHERE id = $1 AND deleted_at IS NULL)`, orgID).Scan(&exists)
	if !exists {
		jsonError(w, "organization not found", http.StatusNotFound)
		return
	}

	// Parse multipart form (max 50MB)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		jsonError(w, "failed to parse form — max 50MB", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "field 'file' is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		jsonError(w, "only .csv files are accepted", http.StatusBadRequest)
		return
	}

	log.Printf("upload started org=%s file=%s size=%d", orgID, header.Filename, header.Size)

	timer := prometheus.NewTimer(uploadDuration.WithLabelValues(orgID))
	defer timer.ObserveDuration()

	// Parse CSV
	rows, parseErrors, err := parseCSV(file)
	if err != nil {
		uploadsTotal.WithLabelValues(orgID, "failed").Inc()
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if len(rows) == 0 {
		uploadsTotal.WithLabelValues(orgID, "empty").Inc()
		jsonResponse(w, http.StatusOK, UploadResult{
			OK:           false,
			OrgID:        orgID,
			RowsReceived: 0,
			Errors:       append([]string{"no valid rows found"}, parseErrors...),
		})
		return
	}

	// Insert into DB
	inserted, err := insertRows(orgID, rows)
	if err != nil {
		uploadsTotal.WithLabelValues(orgID, "failed").Inc()
		jsonError(w, "database error during insert", http.StatusInternalServerError)
		return
	}

	uploadsTotal.WithLabelValues(orgID, "success").Inc()
	rowsIngested.WithLabelValues(orgID).Add(float64(inserted))

	result := UploadResult{
		OK:           true,
		OrgID:        orgID,
		RowsReceived: len(rows),
		RowsInserted: inserted,
		RowsSkipped:  len(rows) - inserted,
		Errors:       parseErrors,
		DurationMs:   time.Since(start).Milliseconds(),
	}

	log.Printf("upload complete org=%s inserted=%d skipped=%d duration=%dms",
		orgID, inserted, result.RowsSkipped, result.DurationMs)

	jsonResponse(w, http.StatusCreated, result)
}

// GET /upload/status?org_id=xxx
func handleStatus(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		jsonError(w, "org_id query param required", http.StatusBadRequest)
		return
	}

	var count int
	var totalRevenue float64
	db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE org_id = $1 AND status = 'completed'
	`, orgID).Scan(&count, &totalRevenue)

	var lastLoaded *time.Time
	row := db.QueryRow(`SELECT MAX(loaded_at) FROM transactions WHERE org_id = $1`, orgID)
	row.Scan(&lastLoaded)

	jsonResponse(w, http.StatusOK, map[string]any{
		"ok":            true,
		"org_id":        orgID,
		"total_txns":    count,
		"total_revenue": fmt.Sprintf("CAD $%.2f", totalRevenue),
		"last_loaded":   lastLoaded,
	})
}

// GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "service": "pipeline"})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResponse(w, status, map[string]any{"ok": false, "error": msg})
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	initMetrics()
	initRateLimit()
	initSentry("pipeline-service")
	initRateLimit()
	initSentry("pipeline-service")
	initDB()
	defer db.Close()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/upload/csv", handleUpload)
	mux.HandleFunc("/upload/status", handleStatus)

	log.Printf("✓ Pipeline service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
