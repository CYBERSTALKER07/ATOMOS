import type { ComponentType } from "react";
import { MotionTokensPreview } from "./previews/MotionTokensPreview";
import { PortalButtonPreview } from "./previews/PortalButtonPreview";
import { PulseTimelinePreview } from "./previews/PulseTimelinePreview";
import { PageChromePreview } from "./previews/PageChromePreview";
import { ExplainBannerPreview } from "./previews/ExplainBannerPreview";
import { FleetRouteMapPreview } from "./previews/FleetRouteMapPreview";
import {
  KpiStatCardPreview,
  PortalCardPreview,
  RoleBadgePreview,
  ScrollSectionPreview,
  StatusChipPreview,
  TopologyGraphPreview,
} from "./previews/MiscPreviews";

export const COMPONENT_PREVIEW_MAP: Record<string, ComponentType> = {
  "motion-tokens": MotionTokensPreview,
  "portal-button": PortalButtonPreview,
  "pulse-timeline": PulseTimelinePreview,
  "portal-card": PortalCardPreview,
  "page-chrome": PageChromePreview,
  "explain-banner": ExplainBannerPreview,
  "kpi-stat-card": KpiStatCardPreview,
  "fleet-route-map": FleetRouteMapPreview,
  "topology-graph": TopologyGraphPreview,
  "status-chip": StatusChipPreview,
  "scroll-section": ScrollSectionPreview,
  "role-badge": RoleBadgePreview,
};
