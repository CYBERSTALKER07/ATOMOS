"use client";

import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { DemandSignal, CreateSignalRequest } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function DemandSignalsPage() {
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
      .catch((err) => setError(err instanceof Error ? err.message : "load_signals_failed"))
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
      title="Demand Signals"
      description="External events and promotions that adjust base AI predictions."
      loading={loading}
      error={error}
      empty={signals.length === 0 && !showForm}
      emptyMessage="No demand signals yet. Add events or promos to adjust predictive inventory."
      actions={
        <button type="button" className="md-btn md-btn-filled md-typescale-label-large px-4 py-2" onClick={() => setShowForm(true)}>
          Add Signal
        </button>
      }
    >
      {showForm && (
        <form onSubmit={addSignal} className="desk-card p-6 mt-6 max-w-2xl">
          <h2 className="bento-card-title mb-4">New Demand Signal</h2>
          <div className="grid gap-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">Type</label>
                <select className="desk-input" value={type} onChange={(e) => setType(e.target.value)} required>
                  <option value="HOLIDAY">Holiday</option>
                  <option value="WEATHER">Weather</option>
                  <option value="EVENT">Event</option>
                  <option value="PROMO">Promotion</option>
                </select>
              </div>
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">Scope</label>
                <select className="desk-input" value={scope} onChange={(e) => setScope(e.target.value)} required>
                  <option value="GLOBAL">Global</option>
                  <option value="REGION">Region</option>
                  <option value="CITY">City</option>
                  <option value="RETAILER">Retailer</option>
                  <option value="RETAILER_SKU">Retailer SKU</option>
                </select>
              </div>
            </div>

            {(scope === "RETAILER" || scope === "RETAILER_SKU") && (
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">Retailer ID</label>
                <input className="desk-input" value={retailerId} onChange={(e) => setRetailerId(e.target.value)} required />
              </div>
            )}

            {scope === "RETAILER_SKU" && (
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">Product ID</label>
                <input className="desk-input" value={productId} onChange={(e) => setProductId(e.target.value)} required />
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">Start Date</label>
                <input type="date" className="desk-input" value={startDate} onChange={(e) => setStartDate(e.target.value)} required />
              </div>
              <div className="flex flex-col gap-1">
                <label className="md-typescale-label-small">End Date</label>
                <input type="date" className="desk-input" value={endDate} onChange={(e) => setEndDate(e.target.value)} required />
              </div>
            </div>

            <div className="flex flex-col gap-1">
              <label className="md-typescale-label-small">Multiplier (e.g. 1.2 for 20% increase)</label>
              <input type="number" step="0.01" min="0.1" className="desk-input" value={multiplier} onChange={(e) => setMultiplier(parseFloat(e.target.value))} required />
            </div>

            <div className="flex flex-col gap-1">
              <label className="md-typescale-label-small">Description</label>
              <input className="desk-input" value={description} onChange={(e) => setDescription(e.target.value)} />
            </div>
          </div>
          <div className="flex items-center gap-4 mt-6">
            <button type="submit" className="md-btn md-btn-filled px-6 py-2">Save</button>
            <button type="button" className="md-btn md-btn-text" onClick={() => setShowForm(false)}>Cancel</button>
          </div>
        </form>
      )}

      {!showForm && signals.length > 0 && (
        <section className="desk-card p-6 mt-6 overflow-x-auto">
          <table className="desk-table w-full">
            <thead>
              <tr style={{ color: "var(--desk-text-secondary)" }}>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Type</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Scope</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Date Range</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Multiplier</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Description</th>
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
