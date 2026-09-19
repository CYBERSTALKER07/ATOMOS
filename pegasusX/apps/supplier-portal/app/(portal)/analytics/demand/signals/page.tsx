"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { DemandSignal, CreateSignalRequest } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function DemandSignalsPage() {
  const t = usePortalT();
  const [signals, setSignals] = useState<DemandSignal[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);

  // Form State
  const [type, setType] = useState("EVENT");
  const [scope, setScope] = useState("GLOBAL");
  const [retailerId, setRetailerId] = useState("");
  const [productId, setProductId] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [multiplier, setMultiplier] = useState(1.0);
  const [description, setDescription] = useState("");

  const load = () => {
    setLoading(true);
    setError(null);
    api
      .getDemandSignals()
      .then((res) => {
        setSignals(res.signals || []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_signals_failed")))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, [refreshTick]);

  const addSignal = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload: CreateSignalRequest = {
        type,
        scope,
        retailerId: retailerId || undefined,
        productId: productId || undefined,
        startDate,
        endDate,
        multiplier,
        description,
      };
      await api.createDemandSignal(payload);
      setShowForm(false);
      // Reset form
      setType("EVENT");
      setScope("GLOBAL");
      setRetailerId("");
      setProductId("");
      setStartDate("");
      setEndDate("");
      setMultiplier(1.0);
      setDescription("");
      load();
    } catch (err: any) {
      alert(err.message || "Failed to create signal");
    }
  };

  return (
    <PageChrome
      icon="campaign"
      title={t("portal.nav.demand_signals")}
      description={t("supplier_portal.residual.text.external_events_and_promotions_that_adjust_base_ai_predictions")}
      loading={loading}
      error={error}
      empty={signals.length === 0 && !showForm}
      emptyMessage={t("supplier_portal.residual.text.no_demand_signals_yet_add_events_or_promos_to_adjust_predictive_")}
      actions={
        <button type="button" className="md-btn md-btn-filled md-typescale-label-large px-4 py-2" onClick={() => setShowForm(true)}>
          Add Signal
        </button>
      }
    >
      {showForm && (
        <form onSubmit={addSignal} className="desk-card p-6 mt-6 max-w-2xl">
          <h2 className="bento-card-title mb-4">{t("supplier_portal.analytics.demand.signals.text.new_demand_signal")}</h2>
          <div className="grid gap-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">{t("supplier_portal.ledger.text.type")}</label>
                <select className="desk-input" value={type} onChange={(e) => setType(e.target.value)} required>
                  <option value="HOLIDAY">{t("supplier_portal.analytics.demand.signals.text.holiday")}</option>
                  <option value="WEATHER">{t("supplier_portal.analytics.demand.signals.text.weather")}</option>
                  <option value="EVENT">{t("supplier_portal.analytics.demand.signals.text.event")}</option>
                  <option value="PROMO">{t("supplier_portal.analytics.demand.signals.text.promotion")}</option>
                </select>
              </div>
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">{t("supplier_portal.analytics.demand.signals.text.scope")}</label>
                <select className="desk-input" value={scope} onChange={(e) => setScope(e.target.value)} required>
                  <option value="GLOBAL">{t("supplier_portal.admin.empathy.hierarchy.global.level")}</option>
                  <option value="REGION">{t("supplier_portal.analytics.demand.signals.text.region")}</option>
                  <option value="CITY">{t("supplier_portal.analytics.demand.signals.text.city")}</option>
                  <option value="RETAILER">{t("supplier_portal.analytics.demand.flywheel.text.retailer")}</option>
                  <option value="RETAILER_SKU">{t("supplier_portal.analytics.demand.signals.text.retailer_sku")}</option>
                </select>
              </div>
            </div>

            {(scope === "RETAILER" || scope === "RETAILER_SKU") && (
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">{t("supplier_portal.chargebacks.text.retailer_id")}</label>
                <input className="desk-input" value={retailerId} onChange={(e) => setRetailerId(e.target.value)} required />
              </div>
            )}

            {scope === "RETAILER_SKU" && (
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">{t("supplier_portal.analytics.demand.signals.text.product_id")}</label>
                <input className="desk-input" value={productId} onChange={(e) => setProductId(e.target.value)} required />
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">{t("supplier_portal.analytics.demand.signals.text.start_date")}</label>
                <input type="date" className="desk-input" value={startDate} onChange={(e) => setStartDate(e.target.value)} required />
              </div>
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">{t("supplier_portal.analytics.demand.signals.text.end_date")}</label>
                <input type="date" className="desk-input" value={endDate} onChange={(e) => setEndDate(e.target.value)} required />
              </div>
            </div>

            <div className="flex flex-col gap-1">
              <label className="md-typescale-label-small">{t("supplier_portal.analytics.demand.signals.text.multiplier_e_g_1_2_for_20_increase")}</label>
              <input type="number" step="0.01" min="0.1" className="desk-input" value={multiplier} onChange={(e) => setMultiplier(parseFloat(e.target.value))} required />
            </div>

            <div className="flex flex-col gap-1">
              <label className="md-typescale-label-small">{t("supplier_portal.analytics.demand.signals.text.description")}</label>
              <input className="desk-input" value={description} onChange={(e) => setDescription(e.target.value)} />
            </div>
          </div>
          <div className="flex items-center gap-4 mt-6">
            <button type="submit" className="md-btn md-btn-filled px-6 py-2">{t("common.action.save")}</button>
            <button type="button" className="md-btn md-btn-text" onClick={() => setShowForm(false)}>{t("common.action.cancel")}</button>
          </div>
        </form>
      )}

      {!showForm && signals.length > 0 && (
        <section className="desk-card p-6 mt-6 overflow-x-auto">
          <table className="desk-table w-full">
            <thead>
              <tr style={{ color: "var(--desk-text-secondary)" }}>
                <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.ledger.text.type")}</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.analytics.demand.signals.text.scope")}</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.analytics.demand.signals.text.date_range")}</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.analytics.demand.signals.text.multiplier")}</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.analytics.demand.signals.text.description")}</th>
              </tr>
            </thead>
            <tbody>
              {signals.map((sig) => (
                <tr key={sig.signalId} style={{ borderTop: "1px solid var(--desk-border)" }}>
                  <td className="p-3 md-typescale-body-medium">{sig.type}</td>
                  <td className="p-3 md-typescale-body-medium">
                    {sig.scope}
                    {sig.retailerId && <div className="text-xs text-gray-500">{sig.retailerId}</div>}
                    {sig.productId && <div className="text-xs text-gray-500">{sig.productId}</div>}
                  </td>
                  <td className="p-3 md-typescale-body-medium font-mono text-sm">{sig.startDate} to {sig.endDate}</td>
                  <td className="p-3 md-typescale-body-medium">{sig.multiplier}x</td>
                  <td className="p-3 md-typescale-body-medium">{sig.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}
    </PageChrome>
  );
}
