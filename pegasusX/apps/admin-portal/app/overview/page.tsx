"use client";

import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { api, type FlagOverride, type Tenant } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { useAdminWS } from "@/lib/ws-provider";
import { deadLetterHealth, deadLetterLabel } from "@/lib/deadLetterHealth";
import { CopyableId } from "@/components/ui/CopyableId";

export default function OverviewPage() {
  const { token } = useAdminToken();
  const { refreshKey } = useAdminWS();

  const [tenants, setTenants] = useState<Tenant[] | null>(null);
  const [pending, setPending] = useState<FlagOverride[] | null>(null);
  const [summary, setSummary] = useState<Awaited<ReturnType<typeof api.outboxSummary>> | null>(null);
  const [runtime, setRuntime] = useState<Record<string, unknown>>({});
  const [events, setEvents] = useState<Array<Record<string, string>>>([]);
  const [deadLetters, setDeadLetters] = useState<Array<Record<string, string | number>>>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // DLQ Drawer & Replay State
  const [selectedDeadLetter, setSelectedDeadLetter] = useState<Record<string, string | number> | null>(null);
  const [replayingId, setReplayingId] = useState<string | null>(null);
  const [replaySuccess, setReplaySuccess] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError("");
    try {
      const [t, f, s, r, ev, dl] = await Promise.all([
        api.listTenants(token),
        api.listPendingFlags(token),
        api.outboxSummary(token),
        api.runtimeOps(token),
        api.outboxEvents(token),
        api.outboxDeadLetters(token),
      ]);
      setTenants(t.tenants || []);
      setPending(f.items || []);
      setSummary(s);
      setRuntime(r || {});
      setEvents(ev.events || []);
      setDeadLetters(dl.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load NOC metrics");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    void loadData();
  }, [loadData, refreshKey]);

  const handleReplay = async (eventId: string) => {
    if (!token) return;
    setReplayingId(eventId);
    setReplaySuccess(null);
    try {
      await api.replayDeadLetter(token, eventId);
      setReplaySuccess(`Event ${eventId} scheduled for immediate replay`);
      setTimeout(() => setReplaySuccess(null), 4000);
      void loadData();
    } catch (e) {
      alert(`Replay failed: ${e instanceof Error ? e.message : "Unknown error"}`);
    } finally {
      setReplayingId(null);
    }
  };

  if (loading && !summary) {
    return (
      <div className="space-y-6">
        <div className="h-8 w-64 rounded bg-gray-200 animate-pulse" />
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-24 rounded-xl bg-gray-200 animate-pulse" />
          ))}
        </div>
      </div>
    );
  }

  const dl = deadLetterHealth(summary);
  const unpublished =
    summary?.unpublished_available || summary?.available
      ? typeof summary?.unpublished_count === "number"
        ? summary.unpublished_count === 0
          ? "0 (Empty)"
          : String(summary.unpublished_count)
        : "unavailable"
      : "unavailable";

  const workersLive =
    typeof runtime.workers_live_cluster === "boolean"
      ? runtime.workers_live_cluster
        ? "Healthy (Live)"
        : "Down (Alert)"
      : "Active";

  return (
    <div className="space-y-8" data-testid="admin-overview-tower">
      {/* Top Banner */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Platform Command Cockpit
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Real-time NOC telemetry, Outbox stream lag, and Dead Letter Queue triage.
          </p>
        </div>
        <button
          type="button"
          onClick={() => void loadData()}
          className="inline-flex items-center gap-1.5 rounded-lg border bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
        >
          ↻ Refresh Pulse
        </button>
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-xs text-red-700">
          {error}
        </div>
      )}

      {replaySuccess && (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-3 text-xs text-emerald-800 font-medium">
          ✓ {replaySuccess}
        </div>
      )}

      {/* KPI Pulse Cards */}
      <section className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <PulseCard
          label="Unpublished Outbox"
          value={unpublished}
          subtext={summary?.oldest_age_seconds ? `Oldest: ${summary.oldest_age_seconds}s` : "Stream clear"}
          warn={summary?.lagging === true}
        />
        <PulseCard
          label="Dead Letter Queue"
          value={deadLetterLabel(dl)}
          subtext="Spanner OutboxDeadLetters"
          warn={dl.kind === "count" && dl.count > 0}
        />
        <PulseCard
          label="Worker Clusters"
          value={workersLive}
          subtext="ai-worker & handoff-service"
          warn={workersLive.includes("Down")}
        />
        <PulseCard
          label="Pending Dual-Control"
          value={pending ? String(pending.length) : "0"}
          subtext="Requires 2nd admin approval"
          warn={pending != null && pending.length > 0}
          link="/governance/flags"
        />
      </section>

      {/* DLQ Triage Center (UA-6) */}
      <section className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div className="flex items-center justify-between border-b pb-3">
          <div>
            <h2 className="text-sm font-bold text-gray-900 uppercase tracking-wide">
              Dead Letter Queue (DLQ) Triage Center
            </h2>
            <p className="text-xs text-gray-500 mt-0.5">
              Poison pills routed away from Kafka partition loops into Cloud Spanner for manual inspection.
            </p>
          </div>
          <span className="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-mono font-semibold text-gray-700">
            {deadLetters.length} Events Pending
          </span>
        </div>

        {deadLetters.length === 0 ? (
          <div className="py-8 text-center text-xs text-gray-500">
            ✓ Zero dead letters. All outbox events committed cleanly to downstream partitions.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b text-gray-500">
                  <th className="py-2 pr-3">Event ID</th>
                  <th className="py-2 pr-3">Aggregate</th>
                  <th className="py-2 pr-3">Failure Reason</th>
                  <th className="py-2 pr-3 text-center">Attempts</th>
                  <th className="py-2 pr-3">Timestamp</th>
                  <th className="py-2 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {deadLetters.map((ev) => (
                  <tr key={String(ev.event_id)} className="hover:bg-gray-50">
                    <td className="py-2.5 pr-3">
                      <CopyableId value={String(ev.event_id)} truncateLength={14} />
                    </td>
                    <td className="py-2.5 pr-3 font-mono text-gray-700">
                      {ev.aggregate_type || "N/A"}/{ev.aggregate_id || "N/A"}
                    </td>
                    <td className="py-2.5 pr-3 text-red-700 max-w-xs truncate font-mono text-[11px]">
                      {String(ev.reason || ev.error || "Kafka publish max retries exceeded")}
                    </td>
                    <td className="py-2.5 pr-3 text-center font-bold">
                      <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] text-red-800">
                        {ev.attempts || 3}
                      </span>
                    </td>
                    <td className="py-2.5 pr-3 text-gray-500 whitespace-nowrap">
                      {String(ev.dead_lettered_at || ev.created_at || "Just now")}
                    </td>
                    <td className="py-2.5 text-right space-x-2 whitespace-nowrap">
                      <button
                        type="button"
                        onClick={() => setSelectedDeadLetter(ev)}
                        className="rounded border bg-white px-2.5 py-1 text-[11px] font-semibold text-gray-700 hover:bg-gray-100"
                      >
                        Inspect
                      </button>
                      <button
                        type="button"
                        disabled={replayingId === String(ev.event_id)}
                        onClick={() => void handleReplay(String(ev.event_id))}
                        className="rounded bg-indigo-600 px-2.5 py-1 text-[11px] font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                      >
                        {replayingId === String(ev.event_id) ? "Replaying…" : "Replay"}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {/* Slide-Over Drawer for Payload Inspection (UA-6) */}
      {selectedDeadLetter && (
        <div className="fixed inset-0 z-50 flex justify-end bg-black/40 backdrop-blur-sm">
          <div className="h-full w-full max-w-xl bg-white p-6 shadow-2xl flex flex-col space-y-4 overflow-y-auto">
            <div className="flex items-center justify-between border-b pb-3">
              <div>
                <h3 className="text-sm font-bold text-gray-900">Dead Letter Event Inspector</h3>
                <p className="text-[11px] font-mono text-gray-500 mt-0.5">
                  ID: {String(selectedDeadLetter.event_id)}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setSelectedDeadLetter(null)}
                className="rounded-lg p-1 text-gray-400 hover:text-gray-700"
              >
                ✕
              </button>
            </div>

            <div className="space-y-3 flex-1 text-xs">
              <div>
                <span className="font-semibold text-gray-700 block mb-1">Target Topic:</span>
                <span className="font-mono bg-gray-100 rounded px-2 py-1 inline-block">
                  {String(selectedDeadLetter.topic_name || "events.orders")}
                </span>
              </div>

              <div>
                <span className="font-semibold text-gray-700 block mb-1">Failure Reason & Stack:</span>
                <div className="rounded-lg bg-red-50 p-3 text-red-800 font-mono text-[11px] break-all border border-red-200">
                  {String(selectedDeadLetter.reason || selectedDeadLetter.error || "Exceeded max delivery attempts (5 retries)")}
                </div>
              </div>

              <div>
                <span className="font-semibold text-gray-700 block mb-1">Raw Event Payload:</span>
                <pre className="rounded-lg bg-gray-900 text-gray-100 p-3 font-mono text-[11px] overflow-x-auto max-h-80 border">
                  {JSON.stringify(selectedDeadLetter, null, 2)}
                </pre>
              </div>
            </div>

            <div className="border-t pt-3 flex justify-between items-center">
              <button
                type="button"
                onClick={() => setSelectedDeadLetter(null)}
                className="rounded border px-4 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50"
              >
                Close Drawer
              </button>
              <button
                type="button"
                disabled={replayingId === String(selectedDeadLetter.event_id)}
                onClick={() => {
                  void handleReplay(String(selectedDeadLetter.event_id));
                  setSelectedDeadLetter(null);
                }}
                className="rounded bg-indigo-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                Replay Event Now
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Unpublished Outbox Stream */}
      <section className="rounded-xl border bg-white p-5 shadow-sm space-y-3">
        <h2 className="text-sm font-bold text-gray-900 uppercase tracking-wide">
          Active Outbox Stream (Spanner In-Flight Buffer)
        </h2>
        {events.length === 0 ? (
          <p className="text-xs text-gray-500 py-3">
            Zero unpublished events buffered in Spanner. All events relayed to Kafka.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b text-gray-500">
                  <th className="py-2 pr-2">Event ID</th>
                  <th className="py-2 pr-2">Aggregate</th>
                  <th className="py-2 pr-2">Topic</th>
                  <th className="py-2">Created</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {events.slice(0, 10).map((ev) => (
                  <tr key={ev.event_id}>
                    <td className="py-2 pr-2">
                      <CopyableId value={ev.event_id} truncateLength={14} />
                    </td>
                    <td className="py-2 pr-2 font-mono text-gray-700">
                      {ev.aggregate_type}/{ev.aggregate_id}
                    </td>
                    <td className="py-2 pr-2 font-mono text-indigo-700">{ev.topic_name}</td>
                    <td className="py-2 text-gray-500 whitespace-nowrap">{ev.created_at}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}

function PulseCard({
  label,
  value,
  subtext,
  warn,
  link,
}: {
  label: string;
  value: string;
  subtext?: string;
  warn?: boolean;
  link?: string;
}) {
  const content = (
    <div
      className={`rounded-xl border p-4 transition-all shadow-sm ${
        warn
          ? "border-amber-300 bg-amber-50/70 text-amber-950"
          : "border-gray-200 bg-white text-gray-900"
      }`}
    >
      <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500 block">
        {label}
      </span>
      <span className="mt-1 text-2xl font-black tracking-tight block font-mono">{value}</span>
      {subtext && <span className="mt-1 text-[11px] text-gray-500 block truncate">{subtext}</span>}
    </div>
  );

  if (link) {
    return <Link href={link}>{content}</Link>;
  }
  return content;
}
