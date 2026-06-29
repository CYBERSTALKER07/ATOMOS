"use client";

import Link from "next/link";
import { useState } from "react";
import FleetLiveMapPanel from "@/components/FleetLiveMapPanel";
import ExceptionWeatherMapPanel from "@/components/ExceptionWeatherMapPanel";
import Icon from "@/components/Icon";
import { PageChrome } from "@/components/PageChrome";

export default function FleetPage() {
  const [tab, setTab] = useState<"fleet" | "exceptions">("fleet");

  return (
    <PageChrome
      icon="fleet"
      title="Fleet & org"
      description="Live sealed-route execution, exception weather, and fleet onboarding."
    >
      <div className="flex gap-2 mb-4">
        <button
          type="button"
          className={`md-btn ${tab === "fleet" ? "md-btn-filled" : "md-btn-tonal"} text-sm`}
          onClick={() => setTab("fleet")}
        >
          Live fleet
        </button>
        <button
          type="button"
          className={`md-btn ${tab === "exceptions" ? "md-btn-filled" : "md-btn-tonal"} text-sm`}
          onClick={() => setTab("exceptions")}
        >
          Exception weather
        </button>
      </div>

      {tab === "fleet" ? (
      <div className="desk-card p-0 overflow-hidden flex flex-col min-h-[420px] mb-4">
        <div
          className="bento-card-header flex justify-between items-center gap-3 px-5 py-4"
          style={{ borderBottom: "1px solid var(--desk-border)", background: "var(--desk-surface-raised)" }}
        >
          <div>
            <h2 className="bento-card-title">Live fleet map</h2>
            <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
              Sealed manifest polylines with live driver positions.
            </p>
          </div>
          <Link href="/manifests" className="md-btn md-btn-text text-sm h-8 px-2 shrink-0 inline-flex items-center gap-1">
            <Icon name="manifests" size={16} />
            Manifests
          </Link>
        </div>
        <FleetLiveMapPanel className="flex-1 min-h-[360px]" />
      </div>
      ) : (
      <div className="desk-card p-0 overflow-hidden flex flex-col min-h-[420px] mb-4">
        <ExceptionWeatherMapPanel className="flex-1" />
      </div>
      )}
      <div className="desk-card p-6 space-y-4">
        <p className="md-typescale-body-medium" style={{ color: "var(--desk-text-secondary)" }}>
          Fleet onboarding runs on the dedicated org-fleet surface with topology validation and idempotent create support.
        </p>
        <Link href="/org-fleet" className="md-btn md-btn-filled inline-flex items-center gap-2">
          <Icon name="person-add" size={18} />
          Open org & fleet onboarding
        </Link>
      </div>
    </PageChrome>
  );
}
