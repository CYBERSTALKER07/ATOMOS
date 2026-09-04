"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import StatusBadge from "@/components/StatusBadge";
import { PageChrome } from "@/components/PageChrome";

type CreditProfile = {
  retailer_id: string;
  supplier_id: string;
  credit_limit_minor: number;
  current_balance_minor: number;
  available_credit_minor: number;
  delinquency_count?: number;
  status: string;
  utilization_bps?: number;
  needs_attention?: boolean;
  updated_at?: string;
};

function formatMinor(n: number): string {
  return (n ?? 0).toLocaleString();
}

export default function CreditCollectionsPage() {
  const t = usePortalT();
  const [profiles, setProfiles] = useState<CreditProfile[]>([]);
  const [statusFilter, setStatusFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [editId, setEditId] = useState<string | null>(null);
  const [limitInput, setLimitInput] = useState("");
  const [reason, setReason] = useState("");

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
      const body = (await res.json()) as { profiles?: CreditProfile[] };
      setProfiles(body.profiles ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_failed"));
      setProfiles([]);
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

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
      <div className="mb-6 p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 flex items-start gap-3">
        <div>
          <h4 className="font-semibold text-sm">Pending Finance Approval</h4>
          <p className="text-sm opacity-80 mt-1">
            You have disputed delivery drafts awaiting final approval. They are excluded from the available balance metrics below until finalized.
          </p>
        </div>
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <select
          className="border border-[var(--border)] rounded-lg px-3 py-1.5 text-sm"
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
        >
          <option value="">ALL</option>
          <option value="ACTIVE">ACTIVE</option>
          <option value="FROZEN">FROZEN</option>
          <option value="BLACKLISTED">BLACKLISTED</option>
          <option value="CLOSED">CLOSED</option>
        </select>
        <button type="button" className="text-sm underline" onClick={() => void load()}>
          Refresh
        </button>
        <span className="text-xs text-[var(--muted)]">{profiles.length} profiles</span>
      </div>

      {error ? <p className="text-sm text-red-600 mb-3">{error}</p> : null}

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-[var(--border)] text-left text-xs uppercase tracking-wide text-[var(--muted)]">
              <th className="py-2 pr-3">{t("supplier_portal.analytics.demand.flywheel.text.retailer")}</th>
              <th className="py-2 pr-3">{t("supplier_portal.compliance.text.status")}</th>
              <th className="py-2 pr-3 text-right">{t("supplier_portal.credit.collections.text.limit")}</th>
              <th className="py-2 pr-3 text-right">{t("supplier_portal.credit.collections.text.balance")}</th>
              <th className="py-2 pr-3 text-right">{t("supplier_portal.credit.collections.text.available")}</th>
              <th className="py-2 pr-3 text-right">{t("supplier_portal.credit.collections.text.util")}</th>
              <th className="py-2 pr-3">{t("supplier_portal.catalog.components.catalog_table.text.actions")}</th>
            </tr>
          </thead>
          <tbody>
            {profiles.length === 0 && !loading ? (
              <tr>
                <td colSpan={7} className="py-8 text-center text-[var(--muted)]">
                  No credit profiles for this filter.
                </td>
              </tr>
            ) : (
              profiles.map((p) => {
                const util =
                  p.utilization_bps != null
                    ? (p.utilization_bps / 100).toFixed(1)
                    : p.credit_limit_minor > 0
                      ? ((p.current_balance_minor * 100) / p.credit_limit_minor).toFixed(1)
                      : "0.0";
                const busy = busyId === p.retailer_id;
                return (
                  <tr
                    key={`${p.retailer_id}:${p.supplier_id}`}
                    className={`border-b border-[var(--border)] ${p.needs_attention ? "bg-amber-50/40" : ""}`}
                  >
                    <td className="py-2.5 pr-3 font-mono text-xs">{p.retailer_id}</td>
                    <td className="py-2.5 pr-3">
                      <StatusBadge state={p.status} />
                      {p.delinquency_count ? (
                        <span className="ml-1 text-xs text-red-600">delinq {p.delinquency_count}</span>
                      ) : null}
                    </td>
                    <td className="py-2.5 pr-3 text-right font-mono">{formatMinor(p.credit_limit_minor)}</td>
                    <td className="py-2.5 pr-3 text-right font-mono font-medium">
                      {formatMinor(p.current_balance_minor)}
                    </td>
                    <td className="py-2.5 pr-3 text-right font-mono text-[var(--muted)]">
                      {formatMinor(p.available_credit_minor)}
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
                            setLimitInput(String(p.credit_limit_minor));
                          }}
                        >
                          Set limit
                        </button>
                      </div>
                      {editId === p.retailer_id ? (
                        <div className="mt-2 flex flex-wrap gap-2 items-center">
                          <input
                            className="border border-[var(--border)] rounded px-2 py-1 text-xs w-28 font-mono"
                            value={limitInput}
                            onChange={(e) => setLimitInput(e.target.value)}
                            placeholder={t("supplier_portal.credit.collections.text.limit_minor")}
                          />
                          <input
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
                              const n = Number(limitInput);
                              if (!Number.isFinite(n) || n < 0) {
                                setError(t("supplier_portal.residual.text.invalid_limit"));
                                return;
                              }
                              void patchProfile(p.retailer_id, {
                                credit_limit_minor: Math.floor(n),
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
    </PageChrome>
  );
}
