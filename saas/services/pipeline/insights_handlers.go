package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// ── Insights handlers — Phase 4 RAG AI ───────────────────────────────────────

func runRAG(input map[string]interface{}) (map[string]interface{}, error) {
	// Inject API key from env
	input["api_key"] = os.Getenv("ANTHROPIC_API_KEY")

	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	cmd := exec.Command("python3", "rag_engine.py", "--stdin")
	cmd.Stdin = bytes.NewReader(data)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("rag_engine stderr: %s", stderr.String())
		return nil, fmt.Errorf("RAG engine failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return result, nil
}

// POST /insights/ask
// Body: {"question": "Why did revenue drop in September?"}
func handleInsightsAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		jsonError(w, "use POST with JSON body or GET with ?q= param", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	var question string

	// Support both GET ?q= and POST JSON
	if r.Method == http.MethodGet {
		question = r.URL.Query().Get("q")
	} else {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		question = body["question"]
	}

	if question == "" {
		jsonError(w, "question required — use GET ?q=your+question or POST {\"question\":\"...\"}", http.StatusBadRequest)
		return
	}

	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = r.Header.Get("X-Org-ID")
	}
	if orgID == "" {
		orgID = "demo"
	}

	result, err := runRAG(map[string]interface{}{
		"mode":     "ask",
		"question": question,
		"org_id":   orgID,
	})
	if err != nil {
		log.Printf("insights/ask error: %v", err)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"ok":       false,
			"question": question,
			"error":    "AI insights engine unavailable",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok":          true,
		"org_id":      orgID,
		"duration_ms": time.Since(start).Milliseconds(),
		"result":      result,
	})
}

// GET /insights/report
func handleInsightsReport(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = "demo"
	}

	result, err := runRAG(map[string]interface{}{
		"mode":   "report",
		"org_id": orgID,
	})
	if err != nil {
		log.Printf("insights/report error: %v", err)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"ok":    false,
			"error": "report generation unavailable",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok":          true,
		"org_id":      orgID,
		"duration_ms": time.Since(start).Milliseconds(),
		"result":      result,
	})
}

// GET /insights/context?q=question
// Returns the raw context without AI (useful for debugging)
func handleInsightsContext(w http.ResponseWriter, r *http.Request) {
	question := r.URL.Query().Get("q")
	if question == "" {
		question = "monthly revenue trend category province"
	}

	result, err := runRAG(map[string]interface{}{
		"mode":     "ask",
		"question": question,
		"api_key":  "", // No API key = returns context only
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"result": result,
	})
}
