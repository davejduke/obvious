"""Economy Engine — Work program generation, resource allocation, and budget tracking."""
from engine.economy.budget_tracker import (
    AlertLevel,
    BudgetEntry,
    BudgetTracker,
    VarianceReport,
)
from engine.economy.resource_allocation import (
    AllocationResult,
    AuditorResource,
    allocate_resources,
)
from engine.economy.templates import (
    TEMPLATE_REGISTRY,
    TemplateStep,
    WorkProgramTemplate,
    get_template,
    list_templates,
)
from engine.economy.work_program import (
    WorkItem,
    WorkItemStatus,
    WorkProgram,
    compute_coverage_gap,
    generate_work_program,
    generate_work_program_from_dag,
)

__all__ = [
    # work_program
    "WorkItem",
    "WorkItemStatus",
    "WorkProgram",
    "compute_coverage_gap",
    "generate_work_program",
    "generate_work_program_from_dag",
    # resource_allocation
    "AllocationResult",
    "AuditorResource",
    "allocate_resources",
    # budget_tracker
    "AlertLevel",
    "BudgetEntry",
    "BudgetTracker",
    "VarianceReport",
    # templates
    "TEMPLATE_REGISTRY",
    "TemplateStep",
    "WorkProgramTemplate",
    "get_template",
    "list_templates",
]
