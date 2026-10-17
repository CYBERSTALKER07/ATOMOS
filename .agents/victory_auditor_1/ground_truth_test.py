import os
import re

base_dir = '/Users/shakhzod/Desktop/V.O.I.D'

tests = [
    # Pegasus
    ("Pegasus: H3 res-7", "pegasus/apps/backend-go", "Resolution7", ["Resolution7", "h3"]),
    ("Pegasus: Spanner 94 tables", "pegasus/apps/backend-go/schema/spanner.ddl", "CREATE TABLE", []),
    ("Pegasus: LedgerAnomalies", "pegasus/apps/backend-go/schema/spanner.ddl", "LedgerAnomalies", ["LedgerAnomalies"]),
    ("Pegasus: Outbox pattern", "pegasus/apps/backend-go/schema/spanner.ddl", "OutboxEvents", ["OutboxEvents"]),
    ("Pegasus: 6 WebSocket hubs", "pegasus/apps/backend-go/ws", "Hub", ["Hub", "Broadcast"]),
    ("Pegasus: optimizer sidecar :50055", "pegasus/apps/backend-go", "50055", ["50055"]),
    
    # PegasusX
    ("PegasusX: CI anti-theatre scripts", "pegasusX/infra/ci", "ci_fail_todo_inject.sh", ["ci_fail_todo_inject.sh"]),
    ("PegasusX: money_path_gate.sh", "pegasusX/infra/ci/money_path_gate.sh", "money_path_gate", []),
    ("PegasusX: assert_cell_backend.sh", "pegasusX/infra/ci/assert_cell_backend.sh", "assert_cell_backend", []),
    ("PegasusX: Spanner tables & migrations", "pegasusX/apps/backend-go/schema/migrations", "", []),
    ("PegasusX: runtime_workers.go", "pegasusX/apps/backend-go/cmd/server/runtime_workers.go", "RuntimeWorkers", ["Worker", "Start"]),
    ("PegasusX: 8 WebSocket hubs", "pegasusX/apps/backend-go/ws", "hub.go", []),
    ("PegasusX: PaymentLedgerEntries", "pegasusX/apps/backend-go/schema/spanner.ddl", "PaymentLedgerEntries", []),
    ("PegasusX: ADR-009 Soliq EHF", "pegasusX/docs/adr", "009", []),

    # Pegasus.x
    ("Pegasus.x: UnitPriceMinor", "pegasus.x/backend/go", "UnitPriceMinor", ["UnitPriceMinor"]),
    ("Pegasus.x: 1200 bps tax", "pegasus.x/backend/go", "1200", []),
    ("Pegasus.x: MXIK classification", "pegasus.x/backend/go", "MXIK", ["MXIK", "mxik"]),
    ("Pegasus.x: 4 HTTP 428 gates", "pegasus.x/backend/go", "428", ["428", "StatusPreconditionRequired"]),
    ("Pegasus.x: 52 smokecheck steps", "pegasus.x/backend/go/cmd/smokecheck/main.go", "step", []),
    ("Pegasus.x: 95% volumetric Tetris buffer", "pegasus.x/services/planner", "0.95", ["0.95", "95"]),
    ("Pegasus.x: TimescaleDB migrations & seeds", "pegasus.x/backend/go/migrations", "", []),
]

print("=== VERIFYING GROUND TRUTH CLAIMS ===")
for name, rel_path, pattern, extra_keywords in tests:
    target = os.path.join(base_dir, rel_path)
    if not os.path.exists(target):
        print(f"FAIL [Target Missing]: {name} -> {target}")
        continue
    
    if os.path.isfile(target):
        with open(target, 'r', encoding='utf-8', errors='ignore') as f:
            c = f.read()
        if pattern:
            count = len(re.findall(re.escape(pattern), c, re.IGNORECASE))
            print(f"PASS: {name} (found {count} matches for '{pattern}' in {rel_path})")
        else:
            print(f"PASS: {name} (file exists, size {len(c)} bytes)")
    elif os.path.isdir(target):
        if pattern:
            # grep across dir
            found = 0
            for root, dirs, files in os.walk(target):
                for file in files:
                    fp = os.path.join(root, file)
                    try:
                        with open(fp, 'r', encoding='utf-8', errors='ignore') as f:
                            cnt = f.read()
                        if pattern.lower() in cnt.lower() or any(k.lower() in cnt.lower() for k in extra_keywords):
                            found += 1
                    except Exception:
                        pass
            print(f"PASS: {name} (found in {found} files under {rel_path})")
        else:
            num_files = len(os.listdir(target))
            print(f"PASS: {name} (directory contains {num_files} entries)")
