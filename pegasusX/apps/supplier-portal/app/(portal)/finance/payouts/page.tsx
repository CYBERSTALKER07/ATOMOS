"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { ApiError } from "@pegasusx/api-client";
import { supplierPayoutGenerateKey } from "@pegasusx/api-client/idempotency";
import type { PayoutBatch, PayoutRailInfo, SupplierPayoutPolicy } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import EmptyState from "@/components/EmptyState";

const api = createSupplierApi();

function fmtMinor(n: number): string {
  return new Intl.NumberFormat("uz-UZ").format(n);
}

export default function SupplierPayoutsPage() {
  const [rail, setRail] = useState<PayoutRailInfo | null>(null);
  const [policy, setPolicy] = useState<SupplierPayoutPolicy | null>(null);
  const [policyReason, setPolicyReason] = useState("");
  const [batches, setBatches] = useState<PayoutBatch[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [periodStart, setPeriodStart] = useState("");
  const [periodEnd, setPeriodEnd] = useState("");
  const [draftMode, setDraftMode] = useState("HQ_SUPPLIER");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [railResp, list] = await Promise.all([api.getPayoutRail(), api.listPayoutBatches()]);
      setRail(railResp);
      setBatches(list.batches ?? []);
      try {
        const p = await api.getPayoutPolicy();
        setPolicy(p);
        setDraftMode(p.payout_mode || "HQ_SUPPLIER");
      } catch {
        setPolicy(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load payouts");
      setBatches([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function generate() {
    if (!periodStart || !periodEnd) {
      setStatus("Period start and end are required");
      return;
    }
    setBusy(true);
    setStatus(null);
    try {
      const scope = supplierScopeId();
      await api.generatePayoutBatch(
        { period_start: periodStart, period_end: periodEnd },
        supplierPayoutGenerateKey(scope, periodStart, periodEnd),
      );
      setStatus("Batch generated — export CSV, process at bank, then mark-paid.");
      await load();
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Generate failed");
    } finally {
      setBusy(false);
    }
  }

  async function exportCsv(batchId: string) {
    setBusy(true);
    setStatus(null);
    try {
      const csv = await api.exportPayoutBatch(batchId);
      const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `payout-${batchId}.csv`;
      a.click();
      URL.revokeObjectURL(url);
      setStatus("CSV downloaded. Process at bank, then mark-paid. Not a live rail.");
      await load();
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Export failed");
    } finally {
      setBusy(false);
    }
  }

  async function markPaid(batchId: string) {
    setBusy(true);
    setStatus(null);
    try {
      const resp = await api.markPayoutBatchPaid(batchId);
      setStatus(resp.message || `Marked ${resp.status}`);
      await load();
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Mark-paid failed");
    } finally {
      setBusy(false);
    }
  }

  async function dispatchLive(batchId: string) {
    setBusy(true);
    setStatus(null);
    try {
      const resp = await api.dispatchPayoutBatch(batchId, true);
      if (resp.code === "no_live_rail" || resp.error === "no_live_rail") {
        setStatus(resp.message || "no_live_rail — export CSV, process at bank, then mark-paid.");
      } else {
        setStatus(resp.message || "Dispatch attempted");
      }
      await load();
    } catch (err) {
      if (err instanceof ApiError) {
        const payload = err.payload as { code?: string; error?: string; message?: string } | null;
        if (payload?.code === "no_live_rail" || payload?.error === "no_live_rail" || err.status === 409) {
          setStatus(payload?.message || "no_live_rail — export CSV, process at bank, then mark-paid.");
          setBusy(false);
          return;
        }
      }
      setStatus(err instanceof Error ? err.message : "Dispatch failed");
    } finally {
      setBusy(false);
    }
  }

  async function savePolicy() {
    if (!policyReason.trim()) {
      setStatus("Reason is required to change payout mode");
      return;
    }
    setBusy(true);
    setStatus(null);
    try {
      const next = await api.patchPayoutPolicy({
        payout_mode: draftMode,
        reason: policyReason.trim(),
      });
      setPolicy(next);
      setDraftMode(next.payout_mode || draftMode);
      setPolicyReason("");
      setStatus("Payout mode saved. Bank-file rail is unchanged (no_live_rail).");
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Policy save failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageChrome
      icon="treasury"
      title="Payout batches"
      description="Bank-file rail only: generate a period, export CSV, process at the bank, then mark-paid. Live dispatch is rejected (no_live_rail). Payout mode is HQ vs warehouse-local — not a live PSP."
      loading={loading}
      error={error}
      actions={
        <button type="button" className="portal-btn portal-btn--ghost text-xs" onClick={() => void load()}>
          Refresh
        </button>
      }
    >
      <section className="desk-card p-6">
        <h2 className="bento-card-title">Rail honesty</h2>
        <p className="md-typescale-body-small mt-2" style={{ color: "var(--desk-text-secondary)" }}>
          {rail?.message || "Bank-file rail: export CSV, process at bank, then mark-paid. live dispatch is rejected (no_live_rail)."}
        </p>
        <p className="text-xs mt-1" style={{ color: "var(--desk-text-secondary)" }}>
          Live: {rail?.is_live ? "yes" : "no"} · {rail?.workflow || "bank_file_export_then_mark_paid"}
        </p>
      </section>

      <section className="desk-card p-6 mt-6">
        <h2 className="bento-card-title">Payout policy</h2>
        <p className="md-typescale-body-small mt-2" style={{ color: "var(--desk-text-secondary)" }}>
          Mode {policy?.payout_mode || "HQ_SUPPLIER"} · source {policy?.source || "DEFAULT"}. Does not enable a live rail.
        </p>
        <div className="mt-3 flex flex-wrap gap-3 items-end">
          <label className="text-sm">
            Mode
            <select
              className="block mt-1 rounded-lg border px-3 py-2 text-sm"
              style={{ borderColor: "var(--color-md-outline-variant)" }}
              value={draftMode}
              onChange={(e) => setDraftMode(e.target.value)}
            >
              <option value="HQ_SUPPLIER">HQ_SUPPLIER</option>
              <option value="WAREHOUSE_LOCAL">WAREHOUSE_LOCAL</option>
            </select>
          </label>
          <label className="text-sm">
            Reason
            <input
              type="text"
              className="block mt-1 rounded-lg border px-3 py-2 text-sm min-w-[16rem]"
              style={{ borderColor: "var(--color-md-outline-variant)" }}
              value={policyReason}
              onChange={(e) => setPolicyReason(e.target.value)}
              placeholder="Required for PATCH"
            />
          </label>
          <button type="button" className="portal-btn portal-btn--primary text-xs" disabled={busy} onClick={() => void savePolicy()}>
            Save mode
          </button>
        </div>
      </section>

      <section className="desk-card p-6 mt-6">
        <h2 className="bento-card-title">Generate period</h2>
        <div className="mt-3 flex flex-wrap gap-3 items-end">
          <label className="text-sm">
            Start
            <input
              type="date"
              className="block mt-1 rounded-lg border px-3 py-2 text-sm"
              style={{ borderColor: "var(--color-md-outline-variant)" }}
              value={periodStart}
              onChange={(e) => setPeriodStart(e.target.value)}
            />
          </label>
          <label className="text-sm">
            End
            <input
              type="date"
              className="block mt-1 rounded-lg border px-3 py-2 text-sm"
              style={{ borderColor: "var(--color-md-outline-variant)" }}
              value={periodEnd}
              onChange={(e) => setPeriodEnd(e.target.value)}
            />
          </label>
          <button type="button" className="portal-btn portal-btn--primary text-xs" disabled={busy} onClick={() => void generate()}>
            Generate
          </button>
        </div>
      </section>

      <section className="desk-card p-6 mt-6 overflow-x-auto">
        <h2 className="bento-card-title">Batches</h2>
        {status ? (
          <p className="md-typescale-body-small mt-2" style={{ color: "var(--desk-text-secondary)" }}>
            {status}
          </p>
        ) : null}
        {batches.length === 0 ? (
          <EmptyState
            headline="No payout batches"
            body="Generate a period to create a bank-file batch. Empty list is {batches:[]}."
          />
        ) : (
          <table className="desk-table w-full text-sm mt-3">
            <thead>
              <tr>
                <th className="text-left px-3 py-2">Period</th>
                <th className="text-left px-3 py-2">Status</th>
                <th className="text-right px-3 py-2">Net (minor)</th>
                <th className="text-right px-3 py-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {batches.map((b) => (
                <tr key={b.batch_id}>
                  <td className="px-3 py-2 font-mono text-xs">
                    {b.period_start} → {b.period_end}
                  </td>
                  <td className="px-3 py-2">{b.status}</td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {fmtMinor(b.net_payout_minor)} {b.currency}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex gap-2 justify-end">
                      <button type="button" className="portal-btn portal-btn--ghost text-xs" disabled={busy} onClick={() => void exportCsv(b.batch_id)}>
                        Export CSV
                      </button>
                      <button type="button" className="portal-btn portal-btn--primary text-xs" disabled={busy} onClick={() => void markPaid(b.batch_id)}>
                        Mark paid
                      </button>
                      <button type="button" className="portal-btn portal-btn--ghost text-xs" disabled={busy} onClick={() => void dispatchLive(b.batch_id)}>
                        Dispatch live
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </PageChrome>
  );
}
