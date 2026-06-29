"use client";

import type { PulseEvent } from "@pegasusx/types";

export type PulseTimelineProps = {
  events: PulseEvent[];
  loading?: boolean;
  emptyLabel?: string;
  onSelect?: (event: PulseEvent) => void;
  className?: string;
};

function formatWhen(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

export function PulseTimeline({
  events,
  loading,
  emptyLabel = "No recent activity yet.",
  onSelect,
  className = "",
}: PulseTimelineProps) {
  if (loading) {
    return <div className={`pulse-timeline pulse-timeline--loading ${className}`.trim()}>Loading pulse…</div>;
  }
  if (!events.length) {
    return <div className={`pulse-timeline pulse-timeline--empty ${className}`.trim()}>{emptyLabel}</div>;
  }
  return (
    <ol className={`pulse-timeline ${className}`.trim()} aria-label="Network pulse timeline">
      {events.map((event) => (
        <li key={event.id} className="pulse-timeline__item">
          <button
            type="button"
            className="pulse-timeline__button"
            onClick={() => onSelect?.(event)}
            disabled={!onSelect}
          >
            <div className="pulse-timeline__meta">
              <span className="pulse-timeline__kind">{event.kind.replaceAll("_", " ")}</span>
              <time className="pulse-timeline__time" dateTime={event.occurred_at}>
                {formatWhen(event.occurred_at)}
              </time>
            </div>
            <div className="pulse-timeline__title">{event.title}</div>
            {event.description ? <div className="pulse-timeline__desc">{event.description}</div> : null}
          </button>
        </li>
      ))}
      <style jsx>{`
        .pulse-timeline {
          list-style: none;
          margin: 0;
          padding: 0;
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
        }
        .pulse-timeline__item {
          margin: 0;
        }
        .pulse-timeline__button {
          width: 100%;
          text-align: left;
          border: 1px solid var(--border, rgba(0, 0, 0, 0.08));
          border-radius: 0.75rem;
          padding: 0.75rem 0.9rem;
          background: var(--surface, #fff);
          cursor: pointer;
        }
        .pulse-timeline__button:disabled {
          cursor: default;
        }
        .pulse-timeline__meta {
          display: flex;
          justify-content: space-between;
          gap: 0.5rem;
          font-size: 0.72rem;
          text-transform: uppercase;
          letter-spacing: 0.04em;
          opacity: 0.7;
        }
        .pulse-timeline__title {
          margin-top: 0.35rem;
          font-weight: 600;
        }
        .pulse-timeline__desc {
          margin-top: 0.25rem;
          font-size: 0.85rem;
          opacity: 0.85;
        }
        .pulse-timeline--loading,
        .pulse-timeline--empty {
          padding: 1rem;
          opacity: 0.75;
        }
      `}</style>
    </ol>
  );
}
