"use client";

import { MOCK_PULSE_EVENTS, type MockPulseEvent } from "@/lib/mock-data/pulse-events";

type PulseEvent = MockPulseEvent;

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"] as const;

/** Deterministic UTC formatting — identical output in Node SSR and the browser. */
function formatWhen(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;

  const month = MONTHS[date.getUTCMonth()];
  const day = date.getUTCDate();
  const hours24 = date.getUTCHours();
  const minutes = String(date.getUTCMinutes()).padStart(2, "0");
  const ampm = hours24 >= 12 ? "PM" : "AM";
  const hours12 = hours24 % 12 || 12;

  return `${month} ${day}, ${hours12}:${minutes} ${ampm} UTC`;
}

function MarketingPulseTimeline({ events }: { events: PulseEvent[] }) {
  return (
    <ol className="pulse-timeline">
      {events.map((event) => (
        <li key={event.id} className="pulse-timeline__item">
          <div className="pulse-timeline__row">
            <span className="pulse-timeline__kind">{event.kind.replace(/_/g, " ")}</span>
            <span className="pulse-timeline__title">{event.title}</span>
            {event.description ? (
              <span className="pulse-timeline__desc">{event.description}</span>
            ) : null}
            <time className="pulse-timeline__time" dateTime={event.occurred_at}>
              {formatWhen(event.occurred_at)}
            </time>
          </div>
        </li>
      ))}
    </ol>
  );
}

export function PulseTimelinePreview() {
  return <MarketingPulseTimeline events={MOCK_PULSE_EVENTS.slice(0, 3)} />;
}
