#!/usr/bin/env bash
# ATOMOS M0.2 / M0.4 — CI immutable ledger & audit table lint.
# Scope: scans ALL non-test Go source under apps/backend-go/ (including cmd/*).
# Fails if any non-append Spanner SDK mutation (InsertOrUpdate, InsertOrUpdateMap,
# InsertOrUpdateStruct, Update, UpdateMap, UpdateStruct, Delete, DeleteRange)
# or SQL DML statement (UPDATE, DELETE FROM, INSERT OR UPDATE, MERGE INTO)
# targets PaymentLedgerEntries, JournalEntries, JournalPostings, ArLedgerEntries,
# AuditEvents, AuditLog, CreditPolicyAudit, or PlatformAdminAudit.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

python3 - <<'PY'
import os
import re
import sys

BACKEND_ROOT = os.path.join("apps", "backend-go")
IMMUTABLE_TABLES = (
    "PaymentLedgerEntries",
    "JournalEntries",
    "JournalPostings",
    "ArLedgerEntries",
    "AuditEvents",
    "AuditLog",
    "CreditPolicyAudit",
    "PlatformAdminAudit",
)
TABLES_ALT = "|".join(IMMUTABLE_TABLES)
FORBIDDEN_OPS = (
    "InsertOrUpdate",
    "InsertOrUpdateMap",
    "InsertOrUpdateStruct",
    "Update",
    "UpdateMap",
    "UpdateStruct",
    "Delete",
    "DeleteRange",
)
OPS_ALT = "|".join(FORBIDDEN_OPS)

RE_DIRECT_CALL = re.compile(
    rf'(?:(?:\b[A-Za-z_][A-Za-z0-9_]*\.)?\b({OPS_ALT}))\s*\(\s*["`]({TABLES_ALT})["`]',
    re.MULTILINE,
)
RE_VAR_BIND = re.compile(
    rf'\b([A-Za-z_][A-Za-z0-9_]*)\s*(?::?=)\s*["`]({TABLES_ALT})["`]'
)
RE_SQL_DML = re.compile(
    rf'(?is)\b(?:DELETE\s+FROM\s+`?({TABLES_ALT})`?|MERGE\s+INTO\s+`?({TABLES_ALT})`?|INSERT\s+OR\s+UPDATE\s+(?:INTO\s+)?`?({TABLES_ALT})`?|UPDATE\s+`?({TABLES_ALT})`?(?:\s+AS\s+[A-Za-z_][A-Za-z0-9_]*)?\s+SET)\b'
)


def strip_comments_keep_lines(src: str):
    """Strip Go // and /* */ comments while preserving newlines and extracting string literals."""
    out = []
    strings = []  # (start_line, content)
    i = 0
    n = len(src)
    line = 1
    while i < n:
        ch = src[i]
        if ch == "\n":
            out.append("\n")
            line += 1
            i += 1
            continue
        # Line comment
        if ch == "/" and i + 1 < n and src[i + 1] == "/":
            i += 2
            while i < n and src[i] != "\n":
                i += 1
            continue
        # Block comment
        if ch == "/" and i + 1 < n and src[i + 1] == "*":
            i += 2
            while i + 1 < n and not (src[i] == "*" and src[i + 1] == "/"):
                if src[i] == "\n":
                    out.append("\n")
                    line += 1
                i += 1
            i += 2
            continue
        # Double-quoted string literal
        if ch == '"':
            start_line = line
            buf = []
            out.append('"')
            i += 1
            while i < n:
                c = src[i]
                out.append(c)
                if c == "\\":
                    if i + 1 < n:
                        buf.append(src[i : i + 2])
                        out.append(src[i + 1])
                        i += 2
                        continue
                if c == '"':
                    i += 1
                    break
                if c == "\n":
                    line += 1
                buf.append(c)
                i += 1
            strings.append((start_line, "".join(buf)))
            continue
        # Raw string literal
        if ch == "`":
            start_line = line
            buf = []
            out.append("`")
            i += 1
            while i < n:
                c = src[i]
                out.append(c)
                if c == "\n":
                    line += 1
                if c == "`":
                    i += 1
                    break
                buf.append(c)
                i += 1
            strings.append((start_line, "".join(buf)))
            continue
        # Rune literal
        if ch == "'":
            out.append("'")
            i += 1
            while i < n:
                c = src[i]
                out.append(c)
                if c == "\\" and i + 1 < n:
                    out.append(src[i + 1])
                    i += 2
                    continue
                if c == "'":
                    i += 1
                    break
                i += 1
            continue
        out.append(ch)
        i += 1
    return "".join(out), strings


def line_at(text: str, offset: int) -> int:
    return text.count("\n", 0, offset) + 1


violations = []
scanned_files = 0

for dirpath, _, filenames in os.walk(BACKEND_ROOT):
    for fn in sorted(filenames):
        if not fn.endswith(".go") or fn.endswith("_test.go"):
            continue
        path = os.path.join(dirpath, fn)
        scanned_files += 1
        with open(path, "r", encoding="utf-8") as fh:
            raw = fh.read()

        if not any(tbl in raw for tbl in IMMUTABLE_TABLES):
            continue

        code, literals = strip_comments_keep_lines(raw)
        lines = raw.splitlines()

        for m in RE_DIRECT_CALL.finditer(code):
            ln = line_at(code, m.start())
            op, tbl = m.group(1), m.group(2)
            snippet = lines[ln - 1].strip() if ln - 1 < len(lines) else m.group(0)
            violations.append(f"{path}:{ln}: forbidden {op} on immutable table {tbl}: {snippet}")

        # Check constant/variable aliases bound to immutable table names in the same file.
        aliases = {m.group(1): m.group(2) for m in RE_VAR_BIND.finditer(code)}
        if aliases:
            alias_alt = "|".join(re.escape(k) for k in aliases)
            re_alias_call = re.compile(
                rf'(?:(?:\b[A-Za-z_][A-Za-z0-9_]*\.)?\b({OPS_ALT}))\s*\(\s*({alias_alt})\b'
            )
            for m in re_alias_call.finditer(code):
                ln = line_at(code, m.start())
                op, ident = m.group(1), m.group(2)
                tbl = aliases[ident]
                snippet = lines[ln - 1].strip() if ln - 1 < len(lines) else m.group(0)
                violations.append(f"{path}:{ln}: forbidden {op} on immutable table {tbl} (via {ident}): {snippet}")

        # Check SQL DML literals targeting immutable tables.
        for start_line, lit in literals:
            for m in RE_SQL_DML.finditer(lit):
                ln = start_line + lit.count("\n", 0, m.start())
                tbl = next(g for g in m.groups() if g)
                dml_op = " ".join(m.group(0).split()).upper()
                violations.append(f"{path}:{ln}: forbidden SQL DML ({dml_op}) on immutable table {tbl}")

if violations:
    print(f"ci_immutable_ledger_gate FAILED ({len(violations)} violation(s) across {scanned_files} non-test Go files):", file=sys.stderr)
    for v in violations:
        print(f"  {v}", file=sys.stderr)
    sys.exit(1)

print(f"immutable-ledger-gate-ok ({scanned_files} non-test Go files scanned)")
PY
