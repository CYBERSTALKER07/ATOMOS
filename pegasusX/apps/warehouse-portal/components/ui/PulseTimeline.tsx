'use client';

import type { PulseEvent } from '@pegasusx/types';

export type PulseTimelineProps = {
  events: PulseEvent[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
  className?: string;
  onSelect?: (event: PulseEvent) => void;
};

function formatWhen(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

export function PulseTimeline({
  events,
  loading = false,
  error = null,
  emptyMessage = 'No recent activity yet.',
  className = '',
  onSelect,
}: PulseTimelineProps) {
  if (loading && events.length === 0) {
    return <div className={`pulse-timeline pulse-timeline--loading ${className}`.trim()}>Loading pulse…</div>;
  }
  if (error) {
    return <div className={`pulse-timeline pulse-timeline--error ${className}`.trim()} role="alert">{error}</div>;
  }
  if (events.length === 0) {
    return <div className={`pulse-timeline pulse-timeline--empty ${className}`.trim()}>{emptyMessage}</div>;
  }
  return (
    <ol className={`pulse-timeline ${className}`.trim()}>
      {events.map((event) => (
        <li key={event.id} className="pulse-timeline__item">
          <button
            type="button"
            className="pulse-timeline__row"
            onClick={() => onSelect?.(event)}
            disabled={!onSelect}
          >
            <span className="pulse-timeline__kind">{event.kind.replace(/_/g, ' ')}</span>
            <span className="pulse-timeline__title">{event.title}</span>
            {event.description ? <span className="pulse-timeline__desc">{event.description}</span> : null}
            <time className="pulse-timeline__time" dateTime={event.occurred_at}>{formatWhen(event.occurred_at)}</time>
          </button>
        </li>
      ))}
    </ol>
  );
}
