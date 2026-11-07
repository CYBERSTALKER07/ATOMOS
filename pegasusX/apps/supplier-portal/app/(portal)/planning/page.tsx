"use client";

import { Suspense, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { planBrainTabFromQuery, type PlanBrainTab } from "@pegasusx/types";
import { PlanBrainTabs } from "@pegasusx/ui-kit/portal";
import { PageChrome } from "@/components/PageChrome";
import PlanningBrainPanel from "@/components/PlanningBrainPanel";
import DigitalBrainPanel from "@/components/DigitalBrainPanel";
import FactoryPlanningOpsPanel from "@/components/settings/planning/FactoryPlanningOpsPanel";
import MeioNetworkPanel from "@/components/MeioNetworkPanel";

const HORIZONS = [7, 14, 28] as const;

export default function PlanningPage() {
  return (
    <Suspense fallback={null}>
      <PlanningPageInner />
    </Suspense>
  );
}

function PlanningPageInner() {
  const router = useRouter();
  const params = useSearchParams();
  const tab = planBrainTabFromQuery(params.get("tab"));
  const sku = params.get("sku") ?? "";
  const retailer = params.get("retailer") ?? "";
  const horizon = Number(params.get("horizon") || 7);

  const replaceQuery = useCallback(
    (next: { tab?: PlanBrainTab; sku?: string; retailer?: string; horizon?: number }) => {
      const q = new URLSearchParams(params.toString());
      if (next.tab) q.set("tab", next.tab);
      if (next.sku != null) {
        if (next.sku) q.set("sku", next.sku);
        else q.delete("sku");
      }
      if (next.retailer != null) {
        if (next.retailer) q.set("retailer", next.retailer);
        else q.delete("retailer");
      }
      if (next.horizon != null) q.set("horizon", String(next.horizon));
      router.replace(`/planning?${q.toString()}`);
    },
    [params, router],
  );

  return (
    <PageChrome
      icon="overview"
      title="Plan & Brain"
      description="Planning what-ifs and Digital Brain belief on one route. Place stays off."
    >
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <PlanBrainTabs value={tab} onChange={(next) => replaceQuery({ tab: next })} />
          <div className="flex flex-wrap items-center gap-2">
            {HORIZONS.map((days) => (
              <button
                key={days}
                type="button"
                className="portal-btn portal-btn--ghost text-sm"
                aria-pressed={horizon === days}
                onClick={() => replaceQuery({ horizon: days })}
              >
                {days}d
              </button>
            ))}
            <input
              className="portal-input"
              defaultValue={sku}
              placeholder="SKU"
              aria-label="SKU"
              onBlur={(e) => replaceQuery({ sku: e.target.value.trim() })}
            />
            <Link href="/settings/planning" className="portal-btn portal-btn--ghost text-sm">
              Planning flags
            </Link>
          </div>
        </div>

        {tab === "brain" ? (
          <DigitalBrainPanel sku={sku} retailerId={retailer} />
        ) : (
          <div className="flex flex-col gap-6">
            <PlanningBrainPanel />
            <FactoryPlanningOpsPanel />
            <MeioNetworkPanel />
          </div>
        )}
      </div>
    </PageChrome>
  );
}
