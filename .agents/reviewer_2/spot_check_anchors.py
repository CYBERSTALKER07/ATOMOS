import json
import os
import re

with open("/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_2/audit_data.json") as f:
    data = json.load(f)

# Pick various links with line numbers and check their exact text on disk
samples = [
    # Pegasus
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/outbox/relay.go", "58"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl", "290"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl", "2151"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/app.go", "74"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/admin/audit_cron.go", "13"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus/package.json", "19"),

    # PegasusX
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/package.json", "5"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/schema/spanner.ddl", "685"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/outbox/relay.go", "14"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/main.go", "417"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/services/optimizer-core/proto/optimizer_core.proto", "14"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/terraform/main.tf", "14"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_placeholder_images.sh", "26"),

    # Pegasus.x
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fiscal/calculator.go", "106"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fiscal/calculator.go", "31"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/soliq/efactura.go", "17"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/soliq/efactura.go", "156"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/compliance/aslbelgisi.go", "21"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/router.go", "441"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/router.go", "494"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/engine/cvrp.py", "33"),
    ("/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/telemetry/geofence.go", "30"),
]

for path, line_str in samples:
    try:
        with open(path, "r", errors="ignore") as f:
            lines = f.readlines()
        ln = int(line_str)
        if 1 <= ln <= len(lines):
            snippet = lines[ln-1].strip()
            print(f"PASS: {path.split('/')[-2]}/{path.split('/')[-1]}:{ln} -> {snippet}")
        else:
            print(f"FAIL: {path}:{ln} (total {len(lines)})")
    except Exception as e:
        print(f"ERROR: {path} -> {e}")
