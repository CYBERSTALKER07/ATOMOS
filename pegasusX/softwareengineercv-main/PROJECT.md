# Project: Pegasus Enterprise Subpage Modernization

## Architecture
- **Framework**: Next.js 14+ (App Router), React 18, TypeScript, Tailwind CSS, Lucide React, Framer Motion.
- **Global Layout Shell**: `ClientLayout.tsx` (Providers: Theme, Language, CookieConsent, ReactLenis, SiteAssistant) wraps root layout. Subpages utilize `FleekPageShell.tsx` rendering `FleekNav` (`SiteNav` -> `PillNav`) and `Footer`.
- **Subpage Rendering Pipeline**:
  `app/[category]/page.tsx` -> `createCategoryHubPage(categoryId)` -> `getCategoryHub(categoryId)` -> `CategoryHubClient` -> `getHubLayoutConfig(hub.id)` -> `HubLayoutRenderer` -> `O9FleekPageLayout`.
- **Target Routes**:
  - `/platform` (Autonomous Logistics & AI Supply Chain Platform Hub)
  - `/capabilities` (Fleet & Execution Capabilities Hub)
  - `/operations` (Operational Workflows & Role Hub)
  - `/technology` (Distributed Architecture & Real-Time Engine Hub)
- **Design Tokens**:
  - Base: High-contrast dark tactical (`bg-black`, `border-white/10`) and light (`bg-[#F8FAFC]`, `border-zinc-200`).
  - Accent: Pegasus Emerald `#10b981` (`text-emerald-400`, `bg-emerald-500`, `border-emerald-500/30`).
  - Typography: Monospace accents (`font-mono`), crisp display headings, chamfered corner notches, tactical status dots.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Enterprise Solution Hero | Category kicker badge with pulse dot, high-impact H1, subheadline, dual CTAs ("Request Demo" / "Take Platform Tour"), Gartner Peer review card, and 4-column proof metric ticker. | M1 | Survey (o9 & Pegasus) |
| 2 | Differentiator Bento Grid | Modular feature grid ("Why Leaders Choose") with icon badges, concise value propositions, visual telemetry preview containers, and tactical corner brackets. | M1 | Survey (o9 & Pegasus) |
| 3 | Quantified Impact & Value Tabs | Interactive horizontal tabs with 4-column KPI cards displaying quantified metrics, context copy, and delta callouts. | M1 | Survey (o9 & Pegasus) |
| 4 | Interactive Use Cases Deep-Dive | Horizontal operational use case selector / carousel with multi-step workflow pipelines and technical SLA specifications. | M1 | Survey (o9 & Pegasus) |
| 5 | Enterprise FAQ Accordion | Accessible WAI-ARIA collapsible accordion answering enterprise architecture, ERP/WMS/TMS integrations, security, and rollout timelines. | M1 | Survey (o9 & Pegasus) |
| 6 | Contextual Tour & Conversion CTA | Bottom dual-card split banner with tactical emerald radial glow, demo booking, and self-guided tour. | M1 | Survey (o9 & Pegasus) |
| 7 | Full Narrative Wiring in O9FleekPageLayout | Integrate all 6 modular components into `O9FleekPageLayout.tsx` so hub layouts render the complete narrative flow. | M1 | Survey (Pegasus Audit) |
| 8 | Platform Subpage Modernization | Real, authentic Pegasus Autonomous Supply Chain Platform data in `hubLayouts.ts` for `/platform` (Bento, Value Tabs, Use Cases, FAQs). | M2 | Requirements R2 |
| 9 | Capabilities Subpage Modernization | Real, authentic Pegasus Fleet, Load-Packing & Dispatch data in `hubLayouts.ts` for `/capabilities` (Bento, Value Tabs, Use Cases, FAQs). | M2 | Requirements R2 |
| 10 | Operations Subpage Modernization | Real, authentic Pegasus 6-Role Operational Workflows & Playbooks data in `hubLayouts.ts` for `/operations` (Bento, Value Tabs, Use Cases, FAQs). | M3 | Requirements R2 |
| 11 | Technology Subpage Modernization | Real, authentic Pegasus Go Modular Monolith, Spanner & Outbox data in `hubLayouts.ts` for `/technology` (Bento, Value Tabs, Use Cases, FAQs). | M3 | Requirements R2 |
| 12 | Responsive Layout Polish | Fluid responsiveness verified across 375px mobile, 768px tablet, 1024px laptop, and 1440px+ desktop. | M4 | Requirements R3 |
| 13 | Dual-Theme Parity & Styling Consistency | Light (`bg-[#F8FAFC]`) and dark (`bg-black`) theme verification with zero contrast or clipping issues. | M4 | Requirements R3 |
| 14 | Build Integrity & Zero-Regression Verification | `npm run build` succeeds with exit code 0, 0 TS errors, 100% routes functional, PillNav and SiteAssistant intact. | M4 | Acceptance Criteria |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 1 | Enterprise Component System & Layout Pipeline | Implement `O9EnterpriseFaq.tsx`, polish `O9DifferentiatorGrid.tsx`, `O9BusinessValueSection.tsx`, `O9CapabilityShowcase.tsx`, `O9HeroSplit.tsx`, `O9SplitTourCTA.tsx`, and wire complete narrative in `O9FleekPageLayout.tsx`. | none | DONE |
| 2 | Platform & Capabilities Content Modernization | Enrich `hubLayouts.ts` with domain-authentic Pegasus data for `/platform` and `/capabilities` (Bento, KPI Tabs, Use Cases, FAQs). | M1 | DONE |
| 3 | Operations & Technology Content Modernization | Enrich `hubLayouts.ts` with domain-authentic Pegasus data for `/operations` and `/technology` (Bento, KPI Tabs, Use Cases, FAQs). | M1 | DONE |
| 4 | Responsive Polish, Dual Theme & Build Verification | Audit responsiveness, theme switching, route health, and execute `npm run build` verification. | M2, M3 | DONE |

## Interface Contracts
### `HubLayoutRenderer` ↔ `O9FleekPageLayout`
```typescript
export interface O9FleekPageLayoutProps {
  hero: React.ReactNode;
  differentiators?: {
    kicker?: string;
    title: string;
    description?: string;
    cards: Array<{
      icon: LucideIcon | string;
      title: string;
      description: string;
      badge?: string;
      previewType?: 'graph' | 'telemetry' | 'code' | 'metric';
    }>;
  };
  businessValue?: {
    kicker?: string;
    title: string;
    description?: string;
    tabs: Array<{
      id: string;
      label: string;
      stats: Array<{
        value: string;
        label: string;
        subtext?: string;
        trend?: string;
      }>;
    }>;
  };
  capabilities?: {
    kicker?: string;
    title: string;
    description?: string;
    items: Array<{
      id: string;
      title: string;
      description: string;
      workflowSteps?: string[];
      sla?: string;
      image?: string;
      tag?: string;
    }>;
  };
  faq?: {
    kicker?: string;
    title: string;
    description?: string;
    items: Array<{
      id: string;
      question: string;
      answer: string;
      category?: string;
    }>;
  };
  details?: React.ReactNode;
  cta?: React.ReactNode;
}
```

## Code Layout
- `app/components/fleek/o9/O9FleekPageLayout.tsx`: Master narrative section composer (Hero -> Bento -> Value Tabs -> Use Cases -> Topic Grid -> FAQ -> CTA).
- `app/components/fleek/o9/O9EnterpriseFaq.tsx`: Reusable Enterprise FAQ Accordion component with tactical styling and ARIA disclosure.
- `app/components/fleek/o9/O9DifferentiatorGrid.tsx`: Bento grid component ("Why Leaders Choose") with icons and preview containers.
- `app/components/fleek/o9/O9BusinessValueSection.tsx`: Value & KPI tabs component with horizontal switching.
- `app/components/fleek/o9/O9CapabilityShowcase.tsx`: Operational use cases and capability showcase component.
- `app/components/fleek/o9/O9HeroSplit.tsx`: High-impact split hero with dual CTAs and proof metric ticker.
- `app/components/fleek/o9/O9SplitTourCTA.tsx`: Bottom conversion banner with demo booking and self-guided tour.
- `app/components/explore/hubs/HubLayoutRenderer.tsx`: Hub layout assembler connecting hub configurations to `O9FleekPageLayout`.
- `app/lib/explore/hubLayouts.ts`: Authoritative domain data configurations for `/platform`, `/capabilities`, `/operations`, `/technology`.
- `app/data/topicPages/`: Topic taxonomy and category hub definitions.
