"use client";

import { useEffect, useState } from "react";
import { api, type FlagOverride, type Tenant } from "@/lib/api";
import { deadLetterHealth, deadLetterLabel } from "@/lib/deadLetterHealth";

export default function CommandBoard({
  token,
  refreshKey = 0,
  onOpenTab,
}: {
  token: string;
  refreshKey?: number;
  onOpenTab: (tab: string) => void;
}) {
  const [tenants, setTenants] = useState<Tenant[] | null>(null);
  const [pending, setPending] = useState<FlagOverride[] | null>(null);
  const [summary, setSummary] = useState<Awaited<ReturnType<typeof api.outboxSummary>> | null>(null);
  const [runtime, setRuntime] = useState<Record<string, unknown> | null>(null);
  const [matchCount, setMatchCount] = useState<number | null>(null);
  const [invoiceCount, setInvoiceCount] = useState<number | null>(null);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setErr("");
      try {
        const [t, f, s, r, m, b] = await Promise.all([
          api.listTenants(token),
          api.listPendingFlags(token),
          api.outboxSummary(token),
          api.runtimeOps(token),
          api.listMatchQueue(token, "PENDING"),
          api.listBillingInvoices(token),
        ]);
        if (cancelled) return;
        setTenants(t.tenants || []);
        setPending(f.items || []);
        setSummary(s);
        setRuntime(r);
        setMatchCount(m.items?.length ?? 0);
        setInvoiceCount(b.invoices?.length ?? 0);
      } catch (e) {
        if (!cancelled) setErr(e instanceof Error ? e.message : "command_load_failed");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token, refreshKey]);

  if (loading) return <p className="text-sm text-gray-600">Loading command…</p>;
  if (err) return <p className="text-sm text-red-700">{err}</p>;

  const dl = deadLetterHealth(summary);
  const unpublishedAvailable = summary?.unpublished_available === true || summary?.available === true;
  const unpublished =
    unpublishedAvailable && typeof summary?.unpublished_count === "number"
      ? summary.unpublished_count === 0
        ? "empty"
        : String(summary.unpublished_count)
      : "unavailable";
  const workers =
    runtime && typeof runtime.workers_live_cluster === "boolean"
      ? runtime.workers_live_cluster
        ? "live"
        : "down"
      : "unavailable";

  const byStatus = {
    PENDING: tenants?.filter((x) => x.Status === "PENDING").length ?? 0,
    APPROVED: tenants?.filter((x) => x.Status === "APPROVED").length ?? 0,
    SUSPENDED: tenants?.filter((x) => x.Status === "SUSPENDED").length ?? 0,
    OFFBOARDED: tenants?.filter((x) => x.Status === "OFFBOARDED").length ?? 0,
  };

  return (
    <div className="space-y-6" data-testid="gs-u-admin-command">
      <section className="grid grid-cols-2 gap-3 md:grid-cols-4" data-testid="gs-u-admin-health">
        <HealthCard label="Unpublished" value={unpublished} onClick={() => onOpenTab("ops")} />
        <HealthCard
          label="Dead letters"
          value={deadLetterLabel(dl)}
          warn={dl.kind === "count"}
          onClick={() => onOpenTab("ops")}
        />
        <HealthCard label="Workers" value={workers} warn={workers === "down"} onClick={() => onOpenTab("ops")} />
        <HealthCard
          label="Pending flags"
          value={pending == null ? "unavailable" : pending.length === 0 ? "empty" : String(pending.length)}
          onClick={() => onOpenTab("flags")}
        />
      </section>

      <section className="rounded border p-4" data-testid="gs-u-admin-tenants">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Tenants</h2>
          <button className="text-xs text-indigo-700 underline" onClick={() => onOpenTab("tenants")}>
            Open table
          </button>
        </div>
        {tenants == null ? (
          <p className="text-sm text-gray-600">unavailable</p>
        ) : tenants.length === 0 ? (
          <p className="text-sm text-gray-600">empty</p>
        ) : (
          <>
            <div className="mb-3 flex flex-wrap gap-2">
              {(["PENDING", "APPROVED", "SUSPENDED", "OFFBOARDED"] as const).map((st) => (
                <span key={st} className="rounded-full border px-2 py-0.5 text-xs">
                  {st} · {byStatus[st]}
                </span>
              ))}
            </div>
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b text-gray-500">
                  <th className="py-1 pr-2">ID</th>
                  <th className="py-1 pr-2">KYB</th>
                  <th className="py-1 pr-2">Pack</th>
                  <th className="py-1 pr-2">Cell</th>
                  <th className="py-1">Registered</th>
                </tr>
              </thead>
              <tbody>
                {tenants.slice(0, 8).map((t) => (
                  <tr key={`${t.TenantType}/${t.TenantID}`} className="border-b border-gray-100">
                    <td className="py-1 pr-2 font-mono">{t.TenantID}</td>
                    <td className="py-1 pr-2">{t.Status}</td>
                    <td className="py-1 pr-2">{t.market_code || "empty"}</td>
                    <td className="py-1 pr-2">{t.home_cell || "empty"}</td>
                    <td className="py-1 text-gray-500">unavailable</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="mt-2 text-xs text-gray-500">
              IsRegistered is tenant-register, not this list — not invented.
            </p>
          </>
        )}
      </section>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <section className="rounded border p-4" data-testid="gs-u-admin-flags">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Pending dual-control</h2>
            <button className="text-xs text-indigo-700 underline" onClick={() => onOpenTab("flags")}>
              Open flags
            </button>
          </div>
          {pending == null ? (
            <p className="text-sm text-gray-600">unavailable</p>
          ) : pending.length === 0 ? (
            <p className="text-sm text-gray-600">empty</p>
          ) : (
            <ul className="space-y-2 text-xs">
              {pending.map((f) => (
                <li key={`${f.FlagKey}|${f.TenantType}|${f.TenantID}`}>
                  <span className="font-mono">{f.FlagKey}</span> · {f.TenantID} · {f.Status}
                </li>
              ))}
            </ul>
          )}
        </section>
        <section className="rounded border p-4" data-testid="gs-u-admin-queues">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Queues</h2>
          <ul className="mt-2 space-y-2 text-sm">
            <li>
              <button className="underline" onClick={() => onOpenTab("match")}>
                Match queue
              </button>
              {": "}
              {matchCount == null ? "unavailable" : matchCount === 0 ? "empty" : String(matchCount)}
            </li>
            <li>
              <button className="underline" onClick={() => onOpenTab("billing")}>
                Billing invoices
              </button>
              {": "}
              {invoiceCount == null ? "unavailable" : invoiceCount === 0 ? "empty" : String(invoiceCount)}
            </li>
            <li>
              <button className="underline" onClick={() => onOpenTab("accuracy")}>
                Planning accuracy
              </button>
              {": "}
              supplier required — no invented mape28 line
            </li>
            <li>
              <button className="underline" onClick={() => onOpenTab("partner")}>
                Partner keys / AS2 / SFTP
              </button>
              {": "}
              tenant required
            </li>
          </ul>
        </section>
      </section>
    </div>
  );
}

function HealthCard({
  label,
  value,
  warn,
  onClick,
}: {
  label: string;
  value: string;
  warn?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded border p-3 text-left ${warn ? "border-amber-400 bg-amber-50" : ""}`}
    >
      <p className="text-[10px] font-semibold uppercase tracking-wide text-gray-500">{label}</p>
      <p className="mt-1 text-lg font-semibold">{value}</p>
    </button>
  );
}
