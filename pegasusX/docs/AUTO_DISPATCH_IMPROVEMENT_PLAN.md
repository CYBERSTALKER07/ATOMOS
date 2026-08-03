# Auto-dispatch — assessment + ordered improvement plan

**Status:** D1–D6 implemented (2026-08-04)  
**Date:** 2026-08-04  
**Scope:** PegasusX / ATOMOS dispatch engine (pure `dispatch` package, Smart Fit, freeze-lock, AI optimizer fallback)  
**Related:** warehouse auto-dispatch worker, `optimizerclient` / `ai-worker`, H3 + VU binpack, continuous replan paths

**Shipped corrections:** Pure path is H3 + scored BinPack + 2-opt (not K-Means). AI preferred for batches ≥12 (`DISPATCH_AI_MIN_STOPS`). `UnitVolumeVU` dual-read + `volume_source` preview honesty. Replan uses same local-search engine with freeze + cooldown. Hierarchical H3 behind `DISPATCH_HIERARCHICAL_H3`.

This document freezes a design review of the current auto-dispatch feature and a concrete, ordered plan to close the honest weaknesses **without** breaking architectural invariants.

---

## 0. Bottom line

For multi-stop **B2B last-mile** with cash, fiscal constraints, and high exception rates, current maturity is the **right level today**.

It is **not** “AI magic.” It is a well-engineered, safety-conscious, hybrid deterministic system with a clean upgrade path to a stronger solver — better than most commercial last-mile auto-dispatch implementations.

**Do not break:**

1. Pure computation stays pure (no Spanner inside scoring / binpack / clustering).
2. Freeze locks remain the single source of truth for human override.
3. AI path always has a deterministic fallback.
4. Retailer orders stay **atomic super-orders by default**.
5. Integer money / single-warehouse rules remain intact.

---

## 1. What is actually strong today

### 1.1 Clean architecture

Core logic lives in a **pure `dispatch` package** (no Spanner, no HTTP, no Kafka). H3 cell lookup, K-Means clustering, bin-packing, vehicle selection, and freeze-lock checks are pure functions.

- Easy to test and reason about.
- Easy to replace the solver later without rewiring I/O.

### 1.2 Pragmatic spatial model

- Uber **H3 res 7** + volumetric units (VU).
- **5% Tetris buffer** (`TetrisBuffer = 0.95`).
- Sensible for dense urban last-mile; not over-engineered.

### 1.3 Smart Fit protocol (heart of the engine)

| Rule | Behavior |
|------|----------|
| 1 | Consolidate into existing **same-cell** route |
| 2 | Greedy first-fit multi-stop at **95%** capacity |
| 3 | Split / overflow when order exceeds max fleet capacity |
| 4 | Explicit **override** path |
| Super-order | Retailer orders treated as **atomic** by default (correct for B2B) |

### 1.4 Safety-first freeze-lock

Manual intervention and AI cannot race each other. Freeze-lock is excellent and must stay.

### 1.5 Graceful degradation

Optional AI optimizer (`ai-worker` via `optimizerclient`) with a hard **~2.5 s timeout**. Any failure falls back to deterministic K-Means + binpack. Production-grade, not marketing.

---

## 2. Honest weaknesses (current reality)

| Area | Current reality | Impact |
|------|-----------------|--------|
| **Solver quality** | Mostly greedy first-fit + K-Means | Good for medium density; loses to real VRP (OR-Tools-class) with tight windows + 30+ stops |
| **Multi-objective scoring** | Mostly volume + H3 proximity | Underuses Driver Score, Allocation PriorityScore, shop-closed risk, fuel/empty miles |
| **Volume accuracy** | `DefaultUnitVolumeVU = 1.0` + L×W×H/5000 | Utilization is optimistic until real per-SKU volume (+ stacking) exists |
| **Continuous vs batch** | Batch “build routes now” is primary | Live continuous replan exists but is less mature than initial dispatch |
| **Scale** | In-memory pure computation | Fine for hundreds of orders; needs hierarchical H3 or AI-primary at higher volume |

---

## 3. Solution paths (keep architecture)

### 3.1 Solver quality

**Problem:** Greedy first-fit + K-Means loses to real VRP under tight windows + 30+ stops.

| Phase | What | Effort | Notes |
|-------|------|--------|-------|
| **A (quick)** | Improve existing heuristic | Low | Add **2-opt / 3-opt** local search after greedy construction. Pure Go, no new deps. |
| **B (recommended first)** | Make AI optimizer **primary** for dense batches | Medium | `optimizerclient` + timeout + fallback already exist. Promote from enhancement → preferred. |
| **C (strong)** | Embed real solver | Higher | OR-Tools (or equivalent) via `ai-worker` gRPC. Pure Go binpack remains hard fallback. |

**Concrete recommendation:** Do **B first**. Raise confidence / batch threshold so AI is default for batches **≥ 12 stops**. Keep pure Go as safety net. Then layer **A** (2-opt) on the pure path so fallback quality rises too.

---

### 3.2 Multi-objective scoring (highest ROI)

**Problem:** Volume + H3 only.

**Solution:** Single pure scoring function every candidate assignment passes through:

```text
Score =
    w1 × VolumeFit            (0–1)
  + w2 × SpatialFit           (H3 distance / max)
  + w3 × PriorityScoreNorm    (from allocation)
  + w4 × DriverScoreNorm
  + w5 × (1 − ShopClosedRisk)
  + w6 × TimeWindowSlack
  − w7 × EmptyMileCost
```

**Implementation steps:**

1. Add `ScoreCandidate(route, order, driver, context)` pure function in `dispatch` package.
2. Pull `PriorityScore` and `DriverScore` from existing packages (already in codebase).
3. Start with fixed weights, e.g. `0.25, 0.20, 0.20, 0.15, 0.10, 0.05, 0.05`. Configurize later.
4. Use the same score in initial BinPack **and** continuous replan.

This turns the dispatcher from “volume filler” into a real multi-objective engine.

---

### 3.3 Volume accuracy

**Problem:** Default 1.0 VU + crude dim formula → optimistic utilization (silent trust killer in UI).

| Stage | Action |
|-------|--------|
| 1 | Add real `VolumePerUnit` (optional `Stackable`, `MaxStackHeight`) on catalog / `SupplierProducts` |
| 2 | Backfill historical products (measured or supplier-declared) |
| 3 | `ComputeOrderVolume` uses real per-SKU volume when present; old formula only as fallback |
| 4 | Later: orientation / non-stackable constraints in binpack |

Until stage 3, utilization numbers remain slightly fictional.

---

### 3.4 Continuous vs batch

**Problem:** Initial dispatch solid; continuous replan weaker.

**Solution:** Reuse the **same** scoring + local-search machinery:

```text
Initial Dispatch  = full BinPack + (optional AI solver)
Continuous Replan = remaining stops only + 2-opt/insert + same multi-objective score
```

**Rules that must stay:**

- Only remaining stops are movable.
- Capacity after partials is respected.
- Freeze lock still blocks concurrent human + AI changes.
- Max replans per route per day + cooldown (as sketched in math docs).

Do **not** invent a second algorithm for replan.

---

### 3.5 Scale

**Problem:** Pure in-memory works for hundreds of orders, not thousands.

| Horizon | Approach |
|---------|----------|
| Short | Keep pure in-memory + H3 (most warehouses stay under limit) |
| Medium | **Hierarchical clustering**: cluster by H3 res 6/5, then normal solver inside each super-cell |
| Long | Large batches permanently on AI worker; pure Go = small-batch + fallback only |

Do **not** rewrite pure functions into a distributed system yet. Hierarchical H3 usually extends life enough.

---

## 4. Recommended execution order (highest leverage first)

| Order | Workstream | Outcome |
|------:|------------|---------|
| **1** | Multi-objective scoring | DriverScore + PriorityScore + shop-closed + slack + empty miles |
| **2** | Real SKU volume master data | Kill default 1.0 VU assumption where data exists |
| **3** | Promote AI optimizer path + 2-opt on pure path | Better dense batches; stronger fallback |
| **4** | Unify continuous replan with same scoring engine | Replan quality ≈ batch quality |
| **5** | Hierarchical H3 only when metrics prove need | Scale without distributed rewrite |

---

## 5. Suggested PR / epic titles (for agents)

Use these as implementation tickets when “start dispatch improvements” is ordered:

1. **D1** — `ScoreCandidate` pure multi-objective scorer + wire into BinPack  
2. **D2** — Catalog `VolumePerUnit` (+ optional stack fields) + `ComputeOrderVolume` dual-read  
3. **D3** — AI optimizer preferred for batch size ≥ 12; pure Go fallback retained  
4. **D4** — 2-opt / 3-opt local search on pure path after greedy construct  
5. **D5** — Continuous replan calls same score + local search; freeze-lock + cooldown preserved  
6. **D6** — Hierarchical H3 pre-cluster (res 5/6) when order count exceeds threshold  

Each PR must keep:

- Pure package free of Spanner/HTTP/Kafka  
- Freeze-lock as sole human/AI mutex  
- Deterministic fallback always available  
- Atomic retailer super-orders by default  

---

## 6. Acceptance / proof ideas

| Workstream | Proof |
|------------|--------|
| D1 scoring | Unit tests: higher PriorityScore / DriverScore wins vs pure volume when weights set |
| D2 volume | Same cart with real VU vs default 1.0 shows different pack result when data present |
| D3 AI primary | Dense fixture (≥12 stops): AI path attempted; timeout/error → pure path marker |
| D4 2-opt | Local search never worsens capacity feasibility; distance/score non-increasing |
| D5 replan | After partial delivery, replan only moves remaining stops; freeze blocks concurrent AI |
| D6 hierarchical | Synthetic 1k orders: hierarchical path completes within SLA; pure path used as leaf |

Markers (optional, SSMR later):

- `PX_E2E_DISPATCH_SCORE_OK`
- `PX_E2E_DISPATCH_VOLUME_MASTER_OK`
- `PX_E2E_DISPATCH_AI_PREFERRED_OK` / `_FALLBACK_OK`
- `PX_E2E_DISPATCH_REPLAN_SAME_SCORE_OK`

---

## 7. Out of scope (for this plan)

- Rewriting dispatch as a distributed microservice mesh  
- Breaking freeze-lock for “faster” AI  
- Non-atomic retailer order splits by default  
- Wave C Retail OS (multi-org / HQ / holds) — separate track  

---

## 8. Next action

When product prioritizes dispatch quality:

1. Start **D1 (multi-objective scoring)** — highest leverage, uses existing signals.  
2. Parallel-track catalog volume schema (**D2**) if data owners can supply numbers.  
3. Only then promote AI path (**D3**) and local search (**D4**).

Agent cue: **“start D1 dispatch scoring”** or **“start auto-dispatch improvement plan.”**
