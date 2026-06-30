"use client";

import { useRef } from "react";
import { useGSAP } from "@gsap/react";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { telemetrySpecs } from "@/content/landing";
import { SectionShell } from "@/components/layout/SectionShell";
import { DemoFleetMap } from "@/components/maps/DemoFleetMap";
import { SpecTable, SectionHeader } from "@/components/docs/SpecTable";
import { TechIconGrid } from "@/components/icons/TechIconGrid";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";

gsap.registerPlugin(ScrollTrigger, useGSAP);

export function LiveTelemetrySection() {
  const sectionRef = useRef<HTMLElement>(null);
  const pathRef = useRef<SVGPathElement>(null);
  const reducedMotion = useReducedMotion();

  useGSAP(
    () => {
      const path = pathRef.current;
      if (!path || reducedMotion) return;
      const length = path.getTotalLength();
      gsap.set(path, { strokeDasharray: length, strokeDashoffset: length });
      gsap.to(path, {
        strokeDashoffset: 0,
        ease: "none",
        scrollTrigger: {
          trigger: sectionRef.current,
          start: "top center",
          end: "bottom center",
          scrub: true,
        },
      });
    },
    { scope: sectionRef, dependencies: [reducedMotion] },
  );

  return (
    <SectionShell id="live-telemetry" minHeight="min-h-screen" className="py-24">
      <section ref={sectionRef} className="mx-auto max-w-7xl px-4 md:px-6">
        <div className="grid gap-10 lg:grid-cols-2">
          <div>
            <SectionHeader
              platformFrame
              label="Live telemetry"
              title="Planned vs actual. Deviation signals. Fleet truth."
              description="Route geometry compared in real time. Fleet maps show live driver markers with truthful stale vs live location semantics."
              titleId="live-telemetry-title"
            />
            <div className="mt-8">
              <SpecTable rows={telemetrySpecs} />
            </div>
            <div className="mt-8">
              <TechIconGrid icons={["map", "websocket", "go"]} columns={3} />
            </div>
            <svg viewBox="0 0 400 120" className="mt-8 w-full max-w-md" aria-hidden>
              <path
                ref={pathRef}
                d="M20,80 C80,20 140,100 200,50 S320,30 380,60"
                fill="none"
                stroke="var(--mkt-text)"
                strokeWidth="2"
              />
              <path
                d="M20,90 C90,40 150,110 210,70 S330,50 380,80"
                fill="none"
                stroke="var(--mkt-muted)"
                strokeWidth="1.5"
                strokeDasharray="6 4"
                opacity="0.5"
              />
            </svg>
          </div>
          <div className="h-[420px] overflow-hidden rounded-lg border border-[var(--mkt-border)]">
            <DemoFleetMap className="h-full w-full" />
          </div>
        </div>
      </section>
    </SectionShell>
  );
}
