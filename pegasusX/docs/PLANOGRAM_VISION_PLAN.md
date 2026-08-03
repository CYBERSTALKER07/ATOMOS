# Planogram & Shelf Vision Plan

**Status:** Implementation plan (product + code-grounded)  
**Repo:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
**Date:** 2026-08-02  
**Maps to:** Next-Layer epic **L11** (`docs/NEXT_LAYER_ECOSYSTEM_PLAN.md`) · Retail OS pack **SECTIONS**  
**Constraint:** **Do not** start CV work before Retail OS stock/POS and Next-Layer L1–L3 priorities. Vision is Mode L polish, not a competing epic.

---

## 0. Executive summary

### Decision (frozen)

| Choice | Verdict |
|--------|---------|
| Fork OSS planogram apps as Pegasus product | **No** |
| Invent novel detectors / compliance math | **No** |
| Ship **non-AI planogram structure** on SECTIONS first | **Yes** |
| Later: **sidecar CV** reusing standard pipeline (detect → embed → grid match) | **Yes** |
| Train/finetune on **Pegasus packshots + pilot shelf photos** | **Required for v2** |

### Product principle

> A planogram is first a **data model** (what should be on which shelf slot).  
> Vision is an optional **auditor** of that model.  
> Solo / CORE retailers never see this. Large stores opt into SECTIONS → PLANOGRAM → optional VISION pack.

### What exists today

| Capability | Status | Anchors |
|------------|--------|---------|
| SECTIONS pack | Shipped | `docs/RETAILER_SECTIONS.md`, `/v1/retailer/sections*` |
| Section ↔ SKU multi-map | Shipped | endcaps allowed |
| Section ↔ staff | Shipped | assist routing |
| Shelf slot / facing / expected qty grid | **Not found** | — |
| Shelf photo capture → compliance job | **Not found** | — |
| CV / YOLO / embedding service | **Not found** | — |

---

## 1. Goals and non-goals

### Goals

1. Let Mode L retailers define **expected shelf layouts** (bay → shelf → slot → SKU).  
2. Support **human compliance** (photo + checklist) before any ML.  
3. When demanded, run **offline/batch vision** that scores gaps / wrong SKU / empty facings.  
4. Keep CV out of `backend-go` hot path (order, POS, dock).  
5. Reuse open algorithm shape from OSS/research; own only Pegasus data + UX + tenancy.

### Non-goals

- Real-time CCTV multi-camera ops center (out of scope forever for v1–v2)  
- Replacing store stock counts with vision OnHand (vision is advisory)  
- Vendor lock-in to Trax/Focal or a single GitHub demo repo  
- Warehouse/factory apps understanding planograms  
- Planogram as a production-blocking gate (unlike Retail OS P0–P5)

---

## 2. Build-vs-adapt strategy

### Use from open source / research (as libraries & ideas)

| Piece | Source pattern | How we use it |
|-------|----------------|---------------|
| Dense shelf detection | SKU-110K + YOLO family | Pretrain / fine-tune **detector** only |
| Product ID | Embeddings + FAISS / CLIP vs packshots | Match boxes → Pegasus `SkuId` / local SKU |
| Grid / shelf bands | DBSCAN or y-band clustering (Retail-Shelf-Monitoring, cvpce) | Map boxes → shelf row + slot |
| Compliance score | Detected grid vs expected JSON (cvpce / papers) | Gap / wrong / missing facing |
| Reference reading | [Alijanloo/Retail-Shelf-Monitoring](https://github.com/Alijanloo/Retail-Shelf-Monitoring), [laitalaj/cvpce](https://github.com/laitalaj/cvpce), [ai-parrot PlanogramCompliance](https://github.com/phenobarbital/ai-parrot), GP-180 / SKU-110K | Copy **pipeline shape**, not product shell |

### Build ourselves (Pegasus-owned)

- Spanner planogram schema + RBAC + pack gates  
- Retailer clients (capture, review, fix tasks)  
- Packshot catalog sync from supplier/local SKUs  
- Sidecar job orchestration (GCS upload → Pub/Sub/Cloud Run → results API)  
- Audit, false-positive feedback loop, pilot ops playbooks  

### Explicitly do not vendor

- Do not merge their desktop CCTV apps into the monorepo  
- Do not call their APIs as SoT  
- Do not block Retail OS on their model licenses without legal review  

---

## 3. Capability packs

| Pack ID | Name | Unlocks | Default | Hard deps | Soft deps |
|---------|------|---------|---------|-----------|------------|
| `SECTIONS` | Sections | Departments, SKU/staff map | Off | STORE_STOCK | TEAM |
| `PLANOGRAM` | Shelf layout | Bays, shelves, slots, expected SKUs, human audits | Off | SECTIONS | TEAM, LOCATIONS |
| `PLANOGRAM_VISION` | Shelf vision | Photo jobs, CV score, auto gap tasks | Off | PLANOGRAM | REPORTS_PRO |

```
enable(PLANOGRAM_VISION):
  require PLANOGRAM (+ SECTIONS + STORE_STOCK)
  warn if packshot coverage < threshold% of mapped SKUs
  warn if no pilot location selected
```

---

## 4. Domain model

### 4.1 Hierarchy

```
RetailerLocation
  └── PlanogramBay[]          # aisle / gondola
        └── PlanogramShelf[]  # row top→bottom
              └── PlanogramSlot[]  # left→right facing positions
                    expected: SkuId | LocalSkuId, facings, min_facings
```

### 4.2 Tables (target)

| Table | Purpose |
|-------|---------|
| `RetailerPlanogramBays` | BayId, LocationId, Name, SortOrder |
| `RetailerPlanogramShelves` | ShelfId, BayId, RowIndex, Label |
| `RetailerPlanogramSlots` | SlotId, ShelfId, ColIndex, ExpectedSkuId, Facings, SkuSource (`PEGASUS`\|`LOCAL`) |
| `RetailerPlanogramVersions` | VersionId, LocationId, Status DRAFT\|PUBLISHED, PublishedAt |
| `RetailerShelfAudits` | AuditId, LocationId, SectionId?, BayId?, Mode HUMAN\|VISION, Status |
| `RetailerShelfAuditPhotos` | PhotoId, AuditId, GcsUri, CapturedBy, CapturedAt |
| `RetailerShelfAuditFindings` | FindingId, AuditId, SlotId?, Type GAP\|WRONG_SKU\|EMPTY\|UNKNOWN, ExpectedSku, DetectedSku?, Confidence, Status OPEN\|ACCEPTED\|DISMISSED |
| `RetailerSkuPackshots` | SkuId, RetailerId?, GcsUri, EmbeddingRef, UpdatedAt |

Published version is SoT for compliance. Edits happen on DRAFT then publish (atomic swap).

### 4.3 Money / stock law

Vision findings **never** auto-adjust `RetailerStockBalances`.  
Optional CTA: “Create count task” / “Create assist ticket” / “Transfer from backroom.”

---

## 5. Phased delivery

### Phase 0 — Process gate (now)

| Item | Action |
|------|--------|
| Priority | Keep under Next-Layer **L11** — after L1–L3 and Retail OS Mode M path |
| Docs | This plan + link from `NEXT_LAYER_ECOSYSTEM_PLAN.md` L11 |
| Legal | Before downloading weights/datasets for training, confirm SKU-110K / model license for commercial use |

**Exit:** Product agrees vision is deferred; SECTIONS remains the only shelf IA.

---

### Phase 1 — Planogram structure (no AI) — **PG1**

#### Purpose

Turn SECTIONS into a real layout: expected SKU per slot, usable for clerks without cameras.

#### Algorithms

```
publish_planogram(location_id, draft_version):
  assert PLANOGRAM pack
  assert actor section.manage or planogram.manage
  validate every slot has sku OR explicitly EMPTY
  set version PUBLISHED; previous → ARCHIVED
  outbox PLANOGRAM_PUBLISHED

compliance_checklist(location_id, bay_id):
  return slots with expected sku + last audit status (human)
```

#### APIs

```
GET/POST   /v1/retailer/locations/{id}/planograms
POST       /v1/retailer/planograms/{versionId}/publish
GET/PATCH  /v1/retailer/planograms/{versionId}/bays|shelves|slots
PUT        /v1/retailer/planograms/{versionId}/slots/bulk
GET        /v1/retailer/planograms/published?location_id=
```

#### UI (parity: desktop primary, mobile usable)

- Desktop: bay editor (grid), drag SKUs from section map / catalog  
- Mobile: read-only layout + “walk aisle” checklist  
- Empty state: “Enable Planogram under Capabilities”  

#### Edges

- Slot SKU not in section map → warn, allow (endcap)  
- Local SKU (Next-Layer L6) allowed when L6 shipped  
- Multi-location: planogram per location, no cross-copy required in v1 (copy bay template = v1.1)

#### PRs

| PR | Scope |
|----|-------|
| **PG1.1** | Pack `PLANOGRAM` + DDL + publish API |
| **PG1.2** | Desktop bay/shelf/slot editor |
| **PG1.3** | Mobile checklist walk |
| **PG1.4** | Parity ledger + e2e smoke |

#### Success

- Manager publishes a 2-shelf bay in &lt; 15 minutes  
- Clerk walks checklist without vision  

---

### Phase 2 — Human photo audit — **PG2**

#### Purpose

Capture evidence and findings **without** ML. Builds dataset for Phase 3.

#### Algorithms

```
start_audit(location, bay, mode=HUMAN):
  create ShelfAudit OPEN
  client uploads photos → GCS (signed URL)
  clerk/manager marks findings per slot (GAP / WRONG / OK)
  close audit → summary counts
  optional: open STOCK count task or ASSIST ticket from finding
```

#### APIs

```
POST /v1/retailer/shelf-audits
POST /v1/retailer/shelf-audits/{id}/photos:sign
POST /v1/retailer/shelf-audits/{id}/findings
POST /v1/retailer/shelf-audits/{id}/close
GET  /v1/retailer/shelf-audits?location_id=
```

#### UI

- Mobile-first camera capture (section lead / stock clerk)  
- Desktop review queue for manager  
- Finding → “Create count” deep link into STORE_STOCK  

#### Edges

- Offline photo queue on device (soft; upload when online)  
- PII: no customer faces required; store staff may appear — retention policy  
- Photo without published planogram → allow free-note audit only  

#### PRs

| PR | Scope |
|----|-------|
| **PG2.1** | Audit + GCS signed upload |
| **PG2.2** | Mobile capture + findings |
| **PG2.3** | Desktop review + export CSV |
| **PG2.4** | Feedback labels stored for future training |

#### Success

- 50+ labeled aisle photos from one pilot store  
- Findings actionable without CV  

---

### Phase 3 — Vision sidecar — **PG3**

#### Purpose

Automate first-pass findings. Human remains SoT for accept/dismiss.

#### Architecture

```
Retailer app
  → POST shelf-audit (mode=VISION) + photos → GCS
  → outbox SHELF_VISION_REQUESTED
       → Kafka / Pub/Sub
            → planogram-vision worker (Cloud Run / GKE Job)
                 1. load published planogram JSON
                 2. load packshot embeddings for expected SKUs (+ hard negatives)
                 3. detect boxes (YOLO)
                 4. embed crops → FAISS/top-k SKU
                 5. cluster to shelves/slots
                 6. compare → findings JSON
            → write RetailerShelfAuditFindings (PENDING_REVIEW)
  → manager accepts/dismisses (same UX as PG2)
```

**Do not** run inference inside API request handlers.

#### Model strategy

| Stage | Approach |
|-------|----------|
| Detector bootstrap | Fine-tune YOLO on SKU-110K (or Ultralytics shelf weights) → domain-adapt on pilot photos |
| SKU ID | Few-shot embeddings from `RetailerSkuPackshots` + supplier images; not a 110k-class closed classifier |
| Compliance | Deterministic grid match vs published slots; confidence thresholds |
| LLM optional | Only as assist for unknown crop captioning — **not** SoT for SKU id |

#### Algorithms

```
vision_job(audit_id):
  assert PLANOGRAM_VISION enabled
  planogram = published(location)
  packshots = embeddings(union expected skus in bay)
  for photo in audit.photos:
    boxes = detect(photo)
    for box in boxes:
      sku, conf = match_embedding(crop(box), packshots)
      assign slot = nearest_slot(box, shelf_bands)
    findings = diff(planogram.slots, assigned)
  persist findings PENDING_REVIEW
  emit SHELF_VISION_COMPLETED
  never mutate stock
```

**Accept finding**

```
accept(finding):
  status ACCEPTED
  optional actions: create count / assist / transfer task
  store as hard training label
```

#### APIs

```
POST /v1/retailer/shelf-audits/{id}/run-vision   # enqueue
GET  /v1/retailer/shelf-audits/{id}/vision-status
# packshots
POST /v1/retailer/skus/{skuId}/packshots
GET  /v1/retailer/planograms/vision/coverage?location_id=  # % skus with packshot
```

#### Ops / infra

| Item | Notes |
|------|-------|
| Runtime | Separate image `planogram-vision` |
| GPU | Optional; start CPU for pilots if latency OK |
| Secrets | Model weights in GCS; no keys in app clients |
| Cost | Per-audit billing meter later; free during pilot |
| Kill switch | Config `PLANOGRAM_VISION_ENABLED=false` |

#### Edges

| Case | Expected |
|------|----------|
| Low packshot coverage | Block run-vision with CTA to upload packshots |
| Occlusion / glare | Low conf → UNKNOWN finding, not wrong SKU |
| New packaging | Dismiss + upload new packshot → re-embed |
| Two facings same SKU | Facings count ≥ min_facings |
| Local SKU without packshot | Exclude from auto match; human-only |
| Latency | Async; UI polls; timeout → FAILED with retry |

#### PRs

| PR | Scope |
|----|-------|
| **PG3.1** | Worker skeleton + GCS/pubsub contract + feature flag |
| **PG3.2** | Detector + embedding match MVP on pilot SKU set (≤200 SKUs) |
| **PG3.3** | Slot assignment + findings writeback |
| **PG3.4** | Manager review UX + precision metrics dashboard |
| **PG3.5** | Retrain loop from accepted/dismissed labels |

#### Success metrics (pilot)

| Metric | Target (first pilot) |
|--------|----------------------|
| Precision on WRONG_SKU (accepted/dismissed) | ≥ 0.80 |
| Recall on obvious GAP (empty slot) | ≥ 0.70 |
| Median job time | &lt; 2 min / bay photo set |
| Stock false mutations | **0** |

---

### Phase 4 — Optional scale — **PG4**

- Multi-photo stitch / panorama  
- Endcap templates library  
- Supplier-shared packshot CDN (with rights)  
- REPORTS_PRO: compliance % by location / section  
- CUSTOMER_ASSIST: auto-ticket “gap on Dairy shelf 2”  
- **Not:** continuous CCTV unless a separate product decision  

---

## 6. Cross-role integration

| Surface | Impact |
|---------|--------|
| Retailer | Only role that authors planograms / runs audits |
| Supplier | Optional packshot contribute later; no compliance UI |
| Warehouse / driver / factory / payload | **None** — no POS/planogram leakage |
| Reorder / sell-through (L3) | May use persistent GAP findings as soft demand hint (future); not in PG3 |

---

## 7. Security & compliance

- Photos in retailer-scoped GCS prefixes; signed URLs short-lived  
- IDOR: all APIs scoped by `RetailerOrgId` + location ACL  
- Vision worker SA: read photos + packshots, write findings only  
- No PAN / payment data in vision path  
- Model/dataset license review before commercial training  
- Staff may appear in aisle photos — retention + delete-on-request  

---

## 8. Testing strategy

| Layer | What |
|-------|------|
| Unit | Slot diff (expected vs detected), facings math, pack enable graph |
| Integration | Publish planogram → audit → findings; vision flag off = 404 |
| CV offline | Golden shelf images + expected JSON fixtures in `planogram-vision` repo/package |
| E2E | Human audit path on SSMR without GPU |
| Negatives | Vision never calls stock adjust APIs |

---

## 9. Suggested implementation order

1. Finish / stabilize SECTIONS usage in pilots (already shipped).  
2. **PG1** structure when a Mode L retailer asks “where should this SKU sit?”  
3. **PG2** photo audits to collect labels.  
4. Only then **PG3** sidecar — start with **one pilot category** (e.g. drinks) not full store.  
5. Expand SKU embedding coverage; never boil the ocean on day one.

---

## 10. Open questions

1. Packshot SoT: retailer-uploaded only, or pull from supplier catalog images?  
2. Minimum facings: enforce in POS/stock or advisory only?  
3. Who may publish planograms: OWNER/ADMIN only, or SECTION_LEAD?  
4. GPU budget for SSMR vs separate vision project in GCP?  
5. Commercial license clearance for SKU-110K / chosen YOLO weights?  
6. Should GAP findings ever suggest auto-order qty (coupling to L3)?  

---

## 11. Risk register

| Risk | Mitigation |
|------|------------|
| Scope steals Retail OS / L1–L3 focus | Hard gate: no PG3 until PG1–PG2 done **and** L1–L3 not slipping |
| Low precision destroys trust | PENDING_REVIEW always; show confidence; dismiss feedback loop |
| Embedding collapse on similar SKUs | Packshot quality bar; human confirm on low conf |
| Cost blowup | Async jobs, pilot SKU cap, kill switch |
| OSS license surprise | Legal check before train |
| Treating vision as inventory SoT | Explicit product law: advisory only |

---

## 12. Documentation deliverables

- This file: `docs/PLANOGRAM_VISION_PLAN.md`  
- `docs/RETAILER_PLANOGRAM.md` (when PG1 ships)  
- `docs/RETAILER_PLANOGRAM_VISION.md` (when PG3 ships)  
- Update `docs/RETAILER_CAPABILITY_PACKS.md` with `PLANOGRAM` / `PLANOGRAM_VISION`  
- Update Next-Layer L11 section to point here  
- Update `ROLE_ROW_PARITY_MATRIX.md` when clients land  

---

## 13. Immediate next steps (after approval)

1. Confirm Phase 0: vision stays deferred; no CV sprint.  
2. When a chain pilot needs layout: schedule **PG1** only.  
3. Do **not** clone Retail-Shelf-Monitoring into the monorepo.  
4. Optionally spike (time-boxed ≤ 2 days): YOLO detect on 10 shelf photos — **spike branch only**, no product merge — to validate feasibility for later PG3.  

---

## Appendix A — Mapping to Next-Layer L11

| L11 item | This plan |
|----------|-----------|
| CUSTOMER_ASSIST | Separate; may consume GAP findings in PG4 |
| Planogram vision | **PG3** |
| Non-AI prerequisite | **PG1–PG2** |
| “Nice for large retail” | Pack-gated; CORE never sees it |

---

## Appendix B — Reference links (external)

- https://github.com/Alijanloo/Retail-Shelf-Monitoring  
- https://github.com/laitalaj/cvpce  
- https://github.com/eg4000/SKU110K_CVPR19  
- https://alessiotonioni.github.io/publication/planogram (GP-180)  
- https://github.com/phenobarbital/ai-parrot (planogram pipelines package)  

---

*End of plan.*
