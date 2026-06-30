"use client";

import { useRef } from "react";
import Link from "next/link";
import { useGSAP } from "@gsap/react";
import gsap from "gsap";
import { ChevronDown } from "lucide-react";
import { heroContent } from "@/content/landing";
import { SectionShell } from "@/components/layout/SectionShell";
import { TextMarquee } from "@/components/void/TextMarquee";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";

gsap.registerPlugin(useGSAP);

const HERO_MARQUEE = ["LOGISTICS", "DISPATCH", "DELIVERY", "NETWORK"];

export function HeroSection() {
  const sectionRef = useRef<HTMLElement>(null);
  const hintRef = useRef<HTMLDivElement>(null);
  const reducedMotion = useReducedMotion();

  useGSAP(
    () => {
      if (reducedMotion || !hintRef.current) return;
      gsap.to(hintRef.current, {
        y: 8,
        duration: 1.2,
        repeat: -1,
        yoyo: true,
        ease: "sine.inOut",
      });
    },
    { scope: sectionRef, dependencies: [reducedMotion] },
  );

  return (
    <SectionShell id="hero" minHeight="min-h-screen" className="relative overflow-hidden">
      <section ref={sectionRef} className="relative z-10 flex min-h-screen flex-col">
        <TextMarquee items={HERO_MARQUEE} separator="✦" className="py-6" speed="slow" />

        <div className="mx-auto flex flex-1 max-w-5xl flex-col items-center justify-center px-4 py-16 text-center md:px-6">
          <p className="void-tag mb-6">The logistics operating system</p>
          <h1 className="void-section-title void-blink-cursor max-w-4xl">
            {heroContent.headline.replace(/\.$/, "")}
          </h1>
          <p className="mt-8 max-w-2xl text-base leading-relaxed text-[var(--mkt-muted)] md:text-lg">
            {heroContent.subheadline}
          </p>
          <div className="mt-12 flex flex-wrap justify-center gap-4">
            <Link
              href={heroContent.primaryCta.href}
              className="mkt-btn mkt-btn-primary min-w-[160px] border-2 border-white font-light uppercase tracking-wide"
            >
              {heroContent.primaryCta.label}
            </Link>
            <Link
              href={heroContent.secondaryCta.href}
              className="mkt-btn mkt-btn-outline min-w-[160px] border-2 font-light uppercase tracking-wide"
            >
              {heroContent.secondaryCta.label}
            </Link>
          </div>
        </div>

        <div ref={hintRef} className="flex flex-col items-center gap-2 pb-10 text-xs font-semibold uppercase tracking-[0.2em] text-[var(--mkt-subtle)]">
          <span>Scroll</span>
          <ChevronDown size={16} />
        </div>

        <TextMarquee items={HERO_MARQUEE} separator="✦" className="border-t border-white/10 py-4" speed="fast" />
      </section>
    </SectionShell>
  );
}
