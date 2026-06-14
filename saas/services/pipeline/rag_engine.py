"""
Phase 4 — RAG AI Insights Engine
FinTech SaaS Pipeline

Flow:
  1. Context builder  — pulls dbt model data into structured context
  2. RAG retrieval    — finds relevant data for the question
  3. Claude API       — generates grounded answer
  4. Executive report — auto-generates summary
"""

import json
import sys
import os
import urllib.request
import urllib.error
from datetime import datetime

# ── Real dbt data context ─────────────────────────────────────────────────────

FINTECH_CONTEXT = {
    "monthly_revenue": [
        {"month": "2025-01", "revenue": 152671.79, "transactions": 312, "customers": 153},
        {"month": "2025-02", "revenue": 127804.21, "transactions": 244, "customers": 141},
        {"month": "2025-03", "revenue": 157035.07, "transactions": 307, "customers": 156},
        {"month": "2025-04", "revenue": 144573.63, "transactions": 271, "customers": 153},
        {"month": "2025-05", "revenue": 117303.60, "transactions": 289, "customers": 147},
        {"month": "2025-06", "revenue": 122352.97, "transactions": 257, "customers": 139},
        {"month": "2025-07", "revenue": 129147.45, "transactions": 296, "customers": 151},
        {"month": "2025-08", "revenue": 143422.08, "transactions": 287, "customers": 155},
        {"month": "2025-09", "revenue": 107275.04, "transactions": 272, "customers": 149},
        {"month": "2025-10", "revenue": 134128.34, "transactions": 285, "customers": 157},
        {"month": "2025-11", "revenue": 105719.53, "transactions": 248, "customers": 146},
        {"month": "2025-12", "revenue": 156466.16, "transactions": 305, "customers": 168},
    ],
    "revenue_by_category": [
        {"category": "transfers",     "revenue": 880047.85, "transactions": 350, "avg": 2514.42},
        {"category": "travel",        "revenue": 322604.38, "transactions": 329, "avg": 980.56},
        {"category": "shopping",      "revenue": 130442.78, "transactions": 333, "avg": 391.72},
        {"category": "healthcare",    "revenue":  91624.64, "transactions": 341, "avg": 268.69},
        {"category": "groceries",     "revenue":  56695.58, "transactions": 378, "avg": 149.99},
        {"category": "utilities",     "revenue":  40951.32, "transactions": 298, "avg": 137.42},
        {"category": "entertainment", "revenue":  31807.80, "transactions": 324, "avg":  98.17},
        {"category": "food_and_drink","revenue":  20719.72, "transactions": 344, "avg":  60.23},
        {"category": "transport",     "revenue":  14540.99, "transactions": 349, "avg":  41.66},
        {"category": "subscriptions", "revenue":   8464.81, "transactions": 327, "avg":  25.89},
    ],
    "revenue_by_province": [
        {"province": "SK", "revenue": 360187.06, "transactions": 737, "customers": 43},
        {"province": "ON", "revenue": 329424.24, "transactions": 663, "customers": 38},
        {"province": "AB", "revenue": 253216.97, "transactions": 502, "customers": 29},
        {"province": "MB", "revenue": 239089.38, "transactions": 535, "customers": 34},
        {"province": "BC", "revenue": 221273.04, "transactions": 507, "customers": 31},
        {"province": "QC", "revenue": 194709.18, "transactions": 429, "customers": 25},
    ],
    "status_summary": [
        {"status": "completed", "count": 4162, "pct": 83.24},
        {"status": "pending",   "count": 625,  "pct": 12.50},
        {"status": "failed",    "count": 213,  "pct": 4.26},
    ],
    "totals": {
        "total_revenue": 1597900.06,
        "total_transactions": 5000,
        "avg_monthly_revenue": 133158.34,
        "peak_month": "2025-03",
        "peak_revenue": 157035.07,
        "lowest_month": "2025-11",
        "lowest_revenue": 105719.53,
        "top_category": "transfers",
        "top_province": "SK",
        "completion_rate": 83.24,
    }
}

# ── Context builder ───────────────────────────────────────────────────────────

def build_context(question: str) -> str:
    """Build relevant context string from dbt data based on the question."""
    q = question.lower()
    ctx_parts = []

    # Always include totals
    t = FINTECH_CONTEXT["totals"]
    ctx_parts.append(f"""
OVERALL PERFORMANCE (2025):
- Total revenue: CAD ${t['total_revenue']:,.2f}
- Total transactions: {t['total_transactions']:,}
- Average monthly revenue: CAD ${t['avg_monthly_revenue']:,.2f}
- Peak month: {t['peak_month']} (${t['peak_revenue']:,.2f})
- Lowest month: {t['lowest_month']} (${t['lowest_revenue']:,.2f})
- Transaction completion rate: {t['completion_rate']}%
- Top category by revenue: {t['top_category']}
- Top province by revenue: {t['top_province']}
""")

    # Monthly data for trend/time questions
    if any(w in q for w in ["month", "trend", "when", "september", "drop", "peak", "low", "high", "quarter", "season"]):
        monthly = FINTECH_CONTEXT["monthly_revenue"]
        ctx_parts.append("MONTHLY REVENUE DATA (2025):")
        for m in monthly:
            ctx_parts.append(f"  {m['month']}: ${m['revenue']:,.2f} | {m['transactions']} txns | {m['customers']} customers")

    # Category data
    if any(w in q for w in ["category", "transfer", "travel", "shopping", "grocery", "health", "entertain", "food", "transport", "subscription", "type"]):
        ctx_parts.append("\nREVENUE BY CATEGORY:")
        for c in FINTECH_CONTEXT["revenue_by_category"]:
            ctx_parts.append(f"  {c['category']}: ${c['revenue']:,.2f} ({c['transactions']} txns, avg ${c['avg']:,.2f})")

    # Province data
    if any(w in q for w in ["province", "region", "sk", "on", "ab", "mb", "bc", "qc", "ontario", "alberta", "british columbia", "quebec", "manitoba", "saskatchewan"]):
        ctx_parts.append("\nREVENUE BY PROVINCE:")
        for p in FINTECH_CONTEXT["revenue_by_province"]:
            ctx_parts.append(f"  {p['province']}: ${p['revenue']:,.2f} ({p['transactions']} txns, {p['customers']} customers)")

    # Status data
    if any(w in q for w in ["fail", "pending", "complet", "success", "status", "error"]):
        ctx_parts.append("\nTRANSACTION STATUS:")
        for s in FINTECH_CONTEXT["status_summary"]:
            ctx_parts.append(f"  {s['status']}: {s['count']:,} ({s['pct']}%)")

    return "\n".join(ctx_parts)

# ── Claude API call ───────────────────────────────────────────────────────────

def ask_claude(question: str, context: str, api_key: str) -> str:
    """Call Claude API with RAG context."""

    system_prompt = """You are a financial data analyst AI for a Canadian FinTech SaaS platform.
You answer questions about business data using ONLY the provided context.
Be specific, cite exact numbers from the data, and give actionable insights.
Keep answers concise but data-rich. Always mention specific CAD amounts and percentages.
If the data doesn't contain enough information to answer, say so clearly."""

    user_message = f"""Based on this FinTech business data:

{context}

Question: {question}

Provide a specific, data-driven answer with exact numbers. Include 1-2 actionable recommendations."""

    payload = json.dumps({
        "model": "claude-sonnet-4-6",
        "max_tokens": 500,
        "system": system_prompt,
        "messages": [{"role": "user", "content": user_message}]
    }).encode("utf-8")

    req = urllib.request.Request(
        "https://api.anthropic.com/v1/messages",
        data=payload,
        headers={
            "Content-Type": "application/json",
            "x-api-key": api_key,
            "anthropic-version": "2023-06-01",
        },
        method="POST"
    )

    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return data["content"][0]["text"]
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8")
        raise RuntimeError(f"Claude API error {e.code}: {body}")

# ── Executive report generator ────────────────────────────────────────────────

def generate_exec_report(api_key: str) -> dict:
    """Auto-generate executive summary from all dbt data."""
    t = FINTECH_CONTEXT["totals"]
    monthly = FINTECH_CONTEXT["monthly_revenue"]

    # Calculate MoM change
    mom_changes = []
    for i in range(1, len(monthly)):
        prev = monthly[i-1]["revenue"]
        curr = monthly[i]["revenue"]
        mom_changes.append({
            "month": monthly[i]["month"],
            "change_pct": round((curr - prev) / prev * 100, 1)
        })

    context = build_context("monthly trend category province performance summary")

    prompt = f"""Generate a brief executive summary (3-4 sentences) for this FinTech platform's 2025 performance.
Focus on: total revenue, key trends, top performers, and 2 specific recommendations.

{context}

Month-over-month changes: {json.dumps(mom_changes)}"""

    summary = ask_claude(prompt, context, api_key)

    return {
        "generated_at": datetime.utcnow().isoformat() + "Z",
        "period": "2025 Full Year",
        "total_revenue": t["total_revenue"],
        "total_transactions": t["total_transactions"],
        "completion_rate": t["completion_rate"],
        "peak_month": t["peak_month"],
        "top_category": t["top_category"],
        "top_province": t["top_province"],
        "ai_summary": summary,
    }

# ── Main entrypoint ───────────────────────────────────────────────────────────

def run(data: dict) -> dict:
    api_key = data.get("api_key") or os.environ.get("ANTHROPIC_API_KEY", "")

    if data.get("mode") == "report":
        if not api_key:
            return {"error": "ANTHROPIC_API_KEY required for AI report"}
        return {"report": generate_exec_report(api_key)}

    question = data.get("question", "")
    if not question:
        return {"error": "question required"}

    context = build_context(question)

    if not api_key:
        # Return context only (no AI key)
        return {
            "question": question,
            "answer": "AI insights require ANTHROPIC_API_KEY. Here is the raw data context:",
            "context": context,
            "ai_available": False,
        }

    answer = ask_claude(question, context, api_key)
    return {
        "question":     question,
        "answer":       answer,
        "context_used": len(context),
        "ai_available": True,
    }


if __name__ == "__main__":
    if "--stdin" in sys.argv:
        raw = sys.stdin.read()
        data = json.loads(raw)
        print(json.dumps(run(data)))
    else:
        # Test without API key — shows context retrieval
        test_questions = [
            "Why did revenue drop in September?",
            "Which province is performing best?",
            "What is our top revenue category?",
        ]
        for q in test_questions:
            ctx = build_context(q)
            print(f"\nQ: {q}")
            print(f"Context length: {len(ctx)} chars")
            print(f"Context preview: {ctx[:200]}...")
            print("---")
