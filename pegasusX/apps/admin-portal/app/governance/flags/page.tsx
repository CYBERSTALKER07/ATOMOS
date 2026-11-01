"use client";

import React, { useEffect, useState } from "react";
import { api, type FlagEval, type FlagOverride } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

const MONEY_FLAGS = [
  "AR_INVOICES_ENABLED",
  "AR_DUNNING_ENABLED",
  "AUTO_ORDER_PLACE_ENABLED",
  "AUTO_ORDER_SHADOW_ENABLED",
  "AUTO_ORDER_SOAK_GATE_DISABLED",
  "FISCAL_PROVIDER",
];

export default function FlagsGovernancePage() {
  const { token } = useAdminToken();

  const [tenantType, setTenantType] = useState("SUPPLIER");
  const [tenantId, setTenantId] = useState("");
  const [flagKey, setFlagKey] = useState(MONEY_FLAGS[0]);
  const [reason, setReason] = useState("");
  const [evalResult, setEvalResult] = useState<FlagEval | null>(null);
  const [pendingFlags, setPendingFlags] = useState<FlagOverride[]>([]);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const isMoney = MONEY_FLAGS.includes(flagKey);

  const fetchPending = async () => {
    if (!token) return;
    try {
      const res = await api.listPendingFlags(token);
      setPendingFlags(res.items || []);
    } catch {
      // Non-fatal if offline
    }
  };

  useEffect(() => {
    fetchPending();
  }, [token]);

  const handleEvaluate = async () => {
    if (!token) return;
    setError(null);
    setLoading(true);
    try {
      const res = await api.evalFlag(token, flagKey, tenantType, tenantId);
      setEvalResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Flag evaluation failed");
    } finally {
      setLoading(false);
    }
  };

  const handleSetFlag = async (enabled: boolean) => {
    if (!token) return;
    if (isMoney && !reason.trim()) {
      setError("A detailed justification rationale is required for money-affecting flags.");
      return;
    }
    setError(null);
    setSuccessMsg(null);
    setLoading(true);

    try {
      const res = await api.setFlag(token, flagKey, tenantType, tenantId, enabled, reason);
      if (res.status === "PENDING") {
        setSuccessMsg("Maker-Checker request queued as PENDING. A different PLATFORM_ADMIN must approve before it takes effect.");
      } else {
        setSuccessMsg("Flag override updated successfully.");
      }
      setReason("");
      await fetchPending();
      await handleEvaluate();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to set flag override");
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (item: FlagOverride) => {
    if (!token) return;
    setError(null);
    setSuccessMsg(null);
    setLoading(true);

    try {
      await api.approveFlag(token, item.FlagKey, item.TenantType, item.TenantID);
      setSuccessMsg(`Flag ${item.FlagKey} approved and activated.`);
      await fetchPending();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to approve flag");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-b pb-4">
        <h1 className="text-xl font-bold tracking-tight text-gray-900">
          Dual-Control Feature Flags & Maker-Checker Governance
        </h1>
        <p className="text-xs text-gray-500 mt-0.5">
          Dual-control governance (UA-5) for money-affecting operational toggles. Changes require independent peer sign-off.
        </p>
      </div>

      {/* Notifications */}
      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 flex items-start gap-2">
          <span className="text-base text-rose-600 font-bold">⚠</span>
          <div>{error}</div>
        </div>
      )}
      {successMsg && (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-xs text-emerald-800 flex items-start gap-2">
          <span className="text-base text-emerald-600 font-bold">✓</span>
          <div>{successMsg}</div>
        </div>
      )}

      {/* Pending Approvals Queue (Maker-Checker Cards UA-5) */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-xs font-bold uppercase tracking-wider text-gray-700">
            Pending Maker-Checker Approvals ({pendingFlags.length})
          </h2>
          <button
            type="button"
            onClick={fetchPending}
            className="text-xs font-semibold text-indigo-600 hover:text-indigo-800"
          >
            Refresh Queue
          </button>
        </div>

        {pendingFlags.length === 0 ? (
          <div className="rounded-xl border border-dashed bg-gray-50/50 p-6 text-center text-xs text-gray-400">
            No pending dual-control flag requests awaiting peer approval.
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {pendingFlags.map((item, idx) => (
              <div key={idx} className="rounded-xl border border-amber-200 bg-amber-50/30 p-4 shadow-sm space-y-3 flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between">
                    <span className="rounded bg-amber-100 px-2 py-0.5 text-[10px] font-bold text-amber-800">
                      PENDING PEER APPROVAL
                    </span>
                    <span className="text-[10px] font-mono text-gray-500">
                      {item.Status}
                    </span>
                  </div>

                  <div className="mt-2 text-sm font-extrabold text-gray-900 font-mono">
                    {item.FlagKey}
                  </div>

                  <div className="text-xs text-gray-600 mt-1 space-y-0.5">
                    <div>Scope: <span className="font-semibold">{item.TenantType}:{item.TenantID || "GLOBAL"}</span></div>
                    <div>Requested Target: <span className={`font-bold ${item.Enabled ? "text-emerald-700" : "text-rose-700"}`}>
                      {item.Enabled ? "ENABLED" : "DISABLED"}
                    </span></div>
                    <div>Updated By: <span className="font-mono text-gray-800">{item.UpdatedBy || "admin_ops"}</span></div>
                  </div>

                  {item.Reason && (
                    <div className="rounded bg-white border border-amber-200 p-2 text-[11px] text-amber-950 mt-2">
                      <strong>Rationale:</strong> {item.Reason}
                    </div>
                  )}
                </div>

                <div className="pt-2 border-t border-amber-200">
                  <button
                    type="button"
                    onClick={() => handleApprove(item)}
                    disabled={loading}
                    className="w-full rounded-lg bg-emerald-600 py-1.5 text-xs font-bold text-white shadow hover:bg-emerald-700 disabled:opacity-50"
                  >
                    1-Click Approve & Activate
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 pt-4 border-t">
        {/* Flag Override Proposer Form */}
        <div className="lg:col-span-6 rounded-xl border bg-white p-5 shadow-sm space-y-4">
          <h2 className="text-xs font-bold uppercase tracking-wider text-gray-500 border-b pb-2">
            Propose Flag Override
          </h2>

          <div className="space-y-3">
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Feature Flag Key</label>
              <select
                value={flagKey}
                onChange={(e) => setFlagKey(e.target.value)}
                className="w-full rounded border px-3 py-2 text-xs font-mono font-semibold"
              >
                {MONEY_FLAGS.map((k) => (
                  <option key={k} value={k}>
                    {k} (Money-Affecting)
                  </option>
                ))}
                <option value="RESCUE_FLEET_AUTO_DISPATCH">RESCUE_FLEET_AUTO_DISPATCH</option>
                <option value="STRICT_IKPU_CLASSIFICATION_ENFORCED">STRICT_IKPU_CLASSIFICATION_ENFORCED</option>
              </select>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Tenant Scope</label>
                <select
                  value={tenantType}
                  onChange={(e) => setTenantType(e.target.value)}
                  className="w-full rounded border px-2.5 py-1.5 text-xs"
                >
                  <option value="SUPPLIER">SUPPLIER</option>
                  <option value="RETAILER">RETAILER</option>
                  <option value="GLOBAL">GLOBAL (All Tenants)</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Tenant ID</label>
                <input
                  type="text"
                  placeholder="Optional (blank for global)"
                  value={tenantId}
                  onChange={(e) => setTenantId(e.target.value)}
                  className="w-full rounded border px-2.5 py-1.5 text-xs font-mono"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Business Justification Rationale {isMoney && <span className="text-rose-600 font-bold">*</span>}
              </label>
              <textarea
                rows={2}
                placeholder="Reason for change, impact on settlement or order volume, incident ticket..."
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                className="w-full rounded border px-3 py-2 text-xs"
              />
            </div>

            <div className="flex items-center gap-3 pt-2">
              <button
                type="button"
                onClick={() => handleSetFlag(true)}
                disabled={loading || !token}
                className="flex-1 rounded-lg bg-emerald-600 py-2 text-xs font-bold text-white shadow hover:bg-emerald-700 disabled:opacity-50"
              >
                Propose Enable (True)
              </button>
              <button
                type="button"
                onClick={() => handleSetFlag(false)}
                disabled={loading || !token}
                className="flex-1 rounded-lg bg-rose-600 py-2 text-xs font-bold text-white shadow hover:bg-rose-700 disabled:opacity-50"
              >
                Propose Disable (False)
              </button>
            </div>
          </div>
        </div>

        {/* Flag Evaluation Inspector */}
        <div className="lg:col-span-6 rounded-xl border bg-white p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between border-b pb-2">
            <h2 className="text-xs font-bold uppercase tracking-wider text-gray-500">
              Live Flag Evaluation Inspector
            </h2>
            <button
              type="button"
              onClick={handleEvaluate}
              disabled={loading || !token}
              className="rounded bg-indigo-50 px-2.5 py-1 text-xs font-semibold text-indigo-700 hover:bg-indigo-100"
            >
              Evaluate Live Head
            </button>
          </div>

          {!evalResult ? (
            <div className="rounded-lg border border-dashed p-8 text-center text-xs text-gray-400">
              Click &quot;Evaluate Live Head&quot; to test the resolved runtime flag boolean for the selected tenant scope.
            </div>
          ) : (
            <div className="space-y-3 text-xs">
              <div className="flex items-center justify-between">
                <span className="font-bold text-gray-700">Effective Resolved State:</span>
                <span
                  className={`rounded-full px-2.5 py-0.5 text-xs font-bold ${
                    evalResult.enabled ? "bg-emerald-100 text-emerald-800" : "bg-rose-100 text-rose-800"
                  }`}
                >
                  {evalResult.enabled ? "ENABLED (TRUE)" : "DISABLED (FALSE)"}
                </span>
              </div>
              <div className="flex justify-between py-1 border-b">
                <span className="text-gray-500">Resolution Source:</span>
                <span className="font-mono font-semibold">{evalResult.source}</span>
              </div>
              <div className="flex justify-between py-1 border-b">
                <span className="text-gray-500">Evaluated Key:</span>
                <span className="font-mono">{evalResult.flag_key}</span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
