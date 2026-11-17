"use client";

import { useState } from "react";
import { ChevronDown, ChevronUp, Clock, RefreshCw, WifiOff } from "lucide-react";

export type DesktopQueuedAction = {
  id: string;
  label: string;
  subtitle?: string;
};

export type DesktopOfflineTrayProps = {
  isOffline: boolean;
  queuedActions?: DesktopQueuedAction[];
  onRetryAll?: () => void | Promise<void>;
  retrying?: boolean;
  offlineMessage?: string;
  className?: string;
};

/**
 * Fixed offline banner + queued-actions tray for Tauri desktop shells.
 * Renders nothing when online and the queue is empty.
 */
export function DesktopOfflineTray({
  isOffline,
  queuedActions = [],
  onRetryAll,
  retrying = false,
  offlineMessage = "You are offline. Cached data is shown where available.",
  className = "",
}: DesktopOfflineTrayProps) {
  const [expanded, setExpanded] = useState(false);
  const queueCount = queuedActions.length;

  if (!isOffline && queueCount === 0) {
    return null;
  }

  const showQueue = queueCount > 0;
  const headline = isOffline
    ? offlineMessage
    : `${queueCount} queued action${queueCount === 1 ? "" : "s"} pending sync`;

  return (
    <div
      className={`desktop-offline-tray ${className}`.trim()}
      style={{
        position: "fixed",
        left: "50%",
        bottom: "1rem",
        transform: "translateX(-50%)",
        zIndex: 60,
        width: "min(36rem, calc(100vw - 2rem))",
        borderRadius: "12px",
        border: "1px solid var(--desk-border, rgba(255,255,255,0.12))",
        background: "var(--desk-surface-raised, rgba(24,24,27,0.96))",
        color: "var(--desk-text-primary, #f4f4f5)",
        boxShadow: "0 12px 40px rgba(0,0,0,0.35)",
        backdropFilter: "blur(12px)",
      }}
      role="status"
      aria-live="polite"
    >
      <div
        className="flex items-start gap-3 px-4 py-3"
        style={{ borderBottom: showQueue && expanded ? "1px solid var(--desk-border, rgba(255,255,255,0.08))" : undefined }}
      >
        <div
          className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full"
          style={{
            background: isOffline
              ? "color-mix(in srgb, var(--desk-danger, #ef4444) 18%, transparent)"
              : "color-mix(in srgb, var(--desk-warning, #f59e0b) 18%, transparent)",
            color: isOffline ? "var(--desk-danger, #ef4444)" : "var(--desk-warning, #f59e0b)",
          }}
        >
          {isOffline ? <WifiOff size={16} aria-hidden /> : <Clock size={16} aria-hidden />}
        </div>

        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium leading-snug">{headline}</p>
          {showQueue && !expanded ? (
            <p className="mt-0.5 text-xs" style={{ color: "var(--desk-text-secondary, #a1a1aa)" }}>
              Tap to review queued items
            </p>
          ) : null}
        </div>

        <div className="flex shrink-0 items-center gap-1">
          {showQueue && onRetryAll ? (
            <button
              type="button"
              onClick={() => void onRetryAll()}
              disabled={retrying || isOffline}
              className="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs font-medium transition-opacity disabled:opacity-50"
              style={{
                background: "var(--desk-accent, #3b82f6)",
                color: "var(--desk-accent-contrast, #fff)",
              }}
              title={isOffline ? "Reconnect to retry queued actions" : "Retry queued actions now"}
            >
              <RefreshCw size={14} className={retrying ? "animate-spin" : ""} aria-hidden />
              Retry
            </button>
          ) : null}
          {showQueue ? (
            <button
              type="button"
              onClick={() => setExpanded((open) => !open)}
              className="inline-flex h-8 w-8 items-center justify-center rounded-lg"
              style={{ color: "var(--desk-text-secondary, #a1a1aa)" }}
              aria-expanded={expanded}
              aria-label={expanded ? "Collapse queued actions" : "Expand queued actions"}
            >
              {expanded ? <ChevronDown size={16} /> : <ChevronUp size={16} />}
            </button>
          ) : null}
        </div>
      </div>

      {showQueue && expanded ? (
        <ul className="max-h-48 overflow-y-auto px-2 py-2">
          {queuedActions.map((action) => (
            <li
              key={action.id}
              className="rounded-lg px-3 py-2 text-sm"
              style={{ color: "var(--desk-text-primary, #f4f4f5)" }}
            >
              <div className="font-medium">{action.label}</div>
              {action.subtitle ? (
                <div className="mt-0.5 text-xs" style={{ color: "var(--desk-text-secondary, #a1a1aa)" }}>
                  {action.subtitle}
                </div>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
