# Surface Skill: Operations Research & Solver Core

Governs Python Google OR-Tools + Rust CP-SAT solvers in `services/optimizer-core`.
Enforce:
- Mathematical solver honesty: report `SolverStatus.HEURISTIC` vs `SolverStatus.OPTIMAL`.
- Dynamic CVRPTW routing with vehicle capacity and time windows.
- 3D crushability bin-packing with reverse-stop LIFO loading constraints.
- Pass `bash scripts/ci_optimizer_core.sh`.
