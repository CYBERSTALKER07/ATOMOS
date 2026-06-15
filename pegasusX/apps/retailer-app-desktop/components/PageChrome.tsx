"use client";

import type { ReactNode } from "react";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "./Skeleton";

type EmptyVariant =
  | "no-data"
  | "no-results"
  | "offline"
  | "restricted"
  | "error"
  | "no-orders"
  | "no-products"
  | "no-predictions"
  | "no-suppliers";

type PageChromeProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  loading?: boolean;
  skeletonVariant?: "dashboard" | "table" | "form" | "catalog";
  error?: string | null;
  empty?: boolean;
  emptyMessage?: string;
  emptyVariant?: EmptyVariant;
  children: ReactNode;
  className?: string;
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
  emptyVariant = "no-data",
  children,
  className = "",
}: PageChromeProps) {
  return (
    <div className={`desk-page min-h-full ${className}`.trim()} style={{ background: "var(--desk-canvas)" }}>
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
        <EmptyState variant="error" headline="Unable to load" body={error} />
      ) : empty ? (
        <EmptyState variant={emptyVariant} headline={emptyMessage} />
      ) : (
        children
      )}
    </div>
  );
}
