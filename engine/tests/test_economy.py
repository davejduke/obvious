"""Unit tests — Economy Engine (work program generation, coverage maximisation)."""
import pytest
from engine.economy.work_program import (
    WorkItem,
    WorkProgram,
    generate_work_program,
    compute_coverage_gap,
)


def make_control(control_id: str, risk_weight: float = 1.0):
    return {
        "control_id": control_id,
        "title": f"Control {control_id}",
        "risk_weight": risk_weight,
        "article_ref": f"NIS2-21a",
    }


class TestGenerateWorkProgram:
    def test_empty_controls_returns_empty_program(self):
        program = generate_work_program([])
        assert len(program.items) == 0
        assert program.coverage_score == 0.0

    def test_single_control_within_budget(self):
        ctrl = make_control("C-001", risk_weight=1.0)
        program = generate_work_program([ctrl], budget_hours=100.0)
        assert len(program.items) == 1
        assert program.coverage_score == pytest.approx(1.0)

    def test_high_risk_prioritised_over_low(self):
        controls = [
            make_control("LOW", risk_weight=0.5),
            make_control("HIGH", risk_weight=4.5),
        ]
        program = generate_work_program(controls, budget_hours=40.0)
        ids = [item.control_id for item in program.items]
        # High risk item should be included first (sorted desc by risk)
        assert ids[0] == "HIGH"

    def test_budget_constraint_respected(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=5.0) for i in range(20)]
        program = generate_work_program(controls, budget_hours=40.0)
        assert program.total_estimated_hours <= 40.0

    def test_coverage_score_range(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=2.0) for i in range(5)]
        program = generate_work_program(controls, budget_hours=100.0)
        assert 0.0 <= program.coverage_score <= 1.0

    def test_coverage_score_increases_with_budget(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=2.0) for i in range(10)]
        low_budget = generate_work_program(controls, budget_hours=20.0)
        high_budget = generate_work_program(controls, budget_hours=500.0)
        assert high_budget.coverage_score >= low_budget.coverage_score

    def test_deterministic_output(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=float(i % 5 + 1)) for i in range(8)]
        p1 = generate_work_program(controls, budget_hours=50.0)
        p2 = generate_work_program(controls, budget_hours=50.0)
        assert [i.control_id for i in p1.items] == [i.control_id for i in p2.items]
        assert p1.coverage_score == p2.coverage_score


class TestCoverageGap:
    def test_all_included_no_gap(self):
        controls = [make_control("A"), make_control("B")]
        program = generate_work_program(controls, budget_hours=1000.0)
        gap = compute_coverage_gap(program, controls)
        assert gap == []

    def test_excluded_controls_reported(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=5.0) for i in range(20)]
        program = generate_work_program(controls, budget_hours=20.0)
        gap = compute_coverage_gap(program, controls)
        included = {item.control_id for item in program.items}
        for ctrl_id in gap:
            assert ctrl_id not in included
