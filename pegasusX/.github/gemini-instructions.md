# Gemini / Google AI — V.O.I.D. instructions

This file is the Gemini counterpart to `.github/copilot-instructions.md`.

**HONESTY OVERRIDE (absolute):** Living product is **`pegasusX/`**. Do not plan or ship from `pegasus/` or frozen `.docx`. Runtime notes in Copilot/ACT/AGENTS files are **not status**.

- Code opened this session is the only status SoT. Docs and **"Wired"** rows are hypotheses.
- Forbidden without file:line: wired, done, production-ready, cloud-ready, we can start connecting cloud.
- Compare docs to code. Code wins. Name THEATRE and DOC DRIFT.
- Cloud / API / infra / deploy: YES only if backend + shipped role-row apps + data flow are REAL and tests passed after re-reading edits. Else NO + ranked blockers.
- Phased work. After a plan lands: re-read edits, re-trace, run unit + integration/CI-equivalent. If it failed, replan.
- Blast radius on every edit. Skills first (`honest-code-gate`, `gap-hunter`), then official docs/web, then proven algorithms; else invent tested in-house logic.

Engineering doctrine (handler shape, outbox, role rows): inherit `.github/copilot-instructions.md` **after** this override, but **never** treat its "Current runtime sync" / additive notes as proof the product is ready.

Always-on Copilot instruction: `.github/instructions/honest-code-gate.instructions.md`.
Canonical docs map: `pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md` (re-verify in code).
