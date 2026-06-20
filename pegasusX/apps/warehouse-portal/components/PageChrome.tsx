"use client";

import type { ReactNode } from "react";
import Icon from "./Icon";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "./Skeleton";
import { PageChrome as KitPageChrome } from "@pegasusx/ui-kit/portal";

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
      renderEmpty={() => <EmptyState variant="no-data" headline={emptyMessage} />}
    >
      {children}
    </KitPageChrome>
  );
}
