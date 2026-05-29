"""
generate_data.py
----------------
Generates realistic mock FinTech transaction data and saves to CSV.
This is the raw source data that feeds the entire pipeline.

Output: data/transactions.csv
"""

import csv
import random
import uuid
from datetime import datetime, timedelta

# ── Seed for reproducibility ──────────────────────────────────────────────────
random.seed(42)

# ── Configuration ─────────────────────────────────────────────────────────────
NUM_CUSTOMERS    = 200
NUM_TRANSACTIONS = 5000
START_DATE       = datetime(2025, 1, 1)
END_DATE         = datetime(2025, 12, 31)
OUTPUT_FILE      = "data/transactions.csv"

# ── Reference data ────────────────────────────────────────────────────────────
CATEGORIES = {
    "food_and_drink":    (5.00,  120.00),   # (min_amount, max_amount)
    "groceries":         (10.00, 300.00),
    "transport":         (2.50,  80.00),
    "entertainment":     (8.00,  200.00),
    "utilities":         (30.00, 250.00),
    "healthcare":        (20.00, 500.00),
    "shopping":          (15.00, 800.00),
    "travel":            (50.00, 2000.00),
    "subscriptions":     (5.00,  50.00),
    "transfers":         (10.00, 5000.00),
}

MERCHANTS = {
    "food_and_drink":  ["Tim Hortons", "McDonald's", "Starbucks", "Subway", "A&W"],
    "groceries":       ["Walmart", "Costco", "Safeway", "Save-On-Foods", "Loblaws"],
    "transport":       ["BC Transit", "Uber", "Lyft", "Esso", "Petro-Canada"],
    "entertainment":   ["Netflix", "Cineplex", "Steam", "Spotify", "Apple TV"],
    "utilities":       ["BC Hydro", "Telus", "Shaw", "Fortis BC", "City of Victoria"],
    "healthcare":      ["Shoppers Drug Mart", "London Drugs", "Pharmasave", "LifeLabs"],
    "shopping":        ["Amazon", "Best Buy", "IKEA", "H&M", "Sport Chek"],
    "travel":          ["Air Canada", "WestJet", "Airbnb", "Expedia", "Booking.com"],
    "subscriptions":   ["Adobe", "Microsoft 365", "GitHub", "AWS", "Notion"],
    "transfers":       ["Interac e-Transfer", "PayPal", "Wise", "RBC Transfer"],
}

STATUSES = ["completed", "completed", "completed", "completed", "pending", "failed"]
# weighted: mostly completed, some pending, few failed

PROVINCES = ["BC", "ON", "AB", "QC", "MB", "SK"]

# ── Generate customers ────────────────────────────────────────────────────────
def generate_customers(n):
    customers = []
    for i in range(n):
        customers.append({
            "customer_id": str(uuid.uuid4()),
            "province":    random.choice(PROVINCES),
            "age_group":   random.choice(["18-25", "26-35", "36-45", "46-55", "55+"]),
            "account_type": random.choice(["chequing", "savings", "premium"]),
        })
    return customers

# ── Generate transactions ─────────────────────────────────────────────────────
def generate_transactions(customers, n):
    transactions = []
    for _ in range(n):
        customer    = random.choice(customers)
        category    = random.choice(list(CATEGORIES.keys()))
        min_amt, max_amt = CATEGORIES[category]
        amount      = round(random.uniform(min_amt, max_amt), 2)
        merchant    = random.choice(MERCHANTS[category])
        status      = random.choice(STATUSES)
        days_offset = random.randint(0, (END_DATE - START_DATE).days)
        txn_date    = START_DATE + timedelta(days=days_offset)

        transactions.append({
            "transaction_id": str(uuid.uuid4()),
            "customer_id":    customer["customer_id"],
            "province":       customer["province"],
            "age_group":      customer["age_group"],
            "account_type":   customer["account_type"],
            "merchant":       merchant,
            "category":       category,
            "amount":         amount,
            "currency":       "CAD",
            "status":         status,
            "txn_date":       txn_date.strftime("%Y-%m-%d"),
            "txn_month":      txn_date.strftime("%Y-%m"),
            "txn_year":       txn_date.year,
        })

    return transactions

# ── Write to CSV ──────────────────────────────────────────────────────────────
def write_csv(transactions, filepath):
    fieldnames = transactions[0].keys()
    with open(filepath, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(transactions)
    print(f"✓ Written {len(transactions)} transactions to {filepath}")

# ── Summary stats ─────────────────────────────────────────────────────────────
def print_summary(transactions):
    total = sum(t["amount"] for t in transactions if t["status"] == "completed")
    by_category = {}
    for t in transactions:
        cat = t["category"]
        by_category[cat] = by_category.get(cat, 0) + 1

    print(f"\n── Dataset summary ──────────────────────────")
    print(f"  Total transactions : {len(transactions)}")
    print(f"  Completed revenue  : CAD ${total:,.2f}")
    print(f"\n  Transactions by category:")
    for cat, count in sorted(by_category.items(), key=lambda x: -x[1]):
        print(f"    {cat:<20} {count:>5}")
    print(f"─────────────────────────────────────────────\n")

# ── Main ──────────────────────────────────────────────────────────────────────
if __name__ == "__main__":
    print("Generating mock FinTech transaction data...")
    customers    = generate_customers(NUM_CUSTOMERS)
    transactions = generate_transactions(customers, NUM_TRANSACTIONS)
    write_csv(transactions, OUTPUT_FILE)
    print_summary(transactions)
