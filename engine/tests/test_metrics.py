"""Unit tests for engine.shared.metrics."""
from __future__ import annotations

from prometheus_client import REGISTRY

from engine.shared.metrics import (
    REQUEST_DURATION,
    REQUEST_TOTAL,
    ERROR_TOTAL,
    ENGAGEMENTS_ACTIVE,
    EVIDENCE_INGESTED,
    CONCLUSIONS_GENERATED,
    QUALITY_GATES_BLOCKED,
    COVERAGE_RATE,
    REASONING_DURATION,
)


class TestMetricRegistrations:
    def test_all_metrics_registered(self) -> None:
        """All required metrics must be registered with the Prometheus collector."""
        names = {m.name for m in REGISTRY.collect()}
        required = {
            "request_duration_seconds",
            "request_total",
            "error_total",
            "aiauditor_engagements_active",
            "aiauditor_evidence_items_ingested_total",
            "aiauditor_conclusions_generated_total",
            "aiauditor_quality_gates_blocked_total",
            "aiauditor_coverage_rate",
            "aiauditor_reasoning_duration_seconds",
        }
        for name in required:
            assert name in names, f"Metric not registered: {name}"


class TestMetricMutations:
    def test_gauge_set_and_inc(self) -> None:
        ENGAGEMENTS_ACTIVE.set(3)
        ENGAGEMENTS_ACTIVE.inc()
        ENGAGEMENTS_ACTIVE.dec()

    def test_counter_inc(self) -> None:
        EVIDENCE_INGESTED.inc()
        CONCLUSIONS_GENERATED.inc()
        QUALITY_GATES_BLOCKED.inc()

    def test_coverage_rate_gauge(self) -> None:
        COVERAGE_RATE.set(0.72)

    def test_histogram_observe(self) -> None:
        REASONING_DURATION.observe(0.456)
        REQUEST_DURATION.labels(
            service="engine", method="POST", path="/reason", status="200"
        ).observe(0.123)

    def test_request_counters(self) -> None:
        REQUEST_TOTAL.labels(
            service="engine", method="POST", path="/reason", status="200"
        ).inc()
        ERROR_TOTAL.labels(
            service="engine", method="POST", path="/reason"
        ).inc()

