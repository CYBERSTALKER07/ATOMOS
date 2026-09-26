"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useState } from "react";
import ControlTowerCommandPanel from "@/components/ControlTowerCommandPanel";
import ExceptionWeatherMapPanel from "@/components/ExceptionWeatherMapPanel";
import Icon from "@/components/Icon";
import { PageChrome } from "@/components/PageChrome";

export default function FleetPage() {
  const t = usePortalT();
  const [tab, setTab] = useState<"fleet" | "exceptions">("fleet");

  return (
    <PageChrome
      icon="fleet"
      title={t("supplier_portal.fleet.text.fleet_and_org")}
      description={t("supplier_portal.residual.text.live_sealed_route_execution_exception_weather_and_fleet_onboardi")}
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
        <ControlTowerCommandPanel />
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
