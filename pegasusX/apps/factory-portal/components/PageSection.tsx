"use client";

import type { ReactNode } from "react";

type PageSectionProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
};

export function PageSection({ title, description, actions, children, className = "" }: PageSectionProps) {
  return (
    <section className={`desk-card overflow-hidden ${className}`.trim()}>
      <div
        className="bento-card-header flex flex-wrap items-start justify-between gap-3 px-5 py-4"
        style={{ borderBottom: "1px solid var(--desk-border)", background: "var(--desk-surface-raised)" }}
      >
        <div className="min-w-0">
          <h2 className="bento-card-title">{title}</h2>
          {description ? (
            <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
              {description}
            </p>
          ) : null}
        </div>
        {actions ? <div className="desk-toolbar shrink-0">{actions}</div> : null}
      </div>
      <div className="p-5 space-y-4">{children}</div>
    </section>
  );
}
