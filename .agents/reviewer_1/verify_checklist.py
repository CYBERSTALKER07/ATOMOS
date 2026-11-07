import os
import re

checklist = [
    # Pegasus
    ("pegasus/agents.md", ["Monorepo Topology", "Zero Theatre", "Spanner", "Outbox", "Developer Workflows"]),
    ("pegasus/docs/ARCHITECTURE.md", ["Macro Dataflow", "Spanner Schema", "Outbox Pattern", "Single-Flight", "H3"]),
    ("pegasus/docs/BACKEND_SERVICES.md", ["Chi Router", "13 Background Cron", "WebSocket Hub", "ai-worker", "optimizer-core"]),
    ("pegasus/docs/FEATURES_AND_PORTALS.md", ["Client Fleet", "Logistics Flow", "Multi-Facility", "Invoices", "H3 Spatial"]),
    ("pegasus/docs/INFRASTRUCTURE.md", ["Terraform", "Kubernetes", "Docker Compose", "Guards", "One-Eye"]),
    
    # PegasusX
    ("pegasusX/agents.md", ["SSMR Doctrine", "Honesty Commandments", "Cloud Spanner", "Outbox", "Workflows"]),
    ("pegasusX/docs/ARCHITECTURE.md", ["SSMR Doctrine", "Cloud Spanner", "Messaging Backbone", "Cell Isolation", "Parity"]),
    ("pegasusX/docs/BACKEND_SERVICES.md", ["Core Go Backend", "24 Runtime Workers", "WebSocket Hub", "AI & Operations Research"]),
    ("pegasusX/docs/FEATURES_AND_ROLE_ROWS.md", ["6 Operational Role-Rows", "ParentOrders Saga", "Ledger", "Soliq EHF", "Promotions"]),
    ("pegasusX/docs/INFRASTRUCTURE.md", ["Terraform", "Kubernetes", "Docker Compose SSMR", "Anti-Theatre CI", "Workflows"]),
    
    # Pegasus.x
    ("pegasus.x/agents.md", ["Sovereign Mission", "Zero Theatre", "Tiyin", "MXIK", "HTTP 428", "Tetris Buffer", "Workflows"]),
    ("pegasus.x/docs/ARCHITECTURE.md", ["Monorepo Topology", "Go 1.26", "TimescaleDB", "S&OP Planning Engine", "Regulatory"]),
    ("pegasus.x/docs/BACKEND_AND_PLANNING.md", ["Core Go", "HTTP 428", "Asynchronous Messaging", "Python S&OP", "CVRP"]),
    ("pegasus.x/docs/FEATURES_AND_APPS.md", ["17 Applications", "Desktop Portals", "Android", "iOS", "Telegram", "E-Imzo", "Asl Belgisi"]),
    ("pegasus.x/docs/INFRASTRUCTURE.md", ["Production Docker Compose", "Servercore Tashkent", "Kubernetes Fleet", "Terraform", "Zero-Downtime"])
]

print("=== CHECKING CHECKLIST ITEMS ACROSS ALL 15 DOCS ===")
all_pass = True
for rel_path, items in checklist:
    full_path = os.path.join('/Users/shakhzod/Desktop/V.O.I.D', rel_path)
    with open(full_path, 'r', encoding='utf-8') as f:
        content = f.read()
    missing = []
    for it in items:
        if not re.search(re.escape(it), content, re.IGNORECASE):
            missing.append(it)
    status = "PASS" if not missing else f"FAIL (missing: {missing})"
    if missing:
        all_pass = False
    print(f"[{status:4s}] {rel_path}")

print("\nOverall Checklist Compliance:", "100% ALL PASS" if all_pass else "FAILURES DETECTED")
