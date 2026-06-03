"""
Feature Store Backfill Job
==========================
Recomputes historical feature values for a date range and upserts
into the feature_values table. Runs weekly via GitHub Actions (Sunday 2am).

Usage:
    python -m backfill.run --feature all --start 7daysago --end today
    python -m backfill.run --feature tx_count_7d --start 2024-01-01 --end 2024-03-01
    python -m backfill.run --feature all --start 2024-01-01 --end today --dry-run
"""

from __future__ import annotations

import argparse
import json
import logging
import os
import sys
from datetime import date, datetime, timedelta
from typing import Generator

import psycopg2
import psycopg2.extras
from psycopg2.extensions import connection as PgConnection

# ── Logging ───────────────────────────────────────────────────────────────────

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s — %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S",
)
log = logging.getLogger("backfill")

# ── Config ────────────────────────────────────────────────────────────────────

BATCH_SIZE = int(os.getenv("BACKFILL_BATCH_SIZE", "500"))


# ── Date helpers ──────────────────────────────────────────────────────────────

def parse_date(value: str) -> date:
    """
    Parse flexible date strings:
      - 'today'       → today
      - '7daysago'    → today - 7 days
      - '30daysago'   → today - 30 days
      - 'YYYY-MM-DD'  → exact date
    """
    value = value.strip().lower()
    if value == "today":
        return date.today()
    if value.endswith("daysago"):
        days = int(value.replace("daysago", ""))
        return date.today() - timedelta(days=days)
    return date.fromisoformat(value)


def date_range(start: date, end: date) -> Generator[date, None, None]:
    """Yield each date from start to end inclusive."""
    current = start
    while current <= end:
        yield current
        current += timedelta(days=1)


# ── DB connection ─────────────────────────────────────────────────────────────

def get_conn() -> PgConnection:
    conn = psycopg2.connect(_get_database_url())
    conn.autocommit = False
    return conn


# ── Feature computers ─────────────────────────────────────────────────────────
# Each function computes feature values for all entities on a given date
# and returns a list of dicts ready for upsert into feature_values.
#
# Shape: [{"entity_id": str, "value": any, "valid_at": datetime}, ...]
#
# Add new features here — register them in FEATURE_REGISTRY below.

def compute_tx_count_7d(conn: PgConnection, as_of: date) -> list[dict]:
    """
    tx_count_7d: number of transactions per customer in the 7 days
    ending on as_of (inclusive). Classic rolling-window feature.
    """
    window_start = as_of - timedelta(days=6)
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute("""
            SELECT
                customer_id::TEXT AS entity_id,
                COUNT(*)          AS tx_count
            FROM raw_transactions
            WHERE txn_date BETWEEN %s AND %s
              AND status = 'completed'
            GROUP BY customer_id
        """, (window_start, as_of))
        rows = cur.fetchall()

    valid_at = datetime.combine(as_of, datetime.min.time())
    return [
        {
            "entity_id": row["entity_id"],
            "value": row["tx_count"],
            "valid_at": valid_at,
        }
        for row in rows
    ]


def compute_avg_tx_amount_30d(conn: PgConnection, as_of: date) -> list[dict]:
    """
    avg_tx_amount_30d: average transaction amount per customer
    in the 30 days ending on as_of. Used in risk scoring.
    """
    window_start = as_of - timedelta(days=29)
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute("""
            SELECT
                customer_id::TEXT    AS entity_id,
                ROUND(AVG(amount), 2) AS avg_amount
            FROM raw_transactions
            WHERE txn_date BETWEEN %s AND %s
              AND status = 'completed'
            GROUP BY customer_id
        """, (window_start, as_of))
        rows = cur.fetchall()

    valid_at = datetime.combine(as_of, datetime.min.time())
    return [
        {
            "entity_id": row["entity_id"],
            "value": float(row["avg_amount"]),
            "valid_at": valid_at,
        }
        for row in rows
    ]


def compute_failed_tx_ratio_7d(conn: PgConnection, as_of: date) -> list[dict]:
    """
    failed_tx_ratio_7d: ratio of failed transactions to total
    in the 7 days ending on as_of. High ratio = risk signal.
    """
    window_start = as_of - timedelta(days=6)
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute("""
            SELECT
                customer_id::TEXT AS entity_id,
                ROUND(
                    COUNT(*) FILTER (WHERE status = 'failed')::NUMERIC
                    / NULLIF(COUNT(*), 0), 4
                ) AS failed_ratio
            FROM raw_transactions
            WHERE txn_date BETWEEN %s AND %s
            GROUP BY customer_id
        """, (window_start, as_of))
        rows = cur.fetchall()

    valid_at = datetime.combine(as_of, datetime.min.time())
    return [
        {
            "entity_id": row["entity_id"],
            "value": float(row["failed_ratio"] or 0),
            "valid_at": valid_at,
        }
        for row in rows
    ]


def compute_top_category_30d(conn: PgConnection, as_of: date) -> list[dict]:
    """
    top_category_30d: the customer's most frequent spend category
    in the 30 days ending on as_of. Useful for personalization.
    """
    window_start = as_of - timedelta(days=29)
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute("""
            SELECT DISTINCT ON (customer_id)
                customer_id::TEXT AS entity_id,
                category          AS top_category
            FROM raw_transactions
            WHERE txn_date BETWEEN %s AND %s
              AND status = 'completed'
            GROUP BY customer_id, category
            ORDER BY customer_id, COUNT(*) DESC
        """, (window_start, as_of))
        rows = cur.fetchall()

    valid_at = datetime.combine(as_of, datetime.min.time())
    return [
        {
            "entity_id": row["entity_id"],
            "value": row["top_category"],
            "valid_at": valid_at,
        }
        for row in rows
    ]


# ── Feature registry ──────────────────────────────────────────────────────────
# Maps feature name → compute function.
# Add new features here — the backfill loop picks them up automatically.

FEATURE_COMPUTERS: dict[str, callable] = {
    "tx_count_7d":          compute_tx_count_7d,
    "avg_tx_amount_30d":    compute_avg_tx_amount_30d,
    "failed_tx_ratio_7d":   compute_failed_tx_ratio_7d,
    "top_category_30d":     compute_top_category_30d,
}


# ── Upsert ────────────────────────────────────────────────────────────────────

def resolve_feature_id(conn: PgConnection, feature_name: str, org_id: str) -> str | None:
    """Look up feature_id from the feature_registry for a given org."""
    with conn.cursor() as cur:
        cur.execute("""
            SELECT id FROM feature_registry
            WHERE name = %s AND org_id = %s AND deleted_at IS NULL
        """, (feature_name, org_id))
        row = cur.fetchone()
    return row[0] if row else None


def upsert_feature_values(
    conn: PgConnection,
    feature_id: str,
    org_id: str,
    rows: list[dict],
    dry_run: bool = False,
) -> int:
    """
    Batch-upsert feature values. Uses ON CONFLICT DO NOTHING —
    safe to re-run for the same date range (idempotent).
    Returns number of rows upserted.
    """
    if not rows:
        return 0

    if dry_run:
        log.info("  [dry-run] would upsert %d rows", len(rows))
        return len(rows)

    inserted = 0
    with conn.cursor() as cur:
        for i in range(0, len(rows), BATCH_SIZE):
            batch = rows[i : i + BATCH_SIZE]
            psycopg2.extras.execute_values(
                cur,
                """
                INSERT INTO feature_values
                    (org_id, feature_id, entity_id, value, valid_at)
                VALUES %s
                ON CONFLICT DO NOTHING
                """,
                [
                    (
                        org_id,
                        feature_id,
                        r["entity_id"],
                        json.dumps(r["value"]),
                        r["valid_at"],
                    )
                    for r in batch
                ],
            )
            inserted += cur.rowcount
            log.info("  upserted batch %d/%d (%d rows)", i // BATCH_SIZE + 1, -(-len(rows) // BATCH_SIZE), cur.rowcount)

    conn.commit()
    return inserted


# ── Backfill orchestrator ─────────────────────────────────────────────────────

def get_all_orgs(conn: PgConnection) -> list[dict]:
    """Return all active orgs. Backfill runs across all tenants."""
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute("""
            SELECT id, name FROM organizations
            WHERE deleted_at IS NULL
            ORDER BY created_at
        """)
        return cur.fetchall()


def backfill_feature(
    conn: PgConnection,
    feature_name: str,
    org: dict,
    start: date,
    end: date,
    dry_run: bool = False,
) -> dict:
    """
    Backfill a single feature for a single org across a date range.
    Returns a summary dict with counts.
    """
    org_id = str(org["id"])
    org_name = org["name"]

    feature_id = resolve_feature_id(conn, feature_name, org_id)
    if feature_id is None:
        log.warning("  feature '%s' not registered for org '%s' — skipping", feature_name, org_name)
        return {"feature": feature_name, "org": org_name, "skipped": True, "reason": "not_registered"}

    compute_fn = FEATURE_COMPUTERS[feature_name]
    total_upserted = 0
    total_computed = 0

    log.info("backfill: feature=%s org=%s range=%s→%s", feature_name, org_name, start, end)

    for day in date_range(start, end):
        rows = compute_fn(conn, day)
        total_computed += len(rows)
        upserted = upsert_feature_values(conn, feature_id, org_id, rows, dry_run=dry_run)
        total_upserted += upserted

    summary = {
        "feature": feature_name,
        "org": org_name,
        "start": str(start),
        "end": str(end),
        "days": (end - start).days + 1,
        "rows_computed": total_computed,
        "rows_upserted": total_upserted,
        "dry_run": dry_run,
    }
    log.info("  done: %s", summary)
    return summary


def run_backfill(
    features: list[str],
    start: date,
    end: date,
    dry_run: bool = False,
) -> list[dict]:
    """Main entry point. Runs backfill for all features × all orgs."""
    conn = get_conn()
    orgs = get_all_orgs(conn)

    if not orgs:
        log.warning("no orgs found — nothing to backfill")
        return []

    log.info(
        "starting backfill: features=%s orgs=%d range=%s→%s dry_run=%s",
        features, len(orgs), start, end, dry_run,
    )

    summaries = []
    for org in orgs:
        for feature_name in features:
            if feature_name not in FEATURE_COMPUTERS:
                log.error("unknown feature: %s — available: %s", feature_name, list(FEATURE_COMPUTERS))
                continue
            try:
                summary = backfill_feature(conn, feature_name, org, start, end, dry_run=dry_run)
                summaries.append(summary)
            except Exception as exc:
                log.error("backfill failed: feature=%s org=%s error=%s", feature_name, org["name"], exc)
                conn.rollback()
                summaries.append({
                    "feature": feature_name,
                    "org": org["name"],
                    "error": str(exc),
                })

    conn.close()
    return summaries


# ── CLI ───────────────────────────────────────────────────────────────────────

def main() -> None:
    parser = argparse.ArgumentParser(description="Feature store backfill job")
    parser.add_argument(
        "--feature",
        required=True,
        help="Feature name to backfill, or 'all' for all registered features.",
    )
    parser.add_argument(
        "--start",
        required=True,
        help="Start date: YYYY-MM-DD, 'today', or '7daysago'",
    )
    parser.add_argument(
        "--end",
        required=True,
        help="End date: YYYY-MM-DD, 'today', or '7daysago'",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Compute features but do not write to DB.",
    )
    args = parser.parse_args()

    start = parse_date(args.start)
    end = parse_date(args.end)

    if start > end:
        log.error("start date %s is after end date %s", start, end)
        sys.exit(1)

    features = list(FEATURE_COMPUTERS.keys()) if args.feature == "all" else [args.feature]

    summaries = run_backfill(features, start, end, dry_run=args.dry_run)

    # Print final report
    print("\n── Backfill Report ──────────────────────────────────────────")
    total_rows = 0
    errors = 0
    for s in summaries:
        if "error" in s:
            errors += 1
            print(f"  ✗ {s['feature']} / {s['org']}: ERROR — {s['error']}")
        elif s.get("skipped"):
            print(f"  ⚠ {s['feature']} / {s['org']}: skipped ({s['reason']})")
        else:
            rows = s.get("rows_upserted", 0)
            total_rows += rows
            dry = " [dry-run]" if s.get("dry_run") else ""
            print(f"  ✔ {s['feature']} / {s['org']}: {rows} rows{dry}")

    print(f"\n  Total rows upserted: {total_rows}")
    print(f"  Errors: {errors}")
    print("─────────────────────────────────────────────────────────────\n")

    if errors:
        sys.exit(1)


if __name__ == "__main__":
    main()
