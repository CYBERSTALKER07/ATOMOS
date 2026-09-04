#!/usr/bin/env bash
# Phase-2 exit gate: enterprise integration surface.
#   1. Phase-1 gate still green (money/law must not regress)
#   2. Partner OpenAPI markers (master-data + idempotency + rotate + POS)
#   3. EDI CONTRL/APERAK + inbound ORDRSP/INVOIC codecs
#   4. GS1 DataMatrix AI scaffolding
#   5. SFTP host-key pin + webhook secret rotation
#   6. CommerceML reference converter smoke
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export SPANNER_EMULATOR_HOST="${SPANNER_EMULATOR_HOST:-localhost:9010}"
export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasusx-local}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-instance}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-db}"

echo "[1/6] Phase-1 gate (regression) ..."
if [[ "${PHASE2_SKIP_REGRESSION:-}" == "1" ]]; then
  echo "skipping phase-1 (PHASE2_SKIP_REGRESSION=1)"
else
  bash scripts/phase1_gate.sh
fi

echo "[2/6] Partner OpenAPI gate (Phase-2 paths + idempotency) ..."
make partner-openapi-gate

echo "[3/6] EDI codecs (CONTRL/APERAK/ORDRSP/INVOIC) ..."
(cd apps/backend-go && go test ./partner/edi/ -run 'TestBuildCONTRL|TestParseORDRSP|TestORDERS|TestOutbound' -count=1)

echo "[4/6] GS1 DataMatrix scaffolding ..."
(cd apps/backend-go && go test ./gs1/ -run 'TestBuildAI|TestNormalizeGTIN' -count=1)

echo "[5/6] SFTP host-key + webhook rotate ..."
(cd apps/backend-go && go test ./partner/ -run 'TestSftpHost|TestRotateWebhook|TestAckAccepted' -count=1)

echo "[6/6] CommerceML reference converter ..."
python3 - <<'PY'
import tempfile, xml.etree.ElementTree as ET
from pathlib import Path
import subprocess, json, os
root = Path(".")
sample = tempfile.NamedTemporaryFile("w", suffix=".xml", delete=False)
sample.write("""<?xml version='1.0' encoding='UTF-8'?>
<КоммерческаяИнформация>
  <Каталог>
    <Товары>
      <Товар>
        <Ид>SKU-1</Ид>
        <Наименование>Test SKU</Наименование>
        <Штрихкод>4006381333931</Штрихкод>
      </Товар>
    </Товары>
  </Каталог>
</КоммерческаяИнформация>
""")
sample.close()
out = tempfile.NamedTemporaryFile("w", suffix=".json", delete=False)
out.close()
subprocess.check_call([
  "python3", "scripts/commerceml_import_ref.py",
  "--import", sample.name, "--out", out.name,
])
data = json.loads(Path(out.name).read_text())
assert data["products"]["items"][0]["external_id"] == "SKU-1"
os.unlink(sample.name)
os.unlink(out.name)
print("commerceml-ref-ok")
PY

echo "phase2-gate-ok"
