"use client";

import { useState } from "react";
import { controlPlaneLayers } from "@/content/landing";
import { PinSection } from "@/components/motion/PinSection";
import { AssetSlot } from "@/components/media/AssetSlot";
import { TechIconLadder } from "@/components/icons/TechIconGrid";
import { SpecTable, SectionHeader } from "@/components/docs/SpecTable";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";
import { ASSET_SLOTS } from "@/lib/constants";
import type { TechIconId } from "@/components/icons/TechIcons";

const LAYER_ICONS: TechIconId[] = controlPlaneLayers.map((l) => l.icon);

export function ControlPlaneSection() {
  const reducedMotion = useReducedMotion();
  const [progress, setProgress] = useState(0);

  const activeIndex = Math.min(
    controlPlaneLayers.length - 1,
    Math.floor(progress * controlPlaneLayers.length),
  );
  const activeLayer = controlPlaneLayers[activeIndex];

  const content = (
    <div className="relative min-h-screen">
      <div className="mx-auto grid min-h-screen max-w-7xl grid-cols-1 items-center gap-10 px-4 py-8 md:grid-cols-2 md:px-6">
        <div>
          <SectionHeader
            platformFrame
            label="Platform"
            title="One platform. Every team in sync."
            description="Dedicated apps for suppliers, warehouses, factories, drivers, retailers, and gate teams — all reading from the same live picture."
            titleId="control-plane-title"
          />
          <div className="mt-8">
            <SpecTable rows={activeLayer?.specs ?? []} />
          </div>
        </div>

        <div className="space-y-6">
          <AssetSlot slotId="object-b-layers" assetPath={ASSET_SLOTS.controlPlane}>
            <TechIconLadder icons={LAYER_ICONS} activeIndex={activeIndex} />
          </AssetSlot>

          {activeLayer ? (
            <div className="mkt-card p-6">
              <p className="font-mono text-xs uppercase tracking-wider text-[var(--mkt-subtle)]">
                Layer {activeIndex + 1} / {controlPlaneLayers.length}
              </p>
              <h3 className="mt-2 text-xl font-semibold">{activeLayer.title}</h3>
              <p className="mt-3 text-sm text-[var(--mkt-muted)]">{activeLayer.body}</p>
              {activeLayer.specs ? (
                <div className="mt-4">
                  <SpecTable rows={activeLayer.specs} />
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );

  if (reducedMotion) {
    return (
      <section id="control-plane" aria-labelledby="control-plane-title" className="py-24">
        <div className="mx-auto max-w-7xl px-4 md:px-6">
          {content}
          <div className="mt-12 space-y-6">
            {controlPlaneLayers.map((layer) => (
              <div key={layer.id} className="mkt-card p-6">
                <h3 className="font-semibold">{layer.title}</h3>
                <p className="mt-2 text-sm text-[var(--mkt-muted)]">{layer.body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>
    );
  }

  return (
    <PinSection
      id="control-plane"
      end="+=200%"
      onProgress={setProgress}
      className="bg-[var(--mkt-bg)]"
    >
      {content}
    </PinSection>
  );
}
