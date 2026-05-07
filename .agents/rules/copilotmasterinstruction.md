---
trigger: always_on
---

# Canonical Source Lock

This file is an adapter and must stay short to avoid policy drift.

Canonical instruction sources:
1. `.github/copilot-instructions.md`
2. `.github/ACT.md`

Precedence:
- If this file differs from either canonical source, canonical sources win.
- Product/runtime doctrine is owned by `.github/copilot-instructions.md`.
- Execution safety and challenge protocol are owned by `.github/ACT.md`.

Non-negotiable reminders:
- Admin Portal is Supplier Portal; ADMIN naming is a technical alias only.
- `pegasus/` is the only canonical source tree.
- For non-trivial work: codebase retrieval first, then docs sync.
- Keep backend, web/desktop, and mobile contracts aligned per role row.
- Never trust scope fields from request bodies; derive scope from claims.

Maintenance rule:
- Do not duplicate large doctrine here.
- Update canonical files first, then adjust this adapter only if needed.