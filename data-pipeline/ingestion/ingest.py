"""
ingest.py
---------
Reads transactions.csv, validates and cleans the data with pandas,
then loads it into PostgreSQL raw_transactions table.

This is the E (Extract) and L (Load) of our ETL pipeline.
The T (Transform) happens in dbt — see transforms/
"""

import pandas as pd
import psycopg2
from psycopg2.extras import execute_values
from datetime import datetime
import os
import sys

# ── Database connection ───────────────────────────────────────────────────────
DB_CONFIG = {
    "host":     "localhost",
    "port":     5432,
    "dbname":   "fintech",
    "user":     "fintech_user",
    "password": "fintech_pass",
}

CSV_PATH = "data/transactions.csv"

# ── Extract ───────────────────────────────────────────────────────────────────
def extract(filepath):
    print(f"── Extract ───────────────────────────────────")
    print(f"  Reading: {filepath}")
    df = pd.read_csv(filepath)
    print(f"  Rows loaded      : {len(df)}")
    print(f"  Columns          : {list(df.columns)}")
    return df

# ── Transform / Validate ──────────────────────────────────────────────────────
def transform(df):
    print(f"\n── Transform ─────────────────────────────────")
    original_count = len(df)

    # 1. Check for nulls
    null_counts = df.isnull().sum()
    if null_counts.any():
        print(f"  ⚠ Nulls found:\n{null_counts[null_counts > 0]}")
    else:
        print(f"  ✓ No null values found")

    # 2. Remove duplicates on transaction_id
    before = len(df)
    df = df.drop_duplicates(subset=["transaction_id"])
    dupes = before - len(df)
    print(f"  ✓ Duplicates removed : {dupes}")

    # 3. Cast types
    df["txn_date"]  = pd.to_datetime(df["txn_date"]).dt.date
    df["amount"]    = pd.to_numeric(df["amount"], errors="coerce")
    df["txn_year"]  = df["txn_year"].astype(int)

    # 4. Filter invalid amounts
    before = len(df)
    df = df[df["amount"] > 0]
    invalid = before - len(df)
    print(f"  ✓ Invalid amounts removed : {invalid}")

    # 5. Normalize status values
    valid_statuses = ["completed", "pending", "failed"]
    before = len(df)
    df = df[df["status"].isin(valid_statuses)]
    bad_status = before - len(df)
    print(f"  ✓ Unknown statuses removed : {bad_status}")

    # 6. Trim string whitespace
    str_cols = ["province", "age_group", "account_type", "merchant", "category", "currency", "status"]
    for col in str_cols:
        df[col] = df[col].str.strip()

    print(f"  ✓ Rows after cleaning : {len(df)} / {original_count}")
    return df

# ── Load ──────────────────────────────────────────────────────────────────────
def load(df, conn):
    print(f"\n── Load ──────────────────────────────────────")

    # Clear existing data for idempotent runs
    with conn.cursor() as cur:
        cur.execute("TRUNCATE TABLE raw_transactions RESTART IDENTITY;")
        print(f"  ✓ Table truncated (idempotent run)")

    # Prepare rows
    rows = [
        (
            row.transaction_id,
            row.customer_id,
            row.province,
            row.age_group,
            row.account_type,
            row.merchant,
            row.category,
            float(row.amount),
            row.currency,
            row.status,
            row.txn_date,
            row.txn_month,
            int(row.txn_year),
        )
        for row in df.itertuples(index=False)
    ]

    insert_sql = """
        INSERT INTO raw_transactions (
            transaction_id, customer_id, province, age_group, account_type,
            merchant, category, amount, currency, status,
            txn_date, txn_month, txn_year
        ) VALUES %s
    """

    with conn.cursor() as cur:
        execute_values(cur, insert_sql, rows, page_size=500)

    conn.commit()
    print(f"  ✓ Rows inserted : {len(rows)}")
    return len(rows)

# ── Log pipeline run ──────────────────────────────────────────────────────────
def log_run(conn, rows_loaded, status, notes=""):
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO pipeline_runs (rows_loaded, status, notes) VALUES (%s, %s, %s)",
            (rows_loaded, status, notes)
        )
    conn.commit()
    print(f"\n  ✓ Pipeline run logged → status: {status}, rows: {rows_loaded}")

# ── Main ──────────────────────────────────────────────────────────────────────
def main():
    start = datetime.now()
    print(f"\n{'='*46}")
    print(f"  FinTech Data Pipeline — Ingestion")
    print(f"  Started: {start.strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{'='*46}\n")

    conn = None
    try:
        # Connect
        conn = psycopg2.connect(**DB_CONFIG)
        print(f"  ✓ Connected to PostgreSQL\n")

        # ETL
        df           = extract(CSV_PATH)
        df_clean     = transform(df)
        rows_loaded  = load(df_clean, conn)

        # Summary
        elapsed = (datetime.now() - start).total_seconds()
        completed_df = df_clean[df_clean["status"] == "completed"]
        total_revenue = completed_df["amount"].sum()

        print(f"\n── Summary ───────────────────────────────────")
        print(f"  Rows loaded      : {rows_loaded}")
        print(f"  Completed revenue: CAD ${total_revenue:,.2f}")
        print(f"  Elapsed          : {elapsed:.2f}s")
        print(f"{'='*46}\n")

        log_run(conn, rows_loaded, "success")

    except Exception as e:
        print(f"\n  ✗ Pipeline failed: {e}")
        if conn:
            log_run(conn, 0, "failed", str(e))
        sys.exit(1)

    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    main()
