import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(process.cwd(), 'patent-dossier');

function readJson(relativePath) {
  return JSON.parse(fs.readFileSync(path.join(root, relativePath), 'utf8'));
}

function writeText(relativePath, content) {
  fs.writeFileSync(path.join(root, relativePath), `${content.trimEnd()}\n`, 'utf8');
}

function escapeTable(value) {
  return String(value ?? '').replaceAll('|', '\\|');
}

function unique(values) {
  return [...new Set(values.filter(Boolean))];
}

function overlaps(first = [], second = []) {
  return first.some((value) => second.includes(value));
}

const figureCatalog = readJson('figure-production-catalog.json');
const figureGroups = readJson('counsel/figure-groupings.json').groups;
const claimFamilies = readJson('counsel/claim-families.json').families;
const crossReferences = readJson('counsel/invention-cross-reference.json').entries;

const groupByShotId = new Map();
for (const group of figureGroups) {
  for (const shotId of group.shotRefs) {
    const existing = groupByShotId.get(shotId) || [];
    existing.push(group);
    groupByShotId.set(shotId, existing);
  }
}

const familyById = new Map(claimFamilies.map((family) => [family.familyId, family]));

function relatedClaimIdsForShot(shot) {
  const direct = Array.isArray(shot.claimFamilyHints) ? shot.claimFamilyHints : [];
  const derived = claimFamilies
    .filter((family) => Array.isArray(family.surfaceRefs) && family.surfaceRefs.includes(shot.surfaceId))
    .map((family) => family.familyId);
  return unique([...direct, ...derived]);
}

function relatedClaimTitlesForShot(shot) {
  return relatedClaimIdsForShot(shot).map((familyId) => familyById.get(familyId)?.title || familyId);
}

function relatedFiguresForFamily(family) {
  return figureCatalog.shots.filter((shot) => relatedClaimIdsForShot(shot).includes(family.familyId));
}

function relatedCrossRefsForFamily(family) {
  return crossReferences.filter((entry) => overlaps(entry.frontend || [], family.surfaceRefs || []) || overlaps(entry.backend || [], family.backendRefs || []));
}

function buildFormalFigurePacket() {
  const groupedCount = figureCatalog.shots.filter((shot) => groupByShotId.has(shot.shotId)).length;
  const ungroupedCount = figureCatalog.shots.length - groupedCount;

  const groupSummaryRows = figureGroups.map((group) => {
    const count = figureCatalog.shots.filter((shot) => (groupByShotId.get(shot.shotId) || []).some((item) => item.groupId === group.groupId)).length;
    return `| ${escapeTable(group.groupId)} | ${escapeTable(group.title)} | ${count} | ${escapeTable(group.goal)} |`;
  });

  const scheduleRows = figureCatalog.shots.map((shot) => {
    const groups = (groupByShotId.get(shot.shotId) || []).map((group) => group.title);
    const claimTitles = relatedClaimTitlesForShot(shot);

    return `| FIG. ${shot.figureNumber} | ${escapeTable(shot.shotId)} | ${escapeTable(shot.surfaceId)} | ${escapeTable(shot.artifactRef)} | ${escapeTable(shot.viewType)} | ${escapeTable(groups.join('; ') || 'Supplemental / ungrouped')} | ${escapeTable(claimTitles.join('; ') || 'Pending counsel assignment')} | ${escapeTable(shot.caption)} |`;
  });

  return `
# Formal Figure Packet: Complete Figure Schedule

## Purpose
This packet is the complete formal figure schedule for the Leviathan patent dossier. It converts the live figure production catalog into a counsel-facing sheet that binds every figure number to a shot identifier, surface, dossier artifact, figure-group context, and claim-family alignment.

## Summary
- Total figures: ${figureCatalog.totalFigures}.
- Rendering profile: ${figureCatalog.renderingProfileRef}.
- Rendering mode: ${figureCatalog.renderingMode}.
- Figures linked to defined counsel groups: ${groupedCount}.
- Supplemental or ungrouped figures: ${ungroupedCount}.
- Canonical auth naming is normalized to web-auth-login and web-auth-register in the current catalog.

## Group Index
| Group ID | Title | Figure Count | Goal |
| --- | --- | ---: | --- |
${groupSummaryRows.join('\n')}

## Complete Schedule
| Figure | Shot ID | Surface ID | Artifact | View Type | Group | Related Claim Families | Caption |
| --- | --- | --- | --- | --- | --- | --- | --- |
${scheduleRows.join('\n')}

## Counsel Notes
- Where a figure currently shows "Pending counsel assignment", the figure remains valid but does not yet have a direct claim-family hint in the live catalog.
- The complete figure packet should be used as the master schedule for illustrators, counsel, and later international filing expansions.
`;
}

const eventSemantics = {
  'claim-family-order-fanout': [
    'Single-cart retailer intent is submitted once and atomically divided into supplier-scoped orders.',
    'MasterInvoice and supplier order rows commit inside one authoritative transaction.',
    'Post-commit order events propagate downstream only after authoritative commit succeeds.'
  ],
  'claim-family-ai-demand-loop': [
    'Historical order patterns are transformed into stored AI prediction rows with trigger timing.',
    'Predictions awaken into visible procurement or auto-order controls.',
    'Human review remains optional over time as machine agents can invoke the same procurement contract.'
  ],
  'claim-family-payment-settlement': [
    'Supplier-scoped payment credentials resolve before hosted or deep-link settlement begins.',
    'Settlement finality is established through webhook verification or sweeper reconciliation.',
    'Driver release or payment completion cues occur only after canonical settlement truth is written.'
  ],
  'claim-family-dispatch-routing': [
    'Loaded orders are selected and externally sequenced into a route order.',
    'Sequence truth is written back as the authoritative route order for driver and payload surfaces.',
    'Dispatch continuity persists across supplier control, payload sealing, and driver route execution.'
  ],
  'claim-family-telemetry-geofence': [
    'Driver coordinates enter a live telemetry hub and fan out to control surfaces.',
    'Redis-backed proximity checks produce approach semantics without mutating canonical order truth.',
    'Geofence signaling remains observability-first and non-blocking until later bounded transitions occur.'
  ],
  'claim-family-warehouse-seal': [
    'Checklist completion is the precondition for payload sealing.',
    'Sealing attaches a delivery token and emits PAYLOAD_SEALED as a machine-readable dispatch event.',
    'Truck readiness and driver departure remain separated by a two-key handshake.'
  ],
  'claim-family-offline-proof': [
    'Offline deliveries are buffered locally and replayed only after connectivity returns.',
    'Redis SETNX dedup and stale-timestamp rejection prevent duplicate credit or stale replay.',
    'Kafka emission is gated after dedup acquisition and any required pre-emit Spanner mutation.'
  ],
  'claim-family-reverse-logistics': [
    'Rejected or damaged delivery outcomes move into a quarantine or reconciliation path.',
    'Supplier-facing restock and write-off controls preserve audit and machine readability.',
    'Amendment and reconciliation outputs remain event-safe for downstream consumers.'
  ],
  'claim-family-human-machine-adapters': [
    'Human-facing controls are bounded state transitions rather than free-form actions.',
    'The same transition contract can later be satisfied by robots, policy engines, or autonomous agents.',
    'Machine substitution preserves downstream legal and operational effect without rewriting the state model.'
  ]
};

const strategicWeight = {
  'claim-family-order-fanout': 'Primary valuation claim',
  'claim-family-human-machine-adapters': 'Primary valuation claim',
};

const crossReferenceTitles = {
  'xref-commerce-to-dispatch': 'Commerce intent to dispatch',
  'xref-payment-to-driver-release': 'Payment settlement to driver release',
  'xref-map-to-proof': 'Live telemetry to proof capture',
  'xref-ai-to-procurement': 'Prediction to procurement activation',
  'xref-warehouse-to-reconciliation': 'Warehouse seal to reconciliation'
};

const representativeClaims = [
  'A logistics orchestration system comprising retailer client devices, supplier control interfaces, driver execution clients, payload-terminal interfaces, and at least one backend orchestration service configured to receive a single retailer checkout intent, atomically divide the single retailer checkout intent into supplier-scoped order records under a shared invoice context, persist the supplier-scoped order records in an authoritative datastore, and emit downstream order events only after authoritative commit succeeds.',
  'The system of claim 1, wherein the backend orchestration service associates supplier-specific settlement credentials with each supplier-scoped order record before payment-session creation so that heterogeneous supplier payment rails resolve through a canonical settlement path while invoice continuity is preserved for the retailer.',
  'A computer-implemented demand activation method comprising ingesting historical order activity, generating future-demand predictions, storing the future-demand predictions with activation timing, and later exposing the future-demand predictions as procurement controls or auto-order controls without requiring manual recreation of the predicted order state.',
  'The method of claim 3, wherein a machine agent invokes the same procurement contract that is exposed to a human operator through a retailer or supplier interface, thereby preserving contract continuity between human-assisted execution and machine-native execution.',
  'A settlement control system comprising a credential vault, a hosted or deep-link payment session generator, a webhook verifier, and a reconciliation sweeper, wherein settlement finality is written as canonical ledger truth before driver-release or payment-complete signals are propagated to delivery execution surfaces.',
  'A dispatch authorization method comprising externally optimizing loaded orders into an ordered route sequence, writing the ordered route sequence back as authoritative route truth, presenting manifest-completion controls to a payload actor, and emitting a machine-readable seal event only after checklist completion is satisfied and a dispatch token is attached.',
  'The method of claim 6, wherein truck readiness and vehicle departure remain separate bounded transitions such that a first transition authorizes a sealed ready state and a second transition activates in-transit execution.',
  'A telemetry and geofence signaling method comprising receiving driver coordinates into a live telemetry hub, distributing the driver coordinates to control interfaces, performing proximity evaluation in a fast-state cache, and emitting approach semantics without directly mutating canonical order state until a later bounded transition is satisfied.',
  'An offline proof synchronization method comprising locally buffering delivery-proof packets, hashing or otherwise binding the delivery-proof packets to manifest context, attempting deduplication through a replay-suppression key, rejecting stale packets older than a threshold age, committing any required pre-emit state mutation before event emission, and releasing the replay-suppression key when downstream emission fails.',
  'A reverse-logistics reconciliation system comprising quarantine-state creation for damaged or rejected inventory, supplier-facing restock or write-off controls, and an audit-safe reconciliation path that preserves downstream event safety while changing inventory disposition.',
  'A machine-substitutable logistics control system comprising human-facing control surfaces that each map to explicit bounded state transitions, wherein the bounded state transitions are satisfiable by either a human actor or a machine actor while preserving the same downstream legal and operational effect.',
  'The system of claim 11, wherein a manifest authorization envelope accepts an actor identifier, an actor kind, proof metadata, a state-before value, and a state-after value so that a warehouse robot can satisfy the same manifest-seal contract previously satisfied by a human payload operator.'
];

function buildRepresentativeClaimSection() {
  return representativeClaims
    .map((claim, index) => `### Claim ${index + 1}\n${claim}`)
    .join('\n\n');
}

function buildCrossReferenceOverview() {
  return crossReferences.map((entry, index) => {
    const title = crossReferenceTitles[entry.entryId] || entry.entryId;
    const frontend = unique(entry.frontend || []).join(', ');
    const backend = unique(entry.backend || []).join(', ');
    const stores = unique(entry.stores || []).join(', ');
    const events = unique(entry.events || []).join(', ');

    return `### ${index + 1}. ${title}\nFrontend surfaces: ${frontend}.\nBackend anchors: ${backend}.\nData substrates: ${stores}.\nRepresentative events: ${events}.\nHuman touchpoint: ${entry.humanTouchpoint}.\nMachine-native path: ${entry.machineNativePath}.`;
  }).join('\n\n');
}

function buildDetailedDescriptionSections() {
  return claimFamilies.map((family, index) => {
    const figures = relatedFiguresForFamily(family);
    const figureRefs = figures.map((shot) => `FIG. ${shot.figureNumber}`).join(', ');
    const frontendSources = unique(family.surfaceRefs || []).join(', ');
    const backendSources = unique(family.backendRefs || []).join(', ');
    const events = (eventSemantics[family.familyId] || ['Counsel event semantics pending additional narrowing.']).map((item) => `- ${item}`).join('\n');
    const evidence = (family.evidence || []).map((item) => `- ${item}`).join('\n');
    const crossRefDetails = relatedCrossRefsForFamily(family);
    const humanTouchpoints = unique(crossRefDetails.map((entry) => entry.humanTouchpoint));
    const machinePaths = unique(crossRefDetails.map((entry) => entry.machineNativePath));
    const humanSection = humanTouchpoints.length ? humanTouchpoints.map((item) => `- ${item}`).join('\n') : '- Human touchpoint will be narrowed during counsel review.';
    const machineSection = machinePaths.length ? machinePaths.map((item) => `- ${item}`).join('\n') : '- Machine-native substitution path will be narrowed during counsel review.';

    return `### ${index + 1}. ${family.title}\n${family.thesis}\n\nRepresentative figures: ${figureRefs || 'Pending figure linkage.'}.\nRepresentative frontend surfaces: ${frontendSources || 'Pending frontend linkage.'}.\nRepresentative backend anchors: ${backendSources || 'Pending backend linkage.'}.\n\nOperational semantics:\n${events}\n\nEvidence and visible manifestations:\n${evidence || '- Pending evidentiary detail.'}\n\nHuman touchpoints:\n${humanSection}\n\nMachine-native substitution path:\n${machineSection}`;
  }).join('\n\n');
}

function buildProvisionalDraft() {
  const figureDescriptions = figureCatalog.shots.map((shot) => `- ${shot.caption}`).join('\n');
  const summaryBullets = [
    'A single retailer intent is atomically converted into supplier-scoped orders while preserving invoice unity, idempotency, and post-commit event determinism.',
    'Historical order behavior is transformed into stored predictions that later awaken into retailer procurement and supplier planning controls.',
    'Supplier-specific settlement credentials, hosted sessions, webhook verification, and sweeper recovery converge on a canonical payment truth before delivery release.',
    'Loaded orders are externally sequenced and written back as authoritative route truth that remains continuous across supplier, payload, and driver surfaces.',
    'Payload manifest completion and truck sealing produce a machine-readable dispatch authorization that remains distinct from later departure activation.',
    'Driver telemetry and proximity semantics remain observability-first, emitting geofence-related signals without prematurely mutating canonical order state.',
    'Offline delivery proof is buffered, deduplicated, freshness-checked, and replayed into authoritative systems only when downstream safety conditions are satisfied.',
    'Reverse-logistics reconciliation preserves quarantine, restock, and write-off pathways while maintaining audit continuity.',
    'Human-facing controls are explicit adapters into bounded contracts that later machine actors can satisfy without rewriting the downstream state model.'
  ].map((item) => `- ${item}`).join('\n');

  return `
# Provisional Patent Draft

## Title
Machine-Orchestrated Multi-Role Logistics Ecosystem With Supplier-Scoped Order Fan-Out, Settlement Recovery, Dispatch Authorization, Replay-Controlled Offline Proof, and Machine-Substitutable State Contracts

## Draft Status
This is a provisional-style working draft assembled from live dossier artifacts, live figure mappings, and implementation-grounded claim families. It is intended to accelerate counsel review, jurisdiction-specific claim drafting, and Uzbekistan-priority plus later PCT packaging.

## Suggested Priority Statement
This draft is structured for a first-filed priority submission in Uzbekistan followed by later international expansion. Formal jurisdiction-specific header fields, inventor declarations, and attorney formatting can be added without changing the technical disclosure below.

## Technical Field
The disclosure relates to computer-implemented logistics orchestration, distributed order processing, supplier-scoped payment settlement, route sequencing, warehouse dispatch authorization, mobile delivery proof synchronization, telemetry signaling, and human-to-machine transition contracts for future autonomous execution.

## Background
Conventional logistics platforms typically divide procurement, dispatch, payment, proof of delivery, inventory reconciliation, and warehouse operations into disconnected products. Those systems often require manual re-entry between supplier, driver, retailer, and warehouse tools; they also struggle to preserve authoritative transaction truth when delivery devices go offline, when multiple suppliers participate in a single retailer basket, or when payment completion and route release must remain synchronized. Existing systems likewise tend to treat human user interfaces as the invention boundary, making later autonomous execution difficult because the downstream contract is not explicitly separated from the current human adapter.

The Leviathan system addresses those deficiencies by organizing the ecosystem as a bounded operating fabric. Retailer intent, supplier order partitions, payment truth, route order, manifest sealing, telemetry, proof replay, and reconciliation are exposed through role-specific surfaces but are governed by a shared contract layer. That shared contract layer allows current human operators to execute the flow while preserving a path for later machine-native execution.

## Abstract
An orchestrated logistics system receives a single retailer checkout intent and atomically partitions the intent into supplier-scoped order records under a shared invoice context. The system resolves supplier-specific settlement credentials, sequences loaded orders into an authoritative route order, and converts manifest completion into a machine-readable dispatch authorization distinct from vehicle departure. Telemetry coordinates are distributed to live control surfaces while geofence-related approach semantics remain non-blocking with respect to canonical order state. Offline delivery proof packets are buffered locally, deduplicated through replay-control keys, checked for freshness, and replayed only after required state mutations and downstream safety conditions are satisfied. Reverse-logistics flows preserve quarantine, restock, and write-off continuity. Human-facing controls operate as adapters into bounded state contracts so that future machine actors, including warehouse robots or autonomous route agents, can satisfy the same contracts without altering downstream legal or operational effect.

## Summary Of The Disclosure
${summaryBullets}

## Representative Claims For Counsel Refinement
${buildRepresentativeClaimSection()}

## Brief Description Of The Drawings
The complete formal figure schedule is maintained in counsel/formal-figure-packet.full.md. The current figures for this draft are as follows:

${figureDescriptions}

## Detailed Description
### System Architecture Overview
In exemplary embodiments, the disclosed system includes a supplier portal, retailer applications, driver applications, and a payload terminal connected to one or more backend orchestration services. Authoritative transactional truth may be stored in a distributed relational datastore such as Google Cloud Spanner. Post-commit propagation may occur through an event fabric such as Kafka. Fast-state functions, including proximity evaluation and replay suppression, may be handled through a cache layer such as Redis. Live coordination, telemetry fan-out, and settlement-status propagation may be distributed through websocket or similar push channels. Local mobile storage may buffer proofs, scans, and correction packets during periods of degraded connectivity.

The disclosed system treats each visible interface as a role-specific shell over bounded state transitions. Retailer, supplier, driver, and payload actors may each invoke the system through different surfaces while still satisfying the same contract layer. This separation allows the system to preserve legal and operational effect even when a later machine actor replaces one or more human entry points.

### Exemplary Cross-System Flows
${buildCrossReferenceOverview()}

### Claim-Family Embodiments
${buildDetailedDescriptionSections()}

## Filing Notes
- The full 142-figure schedule is maintained in counsel/formal-figure-packet.full.md.
- The family-to-figure and family-to-backend mapping is maintained in counsel/claim-chart.full.md.
- Machine-substitution contract language is expanded in counsel/machine-substitution-contract.md.
- Human-authored replay-control ordering and AI-assistance boundaries are preserved in counsel/human-ai-contribution-log.md.
- Cross-surface subsystem mapping is maintained in counsel/invention-cross-reference.json.
`;
}

function buildClaimChart() {
  const sections = claimFamilies.map((family, index) => {
    const figures = relatedFiguresForFamily(family);
    const frontendArtifacts = unique(figures.map((shot) => shot.artifactRef));
    const figureRefs = figures.map((shot) => `FIG. ${shot.figureNumber} (${shot.shotId})`).join(', ');
    const events = (eventSemantics[family.familyId] || ['Counsel event semantics pending additional narrowing.']).map((item) => `- ${item}`).join('\n');
    const evidence = (family.evidence || []).map((item) => `- ${item}`).join('\n');
    const backendSources = (family.backendRefs || []).map((item) => `- ${item}`).join('\n');
    const frontendSources = frontendArtifacts.length ? frontendArtifacts.map((item) => `- patent-dossier/${item}`).join('\n') : '- Pending figure-surface linkage';

    return `
## ${index + 1}. ${family.title}
Family ID: ${family.familyId}
Priority: ${family.priority}${strategicWeight[family.familyId] ? ` | ${strategicWeight[family.familyId]}` : ''}

### Thesis
${family.thesis}

### Exact Figures
${figureRefs || 'No mapped figures yet.'}

### Frontend And Dossier Sources
${frontendSources}

### Backend Contracts And Source Files
${backendSources}

### Event Semantics
${events}

### Evidentiary Signals
${evidence}
`;
  });

  return `
# Claim Chart

## Purpose
This claim chart maps every current claim family to exact figures, dossier artifacts, backend contract anchors, and event semantics. It is intended for counsel refinement, figure review, and conversion into jurisdiction-specific claim language.

## Strategic Notes
- The strongest commercial claim families remain unified retailer intent fan-out and human-operable shells designed for machine substitution.
- Exact figure coverage is driven from the live figure production catalog rather than hand-maintained lists.
- Event semantics are recorded here as system behavior descriptions anchored to the implemented backend and architecture packet.

${sections.join('\n')}
`;
}

writeText('counsel/formal-figure-packet.full.md', buildFormalFigurePacket());
writeText('counsel/claim-chart.full.md', buildClaimChart());
writeText('counsel/provisional-patent-draft.md', buildProvisionalDraft());

console.log('Wrote counsel/formal-figure-packet.full.md');
console.log('Wrote counsel/claim-chart.full.md');
console.log('Wrote counsel/provisional-patent-draft.md');