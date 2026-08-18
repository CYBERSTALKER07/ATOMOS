"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { deadLetterHealth, deadLetterLabel } from "@/lib/deadLetterHealth";

export default function OpsPanel({ token }: { token: string }) {
  const [summary, setSummary] = useState<Record<string, unknown> | null>(null);
  const [runtime, setRuntime] = useState<Record<string, unknown> | null>(null);
  const [events, setEvents] = useState<Array<Record<string, string>>>([]);
  const [deadLetters, setDeadLetters] = useState<Array<Record<string, string | number>>>([]);
  const [deadLetterTotal, setDeadLetterTotal] = useState<ReturnType<typeof deadLetterHealth>>({
    kind: "unavailable",
  });
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setErr("");
      try {
        const [s, r, e, d] = await Promise.all([
          api.outboxSummary(token),
          api.runtimeOps(token),
          api.outboxEvents(token),
          api.outboxDeadLetters(token),
        ]);
        if (cancelled) return;
        setSummary(s as Record<string, unknown>);
        setRuntime(r);
        setEvents(e.events || []);
        setDeadLetters(d.items || []);
        setDeadLetterTotal(deadLetterHealth({ ...s, ...d }));
      } catch (e) {
        if (!cancelled) setErr(e instanceof Error ? e.message : "ops_load_failed");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token]);

  if (loading) return <p className="text-sm text-gray-600">Loading ops…</p>;
  if (err) return <p className="text-sm text-red-700">{err}</p>;

  return (
    <div className="space-y-6">
      <section className="rounded border p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Runtime</h2>
        <pre className="mt-2 overflow-auto rounded bg-gray-50 p-3 text-xs">
          {JSON.stringify(runtime, null, 2)}
        </pre>
      </section>
      <section className="rounded border p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Outbox lag</h2>
        <pre className="mt-2 overflow-auto rounded bg-gray-50 p-3 text-xs">
          {JSON.stringify(summary, null, 2)}
        </pre>
        {summary?.lagging === true && (
          <p className="mt-2 text-sm text-amber-800">Outbox lag exceeds threshold — check worker tier.</p>
        )}
      </section>
      <section className="rounded border p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">
          Unpublished events (sample)
        </h2>
        {events.length === 0 ? (
          <p className="mt-2 text-sm text-gray-600">None or store unavailable.</p>
        ) : (
          <table className="mt-2 w-full text-left text-xs">
            <thead>
              <tr className="border-b text-gray-500">
                <th className="py-1 pr-2">Event</th>
                <th className="py-1 pr-2">Aggregate</th>
                <th className="py-1 pr-2">Topic</th>
                <th className="py-1">Created</th>
              </tr>
            </thead>
            <tbody>
              {events.map((ev) => (
                <tr key={ev.event_id} className="border-b border-gray-100">
                  <td className="py-1 pr-2 font-mono">{ev.event_id}</td>
                  <td className="py-1 pr-2">
                    {ev.aggregate_type}/{ev.aggregate_id}
                  </td>
                  <td className="py-1 pr-2">{ev.topic_name}</td>
                  <td className="py-1">{ev.created_at}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <p className="mt-3 text-xs text-gray-500">
          Kafka topic DLQ remains CLI (<code>cmd/replay-dlq</code>). Spanner dead letters below.
        </p>
      </section>
      <section className="rounded border p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">
          Outbox dead letters (Spanner)
        </h2>
        <p className="mt-1 text-sm" data-testid="gs-u-admin-dlq-count">
          COUNT(*) · {deadLetterLabel(deadLetterTotal)}
          {deadLetterTotal.kind !== "unavailable" ? " — not page length" : ""}
        </p>
        {deadLetterTotal.kind === "unavailable" ? (
          <p className="mt-2 text-sm text-gray-600">unavailable — table or Spanner not wired</p>
        ) : deadLetters.length === 0 ? (
          <p className="mt-2 text-sm text-gray-600">empty</p>
        ) : (
          <table className="mt-2 w-full text-left text-xs">
            <thead>
              <tr className="border-b text-gray-500">
                <th className="py-1 pr-2">Event</th>
                <th className="py-1 pr-2">Aggregate</th>
                <th className="py-1 pr-2">Attempts</th>
                <th className="py-1">Dead-lettered</th>
              </tr>
            </thead>
            <tbody>
              {deadLetters.map((ev) => (
                <tr key={String(ev.event_id)} className="border-b border-gray-100">
                  <td className="py-1 pr-2 font-mono">{ev.event_id}</td>
                  <td className="py-1 pr-2">
                    {ev.aggregate_type}/{ev.aggregate_id}
                  </td>
                  <td className="py-1 pr-2">{ev.attempts}</td>
                  <td className="py-1">{ev.dead_lettered_at}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}
