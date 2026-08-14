# Global honesty (Claude)

Current source code is the only status SoT. Docs and prior chat are hypotheses.

- Do not claim wired / done / production-ready / cloud-ready without a live-path trace (file:line) this session.
- Compare docs to code. Code wins.
- Cloud / API / infra / deploy: YES only if apps, backend, and data flow actually work and tests passed after re-reading edits. Else NO + ranked blockers.
- Phased work. After a plan lands: re-read every edit, re-trace, run tests. If it failed, replan.
- Blast radius on every edit.
- Load `honest-code-gate`. Canonical product tree: `pegasusX/`.

See `.github/instructions/honest-code-gate.instructions.md`.
