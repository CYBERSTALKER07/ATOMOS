"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ApiError } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import { PortalSurface } from "../_components/PortalSurface";
import { errorToMessage, formatMinor, loadFinanceAuthoritySnapshot } from "../../payments/_shared/finance";

const api = createSupplierApi();
const broadcastRoles = ["ALL", "DRIVER", "RETAILER", "PAYLOAD"] as const;

export default function OperationsPage() {
  const [empathy, setEmpathy] = useState<Awaited<ReturnType<typeof api.getSupplierEmpathyAdoption>> | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [broadcastRole, setBroadcastRole] = useState<(typeof broadcastRoles)[number]>("ALL");
  const [orderId, setOrderId] = useState("");
  const [bypassReason, setBypassReason] = useState("");
  const [bypassToken, setBypassToken] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [broadcasting, setBroadcasting] = useState(false);
  const [replenishing, setReplenishing] = useState(false);
  const [bypassing, setBypassing] = useState(false);
  const [confirmBypass, setConfirmBypass] = useState(false);

  useEffect(() => {
    api
      .getSupplierEmpathyAdoption()
      .then(setEmpathy)
      .catch((err) => setError(errorToMessage(err)))
      .finally(() => setLoading(false));
  }, []);

  const onBroadcast = async () => {
    if (!title.trim() || !body.trim()) {
      setError("title_and_body_required");
      return;
    }
    setMessage(null);
    setError(null);
    setBroadcasting(true);
    try {
      const resp = await api.postSupplierBroadcast({ title: title.trim(), body: body.trim(), role: broadcastRole });
      setMessage(`Broadcast sent to supplier room (${resp.supplier_id}).`);
      setTitle("");
      setBody("");
    } catch (err) {
      setError(errorToMessage(err));
    } finally {
      setBroadcasting(false);
    }
  };

  const onReplenishment = async () => {
    setMessage(null);
    setError(null);
    setReplenishing(true);
    try {
      const resp = await api.triggerSupplierReplenishment();
      setMessage(`Replenishment request ${resp.request_id} opened for warehouse ${resp.warehouse_id}.`);
    } catch (err) {
      setError(errorToMessage(err));
    } finally {
      setReplenishing(false);
    }
  };

  const onPaymentBypass = async () => {
    const trimmed = orderId.trim();
    if (!trimmed) {
      setError("order_id required");
      return;
    }
    setMessage(null);
    setError(null);
    setBypassToken(null);
    setBypassing(true);
    try {
      const resp = await api.issueSupplierPaymentBypass({
        order_id: trimmed,
        reason: bypassReason.trim() || undefined,
      });
      setBypassToken(resp.bypass_token);
      setMessage(`Bypass token issued for ${resp.order_id}.`);
      setConfirmBypass(false);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : errorToMessage(err));
    } finally {
      setBypassing(false);
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
        <label className="block md-typescale-label-medium text-[var(--color-md-outline)]">
          Target role
          <select
            className="md-input-outlined w-full mt-1"
            value={broadcastRole}
            onChange={(e) => setBroadcastRole(e.target.value as (typeof broadcastRoles)[number])}
          >
            {broadcastRoles.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          className="md-btn md-btn-filled"
          onClick={onBroadcast}
          disabled={broadcasting || !title.trim() || !body.trim()}
        >
          {broadcasting ? "Sending…" : "Send broadcast"}
        </button>
      </section>

      <section className="md-card p-4 space-y-3">
        <h2 className="md-typescale-title-medium">Replenishment</h2>
        <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
          Opens a warehouse supply request against your primary active warehouse.
        </p>
        <button
          type="button"
          className="md-btn md-btn-tonal"
          onClick={onReplenishment}
          disabled={replenishing}
        >
          {replenishing ? "Triggering…" : "Trigger replenishment"}
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
        <input
          className="md-input-outlined w-full"
          placeholder="Reason (optional)"
          value={bypassReason}
          onChange={(e) => setBypassReason(e.target.value)}
        />
        {confirmBypass ? (
          <div className="space-y-2 border border-[var(--color-md-outline-variant)] p-3">
            <p className="md-typescale-body-medium">
              Issue bypass for <span className="font-mono">{orderId.trim()}</span>? Order must be AWAITING_PAYMENT.
            </p>
            <div className="flex gap-2">
              <button
                type="button"
                className="md-btn md-btn-filled"
                onClick={onPaymentBypass}
                disabled={bypassing}
              >
                {bypassing ? "Issuing…" : "Confirm issue"}
              </button>
              <button
                type="button"
                className="md-btn md-btn-outlined"
                onClick={() => setConfirmBypass(false)}
                disabled={bypassing}
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <button
            type="button"
            className="md-btn md-btn-outlined"
            onClick={() => setConfirmBypass(true)}
            disabled={!orderId.trim()}
          >
            Issue bypass token
          </button>
        )}
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
