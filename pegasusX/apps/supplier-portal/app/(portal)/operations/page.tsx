"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { ApiError, supplierBroadcastKey, supplierPaymentBypassKey } from "@pegasusx/api-client";
import { SUPPLIER_BROADCAST_TEMPLATES } from "@pegasusx/types";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageSection } from "@/components/PageSection";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, readTokenFromCookie } from "@/lib/auth";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import ReplenishmentTraceabilityPanel from "@/components/ReplenishmentTraceabilityPanel";
import { errorToMessage } from "../../payments/_shared/finance";

const api = createSupplierApi();
const broadcastRoles = ["ALL", "DRIVER", "RETAILER", "PAYLOAD"] as const;

function supplierScopeId(): string {
  const token = readTokenFromCookie();
  if (!token) return "supplier";
  const claims = decodeJwtPayload(token);
  return typeof claims?.supplier_id === "string" ? claims.supplier_id : "supplier";
}

export default function OperationsPage() {
  const [empathy, setEmpathy] = useState<Awaited<ReturnType<typeof api.getSupplierEmpathyAdoption>> | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [broadcastRole, setBroadcastRole] = useState<(typeof broadcastRoles)[number]>("ALL");
  const [templateDate, setTemplateDate] = useState("");
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

  useSupplierSessionReconcile(() => {
    if (broadcasting || replenishing || bypassing) {
      setBroadcasting(false);
      setReplenishing(false);
      setBypassing(false);
      setError("Connection restored — verify status before retrying.");
    }
  });

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
      const trimmedTitle = title.trim();
      const trimmedBody = body.trim();
      const resp = await api.postSupplierBroadcast(
        { title: trimmedTitle, body: trimmedBody, role: broadcastRole },
        supplierBroadcastKey(supplierScopeId(), broadcastRole, trimmedTitle, trimmedBody),
      );
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
      const resp = await api.issueSupplierPaymentBypass(
        {
          order_id: trimmed,
          reason: bypassReason.trim() || undefined,
        },
        supplierPaymentBypassKey(trimmed, bypassReason.trim()),
      );
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
    <PageChrome
      icon="operations"
      title="Operations"
      description="Empathy adoption, operator broadcast, replenishment trigger, and payment bypass."
      loading={loading}
      skeletonVariant="form"
      error={error && !empathy ? error : null}
    >
      {empathy ? (
        <KpiStatGrid columns={3}>
          <KpiStatCard label="Predictions" value={empathy.total_predictions} />
          <KpiStatCard label="Waiting" value={empathy.predictions_waiting} />
          <KpiStatCard label="Fired" value={empathy.predictions_fired} />
          <KpiStatCard label="Dormant" value={empathy.predictions_dormant} />
          <KpiStatCard label="Rejected" value={empathy.predictions_rejected} />
        </KpiStatGrid>
      ) : null}

      <PageSection title="Operator broadcast" description="Fan out a message to supplier WS rooms by role.">
        <p className="md-typescale-body-small mb-3 text-[var(--color-md-outline)]">
          Signal ingest health and planning projections live on{" "}
          <Link href={"/settings/planning" as any} className="underline text-[var(--color-md-primary)]">
            Planning settings
          </Link>
          .
        </p>
        <div className="flex flex-wrap gap-2 mb-4">
          {SUPPLIER_BROADCAST_TEMPLATES.map((template) => (
            <button
              key={template.id}
              type="button"
              className="md-btn md-btn-tonal text-xs px-3 py-1.5"
              onClick={() => {
                setTitle(template.title);
                setBody(
                  template.body.replace(
                    "{date}",
                    templateDate.trim() || "the selected date",
                  ),
                );
                setBroadcastRole(
                  broadcastRoles.includes(template.default_role as (typeof broadcastRoles)[number])
                    ? (template.default_role as (typeof broadcastRoles)[number])
                    : "ALL",
                );
              }}
            >
              {template.title}
            </button>
          ))}
        </div>
        <label className="block space-y-1 mb-3 max-w-xs">
          <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
            Template date (optional)
          </span>
          <input
            type="date"
            className="md-input-outlined w-full"
            value={templateDate}
            onChange={(e) => setTemplateDate(e.target.value)}
          />
        </label>
        <div className="space-y-3">
          <label className="block space-y-1">
            <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Title
            </span>
            <input
              className="md-input-outlined w-full"
              placeholder="Title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </label>
          <label className="block space-y-1">
            <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Message
            </span>
            <textarea
              className="md-input-outlined w-full min-h-24"
              placeholder="Message body"
              value={body}
              onChange={(e) => setBody(e.target.value)}
            />
          </label>
          <label className="block space-y-1">
            <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Target role
            </span>
            <select
              className="md-input-outlined w-full"
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
        </div>
      </PageSection>

      <PageSection
        title="Replenishment"
        description="Opens a warehouse supply request against your primary active warehouse."
      >
        <button type="button" className="md-btn md-btn-tonal" onClick={onReplenishment} disabled={replenishing}>
          {replenishing ? "Triggering…" : "Trigger replenishment"}
        </button>
      </PageSection>

      <PageSection title="Payment bypass" description="Issue a one-time driver token for AWAITING_PAYMENT orders.">
        <div className="space-y-3">
          <label className="block space-y-1">
            <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Order ID
            </span>
            <input
              className="md-input-outlined w-full font-mono"
              placeholder="Order ID (AWAITING_PAYMENT)"
              value={orderId}
              onChange={(e) => setOrderId(e.target.value)}
            />
          </label>
          <label className="block space-y-1">
            <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Reason (optional)
            </span>
            <input
              className="md-input-outlined w-full"
              placeholder="Reason"
              value={bypassReason}
              onChange={(e) => setBypassReason(e.target.value)}
            />
          </label>
          {confirmBypass ? (
            <div
              className="space-y-3 p-4 rounded-xl"
              style={{ border: "1px solid var(--desk-border)", background: "var(--desk-surface-raised)" }}
            >
              <p className="md-typescale-body-medium">
                Issue bypass for <span className="font-mono">{orderId.trim()}</span>? Order must be AWAITING_PAYMENT.
              </p>
              <div className="flex flex-wrap gap-2">
                <button type="button" className="md-btn md-btn-filled" onClick={onPaymentBypass} disabled={bypassing}>
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
        </div>
      </PageSection>

      {message ? (
        <p className="md-typescale-body-medium" style={{ color: "var(--desk-success)" }}>
          {message}
        </p>
      ) : null}
      {error && empathy ? (
        <p className="md-typescale-body-medium" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      ) : null}

      <ReplenishmentTraceabilityPanel />
    </PageChrome>
  );
}
