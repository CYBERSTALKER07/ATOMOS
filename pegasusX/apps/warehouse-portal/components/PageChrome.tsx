"use client";

import type { ReactNode } from "react";

type PageChromeProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  emptyMessage?: string;
  children: ReactNode;
};

export function PageChrome({
  title,
  description,
  actions,
  loading,
  error,
  empty,
  emptyMessage = "No data yet.",
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
        <div className="md-card p-8" style={{ color: "var(--desk-text-secondary)" }}>
          Loading…
        </div>
      ) : error ? (
        <div
          className="md-card p-6"
          style={{ color: "var(--desk-danger)", borderColor: "var(--desk-danger)" }}
        >
          {error}
        </div>
      ) : empty ? (
        <div className="md-card p-6" style={{ color: "var(--desk-text-secondary)" }}>
          {emptyMessage}
        </div>
      ) : (
        children
      )}
    </div>
  );
}
