"""Shared finding model for the multi-agent ecosystem audit orchestra."""

from __future__ import annotations

from typing import Literal, TypedDict

PanelName = Literal[
    "data_flow",
    "business_logic",
    "role_parity",
    "money_fiscal",
    "kafka_outbox",
    "redis_cache",
    "code_quality",
    "architecture",
    "security_tenancy",
    "cloud_infra",
    "client_contracts",
    "gap_register_sync",
]

Severity = Literal["P0", "P1", "P2", "P3"]
CoverageClass = Literal["A", "B", "C", "D", "n/a"]

PANEL_NAMES: tuple[PanelName, ...] = (
    "data_flow",
    "business_logic",
    "role_parity",
    "money_fiscal",
    "kafka_outbox",
    "redis_cache",
    "code_quality",
    "architecture",
    "security_tenancy",
    "cloud_infra",
    "client_contracts",
    "gap_register_sync",
)


class AuditFinding(TypedDict, total=False):
    """One evidence-backed finding from a specialist panel."""

    id: str
    panel: PanelName
    severity: Severity
    class_: CoverageClass  # JSON key is "class"; serialize carefully
    title: str
    evidence: list[str]
    gap_id: str | None
    surfaces: list[str]
    recommendation: str
    reopen_forbidden_if_resolved: bool


FINDING_JSON_HINT = """
Return findings as a JSON array. Each object MUST use keys:
  id, panel, severity (P0|P1|P2|P3), class (A|B|C|D|n/a),
  title, evidence (string[] with path/package anchors),
  gap_id (string|null), surfaces (string[]),
  recommendation, reopen_forbidden_if_resolved (bool).
Never invent wired status. Never reopen surfaces.yaml resolved_gap_ids
without fresh code evidence that the fix regressed.
""".strip()
