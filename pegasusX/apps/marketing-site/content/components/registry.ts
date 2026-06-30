import type { ComponentDocMeta } from "./types";

export const COMPONENT_DOCS: ComponentDocMeta[] = [
  {
    slug: "motion-tokens",
    title: "MotionTokens",
    description: "Shared duration, easing, and spring tokens from @pegasusx/motion-tokens.",
    props: [
      { name: "duration", type: "Record<string, number>", description: "Seconds-based duration scale." },
      { name: "easing", type: "Record<string, [number, number, number, number]>", description: "Cubic-bezier tuples." },
      { name: "reducedMotionTransition", type: "(prefersReduced, transition) => Transition", description: "Collapses motion when reduced." },
    ],
    motionSpec: "emphasizedDecelerate [0.05, 0.7, 0.1, 1] for hero reveals; standard [0.2, 0, 0, 1] for UI.",
    usedIn: [
      { role: "Retailer", surface: "Desktop portal" },
      { role: "All", surface: "Native + web transitions" },
    ],
    snippet: `import { duration, easing } from "@pegasusx/motion-tokens";\n\ngsap.to(el, {\n  duration: duration.medium4,\n  ease: easing.emphasizedDecelerate,\n});`,
  },
  {
    slug: "portal-button",
    title: "PortalButton",
    description: "Hand-rolled M3 button classes from portal-ui.css — primary, outline, ghost.",
    props: [
      { name: "className", type: "string", default: "portal-btn portal-btn--primary", description: "Variant via BEM modifier." },
      { name: "disabled", type: "boolean", default: "false", description: "Disables interaction." },
    ],
    motionSpec: "Hover: translateY(-1px), 200ms standard easing.",
    usedIn: [
      { role: "Supplier", surface: "Portal" },
      { role: "Warehouse", surface: "Portal + mobile" },
    ],
    snippet: `<button type="button" className="portal-btn portal-btn--primary">\n  Execute dispatch\n</button>`,
  },
  {
    slug: "pulse-timeline",
    title: "PulseTimeline",
    description: "Scrolling activity strip with PulseEvent rows from @pegasusx/pulse-ui.",
    props: [
      { name: "events", type: "PulseEvent[]", description: "Activity feed items." },
      { name: "loading", type: "boolean", default: "false", description: "Loading state." },
      { name: "onSelect", type: "(event) => void", description: "Row click handler." },
    ],
    motionSpec: "List stagger via motionVariants.listItem — 40ms cap delay.",
    usedIn: [
      { role: "Supplier", surface: "Network pulse panel" },
      { role: "Factory", surface: "Dashboard" },
    ],
    snippet: `import { PulseTimeline } from "@pegasusx/pulse-ui";\n\n<PulseTimeline events={events} />`,
  },
  {
    slug: "portal-card",
    title: "PortalCard",
    description: "desk-card / md-card bento tiles with stat, list, control, anchor sizes.",
    props: [
      { name: "className", type: "string", description: "desk-card + size modifiers." },
      { name: "children", type: "ReactNode", description: "Card content." },
    ],
    motionSpec: "Scroll reveal: opacity + y 8px, medium2 duration.",
    usedIn: [{ role: "Supplier", surface: "Dashboard bento grid" }],
    snippet: `<div className="desk-card p-4">\n  <p className="text-2xl font-light">42</p>\n</div>`,
  },
  {
    slug: "page-chrome",
    title: "PageChrome",
    description: "Operational page shell with title, description, actions, loading/error slots.",
    props: [
      { name: "title", type: "string", description: "Page heading." },
      { name: "description", type: "string", description: "Subtitle." },
      { name: "actions", type: "ReactNode", description: "Header action slot." },
    ],
    motionSpec: "pageEnter variant — opacity + y ±4, short4 duration.",
    usedIn: [{ role: "Warehouse", surface: "All portal pages" }],
    snippet: `import { PageChrome } from "@pegasusx/ui-kit/portal";\n\n<PageChrome title="Dispatch" actions={<Button />}>...</PageChrome>`,
  },
  {
    slug: "explain-banner",
    title: "ExplainStatusBanner",
    description: "Operational error guidance with title, summary, and next_steps.",
    props: [
      { name: "explain", type: "StatusExplain", description: "Structured guidance payload." },
      { name: "errorCode", type: "string", description: "Fallback code lookup." },
    ],
    motionSpec: "modalEnter scale 0.97 for alert surfaces.",
    usedIn: [{ role: "Factory", surface: "Loading bay" }],
    snippet: `import { ExplainStatusBanner } from "@pegasusx/explain-ui";\n\n<ExplainStatusBanner explain={explain} />`,
  },
  {
    slug: "kpi-stat-card",
    title: "KpiStatCard",
    description: "Warehouse-style stat tile with count-up on scroll into view.",
    props: [
      { name: "value", type: "number", description: "Target count." },
      { name: "label", type: "string", description: "Metric label." },
    ],
    motionSpec: "Count-up over 30 frames when IntersectionObserver fires.",
    usedIn: [{ role: "Warehouse", surface: "Insights dashboard" }],
    snippet: `<div className="kpi-stat-card">\n  <p className="kpi-stat-card__value">{value}</p>\n</div>`,
  },
  {
    slug: "fleet-route-map",
    title: "FleetRouteMap",
    description: "MapLibre dark map with route lines and driver markers — demo static routes.",
    props: [
      { name: "routes", type: "DemoFleetRoute[]", description: "GeoJSON-compatible routes." },
      { name: "className", type: "string", description: "Container class." },
    ],
    motionSpec: "Marker pulse via CSS; route draw on scroll for landing section.",
    usedIn: [
      { role: "Warehouse", surface: "FleetLiveMap" },
      { role: "Supplier", surface: "FleetLiveMap" },
    ],
    snippet: `<DemoFleetMap className="h-96 w-full" />`,
  },
  {
    slug: "topology-graph",
    title: "TopologyGraph",
    description: "Supplier topology editor node graph with animated edges.",
    props: [
      { name: "nodes", type: "TopologyNode[]", description: "Warehouse/factory/retailer nodes." },
      { name: "edges", type: "TopologyEdge[]", description: "Supply relationships." },
    ],
    motionSpec: "Edge dashoffset animation, 2s linear loop.",
    usedIn: [{ role: "Supplier", surface: "Topology editor" }],
    snippet: `<TopologyGraph nodes={nodes} edges={edges} />`,
  },
  {
    slug: "status-chip",
    title: "StatusChip",
    description: "M3-style order lifecycle state chips.",
    props: [
      { name: "status", type: "OrderStatus", description: "PENDING | LOADED | IN_TRANSIT | ARRIVED | COMPLETED" },
    ],
    motionSpec: "Color token per state; no motion on chip itself.",
    usedIn: [{ role: "All", surface: "Order lists" }],
    snippet: `<span className="status-chip status-chip--in-transit">IN TRANSIT</span>`,
  },
  {
    slug: "scroll-section",
    title: "ScrollSection",
    description: "Marketing PinSection primitive — pin + reveal API for landing sections.",
    props: [
      { name: "end", type: "string", default: "'+=200%'", description: "ScrollTrigger pin end." },
      { name: "onProgress", type: "(n: number) => void", description: "Scrub progress callback." },
    ],
    motionSpec: "ScrollTrigger scrub: true; disabled when prefers-reduced-motion.",
    usedIn: [{ role: "Marketing", surface: "Landing page" }],
    snippet: `<PinSection id="hero" end="+=200%" onProgress={setProgress}>...</PinSection>`,
  },
  {
    slug: "role-badge",
    title: "RoleBadge",
    description: "Six-role icon + label + color token badge.",
    props: [
      { name: "role", type: "RoleSlug", description: "supplier | warehouse | ..." },
      { name: "color", type: "string", description: "Role accent hex." },
    ],
    motionSpec: "None — static identity chip.",
    usedIn: [{ role: "Marketing", surface: "Roles parade" }],
    snippet: `<span className="role-badge"><span style={{ background: color }} /> Supplier</span>`,
  },
];

export function getComponentDoc(slug: string) {
  return COMPONENT_DOCS.find((doc) => doc.slug === slug);
}
