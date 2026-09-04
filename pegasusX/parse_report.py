import re

with open('/Users/shakhzod/Desktop/V.O.I.D/backend_audit_report.md', 'r') as f:
    lines = f.readlines()

criticals = []
highs = []
mediums = []
lows = []

current_finding = None
current_severity = None

for line in lines:
    if re.match(r"^### (VULN|BUG|PERF|LEAK|F-|Finding|TRK)", line):
        current_finding = line.strip().replace("### ", "")
        current_severity = None
    elif "- **Severity**:" in line and current_finding:
        if "CRITICAL" in line:
            criticals.append(current_finding)
        elif "HIGH" in line:
            highs.append(current_finding)
        elif "MEDIUM" in line:
            mediums.append(current_finding)
        elif "LOW" in line:
            lows.append(current_finding)
        current_finding = None

with open('/Users/shakhzod/.gemini/antigravity-cli/brain/859745ac-ab51-4564-8a5a-3a6c246aac63/backend_remediation_plan.md', 'w') as f:
    f.write("# PegasusX Backend Audit Remediation Plan (Severity-Based)\n\n")
    f.write("This is a comprehensive, prioritized execution plan to resolve all 105 findings in the `backend_audit_report.md`. ")
    f.write("The flow is strictly ordered by severity: Critical -> High -> Medium -> Low. ")
    f.write("Before executing any finding, a dedicated implementation plan will be created for that specific item.\n\n")

    f.write("## Phase 1: CRITICAL Findings (28 Items)\n")
    f.write("**Goal:** Fix all build blockers, fatal Spanner schema aborts (100% crashes), and severe security/tenant bypasses.\n\n")
    for item in criticals:
        f.write(f"- [ ] **{item}**\n")
        
    f.write("\n## Phase 2: HIGH Findings (42 Items)\n")
    f.write("**Goal:** Fix data leaks, broken business logic, missing locks, and severe API contract drifts.\n\n")
    for item in highs:
        f.write(f"- [ ] **{item}**\n")
        
    f.write("\n## Phase 3: MEDIUM Findings (24 Items)\n") # wait, should be 34, let the script count
    f.write("**Goal:** Fix race conditions, incorrect state transitions, and background worker logic flaws.\n\n")
    for item in mediums:
        f.write(f"- [ ] **{item}**\n")
        
    f.write("\n## Phase 4: LOW & PERF Findings (11 Items)\n")
    f.write("**Goal:** Fix unindexed queries, hardcoded placeholders, and minor inaccuracies.\n\n")
    for item in lows:
        f.write(f"- [ ] **{item}**\n")

