"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
import StatusBadge from "@/components/StatusBadge";
import { ObjectHistoryPanel } from "@/components/audit/ObjectHistoryPanel";
import { formatBpsPercent, formatRatioPercentTenths, minorToWholeTrunc } from "@/lib/money";
import { formatWholeSom, wholeSomToMinor } from "@/lib/exception-desk";

type CreditProfile = {
  retailer_id: string;
  supplier_id: string;
  retailer_name?: string;
  display_name?: string;
  tin?: string;
  phone?: string;
  pending_credit_note_count?: number;
  credit_limit_minor: number;
  current_balance_minor: number;
  available_credit_minor: number;
  delinquency_count?: number;
  status: string;
  utilization_bps?: number;
  needs_attention?: boolean;
  updated_at?: string;
};

export default function CollectionsPage() {
  const t = usePortalT();
  const [profiles, setProfiles] = useState<CreditProfile[]>([]);
  const [pendingCreditNoteCount, setPendingCreditNoteCount] = useState(0);
  const [statusFilter, setStatusFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [editId, setEditId] = useState<string | null>(null);
  const [limitInput, setLimitInput] = useState("");
  const [reason, setReason] = useState("");
  const [historyRetailerId, setHistoryRetailerId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const q = new URLSearchParams({ limit: "100" });
      if (statusFilter) q.set("status", statusFilter);
      const res = await supplierFetch(`/v1/supplier/credit-profiles?${q.toString()}`);
      if (!res.ok) {
        throw new Error(`credit_list_${res.status}`);
      }
      const body = (await res.json()) as {
        profiles?: CreditProfile[];
        pending_credit_note_count?: number;
      };
      const rows = body.profiles ?? [];
      setProfiles(rows);
      const rowDraftSum = rows.reduce(
        (acc, p) => acc + (p.pending_credit_note_count ?? 0),
        0,
      );
      setPendingCreditNoteCount(body.pending_credit_note_count ?? rowDraftSum);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_failed"));
      setProfiles([]);
      setPendingCreditNoteCount(0);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function patchProfile(retailerId: string, body: Record<string, unknown>) {
    setBusyId(retailerId);
    setError(null);
    try {
      const res = await supplierFetch("/v1/supplier/retailer-credit-profile", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          retailer_id: retailerId,
          reason: reason || "collections_desk",
          ...body,
        }),
      });
      if (!res.ok) {
        throw new Error(`patch_${res.status}`);
      }
      setEditId(null);
      setLimitInput("");
      setReason("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.patch_failed"));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <PageChrome
      title={t("portal.nav.credit_collections")}
      description={t("supplier_portal.residual.text.supplier_scoped_credit_lines_limits_open_balances_freeze_unfreez")}
      loading={loading}
    >
      {pendingCreditNoteCount > 0 ? (
        <div
          role="status"
          className="mb-6 p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 flex items-start gap-3"
        >
          <div>
            <h4 className="font-semibold text-sm">
              Pending Finance Approval ({pendingCreditNoteCount})
            </h4>
            <p className="text-sm opacity-80 mt-1">
              {pendingCreditNoteCount} disputed delivery draft(s) are awaiting final credit-note approval and are excluded from available balance metrics below until finalized.
            </p>
          </div>
        </div>
      ) : null}

      <div className="mb-4 flex flex-wrap gap-2 items-center">
        {(["", "ACTIVE", "FROZEN", "DELINQUENT"] as const).map((s) => (
          <button
            key={s || "ALL"}
            type="button"
            onClick={() => setStatusFilter(s)}
            className={`px-3 py-1.5 rounded-full text-xs font-medium border transition ${
              statusFilter === s
                ? "bg-[var(--color-md-primary)] text-white border-[var(--color-md-primary)]"
                : "border-[var(--border)] text-[var(--fg)]"
            }`}
          >
            {s || "ALL"}
          </button>
        ))}
        <button
          type="button"
          onClick={() => void load()}
          className="ml-auto px-3 py-1.5 rounded-lg text-xs border border-[var(--border)]"
        >
          Refresh
        </button>
      </div>

      {error ? <p className="mb-3 text-sm text-red-600">{error}</p> : null}

      <div className="overflow-x-auto">
        <table className="w-full text-sm border-collapse">
          <thead>
            <tr className="border-b border-[var(--border)] text-left text-xs uppercase text-[var(--muted)]">
              <th className="py-2 pr-3">{t("supplier_portal.analytics.demand.flywheel.text.retailer")}</th>
              <th className="py-2 pr-3">{t("supplier_portal.compliance.text.status")}</th>
              <th className="py-2 pr-3 text-right">Limit (so&apos;m)</th>
              <th className="py-2 pr-3 text-right">Balance (so&apos;m)</th>
              <th className="py-2 pr-3 text-right">Available (so&apos;m)</th>
              <th className="py-2 pr-3 text-right">{t("supplier_portal.credit.collections.text.util")}</th>
              <th className="py-2 pr-3">{t("supplier_portal.catalog.components.catalog_table.text.actions")}</th>
            </tr>
          </thead>
          <tbody>
            {profiles.length === 0 ? (
              <tr>
                <td colSpan={7} className="py-8 text-center text-[var(--muted)]">
                  No credit profiles for this filter.
                </td>
              </tr>
            ) : (
              profiles.map((p) => {
                const util =
                  p.utilization_bps != null
                    ? formatBpsPercent(p.utilization_bps, 1)
                    : p.credit_limit_minor > 0
                      ? formatRatioPercentTenths(p.current_balance_minor, p.credit_limit_minor)
                      : "0.0";
                const busy = busyId === p.retailer_id;
                const shopName = (p.display_name || p.retailer_name || "").trim();
                return (
                  <tr
                    key={`${p.retailer_id}:${p.supplier_id}`}
                    className={`border-b border-[var(--border)] ${p.needs_attention ? "bg-amber-50/40" : ""}`}
                  >
                    <td className="py-2.5 pr-3">
                      <div className="font-medium text-[var(--fg)]">
                        {shopName || p.retailer_id}
                      </div>
                      <div className="text-xs text-[var(--muted)] flex flex-wrap gap-x-2">
                        {p.tin ? <span className="font-mono">TIN {p.tin}</span> : null}
                        {p.phone ? <span className="font-mono">{p.phone}</span> : null}
                        {shopName ? <span className="font-mono opacity-75">{p.retailer_id}</span> : null}
                      </div>
                    </td>
                    <td className="py-2.5 pr-3">
                      <StatusBadge state={p.status} />
                      {p.delinquency_count ? (
                        <span className="ml-1 text-xs text-red-600">delinq {p.delinquency_count}</span>
                      ) : null}
                    </td>
                    <td className="py-2.5 pr-3 text-right font-mono">{formatWholeSom(p.credit_limit_minor)}</td>
                    <td className="py-2.5 pr-3 text-right font-mono font-medium">
                      {formatWholeSom(p.current_balance_minor)}
                    </td>
                    <td className="py-2.5 pr-3 text-right font-mono text-[var(--muted)]">
                      {formatWholeSom(p.available_credit_minor)}
                    </td>
                    <td className="py-2.5 pr-3 text-right font-mono">{util}%</td>
                    <td className="py-2.5 pr-3">
                      <div className="flex flex-wrap gap-2 items-center">
                        {p.status === "ACTIVE" ? (
                          <button
                            type="button"
                            disabled={busy}
                            className="text-xs underline text-orange-700"
                            onClick={() =>
                              void patchProfile(p.retailer_id, {
                                credit_limit_minor: p.credit_limit_minor,
                                status: "FROZEN",
                              })
                            }
                          >
                            Freeze
                          </button>
                        ) : p.status === "FROZEN" ? (
                          <button
                            type="button"
                            disabled={busy}
                            className="text-xs underline text-emerald-700"
                            onClick={() =>
                              void patchProfile(p.retailer_id, {
                                credit_limit_minor: p.credit_limit_minor,
                                status: "ACTIVE",
                              })
                            }
                          >
                            Unfreeze
                          </button>
                        ) : null}
                        <button
                          type="button"
                          className="text-xs underline"
                          disabled={busy}
                          onClick={() => {
                            setEditId(p.retailer_id);
                            setLimitInput(String(minorToWholeTrunc(p.credit_limit_minor)));
                          }}
                        >
                          Set limit
                        </button>
                        <button
                          type="button"
                          className="text-xs underline text-[var(--muted)]"
                          onClick={() =>
                            setHistoryRetailerId((curr) =>
                              curr === p.retailer_id ? null : p.retailer_id,
                            )
                          }
                        >
                          {t("supplier_portal.audit.open_history")}
                        </button>
                      </div>
                      {editId === p.retailer_id ? (
                        <div className="mt-2 flex flex-wrap gap-2 items-center">
                          <input
                            id="limitinput-input-2"
                            aria-label="Credit limit (so'm)"
                            className="border border-[var(--border)] rounded px-2 py-1 text-xs w-32 font-mono"
                            value={limitInput}
                            onChange={(e) => setLimitInput(e.target.value)}
                            placeholder="Limit in so'm"
                          />
                          <input
                            id="reason-input-1"
                            aria-label="Reason"
                            className="border border-[var(--border)] rounded px-2 py-1 text-xs w-36"
                            value={reason}
                            onChange={(e) => setReason(e.target.value)}
                            placeholder={t("supplier_portal.credit.collections.text.reason")}
                          />
                          <button
                            type="button"
                            className="text-xs font-semibold underline"
                            disabled={busy}
                            onClick={() => {
                              const minor = wholeSomToMinor(limitInput);
                              if (minor === null) {
                                setError(t("supplier_portal.residual.text.invalid_limit"));
                                return;
                              }
                              void patchProfile(p.retailer_id, {
                                credit_limit_minor: minor,
                                status: p.status,
                              });
                            }}
                          >
                            Save
                          </button>
                          <button
                            type="button"
                            className="text-xs underline text-[var(--muted)]"
                            onClick={() => setEditId(null)}
                          >
                            Cancel
                          </button>
                        </div>
                      ) : null}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {historyRetailerId ? (
        <div className="mt-6">
          <ObjectHistoryPanel
            objectType="credit_relationship"
            fallbackObjectTypes={["retailer_credit_profile", "credit_relationships", "credit"]}
            objectId={historyRetailerId}
            compact
            onClose={() => setHistoryRetailerId(null)}
          />
        </div>
      ) : null}
    </PageChrome>
  );
}
