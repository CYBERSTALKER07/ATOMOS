"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { ApiError, supplierBroadcastKey, supplierPaymentBypassKey } from "@pegasusx/api-client";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { createSupplierApi } from "@/lib/api";
import { OperatorBroadcast, broadcastRoles, ReplenishmentAction, PaymentBypass } from "@/components/operations";
import { decodeJwtPayload, readTokenFromCookie } from "@/lib/auth";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import ReplenishmentTraceabilityPanel from "@/components/ReplenishmentTraceabilityPanel";
import { errorToMessage } from "../../payments/_shared/finance";

const api = createSupplierApi();

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

      <OperatorBroadcast
        title={title}
        body={body}
        broadcastRole={broadcastRole}
        templateDate={templateDate}
        broadcasting={broadcasting}
        onTitleChange={setTitle}
        onBodyChange={setBody}
        onBroadcastRoleChange={setBroadcastRole}
        onTemplateDateChange={setTemplateDate}
        onBroadcast={() => void onBroadcast()}
      />

      <ReplenishmentAction
        replenishing={replenishing}
        onReplenishment={() => void onReplenishment()}
      />

      <PaymentBypass
        orderId={orderId}
        bypassReason={bypassReason}
        bypassToken={bypassToken}
        confirmBypass={confirmBypass}
        bypassing={bypassing}
        onOrderIdChange={setOrderId}
        onBypassReasonChange={setBypassReason}
        onConfirmBypassChange={setConfirmBypass}
        onPaymentBypass={() => void onPaymentBypass()}
      />

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
