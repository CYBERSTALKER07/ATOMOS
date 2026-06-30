"use client";

import Link from "next/link";
import { useRef } from "react";
import { useGSAP } from "@gsap/react";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import {
  ecosystemBullets,
  ecosystemComparison,
  pegasusXFootnote,
} from "@/content/pegasus";
import { SectionShell } from "@/components/layout/SectionShell";
import { SpecTable, SectionHeader, BulletList } from "@/components/docs/SpecTable";
import { TechIconGrid } from "@/components/icons/TechIconGrid";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";

gsap.registerPlugin(ScrollTrigger, useGSAP);

export function EcosystemSection() {
  const sectionRef = useRef<HTMLElement>(null);
  const reducedMotion = useReducedMotion();

  useGSAP(
    () => {
      if (reducedMotion || !sectionRef.current) return;
      const cards = sectionRef.current.querySelectorAll("[data-eco-card]");
      gsap.fromTo(
        cards,
        { opacity: 0, y: 24 },
        {
          opacity: 1,
          y: 0,
          stagger: 0.08,
          duration: 0.5,
          ease: "power3.out",
          scrollTrigger: {
            trigger: sectionRef.current,
            start: "top 70%",
          },
        },
      );
    },
    { scope: sectionRef, dependencies: [reducedMotion] },
  );

  return (
    <SectionShell id="ecosystem" minHeight="min-h-screen" className="py-24">
      <section ref={sectionRef} className="mx-auto max-w-7xl px-4 md:px-6">
        <div className="grid gap-12 lg:grid-cols-2">
          <div>
            <SectionHeader
              platformFrame
              label="Your network"
              title="Start with one site. Grow to a full network."
              description="Run a single supplier operation today, or coordinate multiple suppliers across regions — same apps, same workflows, built to scale."
              titleId="ecosystem-title"
            />
            <div className="mt-8">
              <BulletList items={ecosystemBullets} />
            </div>
            <Link href="/platform" className="mkt-btn mkt-btn-outline mt-8 inline-flex">
              See how it works →
            </Link>
          </div>

          <div className="space-y-6">
            <div data-eco-card>
              <SpecTable rows={ecosystemComparison} />
            </div>
            <div data-eco-card>
              <TechIconGrid
                icons={["kubernetes", "go", "spanner", "redis", "kafka", "websocket"]}
                columns={3}
              />
            </div>
            <div data-eco-card className="mkt-card p-6">
              <p className="font-mono text-xs uppercase tracking-wider text-[var(--mkt-subtle)]">
                Growing with you
              </p>
              <p className="mt-3 text-sm text-[var(--mkt-muted)]">
                {pegasusXFootnote}
              </p>
            </div>
          </div>
        </div>
      </section>
    </SectionShell>
  );
}
