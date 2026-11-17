"use client";

import {
  formatOntologyMarkingLabel,
  securityMarkingTone,
  type SecurityMarkingTone,
} from "@/lib/ontology-api";

const TONE_CLASSES: Record<SecurityMarkingTone, string> = {
  finance: "border-amber-600/40 bg-amber-500/10 text-amber-700 dark:text-amber-300",
  pii: "border-sky-600/40 bg-sky-500/10 text-sky-700 dark:text-sky-300",
  quarantine: "border-rose-600/40 bg-rose-500/10 text-rose-700 dark:text-rose-300",
  neutral: "border-neutral-500/40 bg-neutral-500/10 text-neutral-700 dark:text-neutral-300",
};

export function OntologyMarkingBadge({ marking }: { marking: string }) {
  const tone = securityMarkingTone(marking);
  const label = formatOntologyMarkingLabel(marking);
  return (
    <span
      data-marking={marking}
      data-tone={tone}
      className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 font-mono text-[11px] font-semibold tracking-tight ${TONE_CLASSES[tone]}`}
    >
      {label}
    </span>
  );
}

export function OntologyMarkingList({
  markings,
  emptyLabel,
}: {
  markings?: string[];
  emptyLabel?: string;
}) {
  if (!markings || markings.length === 0) {
    return emptyLabel ? (
      <span className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
        {emptyLabel}
      </span>
    ) : null;
  }
  return (
    <div className="flex flex-wrap gap-1.5">
      {markings.map((marking) => (
        <OntologyMarkingBadge key={marking} marking={marking} />
      ))}
    </div>
  );
}
