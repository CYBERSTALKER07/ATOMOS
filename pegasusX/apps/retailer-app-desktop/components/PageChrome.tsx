"use client";

import type { ReactNode } from "react";
import Icon from "./Icon";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "./Skeleton";
import { PageChrome as KitPageChrome } from "@pegasusx/ui-kit/portal";

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
  icon?: string;
  actions?: ReactNode;
  loading?: boolean;
  skeletonVariant?: "dashboard" | "table" | "form" | "catalog";
  error?: string | null;
  empty?: boolean;
  emptyMessage?: string;
  emptyVariant?: EmptyVariant;
  children?: ReactNode;
  className?: string;
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
  emptyVariant = "no-data",
  children,
  className = "",
}: PageChromeProps) {
  return (
    <div className={className || undefined}>
    <KitPageChrome
      title={title}
      description={description}
      icon={icon ? <Icon name={icon} size={22} /> : undefined}
      actions={actions}
      loading={loading}
      error={error}
      empty={empty}
      renderLoading={() => <PageSkeleton variant={skeletonVariant} />}
      renderError={(message) => <EmptyState variant="error" headline="Unable to load" body={message} />}
      renderEmpty={() => <EmptyState variant={emptyVariant} headline={emptyMessage} />}
    >
      {children}
    </KitPageChrome>
    </div>
  );
}
