package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// ── Config ────────────────────────────────────────────────────────────────────

var (
	dbURL         = getEnv("DATABASE_URL", "postgresql://fintech_user:fintech_pass@localhost:5432/fintech?sslmode=disable")
	anthropicKey  = getEnv("ANTHROPIC_API_KEY", "")
	anthropicModel = getEnv("ANTHROPIC_MODEL", "claude-sonnet-4-20250514")
	port          = getEnv("PORT", "8085")
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
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err = db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("✓ Connected to PostgreSQL")
}

// ── Models ────────────────────────────────────────────────────────────────────

type MonthlySummary struct {
	Month   string  `json:"month"`
	Revenue float64 `json:"revenue"`
	Volume  int     `json:"volume"`
}

type CategoryBreakdown struct {
	Category  string  `json:"category"`
	Revenue   float64 `json:"revenue"`
	Volume    int     `json:"volume"`
	AvgAmount float64 `json:"avg_amount"`
}

type InsightRequest struct {
	OrgID string `json:"org_id"`
	Type  string `json:"type"` // summary, anomaly, recommendation
}

type Insight struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}

// ── Anthropic API ─────────────────────────────────────────────────────────────

type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
	System    string             `json:"system"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type InsightOutput struct {
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Confidence float64 `json:"confidence"`
}

func generateInsight(orgID, insightType string, monthly []MonthlySummary, categories []CategoryBreakdown) (*InsightOutput, error) {
	if anthropicKey == "" {
		return &InsightOutput{
			Title:      "AI insights unavailable",
			Content:    "Set ANTHROPIC_API_KEY to enable AI-generated insights.",
			Confidence: 0,
		}, nil
	}

	// Build data context
	dataCtx := fmt.Sprintf("Monthly revenue data:\n")
	for _, m := range monthly {
		dataCtx += fmt.Sprintf("  %s: CAD $%.2f (%d transactions)\n", m.Month, m.Revenue, m.Volume)
	}
	dataCtx += "\nCategory breakdown:\n"
	for _, c := range categories {
		dataCtx += fmt.Sprintf("  %s: CAD $%.2f (%d txns, avg $%.2f)\n", c.Category, c.Revenue, c.Volume, c.AvgAmount)
	}

	// Build prompt based on insight type
	prompts := map[string]string{
		"summary": "Provide a concise executive summary of this organization's financial transaction data. Highlight the most important trends.",
		"anomaly": "Analyze this transaction data for anomalies, unusual patterns, or outliers that a finance team should be aware of.",
		"recommendation": "Based on this transaction data, provide 2-3 actionable recommendations to optimize spending or improve financial health.",
	}
	userPrompt, ok := prompts[insightType]
	if !ok {
		userPrompt = prompts["summary"]
	}

	reqBody := AnthropicRequest{
		Model:     anthropicModel,
		MaxTokens: 500,
		System: `You are a financial analytics AI assistant for a FinTech SaaS platform. 
Analyze transaction data and provide clear, actionable insights for business users.
Respond ONLY with a valid JSON object with exactly these fields:
{"title": "short title under 60 chars", "content": "insight text 2-4 sentences", "confidence": 0.0-1.0}
No markdown, no explanation outside the JSON.`,
		Messages: []AnthropicMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf("%s\n\nData:\n%s", userPrompt, dataCtx),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", anthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic error %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse anthropic response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from anthropic")
	}

	var output InsightOutput
	if err := json.Unmarshal([]byte(anthropicResp.Content[0].Text), &output); err != nil {
		return nil, fmt.Errorf("failed to parse insight JSON: %w", err)
	}

	return &output, nil
}

// ── Data fetching ─────────────────────────────────────────────────────────────

func fetchMonthlyData(orgID string) ([]MonthlySummary, error) {
	rows, err := db.Query(`
		SELECT txn_month, SUM(amount), COUNT(*)
		FROM transactions
		WHERE org_id = $1 AND status = 'completed'
		GROUP BY txn_month
		ORDER BY txn_month DESC
		LIMIT 12
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MonthlySummary
	for rows.Next() {
		var m MonthlySummary
		if err := rows.Scan(&m.Month, &m.Revenue, &m.Volume); err != nil {
			continue
		}
		results = append(results, m)
	}
	return results, nil
}

func fetchCategoryData(orgID string) ([]CategoryBreakdown, error) {
	rows, err := db.Query(`
		SELECT category, SUM(amount), COUNT(*), AVG(amount)
		FROM transactions
		WHERE org_id = $1 AND status = 'completed'
		GROUP BY category
		ORDER BY SUM(amount) DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CategoryBreakdown
	for rows.Next() {
		var c CategoryBreakdown
		if err := rows.Scan(&c.Category, &c.Revenue, &c.Volume, &c.AvgAmount); err != nil {
			continue
		}
		results = append(results, c)
	}
	return results, nil
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// POST /insights/generate
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req InsightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.OrgID == "" {
		jsonError(w, "org_id is required", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		req.Type = "summary"
	}

	// Fetch data for this org
	monthly, err := fetchMonthlyData(req.OrgID)
	if err != nil {
		jsonError(w, "failed to fetch monthly data", http.StatusInternalServerError)
		return
	}
	categories, err := fetchCategoryData(req.OrgID)
	if err != nil {
		jsonError(w, "failed to fetch category data", http.StatusInternalServerError)
		return
	}

	if len(monthly) == 0 {
		jsonError(w, "no transaction data found for this organization", http.StatusNotFound)
		return
	}

	// Generate insight via Anthropic
	output, err := generateInsight(req.OrgID, req.Type, monthly, categories)
	if err != nil {
		log.Printf("AI generation failed: %v", err)
		jsonError(w, "failed to generate insight", http.StatusInternalServerError)
		return
	}

	// Store in DB
	var insightID string
	err = db.QueryRow(`
		INSERT INTO ai_insights (org_id, type, title, content, confidence)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, req.OrgID, req.Type, output.Title, output.Content, output.Confidence).Scan(&insightID)
	if err != nil {
		log.Printf("failed to store insight: %v", err)
	}

	log.Printf("insight generated org=%s type=%s", req.OrgID, req.Type)
	jsonResponse(w, http.StatusCreated, Insight{
		ID:         insightID,
		OrgID:      req.OrgID,
		Type:       req.Type,
		Title:      output.Title,
		Content:    output.Content,
		Confidence: output.Confidence,
		CreatedAt:  time.Now(),
	})
}

// GET /insights?org_id=xxx
func handleList(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		jsonError(w, "org_id query param required", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT id, org_id, type, title, content, confidence, created_at
		FROM ai_insights
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`, orgID)
	if err != nil {
		jsonError(w, "failed to fetch insights", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var insights []Insight
	for rows.Next() {
		var i Insight
		var confidence sql.NullFloat64
		if err := rows.Scan(&i.ID, &i.OrgID, &i.Type, &i.Title, &i.Content, &confidence, &i.CreatedAt); err != nil {
			continue
		}
		if confidence.Valid {
			i.Confidence = confidence.Float64
		}
		insights = append(insights, i)
	}

	if insights == nil {
		insights = []Insight{}
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"insights": insights,
		"total":    len(insights),
	})
}

// GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	aiStatus := "ready"
	if anthropicKey == "" {
		aiStatus = "no api key — set ANTHROPIC_API_KEY"
	}
	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "ai",
		"ai":      aiStatus,
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
	initDB()
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/insights/generate", handleGenerate)
	mux.HandleFunc("/insights", handleList)

	log.Printf("✓ AI service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
