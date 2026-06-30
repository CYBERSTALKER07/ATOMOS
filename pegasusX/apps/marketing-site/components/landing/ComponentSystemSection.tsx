"use client";

import Link from "next/link";
import { useRef } from "react";
import { useGSAP } from "@gsap/react";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { SectionShell } from "@/components/layout/SectionShell";
import { SectionHeader } from "@/components/docs/SpecTable";
import { SOLUTIONS } from "@/lib/constants";
import { solutionContent } from "@/content/solutions";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";

gsap.registerPlugin(ScrollTrigger, useGSAP);

export function ComponentSystemSection() {
  const sectionRef = useRef<HTMLElement>(null);
  const reducedMotion = useReducedMotion();

  useGSAP(
    () => {
      if (reducedMotion || !sectionRef.current) return;
      gsap.fromTo(
        sectionRef.current.querySelectorAll("[data-bento-item]"),
        { opacity: 0, y: 16 },
        {
          opacity: 1,
          y: 0,
          stagger: 0.08,
          duration: 0.45,
          ease: "power3.out",
          scrollTrigger: { trigger: sectionRef.current, start: "top 75%" },
        },
      );
    },
    { scope: sectionRef, dependencies: [reducedMotion] },
  );

  return (
    <SectionShell id="component-system" minHeight="min-h-[80vh]" className="py-24">
      <section ref={sectionRef} className="mx-auto max-w-7xl px-4 md:px-6">
        <SectionHeader
          platformFrame
          label="Solutions"
          title="Start with the problem you need to solve."
          description="Dispatch accuracy, fleet visibility, payment confidence, or network coordination — pick the outcome that matters most to your team."
          titleId="component-system-title"
        />

        <div className="mt-12 grid gap-6 md:grid-cols-2">
          {SOLUTIONS.map((solution) => {
            const content = solutionContent[solution.slug];
            return (
              <Link
                key={solution.slug}
                href={`/solutions/${solution.slug}`}
                data-bento-item
                className="mkt-card block p-6 transition hover:border-[var(--mkt-border-strong)]"
              >
                <h3 className="text-lg font-semibold">{solution.title}</h3>
                <p className="mt-2 text-sm text-[var(--mkt-muted)]">{solution.summary}</p>
                <ul className="mt-4 space-y-1">
                  {content.outcomes.slice(0, 2).map((outcome) => (
                    <li key={outcome} className="text-xs text-[var(--mkt-subtle)]">
                      → {outcome}
                    </li>
                  ))}
                </ul>
              </Link>
            );
          })}
        </div>

        <Link href="/solutions" className="mkt-btn mkt-btn-outline mt-10 inline-flex">
          View all solutions
        </Link>
      </section>
    </SectionShell>
  );
}
