"use client";

import { useRef, useState } from "react";
import Link from "next/link";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { rolesParadeContent } from "@/content/roles";
import { PinSection } from "@/components/motion/PinSection";
import { AssetSlot } from "@/components/media/AssetSlot";
import { SectionHeader, SpecTable, BulletList } from "@/components/docs/SpecTable";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";
import { ASSET_SLOTS } from "@/lib/constants";

export function RolesParadeSection() {
  const cardRef = useRef<HTMLDivElement>(null);
  const reducedMotion = useReducedMotion();
  const [progress, setProgress] = useState(0);
  const [overrideIndex, setOverrideIndex] = useState<number | null>(null);

  const scrollIndex = Math.min(
    rolesParadeContent.length - 1,
    Math.floor(progress * rolesParadeContent.length),
  );
  const activeIndex = overrideIndex ?? scrollIndex;
  const activeRole = rolesParadeContent[activeIndex];

  const goTo = (index: number) => {
    const next = (index + rolesParadeContent.length) % rolesParadeContent.length;
    setOverrideIndex(next);
  };

  const roleSpecs = activeRole
    ? [
        { key: "Surfaces", value: activeRole.surfaces.join(" · ") },
        { key: "Deep dive", value: `/roles/${activeRole.slug}` },
      ]
    : [];

  const carouselNav = (
    <div className="flex items-center justify-between gap-4">
      <p className="font-mono text-xs uppercase tracking-wider text-[var(--mkt-subtle)]">
        {activeRole?.name ?? "Role"} [{activeIndex + 1}/{rolesParadeContent.length}]
      </p>
      <div className="flex items-center gap-2">
        <button
          type="button"
          className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--mkt-border)] transition hover:border-[var(--mkt-border-strong)]"
          aria-label="Previous role"
          onClick={() => goTo(activeIndex - 1)}
        >
          <ChevronLeft size={18} />
        </button>
        <button
          type="button"
          className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--mkt-border)] transition hover:border-[var(--mkt-border-strong)]"
          aria-label="Next role"
          onClick={() => goTo(activeIndex + 1)}
        >
          <ChevronRight size={18} />
        </button>
      </div>
    </div>
  );

  const content = (
    <div className="relative min-h-screen">
      <div className="mx-auto flex min-h-screen max-w-7xl flex-col justify-center gap-10 px-4 py-8 md:px-6">
        <SectionHeader
          platformFrame
          label="Six roles"
          title="One platform for every team in your network."
          description="Suppliers, warehouses, factories, drivers, retailers, and gate teams — each with tools built for their job."
          titleId="six-roles-title"
        />

        {carouselNav}

        <div className="grid gap-8 lg:grid-cols-2">
          <AssetSlot slotId="object-c-roles" assetPath={ASSET_SLOTS.roles} minHeight="min-h-[200px]">
            <div className="grid grid-cols-3 gap-2 p-4 md:grid-cols-6">
              {rolesParadeContent.map((role, i) => (
                <button
                  key={role.slug}
                  type="button"
                  onClick={() => setOverrideIndex(i)}
                  className={`rounded-lg border px-2 py-3 text-center transition-colors ${
                    i === activeIndex
                      ? "border-[var(--mkt-text)] bg-[var(--mkt-elevated)]"
                      : "border-[var(--mkt-border)] opacity-40 hover:opacity-70"
                  }`}
                >
                  <p className="font-mono text-[10px] uppercase tracking-wider">{role.name}</p>
                </button>
              ))}
            </div>
          </AssetSlot>

          {activeRole ? (
            <div ref={cardRef} className="mkt-card p-6">
              <div className="flex items-center gap-3">
                <span className="role-badge__dot" />
                <h3 className="text-xl font-semibold">{activeRole.name}</h3>
              </div>
              <p className="mt-2 text-[var(--mkt-muted)]">{activeRole.tagline}</p>
              <div className="mt-4">
                <BulletList items={activeRole.bullets} />
              </div>
              <div className="mt-6">
                <SpecTable rows={roleSpecs} />
              </div>
              <Link
                href={`/roles/${activeRole.slug}`}
                className="mt-6 inline-flex text-sm font-medium text-[var(--mkt-text)] hover:underline"
              >
                Deep dive → {activeRole.name}
              </Link>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );

  if (reducedMotion) {
    return (
      <section id="six-roles" aria-labelledby="six-roles-title" className="py-24">
        <div className="mx-auto max-w-7xl px-4 md:px-6">
          <SectionHeader
            platformFrame
            label="Six roles"
            title="One platform for every execution persona."
            titleId="six-roles-title"
          />
          <div className="mt-12 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {rolesParadeContent.map((role) => (
              <div key={role.slug} className="mkt-card p-6">
                <h3 className="font-semibold">{role.name}</h3>
                <p className="mt-2 text-sm text-[var(--mkt-muted)]">{role.tagline}</p>
                <Link href={`/roles/${role.slug}`} className="mt-4 inline-block text-sm underline">
                  Learn more →
                </Link>
              </div>
            ))}
          </div>
        </div>
      </section>
    );
  }

  return (
    <PinSection
      id="six-roles"
      end="+=300%"
      onProgress={(p) => {
        setProgress(p);
        setOverrideIndex(null);
      }}
    >
      {content}
    </PinSection>
  );
}
