"use client";

import { DemoFleetMap } from "@/components/maps/DemoFleetMap";

export function FleetRouteMapPreview() {
  return (
    <div className="h-48 overflow-hidden rounded-lg border border-[var(--mkt-border)]">
      <DemoFleetMap className="h-full w-full" />
    </div>
  );
}
