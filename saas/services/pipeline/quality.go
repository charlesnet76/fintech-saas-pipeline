package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"time"
)

// ── Data quality types ────────────────────────────────────────────────────────

type QualityScore struct {
	Score     float64                    `json:"score"`
	Grade     string                     `json:"grade"`
	Breakdown map[string]json.RawMessage `json:"breakdown"`
}

type SchemaResult struct {
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

type Anomaly struct {
	Column  string    `json:"column"`
	Method  string    `json:"method"`
	Count   int       `json:"count"`
	Pct     float64   `json:"pct"`
	Message string    `json:"message"`
	Sample  []float64 `json:"sample"`
}

type AnomalyResult struct {
	Anomalies []Anomaly              `json:"anomalies"`
	Stats     map[string]interface{} `json:"stats"`
}

type QualityReport struct {
	Passed     bool          `json:"passed"`
	Rows       int           `json:"rows"`
	Quality    QualityScore  `json:"quality"`
	Schema     SchemaResult  `json:"schema"`
	Anomalies  AnomalyResult `json:"anomalies"`
	DurationMs int64         `json:"duration_ms"`
}

// ── Run validator ─────────────────────────────────────────────────────────────

func runQualityCheck(rows []CSVRow) (*QualityReport, error) {
	start := time.Now()

	// Convert CSVRow structs to string maps for Python validator
	var rowMaps []map[string]string
	for _, r := range rows {
		rowMaps = append(rowMaps, map[string]string{
			"transaction_id": r.TransactionID,
			"customer_ref":   r.CustomerRef,
			"amount":         strconv.FormatFloat(r.Amount, 'f', 2, 64),
			"category":       r.Category,
			"status":         r.Status,
			"province":       r.Province,
			"merchant":       r.Merchant,
			"currency":       r.Currency,
			"txn_date":       r.TxnDate,
		})
	}

	input, err := json.Marshal(rowMaps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rows: %w", err)
	}

	cmd := exec.Command("python3", "validator.py", "--stdin")
	cmd.Stdin = bytes.NewReader(input)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("validator stderr: %s", stderr.String())
		return nil, fmt.Errorf("validator failed: %w", err)
	}

	var report QualityReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		return nil, fmt.Errorf("failed to parse validator output: %w", err)
	}

	report.DurationMs = time.Since(start).Milliseconds()
	return &report, nil
}

// ── Quality endpoint ──────────────────────────────────────────────────────────

// GET /quality/report?org_id=xxx
func handleQualityReport(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		jsonError(w, "org_id required", http.StatusBadRequest)
		return
	}

	rows, err := getLatestRows(orgID, 1000)
	if err != nil {
		jsonError(w, "failed to fetch rows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(rows) == 0 {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"ok":     true,
			"org_id": orgID,
			"report": nil,
			"message": "no data found for this org",
		})
		return
	}

	report, err := runQualityCheck(rows)
	if err != nil {
		log.Printf("quality check error: %v", err)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"ok":    false,
			"org_id": orgID,
			"error": "quality validator unavailable",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"org_id": orgID,
		"report": report,
	})
}

func getLatestRows(orgID string, limit int) ([]CSVRow, error) {
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT transaction_id, customer_ref, province,
		       age_group, account_type, merchant, category,
		       amount, currency, status, txn_date::text,
		       txn_month, txn_year::text
		FROM transactions
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	dbRows, err := db.Query(query, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer dbRows.Close()

	var rows []CSVRow
	for dbRows.Next() {
		var r CSVRow
		var txnYear string
		err := dbRows.Scan(
			&r.TransactionID, &r.CustomerRef, &r.Province,
			&r.AgeGroup, &r.AccountType, &r.Merchant, &r.Category,
			&r.Amount, &r.Currency, &r.Status, &r.TxnDate,
			&r.TxnMonth, &txnYear,
		)
		if err != nil {
			continue
		}
		rows = append(rows, r)
	}
	return rows, nil
}
