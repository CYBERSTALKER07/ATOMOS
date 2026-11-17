"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
import { CreditEnableModal } from "@/components/CreditEnableModal";
import { ObjectHistoryPanel } from "@/components/audit/ObjectHistoryPanel";
import { minorToWholeTrunc } from "@/lib/money";
import { formatWholeSom, wholeSomToMinor } from "@/lib/exception-desk";

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
  retailer_name?: string;
  display_name?: string;
  tin?: string;
  phone?: string;
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
  const [limitSom, setLimitSom] = useState("1000000");
  const [termsDays, setTermsDays] = useState("30");
  const [busy, setBusy] = useState(false);
  const [historyTarget, setHistoryTarget] = useState<{
    objectType: string;
    fallbackObjectTypes?: string[];
    objectId: string;
  } | null>(null);

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
      const loadedProgram = (await pRes.json()) as Program;
      setProgram(loadedProgram);
      if (loadedProgram.global_default_limit_minor > 0) {
        setLimitSom(String(minorToWholeTrunc(loadedProgram.global_default_limit_minor)));
      }
      if (loadedProgram.global_terms_days > 0) {
        setTermsDays(String(loadedProgram.global_terms_days));
      }
      const body = (await rRes.json()) as { relationships?: Relationship[] };
      setRels(body.relationships ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("supplier_portal.residual.text.load_failed"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function enableProgram(ackAt: string) {
    const defaultLimitMinor = wholeSomToMinor(limitSom);
    if (defaultLimitMinor === null) {
      setError(t("supplier_portal.residual.text.invalid_limit"));
      return;
    }
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
          global_default_limit_minor: defaultLimitMinor,
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
    const creditLimitMinor = wholeSomToMinor(limitSom);
    if (creditLimitMinor === null) {
      setError(t("supplier_portal.residual.text.invalid_limit"));
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
          credit_limit_minor: creditLimitMinor,
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
    const defaultLimitMinor = wholeSomToMinor(limitSom);
    if (defaultLimitMinor === null) {
      setError(t("supplier_portal.residual.text.invalid_limit"));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await supplierFetch("/v1/supplier/credit-program/defaults", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          global_terms_days: Number(termsDays) || 30,
          global_default_limit_minor: defaultLimitMinor,
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
        <div className="flex flex-wrap items-center justify-between gap-2 mb-2">
          <h2 className="text-base font-semibold">{t("supplier_portal.credit.admin_disable.text.program")}</h2>
          {program?.supplier_id ? (
            <button
              type="button"
              className="text-xs underline text-[var(--muted)]"
              onClick={() =>
                setHistoryTarget((curr) =>
                  curr?.objectId === program.supplier_id && curr.objectType === "credit_program"
                    ? null
                    : {
                        objectType: "credit_program",
                        fallbackObjectTypes: ["credit_policy"],
                        objectId: program.supplier_id,
                      },
                )
              }
            >
              {t("supplier_portal.audit.open_history")}
            </button>
          ) : null}
        </div>
        <p className="text-sm text-[var(--muted)] mb-3">
          Status:{" "}
          <strong>{program?.program_enabled ? "ON" : "OFF"}</strong>
          {program?.program_enabled
            ? ` · Net ${program.global_terms_days} · default limit ${formatWholeSom(program.global_default_limit_minor)}`
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
                id="terms-days-input-3"
                aria-label="Terms days"
                className="ml-2 border rounded px-2 py-1 w-20"
                value={termsDays}
                onChange={(e) => setTermsDays(e.target.value)}
              />
            </label>
            <label className="text-sm">
              Default limit (so&apos;m)
              <input
                id="default-limit-som-input"
                aria-label="Default limit (so'm)"
                className="ml-2 border rounded px-2 py-1 w-36 font-mono"
                value={limitSom}
                onChange={(e) => setLimitSom(e.target.value)}
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
              id="retailer-id-input-1"
              aria-label="Retailer ID"
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
                <th className="py-2 pr-3">Limit (so&apos;m)</th>
                <th className="py-2 pr-3">Balance (so&apos;m)</th>
                <th className="py-2 pr-3">{t("supplier_portal.compliance.text.status")}</th>
                <th className="py-2">{t("supplier_portal.catalog.components.catalog_table.text.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {rels.map((r) => {
                const shopName = (r.display_name || r.retailer_name || "").trim();
                return (
                  <tr key={r.retailer_id} className="border-b border-[var(--border)]/60">
                    <td className="py-2 pr-3">
                      <div className="font-medium">{shopName || r.retailer_id}</div>
                      {shopName || r.tin || r.phone ? (
                        <div className="text-xs text-[var(--muted)] font-mono flex flex-wrap gap-x-2">
                          {r.tin ? <span>TIN {r.tin}</span> : null}
                          {r.phone ? <span>{r.phone}</span> : null}
                          {shopName ? <span>{r.retailer_id}</span> : null}
                        </div>
                      ) : null}
                    </td>
                    <td className="py-2 pr-3">Net {r.terms_days}</td>
                    <td className="py-2 pr-3 font-mono">{formatWholeSom(r.credit_limit_minor)}</td>
                    <td className="py-2 pr-3 font-mono">{formatWholeSom(r.current_balance_minor ?? 0)}</td>
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
                      <button
                        type="button"
                        className="underline text-xs text-[var(--muted)]"
                        onClick={() =>
                          setHistoryTarget((curr) =>
                            curr?.objectId === r.retailer_id && curr.objectType === "credit_relationship"
                              ? null
                              : {
                                  objectType: "credit_relationship",
                                  fallbackObjectTypes: ["credit_relationships", "retailer_credit_profile"],
                                  objectId: r.retailer_id,
                                },
                          )
                        }
                      >
                        {t("supplier_portal.audit.open_history")}
                      </button>
                    </td>
                  </tr>
                );
              })}
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

      {historyTarget ? (
        <div className="mt-6">
          <ObjectHistoryPanel
            objectType={historyTarget.objectType}
            fallbackObjectTypes={historyTarget.fallbackObjectTypes}
            objectId={historyTarget.objectId}
            compact
            onClose={() => setHistoryTarget(null)}
          />
        </div>
      ) : null}

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
