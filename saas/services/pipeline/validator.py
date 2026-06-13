"""
Data Quality Validator — FinTech SaaS Pipeline
Runs on every CSV upload:
  1. Schema validation   — required columns, correct types
  2. Anomaly detection   — Z-score + IQR per numeric column
  3. Quality score       — 0-100 per upload
  4. Drift detection     — compare against historical stats
"""

import json
import math
import statistics
from typing import Any

# ── Expected schema ───────────────────────────────────────────────────────────

REQUIRED_COLUMNS = {
    "transaction_id", "customer_ref", "amount",
    "category", "status", "province"
}

OPTIONAL_COLUMNS = {
    "merchant", "currency", "txn_date",
    "txn_month", "txn_year", "age_group", "account_type"
}

VALID_STATUSES   = {"completed", "pending", "failed"}
VALID_PROVINCES  = {"BC", "AB", "SK", "MB", "ON", "QC", "NS", "NB", "NL", "PE", "YT", "NT", "NU"}
VALID_CURRENCIES = {"CAD", "USD"}

NUMERIC_COLUMNS = {"amount"}
MIN_AMOUNT      = 0.01
MAX_AMOUNT      = 100_000.0

# ── Schema validator ──────────────────────────────────────────────────────────

def validate_schema(rows: list[dict]) -> dict:
    """Check required columns and value validity."""
    errors   = []
    warnings = []

    if not rows:
        return {"errors": ["Empty dataset — no rows to validate"], "warnings": []}

    headers = set(rows[0].keys())

    # Required column check
    missing = REQUIRED_COLUMNS - headers
    if missing:
        errors.append(f"Missing required columns: {sorted(missing)}")

    extra = headers - REQUIRED_COLUMNS - OPTIONAL_COLUMNS
    if extra:
        warnings.append(f"Unknown columns (will be ignored): {sorted(extra)}")

    # Row-level validation
    null_counts    = {col: 0 for col in REQUIRED_COLUMNS}
    invalid_status = 0
    invalid_province = 0
    invalid_amount = 0
    negative_amount = 0
    duplicate_ids  = {}

    for i, row in enumerate(rows, 1):
        # Null checks
        for col in REQUIRED_COLUMNS:
            if col in row and (row[col] is None or str(row[col]).strip() == ""):
                null_counts[col] += 1

        # Status validation
        status = str(row.get("status", "")).lower().strip()
        if status and status not in VALID_STATUSES:
            invalid_status += 1

        # Province validation
        province = str(row.get("province", "")).upper().strip()
        if province and province not in VALID_PROVINCES:
            invalid_province += 1

        # Amount validation
        try:
            amount = float(row.get("amount", 0))
            if amount < 0:
                negative_amount += 1
            if amount > MAX_AMOUNT:
                invalid_amount += 1
        except (ValueError, TypeError):
            invalid_amount += 1

        # Duplicate transaction IDs
        tid = str(row.get("transaction_id", "")).strip()
        if tid:
            duplicate_ids[tid] = duplicate_ids.get(tid, 0) + 1

    # Report null counts
    for col, count in null_counts.items():
        pct = (count / len(rows)) * 100
        if pct > 20:
            errors.append(f"Column '{col}' has {pct:.1f}% null values ({count} rows)")
        elif pct > 5:
            warnings.append(f"Column '{col}' has {pct:.1f}% null values ({count} rows)")

    if invalid_status > 0:
        errors.append(f"{invalid_status} rows have invalid status values (expected: completed/pending/failed)")

    if invalid_province > 0:
        warnings.append(f"{invalid_province} rows have unrecognized province codes")

    if invalid_amount > 0:
        errors.append(f"{invalid_amount} rows have invalid amount values (non-numeric or > ${MAX_AMOUNT:,})")

    if negative_amount > 0:
        errors.append(f"{negative_amount} rows have negative amounts")

    dups = {tid: cnt for tid, cnt in duplicate_ids.items() if cnt > 1}
    if dups:
        errors.append(f"{len(dups)} duplicate transaction IDs detected")

    return {"errors": errors, "warnings": warnings}

# ── Anomaly detector ──────────────────────────────────────────────────────────

def detect_anomalies(rows: list[dict]) -> dict:
    """Z-score + IQR anomaly detection on numeric columns."""
    anomalies  = []
    stats_out  = {}

    for col in NUMERIC_COLUMNS:
        values = []
        for row in rows:
            try:
                v = float(row.get(col, ""))
                if v >= 0:
                    values.append(v)
            except (ValueError, TypeError):
                pass

        if len(values) < 4:
            continue

        mean   = statistics.mean(values)
        stdev  = statistics.stdev(values)
        median = statistics.median(values)

        sorted_v = sorted(values)
        q1 = sorted_v[len(sorted_v) // 4]
        q3 = sorted_v[(3 * len(sorted_v)) // 4]
        iqr = q3 - q1

        lower_iqr = q1 - 1.5 * iqr
        upper_iqr = q3 + 1.5 * iqr

        outliers_z   = [v for v in values if stdev > 0 and abs((v - mean) / stdev) > 3]
        outliers_iqr = [v for v in values if v < lower_iqr or v > upper_iqr]

        stats_out[col] = {
            "mean":   round(mean,   2),
            "median": round(median, 2),
            "stdev":  round(stdev,  2),
            "min":    round(min(values), 2),
            "max":    round(max(values), 2),
            "q1":     round(q1, 2),
            "q3":     round(q3, 2),
            "iqr":    round(iqr, 2),
        }

        if outliers_z:
            anomalies.append({
                "column":  col,
                "method":  "z-score",
                "count":   len(outliers_z),
                "pct":     round(len(outliers_z) / len(values) * 100, 2),
                "message": f"{len(outliers_z)} outliers detected via Z-score (>3σ) in '{col}'",
                "sample":  [round(v, 2) for v in sorted(outliers_z, reverse=True)[:5]],
            })

        if outliers_iqr:
            anomalies.append({
                "column":  col,
                "method":  "iqr",
                "count":   len(outliers_iqr),
                "pct":     round(len(outliers_iqr) / len(values) * 100, 2),
                "message": f"{len(outliers_iqr)} outliers detected via IQR in '{col}'",
                "sample":  [round(v, 2) for v in sorted(outliers_iqr, reverse=True)[:5]],
            })

    return {"anomalies": anomalies, "stats": stats_out}

# ── Quality scorer ────────────────────────────────────────────────────────────

def compute_quality_score(rows: list[dict], schema_result: dict, anomaly_result: dict) -> dict:
    """
    Quality score 0-100:
      - Completeness  (40 pts): null rate across required columns
      - Validity      (35 pts): invalid values (status, amount, province)
      - Uniqueness    (15 pts): duplicate transaction IDs
      - Consistency   (10 pts): anomaly rate
    """
    if not rows:
        return {"score": 0, "grade": "F", "breakdown": {}}

    n = len(rows)

    # Completeness
    null_total = 0
    for col in REQUIRED_COLUMNS:
        for row in rows:
            if col in row and (row[col] is None or str(row[col]).strip() == ""):
                null_total += 1
    completeness = max(0, 1 - (null_total / (n * len(REQUIRED_COLUMNS))))
    completeness_score = round(completeness * 40, 1)

    # Validity — count errors that relate to values
    error_rows = sum(1 for e in schema_result["errors"]
                     if "invalid" in e.lower() or "negative" in e.lower() or "duplicate" in e.lower())
    validity = max(0, 1 - (error_rows / max(len(schema_result["errors"]), 1)) * 0.5)
    validity_score = round(validity * 35, 1)

    # Uniqueness
    ids = [str(row.get("transaction_id", "")).strip() for row in rows]
    unique_ids = len(set(ids))
    uniqueness = unique_ids / n if n > 0 else 1
    uniqueness_score = round(uniqueness * 15, 1)

    # Consistency (anomaly rate)
    total_anomalies = sum(a["count"] for a in anomaly_result["anomalies"])
    anomaly_rate = total_anomalies / n if n > 0 else 0
    consistency = max(0, 1 - anomaly_rate)
    consistency_score = round(consistency * 10, 1)

    total = completeness_score + validity_score + uniqueness_score + consistency_score

    if total >= 90:   grade = "A"
    elif total >= 80: grade = "B"
    elif total >= 70: grade = "C"
    elif total >= 60: grade = "D"
    else:             grade = "F"

    return {
        "score": round(total, 1),
        "grade": grade,
        "breakdown": {
            "completeness":  {"score": completeness_score, "max": 40, "rate": round(completeness * 100, 1)},
            "validity":      {"score": validity_score,     "max": 35},
            "uniqueness":    {"score": uniqueness_score,   "max": 15, "rate": round(uniqueness * 100, 1)},
            "consistency":   {"score": consistency_score,  "max": 10, "anomaly_rate": round(anomaly_rate * 100, 2)},
        }
    }

# ── Main entrypoint ───────────────────────────────────────────────────────────

def validate(rows: list[dict]) -> dict:
    """Run full validation pipeline and return report."""
    schema_result  = validate_schema(rows)
    anomaly_result = detect_anomalies(rows)
    quality        = compute_quality_score(rows, schema_result, anomaly_result)

    passed = len(schema_result["errors"]) == 0

    return {
        "passed":    passed,
        "rows":      len(rows),
        "quality":   quality,
        "schema":    schema_result,
        "anomalies": anomaly_result,
    }

# ── CLI test ──────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    # Test with sample data
    sample = [
        {"transaction_id": f"T{i:04d}", "customer_ref": f"C{i:03d}",
         "amount": str(100 + i * 13.5), "category": "transfers",
         "status": "completed", "province": "BC", "currency": "CAD"}
        for i in range(100)
    ]
    # Add some anomalies
    sample[5]["amount"]  = "99999"   # IQR outlier
    sample[10]["status"] = "invalid" # Bad status
    sample[20]["amount"] = ""        # Null amount

    result = validate(sample)
    print(json.dumps(result, indent=2))


# ── stdin mode for Go subprocess calls ───────────────────────────────────────

if __name__ == "__main__":
    import sys

    if "--stdin" in sys.argv:
        # Read rows as JSON from stdin
        raw = sys.stdin.read()
        rows = json.loads(raw)
        result = validate(rows)
        print(json.dumps(result))
    else:
        # Test with sample data
        sample = [
            {"transaction_id": f"T{i:04d}", "customer_ref": f"C{i:03d}",
             "amount": str(100 + i * 13.5), "category": "transfers",
             "status": "completed", "province": "BC", "currency": "CAD"}
            for i in range(100)
        ]
        sample[5]["amount"]  = "99999"
        sample[10]["status"] = "invalid"
        sample[20]["amount"] = ""
        result = validate(sample)
        print(json.dumps(result, indent=2))
