#!/usr/bin/env python3
"""Generate docs/LOAD_TEST_REPORT.md from k6 summary-export JSON artifacts."""

from __future__ import annotations

import json
import sys
from datetime import datetime, timezone
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SLO_FAIL_RATE = 0.05
SLO_BY_PROFILE = {
    "smoke": {"read": 300, "mutation": 800, "supplier": 400},
    "cert": {"read": 3000, "mutation": 8000, "supplier": 2500},
    "stress": {"read": 500, "mutation": 10000, "supplier": 2000},
}


def fail(message: str) -> None:
    print(f"load-cert-error: {message}", file=sys.stderr)
    raise SystemExit(1)


def load_summary(path: Path) -> dict:
    if not path.is_file():
        return {}
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as error:
        fail(f"invalid k6 summary JSON {path}: {error}")


def slo_for_profile(profile: str) -> dict[str, int]:
    return SLO_BY_PROFILE.get(profile, SLO_BY_PROFILE["smoke"])


def metric_value(summary: dict, name: str, field: str) -> float | None:
    metrics = summary.get("metrics", {})
    metric = metrics.get(name, {})
    if not metric:
        return None

    values = metric.get("values", {})
    if field in values:
        return float(values[field])
    if field in metric:
        return float(metric[field])
    if field == "rate" and "value" in metric:
        return float(metric["value"])
    return None


def k6_thresholds_breached(summary: dict) -> list[str]:
    """k6 summary-export marks breached thresholds with true."""
    breached: list[str] = []
    for metric_name, metric in summary.get("metrics", {}).items():
        thresholds = metric.get("thresholds") or {}
        for expr, crossed in thresholds.items():
            if crossed is True:
                breached.append(f"{metric_name} {expr}")
    return breached


def pass_threshold(actual: float | None, target: float, comparator: str) -> str:
    if actual is None:
        return "n/a"
    if comparator == "lt":
        return "PASS" if actual < target else "FAIL"
    if comparator == "lte":
        return "PASS" if actual <= target else "FAIL"
    return "n/a"


def main() -> None:
    if len(sys.argv) < 2:
        fail("usage: generate_report.py <artifact_dir>")

    artifact_dir = Path(sys.argv[1]).resolve()
    retailer = load_summary(artifact_dir / "k6-retailer.json")
    supplier = load_summary(artifact_dir / "k6-supplier.json")

    profile = (artifact_dir / "profile.txt").read_text(encoding="utf-8").strip() if (
        artifact_dir / "profile.txt"
    ).is_file() else "smoke"
    base_url = (artifact_dir / "base_url.txt").read_text(encoding="utf-8").strip() if (
        artifact_dir / "base_url.txt"
    ).is_file() else "unknown"

    slo = slo_for_profile(profile)

    p99_read = metric_value(retailer, "http_req_duration{endpoint:read}", "p(99)")
    p99_mut = metric_value(retailer, "http_req_duration{endpoint:mutation}", "p(99)")
    fail_rate = metric_value(retailer, "http_req_failed", "rate")

    vus = metric_value(retailer, "vus", "max")
    if vus is None:
        vus = metric_value(retailer, "vus", "value")

    read_pass = pass_threshold(p99_read, slo["read"], "lt")
    mut_pass = pass_threshold(p99_mut, slo["mutation"], "lt")
    fail_pass = pass_threshold(fail_rate, SLO_FAIL_RATE, "lte")

    supplier_p99 = metric_value(supplier, "http_req_duration{endpoint:read}", "p(99)")
    supplier_pass = pass_threshold(supplier_p99, slo["supplier"], "lt") if supplier else "n/a"

    k6_breaches = k6_thresholds_breached(retailer) + k6_thresholds_breached(supplier)
    k6_pass = "PASS" if not k6_breaches else "FAIL"

    overall = "PASS"
    for status in (read_pass, mut_pass, fail_pass, supplier_pass, k6_pass):
        if status == "FAIL":
            overall = "FAIL"
            break

    breach_lines = ""
    if k6_breaches:
        breach_lines = "\n".join(f"- `{line}`" for line in k6_breaches)

    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    report_path = artifact_dir / "LOAD_TEST_REPORT.md"
    notes = [
        "- Profile `smoke` is the local/CI gate (`make load-cert`).",
        "- Profile `cert` uses relaxed mutation/supplier SLO on local SSMR (emulator); staging uses production targets in `docs/LOAD_TEST_SLO.md`.",
        "- Cert/stress bootstrap `LOAD_RETAILER_POOL_SIZE` distinct retailer JWTs so per-actor rate limits reflect real concurrency.",
        "- SLO source: `docs/LOAD_TEST_SLO.md`.",
    ]
    if k6_breaches:
        notes.insert(0, "- **k6 threshold breaches:**")
        notes = [notes[0], *["  " + line for line in breach_lines.splitlines()], *notes[1:]]

    report_path.write_text(
        "\n".join(
            [
                "# pegasusX load certification report",
                "",
                f"- **Date:** {now}",
                f"- **Profile:** `{profile}`",
                f"- **Base URL:** `{base_url}`",
                f"- **Artifacts:** `{artifact_dir}`",
                f"- **Overall:** **{overall}**",
                f"- **k6 thresholds:** **{k6_pass}**",
                "",
                "| Metric | Target | Observed | Pass |",
                "|--------|--------|----------|------|",
                f"| Retailer VUs (max) | profile-defined | {int(vus) if vus is not None else 'n/a'} | |",
                f"| p99 read (tracking/cart) | < {slo['read']} ms | "
                f"{round(p99_read) if p99_read is not None else 'n/a'} ms | {read_pass} |",
                f"| p99 mutation (order create) | < {slo['mutation']} ms | "
                f"{round(p99_mut) if p99_mut is not None else 'n/a'} ms | {mut_pass} |",
                f"| HTTP failure rate | <= {SLO_FAIL_RATE * 100:.0f}% | "
                f"{(fail_rate * 100) if fail_rate is not None else 'n/a'}% | {fail_pass} |",
                f"| Supplier p99 read | < {slo['supplier']} ms | "
                f"{round(supplier_p99) if supplier_p99 is not None else 'n/a'} ms | {supplier_pass} |",
                "",
                "## Notes",
                "",
                *notes,
                "",
            ]
        ),
        encoding="utf-8",
    )

    latest = ROOT / "docs" / "LOAD_TEST_REPORT.md"
    latest.write_text(report_path.read_text(encoding="utf-8"), encoding="utf-8")

    print(f"report={report_path}")
    print(f"overall={overall}")
    if overall != "PASS":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
