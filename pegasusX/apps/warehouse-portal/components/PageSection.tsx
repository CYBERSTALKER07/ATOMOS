"use client";

import type { ReactNode } from "react";

type PageSectionProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  bay?: "ops" | "inventory" | "fleet" | "finance";
};

const BAY_CLASS: Record<NonNullable<PageSectionProps["bay"]>, string> = {
  ops: "wh-bay--ops",
  inventory: "wh-bay--inventory",
  fleet: "wh-bay--fleet",
  finance: "wh-bay--finance",
};

export function PageSection({
  title,
  description,
  actions,
  children,
  className = "",
  bay = "ops",
}: PageSectionProps) {
  return (
    <section className={`wh-bay-panel mt-6 ${BAY_CLASS[bay]} ${className}`.trim()}>
      <div className="wh-section-head">
        <div className="min-w-0">
          <h2 className="wh-section-title">{title}</h2>
          {description ? <p className="wh-section-desc">{description}</p> : null}
        </div>
        {actions ? <div className="desk-toolbar shrink-0">{actions}</div> : null}
      </div>
      <div className="p-5 space-y-4">{children}</div>
    </section>
  );
}
