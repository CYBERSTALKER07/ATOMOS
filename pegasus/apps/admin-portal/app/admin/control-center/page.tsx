"use client";

import { useMemo, useState } from "react";
import { apiFetch } from "@/lib/auth";
import { useToast } from "@/components/Toast";
import { useLocale } from "@/hooks/useLocale";

type BroadcastRole = "ALL" | "RETAILER" | "DRIVER";

function parseBroadcastData(raw: string, invalidJsonMessage: string): Record<string, string> {
  if (!raw.trim()) return {};

  const parsed = JSON.parse(raw) as unknown;
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(invalidJsonMessage);
  }

  const output: Record<string, string> = {};
  for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
    output[key] = typeof value === "string" ? value : JSON.stringify(value);
  }
  return output;
}

export default function ControlCenterPage() {
  const { toast } = useToast();
  const { t } = useLocale();

  const [orderId, setOrderId] = useState("");
  const [bypassReason, setBypassReason] = useState("");
  const [bypassToken, setBypassToken] = useState("");
  const [bypassLoading, setBypassLoading] = useState(false);

  const [sessionId, setSessionId] = useState("");
  const [reconcileResult, setReconcileResult] = useState<string>("");
  const [reconcileLoading, setReconcileLoading] = useState(false);

  const [broadcastTitle, setBroadcastTitle] = useState("");
  const [broadcastBody, setBroadcastBody] = useState("");
  const [broadcastRole, setBroadcastRole] = useState<BroadcastRole>("ALL");
  const [broadcastData, setBroadcastData] = useState('{"source":"admin-control-center"}');
  const [broadcastLoading, setBroadcastLoading] = useState(false);
  const [broadcastSummary, setBroadcastSummary] = useState("");

  const [replenishLoading, setReplenishLoading] = useState(false);
  const [replenishStatus, setReplenishStatus] = useState("");

  const canIssueBypass = useMemo(() => orderId.trim().length > 0, [orderId]);
  const canReconcile = useMemo(() => sessionId.trim().length > 0, [sessionId]);
  const canBroadcast = useMemo(
    () => broadcastTitle.trim().length > 0 && broadcastBody.trim().length > 0,
    [broadcastTitle, broadcastBody],
  );

  const issuePaymentBypass = async () => {
    if (!canIssueBypass) return;

    setBypassLoading(true);
    setBypassToken("");
    try {
      const response = await apiFetch("/v1/admin/orders/payment-bypass", {
        method: "POST",
        body: JSON.stringify({
          order_id: orderId.trim(),
          reason: bypassReason.trim(),
        }),
      });

      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(payload?.error || `HTTP ${response.status}`);
      }

      const token = String(payload?.bypass_token || "");
      setBypassToken(token);
      toast(t("supplier_portal.admin.control_center.toast.bypass_issued", { order_id: orderId.trim() }), "success");
    } catch (error) {
      const message = error instanceof Error ? error.message : t("supplier_portal.admin.control_center.error.bypass_failed");
      toast(message, "error");
    } finally {
      setBypassLoading(false);
    }
  };

  const reconcileSession = async () => {
    if (!canReconcile) return;

    setReconcileLoading(true);
    setReconcileResult("");
    try {
      const response = await apiFetch("/v1/admin/payment/reconcile", {
        method: "POST",
        body: JSON.stringify({ session_id: sessionId.trim() }),
      });

      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(payload?.error || `HTTP ${response.status}`);
      }

      setReconcileResult(JSON.stringify(payload, null, 2));
      toast(t("supplier_portal.admin.control_center.toast.reconcile_completed", { session_id: sessionId.trim() }), "success");
    } catch (error) {
      const message = error instanceof Error ? error.message : t("supplier_portal.admin.control_center.error.reconcile_failed");
      toast(message, "error");
    } finally {
      setReconcileLoading(false);
    }
  };

  const sendBroadcast = async () => {
    if (!canBroadcast) return;

    setBroadcastLoading(true);
    setBroadcastSummary("");
    try {
      const data = parseBroadcastData(
        broadcastData,
        t("supplier_portal.admin.control_center.error.broadcast_data_must_be_object"),
      );
      const response = await apiFetch("/v1/admin/broadcast", {
        method: "POST",
        body: JSON.stringify({
          title: broadcastTitle.trim(),
          body: broadcastBody.trim(),
          role: broadcastRole,
          data,
        }),
      });

      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(payload?.error || `HTTP ${response.status}`);
      }

      const recipients = Number(payload?.recipients || 0);
      const sent = Number(payload?.fcm_sent || 0);
      const failed = Number(payload?.fcm_failed || 0);
      setBroadcastSummary(
        t("supplier_portal.admin.control_center.broadcast.summary", {
          recipients,
          sent,
          failed,
        }),
      );
      toast(t("supplier_portal.admin.control_center.toast.broadcast_completed"), "success");
    } catch (error) {
      const message = error instanceof Error ? error.message : t("supplier_portal.admin.control_center.error.broadcast_failed");
      toast(message, "error");
    } finally {
      setBroadcastLoading(false);
    }
  };

  const triggerReplenishment = async () => {
    setReplenishLoading(true);
    setReplenishStatus("");
    try {
      const response = await apiFetch("/v1/admin/replenishment/trigger", {
        method: "POST",
        body: JSON.stringify({}),
      });

      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(payload?.error || `HTTP ${response.status}`);
      }

      setReplenishStatus(String(payload?.status || "CYCLE_COMPLETE"));
      toast(t("supplier_portal.admin.control_center.toast.replenishment_triggered"), "success");
    } catch (error) {
      const message = error instanceof Error ? error.message : t("supplier_portal.admin.control_center.error.replenishment_failed");
      toast(message, "error");
    } finally {
      setReplenishLoading(false);
    }
  };

  return (
    <div className="min-h-full p-6 md:p-10 space-y-6" style={{ background: "var(--background)", color: "var(--foreground)" }}>
      <header className="space-y-2">
        <h1 className="md-typescale-headline-medium">{t("supplier_portal.admin.control_center.title")}</h1>
        <p className="md-typescale-body-medium" style={{ color: "var(--muted)" }}>
          {t("supplier_portal.admin.control_center.subtitle")}
        </p>
      </header>

      <section className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="md-card md-card-elevated p-5 space-y-4">
          <div>
            <h2 className="md-typescale-title-medium">{t("supplier_portal.admin.control_center.bypass.title")}</h2>
            <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
              {t("supplier_portal.admin.control_center.bypass.description")}
            </p>
          </div>

          <div className="space-y-3">
            <input
              className="md-input-outlined w-full"
              placeholder={t("supplier_portal.admin.control_center.field.order_id")}
              value={orderId}
              onChange={(event) => setOrderId(event.target.value)}
            />
            <input
              className="md-input-outlined w-full"
              placeholder={t("supplier_portal.admin.control_center.field.reason")}
              value={bypassReason}
              onChange={(event) => setBypassReason(event.target.value)}
            />
          </div>

          <div className="flex items-center gap-3">
            <button
              className="md-btn md-btn-filled px-4 py-2"
              onClick={issuePaymentBypass}
              disabled={bypassLoading || !canIssueBypass}
            >
              {bypassLoading
                ? t("supplier_portal.admin.control_center.action.issuing")
                : t("supplier_portal.admin.control_center.action.issue_token")}
            </button>
            {bypassToken ? (
              <span className="md-typescale-label-large" style={{ color: "var(--color-md-primary)" }}>
                {t("supplier_portal.admin.control_center.bypass.token", { token: bypassToken })}
              </span>
            ) : null}
          </div>
        </div>

        <div className="md-card md-card-elevated p-5 space-y-4">
          <div>
            <h2 className="md-typescale-title-medium">{t("supplier_portal.admin.control_center.reconcile.title")}</h2>
            <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
              {t("supplier_portal.admin.control_center.reconcile.description")}
            </p>
          </div>

          <div className="space-y-3">
            <input
              className="md-input-outlined w-full"
              placeholder={t("supplier_portal.admin.control_center.field.session_id")}
              value={sessionId}
              onChange={(event) => setSessionId(event.target.value)}
            />
          </div>

          <div className="space-y-3">
            <button
              className="md-btn md-btn-tonal px-4 py-2"
              onClick={reconcileSession}
              disabled={reconcileLoading || !canReconcile}
            >
              {reconcileLoading
                ? t("supplier_portal.admin.control_center.action.reconciling")
                : t("supplier_portal.admin.control_center.action.run_reconcile")}
            </button>
            {reconcileResult ? (
              <pre className="md-typescale-body-small p-3 overflow-auto rounded-md" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
                {reconcileResult}
              </pre>
            ) : null}
          </div>
        </div>

        <div className="md-card md-card-elevated p-5 space-y-4">
          <div>
            <h2 className="md-typescale-title-medium">{t("supplier_portal.admin.control_center.broadcast.title")}</h2>
            <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
              {t("supplier_portal.admin.control_center.broadcast.description")}
            </p>
          </div>

          <div className="space-y-3">
            <input
              className="md-input-outlined w-full"
              placeholder={t("supplier_portal.admin.control_center.field.title")}
              value={broadcastTitle}
              onChange={(event) => setBroadcastTitle(event.target.value)}
            />
            <textarea
              className="md-input-outlined w-full min-h-24"
              placeholder={t("supplier_portal.admin.control_center.field.body")}
              value={broadcastBody}
              onChange={(event) => setBroadcastBody(event.target.value)}
            />
            <select
              className="md-input-outlined w-full"
              value={broadcastRole}
              onChange={(event) => setBroadcastRole(event.target.value as BroadcastRole)}
            >
              <option value="ALL">{t("supplier_portal.admin.control_center.role.all")}</option>
              <option value="RETAILER">{t("supplier_portal.admin.control_center.role.retailer")}</option>
              <option value="DRIVER">{t("supplier_portal.admin.control_center.role.driver")}</option>
            </select>
            <textarea
              className="md-input-outlined w-full min-h-24 font-mono"
              placeholder='{"key":"value"}'
              value={broadcastData}
              onChange={(event) => setBroadcastData(event.target.value)}
            />
          </div>

          <div className="flex items-center gap-3">
            <button
              className="md-btn md-btn-filled px-4 py-2"
              onClick={sendBroadcast}
              disabled={broadcastLoading || !canBroadcast}
            >
              {broadcastLoading
                ? t("supplier_portal.admin.control_center.action.broadcasting")
                : t("supplier_portal.admin.control_center.action.send_broadcast")}
            </button>
            {broadcastSummary ? (
              <span className="md-typescale-label-medium" style={{ color: "var(--muted)" }}>
                {broadcastSummary}
              </span>
            ) : null}
          </div>
        </div>

        <div className="md-card md-card-elevated p-5 space-y-4">
          <div>
            <h2 className="md-typescale-title-medium">{t("supplier_portal.admin.control_center.replenishment.title")}</h2>
            <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
              {t("supplier_portal.admin.control_center.replenishment.description")}
            </p>
          </div>

          <div className="flex items-center gap-3">
            <button
              className="md-btn md-btn-outlined px-4 py-2"
              onClick={triggerReplenishment}
              disabled={replenishLoading}
            >
              {replenishLoading
                ? t("supplier_portal.admin.control_center.action.running")
                : t("supplier_portal.admin.control_center.action.trigger_cycle")}
            </button>
            {replenishStatus ? (
              <span className="md-typescale-label-large" style={{ color: "var(--color-md-primary)" }}>
                {replenishStatus}
              </span>
            ) : null}
          </div>
        </div>
      </section>
    </div>
  );
}
