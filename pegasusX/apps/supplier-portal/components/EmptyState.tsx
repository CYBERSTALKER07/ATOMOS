"use client";

import type { ReactNode } from "react";
import Icon from "./Icon";

type EmptyStateProps = {
  icon?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
  children?: ReactNode;
};

export default function EmptyState({
  icon = "inventory",
  headline,
  body,
  action,
  onAction,
  children,
}: EmptyStateProps) {
  return (
    <div className="md-empty-state desk-card" style={{ minHeight: 240 }}>
      <div
        className="flex items-center justify-center w-14 h-14 rounded-full mb-2"
        style={{ background: "var(--desk-surface-subtle)", color: "var(--desk-text-tertiary)" }}
      >
        <Icon name={icon} size={28} />
      </div>
      <h3 className="md-typescale-title-large" style={{ color: "var(--desk-text-primary)", margin: 0 }}>
        {headline}
      </h3>
      {body ? (
        <p className="md-typescale-body-medium max-w-md" style={{ color: "var(--desk-text-secondary)" }}>
          {body}
        </p>
      ) : null}
      {children}
      {action && onAction ? (
        <button type="button" className="desk-btn-primary mt-2" onClick={onAction}>
          {action}
        </button>
      ) : null}
    </div>
  );
}
