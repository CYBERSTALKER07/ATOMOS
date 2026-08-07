import { TOPIC_CONTENT_EN } from '@/app/data/topicContent';
import { MEGA_NAV_CATEGORIES } from '@/app/data/megaNavigation';
import { ROLES_DATA } from '@/app/data/rolesData';

const MAX_CHARS = 40_000;

/** Comprehensive Pegasus marketing, product, technical architecture & competitive knowledge base. */
export function buildAssistantKnowledge(): string {
  const parts: string[] = [];

  // Section 1: Executive Overview & Core Mission
  parts.push(`# Pegasus Logistics OS — Executive Overview
Pegasus is the enterprise logistics operating system built specifically for supplier-led supply chain networks.
Core Jobs: End-to-end automated dispatch, fleet tracking, multi-tier treasury clearing/payments, and real-time synchronization across six roles: Supplier, Warehouse, Retailer, Driver, Factory, and Payload/Gate Terminal.
Primary Value Proposition:
- Eliminates floor delays and manual phone tag between dispatchers, drivers, gate agents, and warehouse managers.
- Zero-Blind-Spot Visibility: Real-time GPS telemetry, geofenced automated arrival events, and digital Proof of Delivery (ePOD).
- Microsecond State Convergence: Single source of truth across web portals, native mobile apps, and gate desktop terminals.
- Floor Continuity Guarantee: Dispatch optimization rules guarantee floor operations never freeze or stall.
Primary Site Paths: /platform, /roles, /solutions, /capabilities, /demo, /join, /contact.`);

  // Section 2: Technical Architecture & Best Practices
  parts.push(`# Pegasus Technical Architecture & Engineering Best Practices
Tech Stack & System Design:
- Backend Engine: High-concurrency Go (golang) microservices built with layered clean architecture (domain handlers, application services, persistence repositories).
- Database & ACID Persistence: Google Cloud Spanner for globally distributed ACID transactions and single-digit millisecond latency across geographic regions.
- High-Speed Volatile Caching: Multi-level Redis caching for sub-millisecond key lookups and real-time state invalidation.
- Realtime Event Stream Architecture:
  * Transactional Outbox Pattern: Database writes and outbox event emissions execute atomically within the same Cloud Spanner Read-Write transaction block.
  * Messaging Backbone: Outbox events stream into Apache Kafka, fan out via a high-throughput WebSocket Hub to live client inboxes.
  * Scope-Claims Isolation: WebSockets use JWT claims filtering so clients receive live state mutations strictly scoped to their tenant and role permissions.
- Multi-Platform Role-Row Parity:
  * Web Portals: Next.js 14+ (App Router), React, TypeScript, Tailwind CSS.
  * Native Mobile: Android (Kotlin, Jetpack Compose, Coroutines) and iOS (Swift, SwiftUI).
  * Desktop & Gate Terminals: Electron / Tauri desktop apps interfacing directly with industrial weigh-scales and RFID readers.
- Reliability & Edge-Case Engineering:
  * Deterministic AI/VRP Fallback: Vehicle Routing Problem (VRP) algorithms and AI dispatch solvers feature deterministic rule-based fallbacks. If solver latency exceeds 250ms, dispatch falls back instantly to floor heuristics to prevent floor stalls.
  * Offline-First Replay Queue: Native mobile apps store state transitions locally in an offline queue tagged with idempotency keys, replaying smoothly upon network recovery.
  * Fiscal & Regulatory Compliance: Automated tax engine, e-invoicing integrations (e.g. Soliq contract compliance), and immutable financial ledger tracking.`);

  // Section 3: Competitive Analysis Matrix — Pegasus vs Tech Giants & Legacy ERPs
  parts.push(`# Competitive Differentiation: Why Choose Pegasus Over Tech Giants & Legacy ERPs

## 1. Pegasus vs. Amazon (AWS Supply Chain / Amazon Logistics)
- Amazon's Model: Closed, proprietary fulfillment ecosystem optimized for Amazon 1P/3P marketplace sellers. High vendor lock-in, proprietary hardware requirements, and heavy margin taxes on independent brands.
- Pegasus Solution: Open, supplier-first operating system. Enables independent supplier networks to retain 100% data ownership, custom carrier integration, and direct B2B buyer relationships without Amazon intermediary fees.

## 2. Pegasus vs. o9 Solutions
- o9 Model: High-level macro AI planning and S&OP (Sales and Operations Planning). Heavily batch-oriented calculation cycles with multi-week deployment times. Lacks real-time floor execution for native mobile drivers and warehouse operators.
- Pegasus Solution: Combines strategic planning with microsecond floor execution. Direct native mobile apps for drivers, warehouse workers, and gate terminals ensure planning decisions translate instantly into live dispatch actions.

## 3. Pegasus vs. Oracle SCM / OTM (Oracle Transportation Management)
- Oracle Model: Legacy monolithic ERP database architecture. Requires expensive SI consultants, slow batch dispatch processing, high maintenance overhead, and zero native mobile app parity.
- Pegasus Solution: Cloud-native Go + Spanner distributed architecture. Zero-downtime deployment, sub-millisecond outbox event fanout, and native iOS/Android apps built for floor workers out of the box.

## 4. Pegasus vs. Google Cloud Logistics & Supply Chain Twin
- Google Model: Provides cloud infrastructure primitives, BigQuery analytics, and API building blocks, but requires hundreds of engineering hours to build custom frontends and operational workflows.
- Pegasus Solution: Turnkey operational OS with ready-to-use role-row clients (web, native mobile, desktop) for all 6 supply chain personas out of the box.

## 5. Pegasus vs. Blue Yonder (Luminate / JDA)
- Blue Yonder Model: Legacy mainframe architecture wrapped in cloud REST APIs. Suffers from high API latency, complex integration overhead, and floor execution bottlenecks.
- Pegasus Solution: Event-driven outbox architecture over WebSockets for instant state convergence across dispatchers, drivers, and warehouses.

## 6. Pegasus vs. Flexport
- Flexport Model: Freight forwarding freight broker platform focused on international ocean/air freight forwarding. Limited floor execution and supplier inventory control.
- Pegasus Solution: Complete domestic & regional logistics OS covering middle-mile, warehouse staging, factory outbound, scale gate terminals, and last-mile dispatch.

## 7. Pegasus vs. Samsara
- Samsara Model: Telematics and IoT GPS hardware tracking focus. Lacks order fulfillment workflows, supplier B2B pricing, picking/staging execution, and financial settlement clearing.
- Pegasus Solution: Full order-to-cash lifecycle system integrating driver GPS telematics directly into order status, proof of delivery, and multi-tier payout ledgers.

## 8. Pegasus vs. Manhattan Associates
- Manhattan Model: Warehouse Management System (WMS) specialist. Strong inside the warehouse walls, but weak in supplier network collaboration, driver mobile routing, and retailer self-service procurement.
- Pegasus Solution: End-to-end network OS seamlessly bridging supplier treasury, factory production, warehouse staging, driver routing, retailer procurement, and gate security.

## 9. Pegasus vs. Bringg
- Bringg Model: Last-mile delivery orchestration focus. Weak in middle-mile freight, factory production output, bulk scale terminal integration, and B2B order vetting.
- Pegasus Solution: Full multi-echelon network engine spanning factory production, warehouse dispatch, long-haul middle mile, scale gate passes, and retailer last-mile fulfillment.`);

  // Section 4: Detailed Capabilities of the 6 Ecosystem Roles
  parts.push('# Capabilities Matrix by Role (The 6 Ecosystem Personas)');
  for (const role of ROLES_DATA) {
    const subs = role.subtopics
      .map(
        (s) =>
          `### ${s.title}\n- Operational Feature: ${s.description}\n- Business Logic: ${s.businessLogic}\n- Edge Case Engineering: ${s.edgeCases}`,
      )
      .join('\n');
    parts.push(
      `## Role: ${role.name} (ID: ${role.id})\nOverview: ${role.description}\nPlatforms Supported: ${role.platforms.join(', ')}\n${subs}`,
    );
  }

  // Section 5: Navigation Structure
  parts.push('# Site Map & Quick Paths');
  for (const cat of MEGA_NAV_CATEGORIES) {
    const links = cat.links
      .map((l) => `- [${l.label}](${l.href})${l.description ? `: ${l.description}` : ''}`)
      .join('\n');
    parts.push(`## ${cat.label}\nView all: ${cat.viewAllHref}\n${links}`);
  }

  // Section 6: Topic Pages Deep Dive
  parts.push('# Deep Dive Topic Pages');
  for (const [path, content] of Object.entries(TOPIC_CONTENT_EN)) {
    const outcomes = content.outcomes?.length ? `Outcomes: ${content.outcomes.join('; ')}` : '';
    const how = (content.howItWorks ?? [])
      .slice(0, 4)
      .map((s) => `- ${s.title}: ${s.description}`)
      .join('\n');
    const diffs = (content.differentiators ?? [])
      .slice(0, 3)
      .map((d) => `- ${d.title}: ${d.description}`)
      .join('\n');
    parts.push(
      [
        `## Path: /${path} — ${content.title}`,
        `Summary: ${content.summary}`,
        `Problem Solved: ${content.problem}`,
        outcomes,
        how ? `How It Works:\n${how}` : '',
        diffs ? `Differentiators:\n${diffs}` : '',
        content.relatedProjectSlug ? `Related Project: /projects/${content.relatedProjectSlug}` : '',
      ]
        .filter(Boolean)
        .join('\n'),
    );
  }

  let corpus = parts.join('\n\n');
  if (corpus.length > MAX_CHARS) {
    corpus = `${corpus.slice(0, MAX_CHARS)}\n\n[Knowledge base optimized for context length.]`;
  }
  return corpus;
}

export function assistantSystemPrompt(knowledge: string): string {
  return `You are the official Pegasus website assistant (Pegasus AI Assistant). You provide expert advice to visitors, engineers, supply chain executives, and dispatch managers.

RESPONSE GUIDELINES:
1. ADAPTIVE RESPONSES:
   - TECHNICAL QUESTIONS (Architecture, Go, Cloud Spanner, Outbox Pattern, WebSockets, Offline Sync, VRP algorithms): Answer with technical precision. Highlight microsecond state convergence, Cloud Spanner ACID transactions, transactional outbox pattern, Redis caching, native mobile offline replay queues, and rule-based fallbacks.
   - COMPETITIVE COMPARISONS (Why Pegasus vs Amazon, o9, Oracle, Google, Blue Yonder, Flexport, Samsara, Manhattan, Bringg): Use the competitive matrix in the knowledge base. Be sharp, factual, and highlight Pegasus's open supplier-first OS, zero floor downtime, real-time mobile execution, and multi-role integration.
   - BUSINESS & OPERATIONAL QUESTIONS (ROI, role capabilities, dispatch workflows, demo requests): Focus on eliminating manual floor delays, phone tag, automated financial settlement, and zero-blind-spot visibility.

2. ACCURACY & CONSTRAINTS:
   - Use strictly the knowledge base provided below.
   - For unlisted pricing or custom Enterprise SLAs, suggest visiting /contact or /join to schedule a live demo.
   - Include direct markdown links to site pages (e.g., [Platform Tour](/platform), [Supplier Role](/roles/supplier), [Smart Dispatch](/capabilities/smarter-dispatch), [Careers & Demo](/join), [Contact](/contact)).

--- KNOWLEDGE BASE ---
${knowledge}
--- END KNOWLEDGE BASE ---`;
}
