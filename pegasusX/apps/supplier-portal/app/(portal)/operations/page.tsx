"use client";

import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierEmpathyAdoption } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function OperationsPage() {
  const [empathy, setEmpathy] = useState<SupplierEmpathyAdoption | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [orderId, setOrderId] = useState("");
  const [bypassToken, setBypassToken] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .getSupplierEmpathyAdoption()
      .then(setEmpathy)
      .catch((err) => setError(err instanceof Error ? err.message : "empathy_load_failed"))
      .finally(() => setLoading(false));
  }, []);

  const onBroadcast = async () => {
    setMessage(null);
    setError(null);
    try {
      const resp = await api.postSupplierBroadcast({ title, body, role: "ALL" });
      setMessage(`Broadcast sent to supplier room (${resp.supplier_id}).`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "broadcast_failed");
    }
  };

  const onReplenishment = async () => {
    setMessage(null);
    setError(null);
    try {
      const resp = await api.triggerSupplierReplenishment();
      setMessage(`Replenishment request ${resp.request_id} opened for warehouse ${resp.warehouse_id}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "replenishment_failed");
    }
  };

  const onPaymentBypass = async () => {
    setMessage(null);
    setError(null);
    setBypassToken(null);
    try {
      const resp = await api.issueSupplierPaymentBypass({ order_id: orderId.trim() });
      setBypassToken(resp.bypass_token);
      setMessage(`Bypass token issued for ${resp.order_id}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "payment_bypass_failed");
    }
  };

  return (
    <PortalSurface
      title="Operations"
      description="Empathy adoption, operator broadcast, replenishment trigger, and payment bypass."
      loading={loading}
      error={error}
    >
      {empathy ? (
        <div className="md-card p-4 grid grid-cols-2 md:grid-cols-3 gap-4">
          <div>
            <p className="md-typescale-label-small text-[var(--color-md-outline)]">Predictions</p>
            <p className="md-kpi-value">{empathy.total_predictions}</p>
          </div>
          <div>
            <p className="md-typescale-label-small text-[var(--color-md-outline)]">Waiting</p>
            <p className="md-kpi-value">{empathy.predictions_waiting}</p>
          </div>
          <div>
            <p className="md-typescale-label-small text-[var(--color-md-outline)]">Fired</p>
            <p className="md-kpi-value">{empathy.predictions_fired}</p>
          </div>
          <div>
            <p className="md-typescale-label-small text-[var(--color-md-outline)]">Dormant</p>
            <p className="md-kpi-value">{empathy.predictions_dormant}</p>
          </div>
          <div>
            <p className="md-typescale-label-small text-[var(--color-md-outline)]">Rejected</p>
            <p className="md-kpi-value">{empathy.predictions_rejected}</p>
          </div>
        </div>
      ) : null}

      <section className="md-card p-4 space-y-3">
        <h2 className="md-typescale-title-medium">Operator broadcast</h2>
        <input
          className="md-input-outlined w-full"
          placeholder="Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <textarea
          className="md-input-outlined w-full min-h-24"
          placeholder="Message body"
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
        <button type="button" className="md-btn md-btn-filled" onClick={onBroadcast}>
          Send broadcast
        </button>
      </section>

      <section className="md-card p-4 space-y-3">
        <h2 className="md-typescale-title-medium">Replenishment</h2>
        <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
          Opens a warehouse supply request against your primary active warehouse.
        </p>
        <button type="button" className="md-btn md-btn-tonal" onClick={onReplenishment}>
          Trigger replenishment
        </button>
      </section>

      <section className="md-card p-4 space-y-3">
        <h2 className="md-typescale-title-medium">Payment bypass</h2>
        <input
          className="md-input-outlined w-full"
          placeholder="Order ID (AWAITING_PAYMENT)"
          value={orderId}
          onChange={(e) => setOrderId(e.target.value)}
        />
        <button type="button" className="md-btn md-btn-outlined" onClick={onPaymentBypass}>
          Issue bypass token
        </button>
        {bypassToken ? (
          <p className="md-typescale-body-medium">
            Driver token: <span className="font-mono">{bypassToken}</span>
          </p>
        ) : null}
      </section>

      {message ? <p className="md-typescale-body-medium text-[var(--color-md-success)]">{message}</p> : null}
    </PortalSurface>
  );
}
