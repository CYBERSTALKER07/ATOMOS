import re
with open('/Users/shakhzod/Desktop/V.O.I.D/backend_audit_report.md', 'r') as f:
    lines = f.readlines()
counts = {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0}
for line in lines:
    if "- **Severity**:" in line:
        for k in counts.keys():
            if k in line:
                counts[k] += 1
print(counts)
