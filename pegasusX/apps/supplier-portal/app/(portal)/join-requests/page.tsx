"use client";

import { useMemo, useState } from "react";
import { usePortalT } from "@/lib/i18n";
import { useJoinRequests } from "@pegasusx/api-react";
import { joinRequestDecideKey } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { SUPPLIER_JOIN_REQUEST_REFRESH_EVENTS } from "@/lib/supplier-ws-events";
import { useSupplierWsRefresh } from "@/lib/use-supplier-ws-refresh";
import { PageChrome } from "@/components/PageChrome";

export default function SupplierJoinRequestsPage() {
  const t = usePortalT();
  const api = useMemo(() => createSupplierApi(), []);
  const [statusFilter, setStatusFilter] = useState<string>("PENDING");
  const [reasonById, setReasonById] = useState<Record<string, string>>({});
  const [busyId, setBusyId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const { data: requests, loading, error, refresh } = useJoinRequests(
    (s) => api.listSupplierJoinRequests(s),
    statusFilter,
    { pollMs: 20_000 },
  );

  useSupplierWsRefresh(
    () => {
      void refresh(true);
    },
    {
      eventTypes: SUPPLIER_JOIN_REQUEST_REFRESH_EVENTS,
      debounceMs: 400,
    },
  );

  async function decide(requestId: string, decision: "approve" | "reject") {
    if (busyId) return;
    setBusyId(requestId);
    setActionError(null);
    setNotice(null);
    try {
      const reason = reasonById[requestId] ?? "";
      const key = joinRequestDecideKey(requestId, decision);
      if (decision === "approve") {
        await api.approveSupplierJoinRequest(requestId, reason, key);
      } else {
        await api.rejectSupplierJoinRequest(requestId, reason, key);
      }
      setNotice(t("supplier_portal.join_requests.decided_notice"));
      await refresh(true);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <PageChrome
      icon="person-add"
      title={t("supplier_portal.join_requests.inbox_title")}
      description={t("supplier_portal.join_requests.inbox_subtitle")}
      loading={loading && requests.length === 0}
      error={actionError || error}
      actions={
        <select
          className="md-input-outlined px-3 py-1.5 text-sm"
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
        >
          <option value="PENDING">{t("supplier_portal.join_requests.status_pending")}</option>
          <option value="APPROVED">{t("supplier_portal.join_requests.status_approved")}</option>
          <option value="REJECTED">{t("supplier_portal.join_requests.status_rejected")}</option>
        </select>
      }
    >
      {notice && (
        <div className="mb-4 rounded-lg border border-emerald-500/40 bg-emerald-500/10 p-3 text-sm text-emerald-700 dark:text-emerald-300">
          {notice}
        </div>
      )}

      {requests.length === 0 ? (
        <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-6 text-sm text-[var(--muted)]">
          {t("supplier_portal.join_requests.empty_inbox")}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {requests.map((req) => (
            <div
              key={req.request_id}
              className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-4 flex flex-col md:flex-row md:items-center md:justify-between gap-4"
            >
              <div className="flex flex-col gap-1 text-sm">
                <div className="font-semibold">
                  {req.requester_name || req.requester_phone} ({req.requester_phone})
                </div>
                <div className="text-xs text-[var(--muted)]">
                  INN {req.normalized_inn} · {t("supplier_portal.join_requests.role_sought")}:{" "}
                  <span className="font-mono">{req.role_sought}</span> · {req.status}
                </div>
              </div>

              {req.status === "PENDING" && (
                <div className="flex flex-wrap items-center gap-2">
                  <input
                    type="text"
                    className="md-input-outlined px-3 py-1.5 text-xs"
                    placeholder={t("supplier_portal.join_requests.reject_reason_placeholder")}
                    value={reasonById[req.request_id] ?? ""}
                    onChange={(e) =>
                      setReasonById((prev) => ({ ...prev, [req.request_id]: e.target.value }))
                    }
                  />
                  <button
                    type="button"
                    disabled={busyId === req.request_id}
                    onClick={() => void decide(req.request_id, "approve")}
                    className="md-btn md-btn-filled px-4 py-1.5 text-xs"
                  >
                    {t("supplier_portal.join_requests.approve_button")}
                  </button>
                  <button
                    type="button"
                    disabled={busyId === req.request_id}
                    onClick={() => void decide(req.request_id, "reject")}
                    className="md-btn md-btn-outlined px-4 py-1.5 text-xs text-rose-600"
                  >
                    {t("supplier_portal.join_requests.reject_button")}
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </PageChrome>
  );
}
