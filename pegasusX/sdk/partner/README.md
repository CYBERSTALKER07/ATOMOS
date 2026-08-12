# Partner OpenAPI SDK generation (Phase 2)

Replaces gradual expansion of the hand-written partner client with
OpenAPI-generated stubs for the machine surface in
`contracts/partner.openapi.yaml`.

## Generate

```bash
# TypeScript (fetch)
bash scripts/gen_partner_sdk.sh ts

# Go client
bash scripts/gen_partner_sdk.sh go
```

Outputs land in (and are committed at):

- `sdk/partner/ts/` — TypeScript (fetch) client (`@pegasusx/partner-sdk`)
- `sdk/partner/go/` — Go client (`github.com/pegasusx/pegasusx/sdk/partner/go`, in `go.work`)

Requires Docker (uses `openapitools/openapi-generator-cli`). Human JWT core
remains on the hand client until `contracts/jwt-core.openapi.yaml` coverage
expands.

## Verify contract drift

```bash
make partner-openapi-gate
```
