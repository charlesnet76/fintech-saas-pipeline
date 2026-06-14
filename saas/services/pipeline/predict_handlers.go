package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"
)

// ── Predict handlers — Phase 3 predictive analytics ──────────────────────────

func runPredict(input map[string]interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	cmd := exec.Command("python3", "predict.py", "--stdin")
	cmd.Stdin = bytes.NewReader(data)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("predict.py stderr: %s", stderr.String())
		return nil, fmt.Errorf("predict failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return result, nil
}

// GET /predict/all — run all models
func handlePredictAll(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = "demo"
	}

	input, err := buildPredictInput(orgID)
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"ok": false, "error": err.Error(),
		})
		return
	}

	result, err := runPredict(input)
	if err != nil {
		log.Printf("predict/all error: %v", err)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"ok": false, "error": "prediction engine unavailable",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok":          true,
		"org_id":      orgID,
		"duration_ms": time.Since(start).Milliseconds(),
		"results":     result,
	})
}

// GET /predict/forecast
func handlePredictForecast(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" { orgID = "demo" }

	monthly, err := getMonthlyRevenue(orgID)
	if err != nil || len(monthly) == 0 {
		monthly = getDemoMonthly()
	}

	result, err := runPredict(map[string]interface{}{"monthly": monthly})
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "org_id": orgID, "forecast": result["forecast"]})
}

// GET /predict/churn
func handlePredictChurn(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" { orgID = "demo" }

	customers, err := getCustomerStats(orgID)
	if err != nil || len(customers) == 0 {
		customers = getDemoCustomers()
	}

	result, err := runPredict(map[string]interface{}{"customers": customers})
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "org_id": orgID, "churn": result["churn"]})
}

// GET /predict/segments
func handlePredictSegments(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" { orgID = "demo" }

	customers, err := getCustomerStats(orgID)
	if err != nil || len(customers) == 0 {
		customers = getDemoCustomers()
	}

	result, err := runPredict(map[string]interface{}{"customers": customers})
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "org_id": orgID, "segments": result["segments"]})
}

// GET /predict/fraud
func handlePredictFraud(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" { orgID = "demo" }

	rows, err := getLatestRows(orgID, 500)
	var txns []map[string]interface{}
	if err != nil || len(rows) == 0 {
		txns = getDemoTransactions()
	} else {
		for _, r := range rows {
			txns = append(txns, map[string]interface{}{
				"transaction_id": r.TransactionID,
				"amount":         r.Amount,
			})
		}
	}

	result, err := runPredict(map[string]interface{}{"transactions": txns})
	if err != nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "org_id": orgID, "fraud": result["fraud"]})
}

// ── DB helpers ────────────────────────────────────────────────────────────────

func buildPredictInput(orgID string) (map[string]interface{}, error) {
	monthly, _ := getMonthlyRevenue(orgID)
	customers, _ := getCustomerStats(orgID)
	rows, _ := getLatestRows(orgID, 500)

	if len(monthly) == 0 { monthly = getDemoMonthly() }
	if len(customers) == 0 { customers = getDemoCustomers() }

	var txns []map[string]interface{}
	if len(rows) == 0 {
		txns = getDemoTransactions()
	} else {
		for _, r := range rows {
			txns = append(txns, map[string]interface{}{
				"transaction_id": r.TransactionID,
				"amount":         r.Amount,
			})
		}
	}

	return map[string]interface{}{
		"monthly":      monthly,
		"customers":    customers,
		"transactions": txns,
	}, nil
}

func getMonthlyRevenue(orgID string) ([]map[string]interface{}, error) {
	if db == nil { return nil, fmt.Errorf("no db") }
	rows, err := db.Query(`
		SELECT to_char(date_trunc('month', created_at), 'YYYY-MM') as month,
		       SUM(amount) as revenue
		FROM transactions WHERE org_id = $1
		GROUP BY 1 ORDER BY 1`, orgID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []map[string]interface{}
	for rows.Next() {
		var month string
		var revenue float64
		if err := rows.Scan(&month, &revenue); err != nil { continue }
		result = append(result, map[string]interface{}{"month": month, "revenue": revenue})
	}
	return result, nil
}

func getCustomerStats(orgID string) ([]map[string]interface{}, error) {
	if db == nil { return nil, fmt.Errorf("no db") }
	rows, err := db.Query(`
		SELECT customer_ref,
		       COUNT(*) as txn_count,
		       SUM(amount) as total_spend,
		       AVG(amount) as avg_amount,
		       EXTRACT(EPOCH FROM (NOW() - MAX(created_at)))/86400 as days_since_last,
		       AVG(CASE WHEN status='failed' THEN 1.0 ELSE 0.0 END) as failed_ratio
		FROM transactions WHERE org_id = $1
		GROUP BY customer_ref`, orgID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []map[string]interface{}
	for rows.Next() {
		var cref string
		var txnCount int
		var totalSpend, avgAmount, daysSince, failedRatio float64
		if err := rows.Scan(&cref, &txnCount, &totalSpend, &avgAmount, &daysSince, &failedRatio); err != nil { continue }
		result = append(result, map[string]interface{}{
			"customer_ref":   cref,
			"txn_count":      txnCount,
			"total_spend":    totalSpend,
			"avg_amount":     avgAmount,
			"days_since_last": daysSince,
			"failed_ratio":   failedRatio,
		})
	}
	return result, nil
}

// ── Demo data (used when no DB available) ─────────────────────────────────────

func getDemoMonthly() []map[string]interface{} {
	return []map[string]interface{}{
		{"month": "2025-01", "revenue": 152671.79},
		{"month": "2025-02", "revenue": 127804.21},
		{"month": "2025-03", "revenue": 157035.07},
		{"month": "2025-04", "revenue": 144573.63},
		{"month": "2025-05", "revenue": 117303.60},
		{"month": "2025-06", "revenue": 122352.97},
		{"month": "2025-07", "revenue": 129147.45},
		{"month": "2025-08", "revenue": 143422.08},
		{"month": "2025-09", "revenue": 107275.04},
		{"month": "2025-10", "revenue": 134128.34},
		{"month": "2025-11", "revenue": 105719.53},
		{"month": "2025-12", "revenue": 156466.16},
	}
}

func getDemoCustomers() []map[string]interface{} {
	return []map[string]interface{}{
		{"customer_ref": "C001", "txn_count": 15, "days_since_last": 5,  "avg_amount": 450, "total_spend": 6750,  "failed_ratio": 0.05},
		{"customer_ref": "C002", "txn_count": 3,  "days_since_last": 75, "avg_amount": 120, "total_spend": 360,   "failed_ratio": 0.33},
		{"customer_ref": "C003", "txn_count": 8,  "days_since_last": 12, "avg_amount": 280, "total_spend": 2240,  "failed_ratio": 0.10},
		{"customer_ref": "C004", "txn_count": 1,  "days_since_last": 90, "avg_amount": 45,  "total_spend": 45,    "failed_ratio": 0.50},
		{"customer_ref": "C005", "txn_count": 22, "days_since_last": 3,  "avg_amount": 890, "total_spend": 19580, "failed_ratio": 0.02},
	}
}

func getDemoTransactions() []map[string]interface{} {
	txns := make([]map[string]interface{}, 50)
	for i := range txns {
		txns[i] = map[string]interface{}{
			"transaction_id": fmt.Sprintf("T%04d", i),
			"amount":         100.0 + float64(i)*17.5,
		}
	}
	txns = append(txns,
		map[string]interface{}{"transaction_id": "T9999", "amount": 98500.0},
		map[string]interface{}{"transaction_id": "T9998", "amount": 87300.0},
	)
	return txns
}
