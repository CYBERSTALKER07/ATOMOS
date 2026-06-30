import Link from "next/link";
import type { Route } from "next";
import { ArrowRight, Compass, MessageSquare } from "lucide-react";
import { experiencePaths } from "@/content/landing/enterprise";
import { SectionShell } from "@/components/layout/SectionShell";
import { SectionHeader } from "@/components/docs/SpecTable";

const ICONS = {
  tour: Compass,
  demo: MessageSquare,
} as const;

export function ExperienceSection() {
  return (
    <SectionShell id="experience" className="py-20 md:py-24">
      <div className="mx-auto max-w-7xl px-4 md:px-6">
        <SectionHeader
          platformFrame
          label="Experience Pegasus"
          title="Two ways to get started."
          description="Explore the platform on your own, or talk to our team about your dispatch, tracking, and payment needs."
          titleId="experience-title"
        />

        <div className="mt-12 grid gap-6 md:grid-cols-2">
          {experiencePaths.map((path) => {
            const Icon = ICONS[path.id];
            return (
              <Link
                key={path.id}
                href={path.href as Route}
                className="mkt-card group flex flex-col p-8 transition hover:border-[var(--mkt-border-strong)]"
              >
                <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-[var(--mkt-border)]">
                  <Icon size={20} className="text-[var(--mkt-text)]" />
                </div>
                <h3 className="mt-6 text-xl font-semibold">{path.title}</h3>
                <p className="mt-3 flex-1 text-sm leading-relaxed text-[var(--mkt-muted)]">
                  {path.description}
                </p>
                <span className="mt-8 inline-flex items-center gap-2 text-sm font-semibold">
                  {path.cta}
                  <ArrowRight
                    size={16}
                    className="transition-transform group-hover:translate-x-1"
                  />
                </span>
              </Link>
            );
          })}
        </div>
      </div>
    </SectionShell>
  );
}
