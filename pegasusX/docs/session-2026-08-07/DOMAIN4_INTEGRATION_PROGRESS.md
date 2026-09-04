# Domain 4 — Integration (P1–P3) · Progress

> **HISTORICAL / FROZEN — session progress note; do not treat as current gap SoT.**
> Living residuals: [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md).


Date: 2026-08-11 · Roadmap ref: `/Users/shakhzod/.cursor/plans/Ecosystem Capability Roadmap-7cbf327a.plan.md` (Domain 4).

## 1. Real GS1 DataMatrix (ECC200) encoder — replaces the placeholder

The previous `gs1.EncodeDataMatrixModules` was a self-described **placeholder**
("NOT a certified GS1 DataMatrix encoder"). Replaced with a genuine
ISO/IEC 16022 (ECC200) implementation in `gs1/ecc200.go`:

- **ASCII encodation** — double-digit pairing (`12` → 142), literal +1, Upper
  Shift (235) for extended ASCII, and the FNC1 codeword (232) for GS1 group
  separators.
- **Reed–Solomon over GF(256)** with the ECC200 generator polynomial
  `x^8+x^5+x^3+x^2+1` (0x12D), correct log/exp tables and per-symbol ECC counts.
- **Standard module placement** — the utah shapes + four corner cases via the
  canonical up/down 2-column sweep, with row/column wrap.
- **Finder/timing assembly** — solid left/bottom finder, alternating top/right
  clock tracks, correct corner handling, multi-region support up to 44×44
  (144 data codewords; larger payloads route to the printer-native `^BX` path).
- `BuildAIElementStringFNC1` builds the FNC1-separated element string the
  encoder needs for unambiguous variable-length AI parsing ((10) lot, (21) serial).
- `EncodeDataMatrixModules` now delegates to the real encoder; `payload_too_large`
  fails closed for over-44×44 payloads (use ZPL `^BX`).

Tests (`gs1/ecc200_test.go`): digit pairing vector, pad randomization
determinism, RS determinism/non-triviality, finder/timing structural checks,
valid symbol sizing, FNC1 string round-trip, oversize rejection. All pass.

## 2. Committed partner SDKs (no longer README-only)

- Generated both clients from `contracts/partner.openapi.yaml` (Docker
  openapi-generator v7.10.0) and committed them:
  - `sdk/partner/ts/` — TypeScript fetch client; added `package.json`
    (`@pegasusx/partner-sdk`) + `tsconfig.json`; type-checks clean.
  - `sdk/partner/go/` — Go client; fixed the placeholder module path to
    `github.com/CYBERSTALKER07/ATOMOS/sdk/partner/go`; compiles standalone.
- Fixed `scripts/gen_partner_sdk.sh` — the TS output path mismatch
  (`OUT=.../typescript` vs docker `-o .../${LANG}=ts`) that left the documented
  `typescript/` dir empty.
- `sdk/partner/README.md` updated to the committed layout.

## Verification

- `make partner-openapi-gate` — **ok** (contract drift gate passes).
- `go test ./partner/... ./gs1/` — all pass (`partner`, `partner/as2`,
  `partner/edi`, `gs1`).
- `go build ./...` (backend) — clean.
- TS SDK `tsc --noEmit` — clean; Go SDK `go build` (GOWORK=off) — clean.

## Already-wired (verified, not rebuilt)

- AS2 transport: `partner/as2/` (MIME, MDN, client, crypto) — tests pass.
- EDI: `partner/edi` (CONTRL/APERAK, ORDRSP/INVOIC) — tests pass.

## Out of scope / follow-ups (P3, need external certs/slabs)

- **Drummond AS2 certification** — requires a Drummond testbed account; cannot be
  closed from the codebase. The AS2 client is wired and tested; certification is
  an ops/commercial step.
- **1C/CommerceML chain** — `scripts/commerceml_import_ref.py` exists; a full
  two-way CommerceML 3.x exchange (catalog ↔ orders) with a live 1C instance is a
  P3 integration engagement, not a code gap.
