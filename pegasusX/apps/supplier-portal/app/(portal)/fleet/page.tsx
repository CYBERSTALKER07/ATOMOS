"use client";

import Link from "next/link";
import FleetLiveMapPanel from "@/components/FleetLiveMapPanel";
import { PortalSurface } from "../_components/PortalSurface";

export default function FleetPage() {
  return (
    <PortalSurface
      title="Fleet & org"
      description="Live sealed-route execution and fleet onboarding."
    >
      <div className="md-card p-0 overflow-hidden flex flex-col min-h-[420px] mb-4">
        <div
          className="p-4 border-b flex justify-between items-center gap-3"
          style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
        >
          <div>
            <h2 className="md-typescale-title-medium">Live fleet map</h2>
            <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
              Sealed manifest polylines with live driver positions.
            </p>
          </div>
          <Link href="/manifests" className="md-btn md-btn-text text-sm h-8 px-2 shrink-0">
            Manifests
          </Link>
        </div>
        <FleetLiveMapPanel className="flex-1 min-h-[360px]" />
      </div>
      <div className="md-card p-6 space-y-4">
        <p className="md-typescale-body-medium">
          Fleet onboarding runs on the dedicated org-fleet surface with topology validation and idempotent create support.
        </p>
        <Link href="/org-fleet" className="md-btn md-btn-filled inline-flex">
          Open org & fleet onboarding
        </Link>
      </div>
    </PortalSurface>
  );
}
