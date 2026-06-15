"use client";

import type { ReactNode } from "react";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "./Skeleton";

type PageChromeProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  loading?: boolean;
  skeletonVariant?: "dashboard" | "table" | "form";
  error?: string | null;
  empty?: boolean;
  emptyMessage?: string;
  emptyIcon?: string;
  children: ReactNode;
};

export function PageChrome({
  title,
  description,
  actions,
  loading,
  skeletonVariant = "dashboard",
  error,
  empty,
  emptyMessage = "No data yet.",
  emptyIcon = "inventory",
  children,
}: PageChromeProps) {
  return (
    <div className="desk-page">
      <div className="desk-page-header">
        <div>
          <h1 className="desk-page-title">{title}</h1>
          {description ? <p className="desk-page-subtitle">{description}</p> : null}
        </div>
        {actions ? <div className="desk-toolbar">{actions}</div> : null}
      </div>

      {loading ? (
        <PageSkeleton variant={skeletonVariant} />
      ) : error ? (
        <EmptyState icon="error" headline="Unable to load" body={error} />
      ) : empty ? (
        <EmptyState icon={emptyIcon} headline={emptyMessage} />
      ) : (
        children
      )}
    </div>
  );
}

/** @deprecated Use PageChrome — thin alias for gradual migration */
export function PortalSurface(props: PageChromeProps) {
  return <PageChrome {...props} />;
}
