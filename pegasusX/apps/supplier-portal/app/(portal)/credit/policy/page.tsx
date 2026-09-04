"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
import { CreditEnableModal } from "@/components/CreditEnableModal";

type Program = {
  supplier_id: string;
  program_enabled: boolean;
  global_terms_days: number;
  global_grace_days: number;
  global_default_limit_minor: number;
  timezone?: string;
};

type Relationship = {
  retailer_id: string;
  credit_enabled: boolean;
  terms_days: number;
  grace_period_days: number;
  credit_limit_minor: number;
  use_global_defaults: boolean;
  profile_status?: string;
  current_balance_minor?: number;
  available_credit_minor?: number;
  on_hold?: boolean;
};

export default function CreditPolicyPage() {
  const t = usePortalT();
  const [program, setProgram] = useState<Program | null>(null);
  const [rels, setRels] = useState<Relationship[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modal, setModal] = useState<"program" | "retailer" | null>(null);
  const [retailerId, setRetailerId] = useState("");
  const [limitMinor, setLimitMinor] = useState("1000000");
  const [termsDays, setTermsDays] = useState("30");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [pRes, rRes] = await Promise.all([
        supplierFetch("/v1/supplier/credit-program"),
        supplierFetch("/v1/supplier/credit-relationships"),
      ]);
      if (!pRes.ok) throw new Error(`program_${pRes.status}`);
      if (!rRes.ok) throw new Error(`rels_${rRes.status}`);
      setProgram((await pRes.json()) as Program);
      const body = (await rRes.json()) as { relationships?: Relationship[] };
      setRels(body.relationships ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("supplier_portal.residual.text.load_failed"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function enableProgram(ackAt: string) {
    setBusy(true);
    setError(null);
    try {
      const res = await supplierFetch("/v1/supplier/credit-program", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          warning_ack: true,
          warning_ack_at: ackAt,
          global_terms_days: Number(termsDays) || 30,
          global_default_limit_minor: Number(limitMinor) || 0,
        }),
      });
      if (!res.ok) throw new Error(`enable_program_${res.status}`);
      setModal(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("supplier_portal.residual.text.enable_failed"));
    } finally {
      setBusy(false);
    }
  }

  async function enableRetailer(ackAt: string) {
    const rid = retailerId.trim();
    if (!rid) {
      setError(t("supplier_portal.residual.text.retailer_id_required"));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await supplierFetch(`/v1/supplier/credit-relationships/${encodeURIComponent(rid)}/enable`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          warning_ack: true,
          warning_ack_at: ackAt,
          terms_days: Number(termsDays) || 30,
          credit_limit_minor: Number(limitMinor) || 0,
          use_global_defaults: false,
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `enable_rel_${res.status}`);
      }
      setModal(null);
      setRetailerId("");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("supplier_portal.residual.text.enable_failed"));
    } finally {
      setBusy(false);
    }
  }

  async function patchDefaults() {
    setBusy(true);
    try {
      const res = await supplierFetch("/v1/supplier/credit-program/defaults", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          global_terms_days: Number(termsDays) || 30,
          global_default_limit_minor: Number(limitMinor) || 0,
        }),
      });
      if (!res.ok) throw new Error(`defaults_${res.status}`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("supplier_portal.residual.text.defaults_failed"));
    } finally {
      setBusy(false);
    }
  }

  async function hold(rid: string, unhold: boolean) {
    setBusy(true);
    try {
      const path = unhold ? "unhold" : "hold";
      const res = await supplierFetch(
        `/v1/supplier/credit-relationships/${encodeURIComponent(rid)}/${path}`,
        { method: "POST" },
      );
      if (!res.ok) throw new Error(`${path}_${res.status}`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("supplier_portal.residual.text.hold_failed"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageChrome
      title={t("portal.nav.credit_policy")}
      description={t("supplier_portal.residual.text.enable_the_supplier_credit_program_and_per_retailer_net_terms_di")}
      loading={loading}
    >
      {error ? <p className="mb-3 text-sm text-red-600">{error}</p> : null}

      <section className="mb-8">
        <h2 className="text-base font-semibold mb-2">{t("supplier_portal.credit.admin_disable.text.program")}</h2>
        <p className="text-sm text-[var(--muted)] mb-3">
          Status:{" "}
          <strong>{program?.program_enabled ? "ON" : "OFF"}</strong>
          {program?.program_enabled
            ? ` · Net ${program.global_terms_days} · default limit ${program.global_default_limit_minor}`
            : ""}
        </p>
        {!program?.program_enabled ? (
          <button
            type="button"
            className="rounded-lg bg-[var(--color-md-primary)] text-white px-3 py-1.5 text-sm"
            onClick={() => setModal("program")}
          >
            Enable credit program
          </button>
        ) : (
          <div className="flex flex-wrap gap-2 items-end">
            <label className="text-sm">
              Terms days
              <input
                className="ml-2 border rounded px-2 py-1 w-20"
                value={termsDays}
                onChange={(e) => setTermsDays(e.target.value)}
              />
            </label>
            <label className="text-sm">
              Default limit (minor)
              <input
                className="ml-2 border rounded px-2 py-1 w-32"
                value={limitMinor}
                onChange={(e) => setLimitMinor(e.target.value)}
              />
            </label>
            <button
              type="button"
              className="rounded-lg border px-3 py-1.5 text-sm"
              onClick={() => void patchDefaults()}
              disabled={busy}
            >
              Save defaults
            </button>
          </div>
        )}
      </section>

      <section>
        <h2 className="text-base font-semibold mb-2">{t("supplier_portal.credit.policy.text.retailer_relationships")}</h2>
        <div className="mb-4 flex flex-wrap gap-2 items-end">
          <label className="text-sm">
            Retailer ID
            <input
              className="ml-2 border rounded px-2 py-1 w-56"
              value={retailerId}
              onChange={(e) => setRetailerId(e.target.value)}
              placeholder={t("supplier_portal.credit.policy.text.ret")}
            />
          </label>
          <button
            type="button"
            className="rounded-lg bg-[var(--color-md-primary)] text-white px-3 py-1.5 text-sm disabled:opacity-40"
            disabled={!program?.program_enabled}
            onClick={() => setModal("retailer")}
          >
            Enable retailer credit
          </button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left border-b border-[var(--border)]">
                <th className="py-2 pr-3">{t("supplier_portal.analytics.demand.flywheel.text.retailer")}</th>
                <th className="py-2 pr-3">{t("supplier_portal.credit.policy.text.terms")}</th>
                <th className="py-2 pr-3">{t("supplier_portal.credit.collections.text.limit")}</th>
                <th className="py-2 pr-3">{t("supplier_portal.credit.collections.text.balance")}</th>
                <th className="py-2 pr-3">{t("supplier_portal.compliance.text.status")}</th>
                <th className="py-2">{t("supplier_portal.catalog.components.catalog_table.text.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {rels.map((r) => (
                <tr key={r.retailer_id} className="border-b border-[var(--border)]/60">
                  <td className="py-2 pr-3 font-mono text-xs">{r.retailer_id}</td>
                  <td className="py-2 pr-3">Net {r.terms_days}</td>
                  <td className="py-2 pr-3">{r.credit_limit_minor}</td>
                  <td className="py-2 pr-3">{r.current_balance_minor ?? 0}</td>
                  <td className="py-2 pr-3">{r.profile_status ?? (r.credit_enabled ? "ON" : "OFF")}</td>
                  <td className="py-2 flex gap-2">
                    {r.profile_status === "FROZEN" ? (
                      <button type="button" className="underline text-xs" onClick={() => void hold(r.retailer_id, true)}>
                        Unhold
                      </button>
                    ) : (
                      <button type="button" className="underline text-xs" onClick={() => void hold(r.retailer_id, false)}>
                        Hold
                      </button>
                    )}
                    <span className="text-xs text-[var(--muted)]" title={t("supplier_portal.credit.policy.text.self_serve_disable_blocked")}>
                      Disable → support
                    </span>
                  </td>
                </tr>
              ))}
              {rels.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-4 text-[var(--muted)]">
                    No credit-enabled retailers yet.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>

      <CreditEnableModal
        open={modal === "program"}
        busy={busy}
        onCancel={() => setModal(null)}
        onConfirm={(ackAt) => void enableProgram(ackAt)}
      />
      <CreditEnableModal
        open={modal === "retailer"}
        busy={busy}
        confirmToken="ENABLE"
        onCancel={() => setModal(null)}
        onConfirm={(ackAt) => void enableRetailer(ackAt)}
      />
    </PageChrome>
  );
}
