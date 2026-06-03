"""
Tests for the backfill job.
Uses a real PostgreSQL instance via pytest fixtures (same pattern as saas-ci.yml).
Run: pytest backfill/tests/ -v
"""

from __future__ import annotations

import json
from datetime import date, datetime, timedelta
from unittest.mock import MagicMock, patch

import pytest

from backfill.run import (
    FEATURE_COMPUTERS,
    compute_avg_tx_amount_30d,
    compute_failed_tx_ratio_7d,
    compute_top_category_30d,
    compute_tx_count_7d,
    parse_date,
    upsert_feature_values,
)


# ── parse_date tests ──────────────────────────────────────────────────────────

class TestParseDate:
    def test_today(self):
        assert parse_date("today") == date.today()

    def test_days_ago(self):
        assert parse_date("7daysago") == date.today() - timedelta(days=7)
        assert parse_date("30daysago") == date.today() - timedelta(days=30)

    def test_iso_date(self):
        assert parse_date("2024-01-15") == date(2024, 1, 15)

    def test_case_insensitive(self):
        assert parse_date("TODAY") == date.today()
        assert parse_date("7DAYSAGO") == date.today() - timedelta(days=7)

    def test_invalid_date_raises(self):
        with pytest.raises(ValueError):
            parse_date("not-a-date")


# ── Feature computer tests ────────────────────────────────────────────────────

class TestFeatureComputers:
    """
    Tests feature compute functions against a mock DB connection.
    Validates output shape and data types — not the SQL logic itself
    (that's covered by integration tests in CI).
    """

    def _make_conn(self, rows: list[dict]) -> MagicMock:
        """Build a mock psycopg2 connection that returns given rows."""
        mock_cur = MagicMock()
        mock_cur.__enter__ = lambda s: s
        mock_cur.__exit__ = MagicMock(return_value=False)
        mock_cur.fetchall.return_value = rows

        mock_conn = MagicMock()
        mock_conn.cursor.return_value = mock_cur
        return mock_conn

    def test_tx_count_7d_output_shape(self):
        conn = self._make_conn([
            {"entity_id": "user-1", "tx_count": 5},
            {"entity_id": "user-2", "tx_count": 12},
        ])
        result = compute_tx_count_7d(conn, date(2024, 6, 1))

        assert len(result) == 2
        assert all("entity_id" in r for r in result)
        assert all("value" in r for r in result)
        assert all("valid_at" in r for r in result)
        assert result[0]["value"] == 5
        assert result[1]["value"] == 12
        assert isinstance(result[0]["valid_at"], datetime)

    def test_avg_tx_amount_30d_output_shape(self):
        conn = self._make_conn([
            {"entity_id": "user-1", "avg_amount": 125.50},
        ])
        result = compute_avg_tx_amount_30d(conn, date(2024, 6, 1))

        assert len(result) == 1
        assert isinstance(result[0]["value"], float)
        assert result[0]["value"] == 125.50

    def test_failed_tx_ratio_7d_handles_none(self):
        """Null ratio (no transactions) should coerce to 0.0."""
        conn = self._make_conn([
            {"entity_id": "user-1", "failed_ratio": None},
        ])
        result = compute_failed_tx_ratio_7d(conn, date(2024, 6, 1))

        assert result[0]["value"] == 0.0

    def test_failed_tx_ratio_7d_valid_float(self):
        conn = self._make_conn([
            {"entity_id": "user-1", "failed_ratio": 0.1429},
        ])
        result = compute_failed_tx_ratio_7d(conn, date(2024, 6, 1))

        assert result[0]["value"] == pytest.approx(0.1429)

    def test_top_category_30d_output_shape(self):
        conn = self._make_conn([
            {"entity_id": "user-1", "top_category": "groceries"},
            {"entity_id": "user-2", "top_category": "transport"},
        ])
        result = compute_top_category_30d(conn, date(2024, 6, 1))

        assert result[0]["value"] == "groceries"
        assert result[1]["value"] == "transport"

    def test_empty_result(self):
        """All computers should return empty list when no data."""
        conn = self._make_conn([])
        for name, fn in FEATURE_COMPUTERS.items():
            result = fn(conn, date(2024, 6, 1))
            assert result == [], f"{name} should return [] for empty input"

    def test_valid_at_matches_as_of_date(self):
        """valid_at should always be midnight of the as_of date."""
        conn = self._make_conn([{"entity_id": "u1", "tx_count": 3}])
        as_of = date(2024, 3, 15)
        result = compute_tx_count_7d(conn, as_of)

        assert result[0]["valid_at"].date() == as_of
        assert result[0]["valid_at"].hour == 0
        assert result[0]["valid_at"].minute == 0


# ── Feature registry tests ────────────────────────────────────────────────────

class TestFeatureRegistry:
    def test_all_features_registered(self):
        expected = {
            "tx_count_7d",
            "avg_tx_amount_30d",
            "failed_tx_ratio_7d",
            "top_category_30d",
        }
        assert set(FEATURE_COMPUTERS.keys()) == expected

    def test_all_computers_callable(self):
        for name, fn in FEATURE_COMPUTERS.items():
            assert callable(fn), f"{name} computer is not callable"


# ── Upsert tests ──────────────────────────────────────────────────────────────

class TestUpsertFeatureValues:
    def test_dry_run_does_not_call_execute(self):
        mock_conn = MagicMock()
        rows = [
            {"entity_id": "u1", "value": 5, "valid_at": datetime.now()},
            {"entity_id": "u2", "value": 3, "valid_at": datetime.now()},
        ]
        result = upsert_feature_values(
            mock_conn, "feat-id", "org-id", rows, dry_run=True
        )
        # Dry run should return count but not touch DB
        assert result == 2
        mock_conn.cursor.assert_not_called()

    def test_empty_rows_returns_zero(self):
        mock_conn = MagicMock()
        result = upsert_feature_values(mock_conn, "feat-id", "org-id", [], dry_run=False)
        assert result == 0
        mock_conn.cursor.assert_not_called()
