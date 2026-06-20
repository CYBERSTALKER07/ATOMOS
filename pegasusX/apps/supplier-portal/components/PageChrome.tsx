"use client";

import type { ReactNode } from "react";
import Icon from "./Icon";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "./Skeleton";

type PageChromeProps = {
  title: string;
  description?: string;
  icon?: string;
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
  icon,
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
      <div className={`desk-page-header${icon ? " desk-page-header--with-icon" : ""}`}>
        {icon ? (
          <div className="desk-page-header-icon" aria-hidden>
            <Icon name={icon} size={22} />
          </div>
        ) : null}
        <div className="flex flex-1 flex-wrap items-start justify-between gap-4 min-w-0">
          <div className="min-w-0">
            <h1 className="desk-page-title">{title}</h1>
            {description ? <p className="desk-page-subtitle">{description}</p> : null}
          </div>
          {actions ? <div className="desk-toolbar shrink-0">{actions}</div> : null}
        </div>
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

/** @deprecated Use PageChrome */
export const PortalSurface = PageChrome;
