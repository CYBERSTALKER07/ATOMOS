---
name: patent-architect
description: 'Transform the V.O.I.D. monorepo into patent-application assets: legally agnostic specification narratives, UI line-art bounding-box JSON, algorithm diagrams, alternative embodiment matrices, and comprehensive feature catalogs. Use for patent generation, diagram extraction, IP lockdown, claim-support drafting, and UI line-art JSON generation.'
argument-hint: 'Describe scope: full monorepo or target apps/domains, output priority, and batch size before halt checkpoints.'
---

# Patent Architect (V.O.I.D. Ecosystem IP Extraction)

## Mission

Convert repository implementation reality into patent-ready artifacts without legal conclusions and without implementation code leakage in explanatory narratives.

## When To Use

Use this skill when the user asks for:

- patent generation
- IP extraction
- diagram extraction
- claim-support documentation
- embodiment matrices
- UI line-art JSON generation
- ecosystem IP lockdown documentation

## Required Inputs

Collect these inputs before execution if not provided:

1. Scope: full monorepo or selected apps and backend domains.
2. Priority: Output 1, 2, 3, or 4 first.
3. Batch size: how many apps or domains per pass before halting.
4. Figure namespace preference: global figure series or per-domain figure series.

If missing, default to full monorepo, Output 1 first, one domain batch per pass, global figure numbering.

## Non-Negotiable Directives

### 1. Tone And Format

- Narrative output must read like formal patent specification prose.
- Explanatory markdown must not include source code excerpts.
- In explanatory markdown, avoid programming symbols and implementation-style naming.
- Translate technical names into patent language equivalents.
- Keep wording legally agnostic and jurisdiction-neutral.

### 2. Completeness

- Exhaustive extraction is mandatory.
- Do not sample a subset when the user requests full extraction.
- If evidence is partial, label the section as partial and continue extraction.

### 3. Artifact Separation

- Narrative files contain patent prose only.
- Structured files contain JSON or Mermaid assets only.
- Do not mix narrative and machine-structured formats in the same section.

## Repository Sweep Protocol

### Step 1. Acknowledge And Initialize

- Confirm Patent Architect protocol start.
- Echo selected scope, output priority, and halt cadence.

### Step 2. Build The Internal Map

Sweep and map these sources:

- backend and worker logic from Go files and schema files
- event and state flows from Kafka and outbox paths
- geospatial and optimization logic from proximity and routing modules
- web surfaces from TypeScript and Next.js app routes
- Android surfaces from Kotlin and Compose screens
- iOS surfaces from Swift and SwiftUI views
- payload and terminal flows across relevant apps

### Step 3. Branching Logic

- If scope is full monorepo: process by role-domain waves.
- If scope is app-specific: finish all outputs for that app before switching.
- If token pressure is high: split by output type, then halt for confirmation.
- If conflicts appear between code and schema: report drift and continue with best-evidence extraction.

### Step 4. Evidence Traceability

For each extracted feature, track:

- source surface or domain
- lifecycle stage
- operator role involvement
- node dependency
- continuity behavior under missing-node permutations

## Output Package Specification

Generate these deliverables in every run unless user narrows scope.

### Output 1. UI Bounding-Box JSON For Line-Art

Create: patent-ui-layouts.json

Requirements:

- include every discovered screen across admin portal, payload terminal, driver, and retailer surfaces in scope
- describe layout using relational placement and percentages
- avoid absolute pixel values
- assign stable figure numbers
- include component identifiers and semantic element lists

Required object shape:

```json
{
  "app": "Payload Terminal",
  "screen": "Active Manifest View",
  "patent_figure_number": "Fig. 4A",
  "layout_prompt": "A monochrome line-art wireframe of a mobile interface.",
  "components": [
    { "id": "102", "type": "Header", "position": "Top 10%", "elements": ["Status Indicator", "H3 Hex ID"] },
    { "id": "104", "type": "Interactive List", "position": "Middle 70%", "elements": ["Draggable Order Cards"] }
  ]
}
```

### Output 2. Core Algorithm Diagrams

Create:

- patent-core-algorithms.md
- patent-core-algorithms-mindmaps.json

Required systems:

1. H3 Hexagon Grid Dispatch System
2. Freeze Lock and Manual Override State Machine
3. Payloader Idempotency and Transactional Outbox
4. Predictive Preorder Demand Engine

Requirements:

- Mermaid flowchart per system in markdown output
- companion JSON mind-map per system with nodes, transitions, guard conditions, and recovery paths
- each diagram includes figure label and short patent caption

### Output 3. Alternative Embodiment Permutation Matrix

Create: patent-alternative-embodiments.md

Required embodiment families:

- Base Claim: Supplier -> Warehouse with Payloader -> Driver -> Retailer
- Alternative Embodiment A: Supplier -> Driver -> Retailer
- Alternative Embodiment B: Supplier -> Warehouse -> Driver -> Retailer without payloader application involvement
- Alternative Embodiment C: Factory -> Retailer direct fulfillment

Directive:

- For every extracted feature, explain lifecycle continuity and data integrity when nodes are absent or bypassed.
- Include guardrails for state integrity, reconciliation integrity, and event consistency.

### Output 4. Comprehensive Feature Catalog Tables

Create: patent-feature-catalog.md

Generate exhaustive markdown tables with these required columns:

- Feature Name
- Technical Mechanism (Internal)
- Patent Description (Official)
- Node Dependency

Requirements:

- one row per operational feature
- group tables by domain and application surface
- maintain consistent terminology across all tables

## Execution Workflow

1. Initialize and confirm selected scope.
2. Run repository sweep for the active batch.
3. Generate Output 1 through Output 4 for that batch.
4. Halt and request confirmation before moving to next batch.
5. Continue batch-by-batch until requested scope is complete.

## Quality Gates

A run is complete only when all conditions are true:

1. Narrative outputs contain no source code excerpts.
2. Output 1 includes every screen in current batch scope.
3. Output 2 covers all four required core systems.
4. Output 3 includes base and all required alternative embodiments.
5. Output 4 includes complete feature rows with all required columns.
6. Figure numbering is unique and stable within the run.
7. Missing evidence is explicitly marked and queued for next sweep.

## Weak-Signal And Ambiguity Policy

Ask clarification immediately when any of these are ambiguous:

- desired legal depth versus engineering depth
- jurisdiction-specific claim style requirements
- preferred figure numbering convention
- strict app order for extraction sequence
- whether to include deprecated or legacy routes in the patent package

If user does not specify, use neutral defaults and proceed.

## Suggested Invocation Prompts

- Generate full Patent Architect package for the complete V.O.I.D. monorepo, batch by app family.
- Produce only Output 1 and Output 2 for payload and driver surfaces, then halt.
- Build alternative embodiment matrix first for warehouse-bypass and micro-fulfillment paths.
- Regenerate patent feature catalog for backend dispatch, payment, and telemetry domains only.
