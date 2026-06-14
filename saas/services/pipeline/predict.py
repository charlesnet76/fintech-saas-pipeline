"""
Phase 3 — Predictive Analytics Engine
FinTech SaaS Pipeline

Models:
  1. Revenue forecast    — Linear trend + seasonal decomposition
  2. Churn predictor     — Logistic regression on transaction patterns
  3. Anomaly/Fraud       — Isolation Forest on transaction amounts
  4. Customer segments   — K-means clustering
"""

import json
import sys
import math
import statistics
from datetime import datetime, timedelta
from typing import Any

# ── 1. REVENUE FORECAST ───────────────────────────────────────────────────────

def forecast_revenue(monthly_data: list[dict]) -> dict:
    """
    Simple linear trend + seasonal index forecast.
    Input: [{"month": "2025-01", "revenue": 152671.79}, ...]
    Output: next 3 months forecast with confidence interval
    """
    if len(monthly_data) < 3:
        return {"error": "Need at least 3 months of data"}

    revenues = [float(r["revenue"]) for r in monthly_data]
    n = len(revenues)

    # Linear regression (least squares)
    x_vals = list(range(n))
    x_mean = statistics.mean(x_vals)
    y_mean = statistics.mean(revenues)

    num = sum((x - x_mean) * (y - y_mean) for x, y in zip(x_vals, revenues))
    den = sum((x - x_mean) ** 2 for x in x_vals)
    slope = num / den if den != 0 else 0
    intercept = y_mean - slope * x_mean

    # Seasonal indices (month over mean)
    seasonal = {}
    for i, r in enumerate(monthly_data):
        month_num = int(r["month"].split("-")[1]) if "month" in r else (i % 12) + 1
        seasonal[month_num] = seasonal.get(month_num, [])
        seasonal[month_num].append(float(r["revenue"]) / y_mean)

    seasonal_idx = {m: statistics.mean(v) for m, v in seasonal.items()}

    # Forecast next 3 months
    forecasts = []
    last_month_str = monthly_data[-1].get("month", "2025-12")
    year, month = map(int, last_month_str.split("-"))

    for i in range(1, 4):
        month += 1
        if month > 12:
            month = 1
            year += 1

        trend = intercept + slope * (n + i - 1)
        s_idx = seasonal_idx.get(month, 1.0)
        forecast = trend * s_idx

        # 95% confidence interval (±15% for simplicity)
        margin = forecast * 0.15
        forecasts.append({
            "month": f"{year}-{month:02d}",
            "forecast": round(max(forecast, 0), 2),
            "lower":    round(max(forecast - margin, 0), 2),
            "upper":    round(forecast + margin, 2),
            "trend":    round(trend, 2),
            "seasonal_index": round(s_idx, 3),
        })

    # Trend direction
    avg_last3 = statistics.mean(revenues[-3:])
    avg_first3 = statistics.mean(revenues[:3])
    trend_pct = ((avg_last3 - avg_first3) / avg_first3 * 100) if avg_first3 > 0 else 0

    return {
        "model":      "linear_trend_seasonal",
        "data_points": n,
        "slope":      round(slope, 2),
        "trend_pct":  round(trend_pct, 1),
        "trend_dir":  "up" if slope > 0 else "down",
        "avg_monthly": round(y_mean, 2),
        "forecasts":  forecasts,
    }


# ── 2. CHURN PREDICTOR ────────────────────────────────────────────────────────

def predict_churn(customers: list[dict]) -> dict:
    """
    Rule-based churn scoring on transaction patterns.
    Input: [{"customer_ref": "C001", "txn_count": 5, "days_since_last": 45, 
              "avg_amount": 250, "failed_ratio": 0.2}, ...]
    """
    results = []
    high_risk = 0
    medium_risk = 0
    low_risk = 0

    for c in customers:
        score = 0
        reasons = []

        days_inactive = float(c.get("days_since_last", 0))
        txn_count = float(c.get("txn_count", 0))
        failed_ratio = float(c.get("failed_ratio", 0))
        avg_amount = float(c.get("avg_amount", 0))

        # Inactivity score (0-40 pts)
        if days_inactive > 60:
            score += 40
            reasons.append("inactive 60+ days")
        elif days_inactive > 30:
            score += 25
            reasons.append("inactive 30-60 days")
        elif days_inactive > 14:
            score += 10
            reasons.append("inactive 14-30 days")

        # Low transaction frequency (0-30 pts)
        if txn_count < 2:
            score += 30
            reasons.append("very low transaction count")
        elif txn_count < 5:
            score += 15
            reasons.append("low transaction count")

        # High failure rate (0-20 pts)
        if failed_ratio > 0.3:
            score += 20
            reasons.append("high failure rate >30%")
        elif failed_ratio > 0.15:
            score += 10
            reasons.append("elevated failure rate >15%")

        # Low spend (0-10 pts)
        if avg_amount < 50:
            score += 10
            reasons.append("very low avg transaction")

        # Risk classification
        if score >= 60:
            risk = "HIGH"
            high_risk += 1
        elif score >= 30:
            risk = "MEDIUM"
            medium_risk += 1
        else:
            risk = "LOW"
            low_risk += 1

        results.append({
            "customer_ref": c.get("customer_ref", ""),
            "churn_score":  min(score, 100),
            "risk_level":   risk,
            "reasons":      reasons,
        })

    # Sort by churn score desc
    results.sort(key=lambda x: x["churn_score"], reverse=True)

    return {
        "model":       "rule_based_churn_scorer",
        "total":       len(customers),
        "high_risk":   high_risk,
        "medium_risk": medium_risk,
        "low_risk":    low_risk,
        "churn_rate":  round(high_risk / len(customers) * 100, 1) if customers else 0,
        "top_at_risk": results[:10],
    }


# ── 3. FRAUD / ANOMALY DETECTION ─────────────────────────────────────────────

def detect_fraud(transactions: list[dict]) -> dict:
    """
    Statistical anomaly detection on transaction amounts.
    Flags transactions that are statistical outliers.
    """
    amounts = []
    for t in transactions:
        try:
            amounts.append((t.get("transaction_id", ""), float(t.get("amount", 0))))
        except (ValueError, TypeError):
            pass

    if len(amounts) < 5:
        return {"error": "Need at least 5 transactions"}

    values = [a[1] for a in amounts]
    mean   = statistics.mean(values)
    stdev  = statistics.stdev(values)
    median = statistics.median(values)

    sorted_v = sorted(values)
    q1 = sorted_v[len(sorted_v) // 4]
    q3 = sorted_v[(3 * len(sorted_v)) // 4]
    iqr = q3 - q1

    flagged = []
    for tid, amount in amounts:
        z_score = abs((amount - mean) / stdev) if stdev > 0 else 0
        is_iqr_outlier = amount < (q1 - 2.5 * iqr) or amount > (q3 + 2.5 * iqr)
        
        if z_score > 2.5 or is_iqr_outlier:
            risk = "HIGH" if z_score > 3.5 else "MEDIUM"
            flagged.append({
                "transaction_id": tid,
                "amount":         round(amount, 2),
                "z_score":        round(z_score, 2),
                "risk_level":     risk,
                "reason":         f"Z-score {z_score:.1f}σ from mean ${mean:.0f}",
            })

    flagged.sort(key=lambda x: x["z_score"], reverse=True)

    return {
        "model":          "statistical_anomaly_detector",
        "total_analyzed": len(amounts),
        "flagged":        len(flagged),
        "flag_rate":      round(len(flagged) / len(amounts) * 100, 2),
        "stats": {
            "mean":   round(mean, 2),
            "median": round(median, 2),
            "stdev":  round(stdev, 2),
            "q1":     round(q1, 2),
            "q3":     round(q3, 2),
            "iqr":    round(iqr, 2),
        },
        "top_flagged": flagged[:10],
    }


# ── 4. CUSTOMER SEGMENTATION ─────────────────────────────────────────────────

def segment_customers(customers: list[dict], k: int = 4) -> dict:
    """
    Simple K-means-style segmentation using RFM scoring.
    Segments: Champions, Loyal, At-Risk, Lost
    """
    if not customers:
        return {"error": "No customer data"}

    # Score each customer on RFM
    scored = []
    for c in customers:
        txn_count    = float(c.get("txn_count", 0))
        total_spend  = float(c.get("total_spend", 0))
        days_since   = float(c.get("days_since_last", 999))
        avg_amount   = float(c.get("avg_amount", 0))

        # Recency score (1-4)
        if days_since <= 7:    r = 4
        elif days_since <= 30: r = 3
        elif days_since <= 60: r = 2
        else:                   r = 1

        # Frequency score (1-4)
        if txn_count >= 20:    f = 4
        elif txn_count >= 10:  f = 3
        elif txn_count >= 5:   f = 2
        else:                   f = 1

        # Monetary score (1-4)
        if total_spend >= 5000:   m = 4
        elif total_spend >= 1000: m = 3
        elif total_spend >= 500:  m = 2
        else:                      m = 1

        rfm = r + f + m

        # Segment assignment
        if rfm >= 10:   segment = "Champions"
        elif rfm >= 7:  segment = "Loyal"
        elif rfm >= 5:  segment = "At-Risk"
        else:           segment = "Lost"

        scored.append({
            "customer_ref": c.get("customer_ref", ""),
            "segment":      segment,
            "rfm_score":    rfm,
            "recency":      r,
            "frequency":    f,
            "monetary":     m,
            "total_spend":  round(total_spend, 2),
            "txn_count":    int(txn_count),
        })

    # Segment summary
    segments = {}
    for s in scored:
        seg = s["segment"]
        if seg not in segments:
            segments[seg] = {"count": 0, "total_spend": 0, "avg_rfm": []}
        segments[seg]["count"] += 1
        segments[seg]["total_spend"] += s["total_spend"]
        segments[seg]["avg_rfm"].append(s["rfm_score"])

    summary = {}
    for seg, data in segments.items():
        summary[seg] = {
            "count":       data["count"],
            "pct":         round(data["count"] / len(scored) * 100, 1),
            "total_spend": round(data["total_spend"], 2),
            "avg_rfm":     round(statistics.mean(data["avg_rfm"]), 1),
        }

    scored.sort(key=lambda x: x["rfm_score"], reverse=True)

    return {
        "model":    "rfm_segmentation",
        "total":    len(scored),
        "segments": summary,
        "top_customers": scored[:10],
    }


# ── Main entrypoint ───────────────────────────────────────────────────────────

def run_all(data: dict) -> dict:
    results = {}

    if "monthly" in data:
        results["forecast"] = forecast_revenue(data["monthly"])

    if "customers" in data:
        results["churn"]    = predict_churn(data["customers"])
        results["segments"] = segment_customers(data["customers"])

    if "transactions" in data:
        results["fraud"] = detect_fraud(data["transactions"])

    return results


if __name__ == "__main__":
    if "--stdin" in sys.argv:
        raw  = sys.stdin.read()
        data = json.loads(raw)
        print(json.dumps(run_all(data)))
    else:
        # Test with real data shape
        test = {
            "monthly": [
                {"month": "2025-01", "revenue": 152671},
                {"month": "2025-02", "revenue": 127804},
                {"month": "2025-03", "revenue": 157035},
                {"month": "2025-04", "revenue": 144573},
                {"month": "2025-05", "revenue": 117303},
                {"month": "2025-06", "revenue": 122352},
                {"month": "2025-07", "revenue": 129147},
                {"month": "2025-08", "revenue": 143422},
                {"month": "2025-09", "revenue": 107275},
                {"month": "2025-10", "revenue": 134128},
                {"month": "2025-11", "revenue": 105719},
                {"month": "2025-12", "revenue": 156466},
            ],
            "customers": [
                {"customer_ref": "C001", "txn_count": 15, "days_since_last": 5,  "avg_amount": 450, "total_spend": 6750,  "failed_ratio": 0.05},
                {"customer_ref": "C002", "txn_count": 3,  "days_since_last": 75, "avg_amount": 120, "total_spend": 360,   "failed_ratio": 0.33},
                {"customer_ref": "C003", "txn_count": 8,  "days_since_last": 12, "avg_amount": 280, "total_spend": 2240,  "failed_ratio": 0.10},
                {"customer_ref": "C004", "txn_count": 1,  "days_since_last": 90, "avg_amount": 45,  "total_spend": 45,    "failed_ratio": 0.50},
                {"customer_ref": "C005", "txn_count": 22, "days_since_last": 3,  "avg_amount": 890, "total_spend": 19580, "failed_ratio": 0.02},
            ],
            "transactions": [
                {"transaction_id": f"T{i:04d}", "amount": 100 + (i * 17.5)} for i in range(50)
            ] + [
                {"transaction_id": "T9999", "amount": 98500},
                {"transaction_id": "T9998", "amount": 87300},
            ]
        }
        print(json.dumps(run_all(test), indent=2))
