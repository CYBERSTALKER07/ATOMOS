"use client";

import { useEffect, useState } from "react";
import { api, type BillingInvoice, type BillingFeeSchedule } from "@/lib/api";

export default function BillingPanel({ token }: { token: string }) {
  const [invoices, setInvoices] = useState<BillingInvoice[]>([]);
  const [schedules, setSchedules] = useState<BillingFeeSchedule[]>([]);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [runMsg, setRunMsg] = useState("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setErr("");
      try {
        const [inv, sch] = await Promise.all([api.listBillingInvoices(token), api.listBillingFeeSchedules(token)]);
        if (cancelled) return;
        setInvoices(inv.invoices || []);
        setSchedules(sch.fee_schedules || []);
      } catch (e) {
        if (!cancelled) setErr(e instanceof Error ? e.message : "billing_load_failed");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token]);

  if (loading) return <p className="text-sm text-gray-600">Loading billing…</p>;
  if (err) return <p className="text-sm text-red-700">{err}</p>;

  return (
    <div className="space-y-6">
      <section className="rounded border p-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Monthly run</h2>
          <button
            disabled={running}
            className="rounded border px-3 py-1 text-sm disabled:opacity-50"
            onClick={async () => {
              setRunning(true);
              setRunMsg("");
              try {
                const res = await api.runMonthlyBilling(token);
                setRunMsg(`billed ${res.billed} for ${res.month}`);
              } catch (e) {
                setRunMsg(e instanceof Error ? e.message : "run_failed");
              } finally {
                setRunning(false);
              }
            }}
          >
            Run previous month
          </button>
        </div>
        {runMsg ? <p className="mt-2 text-sm text-gray-700">{runMsg}</p> : null}
        <p className="mt-2 text-xs text-gray-500">Zero fee schedules skip (no silent charges). Requires AR_INVOICES_ENABLED.</p>
      </section>
      <section className="rounded border p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Invoices</h2>
        {invoices.length === 0 ? (
          <p className="mt-2 text-sm text-gray-600">No platform invoices.</p>
        ) : (
          <table className="mt-2 w-full text-left text-xs">
            <thead>
              <tr className="border-b text-gray-500">
                <th className="py-1 pr-2">Invoice</th>
                <th className="py-1 pr-2">Supplier</th>
                <th className="py-1 pr-2">Status</th>
                <th className="py-1">Balance</th>
              </tr>
            </thead>
            <tbody>
              {invoices.map((inv) => (
                <tr key={inv.invoice_id} className="border-b border-gray-100">
                  <td className="py-1 pr-2 font-mono">{inv.invoice_id}</td>
                  <td className="py-1 pr-2">{inv.billed_supplier_id}</td>
                  <td className="py-1 pr-2">{inv.status}</td>
                  <td className="py-1">
                    {inv.balance_minor} {inv.currency}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
      <section className="rounded border p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Fee schedules</h2>
        {schedules.length === 0 ? (
          <p className="mt-2 text-sm text-gray-600">No schedules — monthly run will skip.</p>
        ) : (
          <table className="mt-2 w-full text-left text-xs">
            <thead>
              <tr className="border-b text-gray-500">
                <th className="py-1 pr-2">Tier</th>
                <th className="py-1 pr-2">Supplier</th>
                <th className="py-1 pr-2">Per order</th>
                <th className="py-1">GMV bps</th>
              </tr>
            </thead>
            <tbody>
              {schedules.map((s) => (
                <tr key={s.fee_schedule_id} className="border-b border-gray-100">
                  <td className="py-1 pr-2">{s.tier}</td>
                  <td className="py-1 pr-2">{s.supplier_id || "tier default"}</td>
                  <td className="py-1 pr-2">{s.per_order_minor}</td>
                  <td className="py-1">{s.gmv_bps}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}
