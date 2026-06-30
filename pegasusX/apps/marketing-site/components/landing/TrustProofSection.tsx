import Link from "next/link";
import type { Route } from "next";
import { trustProof } from "@/content/landing/enterprise";
import { SectionShell } from "@/components/layout/SectionShell";
import { SectionHeader, SpecTable } from "@/components/docs/SpecTable";

export function TrustProofSection() {
  return (
    <SectionShell id="trust-proof" className="border-t border-[var(--mkt-border)] py-20 md:py-24">
      <div className="mx-auto max-w-7xl px-4 md:px-6">
        <div className="grid gap-10 lg:grid-cols-2 lg:items-center">
          <div>
            <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-[var(--mkt-subtle)]">
              [{trustProof.tag}]
            </p>
            <SectionHeader
              platformFrame
              label="Why teams trust Pegasus"
              title={trustProof.title}
              description={trustProof.summary}
              titleId="trust-proof-title"
            />
            <Link href={trustProof.cta.href as Route} className="mkt-btn mkt-btn-outline mt-8 inline-flex">
              {trustProof.cta.label} →
            </Link>
          </div>

          <div className="mkt-card p-6 md:p-8">
            <SpecTable rows={trustProof.highlights} />
            <p className="mt-6 text-xs leading-relaxed text-[var(--mkt-subtle)]">
              Built for operators who need every screen to agree — during peak dispatch,
              high-volume order days, and end-of-day reconciliation.
            </p>
          </div>
        </div>
      </div>
    </SectionShell>
  );
}
