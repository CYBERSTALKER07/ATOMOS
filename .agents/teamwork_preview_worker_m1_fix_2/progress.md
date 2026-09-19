# Progress Tracker

Last visited: 2026-08-21T15:48:30Z

- [x] Initial investigation and discovery of all `reatilerapp` occurrences
- [x] Fix typo in `pegasusX/scripts/build_all_native_local.sh`
- [x] Fix typo in `pegasusX/scripts/ci_ios_apps.sh`
- [x] Fix typo in `pegasusX/.github/workflows/ci.yml`
- [x] Fix typo in `pegasusX/packages/i18n/scripts/wire-mobile-resources.mjs`
- [x] Fix typo in `pegasusX/packages/i18n/scripts/wire-mobile-interpolations.mjs`
- [x] Fix typo in `generate_icons.py`
- [x] Fix remaining occurrences in `pegasusX/packages/i18n/generated/`, docs, and `pegasusX/modify_files.py`
- [x] Fix occurrence in `pegasus/replace_gateways.sh`
- [x] Verify 0 matches of `reatilerapp` in `pegasusX/`, `.github/`, `*.py`, `*.sh`
- [x] Run backend-go build and test suite
- [x] Write `changes.md` and `handoff.md`


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
