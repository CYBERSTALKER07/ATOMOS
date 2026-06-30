# pegasusX

Single-tenant logistics stack. Sibling project to `pegasus/`.

**Relationship to Pegasus**
- `pegasus/` remains the canonical multi-supplier ecosystem and the reference implementation.
- `pegasusX/` is an independent, fully separated stack that targets one supplier and many retailers.
- Code, builds, infra, and tooling are not shared. Architecture language, contracts, and role/app rows are intentionally aligned so features can be compared one-to-one and so a later migration into Pegasus stays mechanical.

**Tenancy Doctrine**
- Single-supplier is a deployment constraint, not a schema simplification.
- `SupplierId` remains on every supplier-owned row, claim, DTO, and event payload.
- One supplier is seeded at bootstrap. Supplier discovery and supplier-selection UX are hidden, not removed.
- Retailers remain self-registered outside supplier scope.

**Role / App Matrix (launch scope)**
| Role | Surfaces |
|---|---|
| SUPPLIER | supplier-portal (web + Tauri desktop) |
| RETAILER | retailer-app-android, retailer-app-ios, retailer-app-desktop |
| DRIVER | driver-app-android, driver-app-ios |
| WAREHOUSE | warehouse-portal, warehouse-app-android, warehouse-app-ios |
| FACTORY | factory-portal, factory-app-android, factory-app-ios |
| PAYLOAD | payload-terminal, payload-app-ios, payload-app-android |
| MARKETING | marketing-site (Next.js, port 3004) |
| SYSTEM | backend-go, ai-worker |

**Repo Layout**
```
pegasusX/
├── .github/          # project-local doctrine (ACT, copilot, gemini)
├── apps/             # role-row applications
├── packages/         # shared types, api-client, validation, i18n, ui-kit, config
├── contracts/        # canonical event/schema artifacts (events.schema.json)
├── infra/            # docker-compose, terraform, k8s
├── context/          # architecture, design system, technology inventory
├── docs/             # operational docs
└── scripts/          # build, guard, codegen
```

**See**
- [context/purpose.md](context/purpose.md)
- [context/architecture.md](context/architecture.md)
- [context/parity-ledger.md](context/parity-ledger.md)
- [.github/copilot-instructions.md](.github/copilot-instructions.md)
