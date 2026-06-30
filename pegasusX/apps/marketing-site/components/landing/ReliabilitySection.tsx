"use client";

import { useRef } from "react";
import { useGSAP } from "@gsap/react";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import {
  reliabilityBullets,
  reliabilitySpecs,
  trustStrip,
} from "@/content/pegasus";
import { SectionShell } from "@/components/layout/SectionShell";
import { SpecTable, SectionHeader, BulletList } from "@/components/docs/SpecTable";
import { TechIconGrid } from "@/components/icons/TechIconGrid";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";

gsap.registerPlugin(ScrollTrigger, useGSAP);

export function ReliabilitySection() {
  const sectionRef = useRef<HTMLElement>(null);
  const reducedMotion = useReducedMotion();

  useGSAP(
    () => {
      if (reducedMotion || !sectionRef.current) return;
      ScrollTrigger.batch("[data-reliability-card]", {
        start: "top 85%",
        onEnter: (elements) => {
          gsap.fromTo(
            elements,
            { opacity: 0, y: 20 },
            { opacity: 1, y: 0, stagger: 0.06, duration: 0.4, ease: "power3.out" },
          );
        },
      });
    },
    { scope: sectionRef, dependencies: [reducedMotion] },
  );

  return (
    <SectionShell id="reliability" minHeight="min-h-[80vh]" className="py-24">
      <section ref={sectionRef} className="mx-auto max-w-7xl px-4 md:px-6">
        <SectionHeader
          platformFrame
          label="Trust"
          title="Built for operations that can't afford mistakes."
          description="Accurate order status, safe payments, honest tracking, and human override when your team needs to step in."
          titleId="reliability-title"
        />

        <div className="mt-12 grid gap-8 lg:grid-cols-2">
          <div data-reliability-card>
            <SpecTable rows={reliabilitySpecs} />
          </div>
          <div data-reliability-card>
            <BulletList items={reliabilityBullets} />
            <div className="mt-8">
              <TechIconGrid icons={["spanner", "redis", "kafka", "go"]} columns={4} />
            </div>
          </div>
        </div>

        <div className="mt-12 flex flex-wrap gap-3 border-t border-[var(--mkt-border)] pt-8">
          {trustStrip.map((item) => (
            <span key={item} className="role-badge font-mono text-[10px] uppercase tracking-wider">
              {item}
            </span>
          ))}
        </div>
      </section>
    </SectionShell>
  );
}
