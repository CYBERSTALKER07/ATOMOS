# 01 — Design — Phase 0

## Branch strategy

| Rule | Detail |
|------|--------|
| One phase branch | `phase/G1-money-law`, `phase/G2-physical`, … |
| No mixed gates | Do not land G5 commits on a G1 branch |
| Base | Prefer `main` / agreed integration branch after phase proof |
| Working tree | Uncommitted B7 should land before or as first commit on G1 if still dirty |

## Commit message convention

```
phase(G1.A): AR cash pay-down same-txn

phase(G1.C): remove driver state-patch client calls
```

## Sub-phase ownership

Within a phase, sub-phases (A/B/C) may use **one implementer stream** (or one agent) per package cluster to avoid merge thrash.

## Algorithm playbook pointer

See MASTER program §1.5 — prefer existing primitives → OSS/industry math → own design.
