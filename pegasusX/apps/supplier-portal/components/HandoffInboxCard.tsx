"use client";

import type { HandoffCardMetadata } from "@pegasusx/types";
import Link from "next/link";

export function HandoffInboxCard({ handoff }: { handoff: HandoffCardMetadata }) {
  return (
    <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-3 mt-2">
      <div className="text-xs uppercase tracking-wide opacity-60">Handoff</div>
      <div className="font-semibold mt-1">{handoff.title}</div>
      {handoff.subtitle ? <div className="text-sm opacity-80 mt-1">{handoff.subtitle}</div> : null}
      {handoff.primary_link ? (
        <Link href={handoff.primary_link} className="portal-btn portal-btn--primary text-xs mt-2 inline-flex">
          {handoff.primary_cta || "Open"}
        </Link>
      ) : null}
    </div>
  );
}
