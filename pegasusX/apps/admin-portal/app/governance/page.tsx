"use client";

import React, { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import {
  api,
  type AuditChainAnchor,
  type AuditChainVerifyResult,
  type AuditLegalHold,
  type AuditV2Event,
  type IamSodPolicy,
  type IamSodViolation,
  type Tenant,
} from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

function currentUtcDayBucket(): string {
  return new Date().toISOString().slice(0, 10);
}

function shortHash(hash?: string): string {
  const clean = (hash || "").trim();
  if (!clean) return "—";
  if (clean.length <= 16) return clean;
  return `${clean.slice(0, 10)}…${clean.slice(-6)}`;
}

export default function AdminGovernancePage() {
  return (
    <Suspense fallback={null}>
      <AdminGovernanceInner />
    </Suspense>
  );
}

function AdminGovernanceInner() {
  const { token } = useAdminToken();
  const searchParams = useSearchParams();

  const [selectedEntity, setSelectedEntity] = useState<string>(
    () => (searchParams.get("entity") || "").trim(),
  );
  const [dayBucket, setDayBucket] = useState<string>(
    () => (searchParams.get("date") || "").trim() || currentUtcDayBucket(),
  );

  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [anchors, setAnchors] = useState<AuditChainAnchor[]>([]);
  const [verifyResults, setVerifyResults] = useState<Record<string, AuditChainVerifyResult>>({});
  const [holds, setHolds] = useState<AuditLegalHold[]>([]);
  const [sodPolicies, setSodPolicies] = useState<IamSodPolicy[]>([]);
  const [sodViolations, setSodViolations] = useState<IamSodViolation[]>([]);
  const [events, setEvents] = useState<AuditV2Event[]>([]);

  // Legal hold quick form
  const [holdObjectType, setHoldObjectType] = useState("order");
  const [holdObjectId, setHoldObjectId] = useState("");
  const [holdReason, setHoldReason] = useState("");

  const [loading, setLoading] = useState(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const supplierTenants = useMemo(
    () =>
      tenants.filter(
        (t) => (t.TenantType || "").toUpperCase() === "SUPPLIER",
      ),
    [tenants],
  );

  const loadCrossTenantHealth = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError(null);
    try {
      const entityFilter = selectedEntity.trim() || undefined;
      const cleanDate = dayBucket.trim() || currentUtcDayBucket();

      const [
        tenantsRes,
        anchorsRes,
        holdsRes,
        violationsRes,
        policiesRes,
        eventsRes,
      ] = await Promise.allSettled([
        api.listTenants(token, "", 100),
        api.listAuditAnchors(token, { entity: entityFilter, date: cleanDate, limit: 100 }),
        api.listAuditLegalHolds(token, { entity: entityFilter, limit: 100 }),
        api.listSodViolations(token, { entity: entityFilter, limit: 100 }),
        api.listSodPolicies(token, entityFilter),
        api.listAuditV2Events(token, { entity: entityFilter, date: cleanDate, limit: 50 }),
      ]);

      if (tenantsRes.status === "fulfilled") {
        const list = tenantsRes.value.tenants || [];
        setTenants(list);
        if (!selectedEntity.trim()) {
          const firstSupplier = list.find(
            (t) => (t.TenantType || "").toUpperCase() === "SUPPLIER",
          );
          if (firstSupplier?.TenantID) {
            setSelectedEntity(firstSupplier.TenantID);
          }
        }
      }
      if (anchorsRes.status === "fulfilled") {
        setAnchors(anchorsRes.value.anchors || []);
      } else {
        throw anchorsRes.reason;
      }
      if (holdsRes.status === "fulfilled") {
        setHolds(holdsRes.value.holds || []);
      }
      if (violationsRes.status === "fulfilled") {
        setSodViolations(violationsRes.value.violations || []);
      }
      if (policiesRes.status === "fulfilled") {
        setSodPolicies(policiesRes.value.policies || []);
      }
      if (eventsRes.status === "fulfilled") {
        setEvents(eventsRes.value.events || []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load governance health");
    } finally {
      setLoading(false);
    }
  }, [dayBucket, selectedEntity, token]);

  useEffect(() => {
    void loadCrossTenantHealth();
  }, [loadCrossTenantHealth]);

  const handleVerifyTenant = async (entityId: string) => {
    if (!token || !entityId.trim()) return;
    setBusyAction(`verify:${entityId}`);
    setError(null);
    setNotice(null);
    try {
      const res = await api.verifyAuditChain(token, entityId.trim(), dayBucket.trim());
      setVerifyResults((prev) => ({ ...prev, [entityId.trim()]: res }));
      setNotice(
        res.valid
          ? `Chain verified for ${entityId.trim()} (${res.day_bucket}, ${res.event_count} events).`
          : `Chain verification FAILED for ${entityId.trim()}: ${res.failure_reason || res.broken_event_id || "mismatch"}`,
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Chain verification failed");
    } finally {
      setBusyAction(null);
    }
  };

  const handleVerifyAllTenants = async () => {
    if (!token) return;
    const targets = Array.from(
      new Set([
        ...supplierTenants.map((t) => t.TenantID),
        ...anchors.map((a) => a.tenant_id),
        ...(selectedEntity.trim() ? [selectedEntity.trim()] : []),
      ].filter(Boolean)),
    );
    if (targets.length === 0) return;

    setBusyAction("verify-all");
    setError(null);
    setNotice(null);
    try {
      const checks = await Promise.allSettled(
        targets.map((tid) => api.verifyAuditChain(token, tid, dayBucket.trim())),
      );
      const next: Record<string, AuditChainVerifyResult> = {};
      let validCount = 0;
      let brokenCount = 0;
      for (const c of checks) {
        if (c.status === "fulfilled") {
          next[c.value.tenant_id] = c.value;
          if (c.value.valid) validCount += 1;
          else brokenCount += 1;
        }
      }
      setVerifyResults((prev) => ({ ...prev, ...next }));
      setNotice(
        `Verified ${validCount + brokenCount} tenant chains for ${dayBucket}: ${validCount} valid, ${brokenCount} broken.`,
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Batch chain verification failed");
    } finally {
      setBusyAction(null);
    }
  };

  const handleWriteAnchor = async (entityId: string) => {
    if (!token || !entityId.trim()) return;
    setBusyAction(`anchor:${entityId}`);
    setError(null);
    setNotice(null);
    try {
      const anchor = await api.writeAuditAnchor(token, entityId.trim(), dayBucket.trim());
      setNotice(`Daily root anchor written for ${anchor.tenant_id} (${anchor.day_bucket}).`);
      await loadCrossTenantHealth();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to write daily anchor");
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateHold = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token || !selectedEntity.trim() || !holdObjectId.trim() || !holdReason.trim()) {
      return;
    }
    setBusyAction("create-hold");
    setError(null);
    setNotice(null);
    try {
      const hold = await api.createAuditLegalHold(token, {
        entity_id: selectedEntity.trim(),
        object_type: holdObjectType.trim() || "order",
        object_id: holdObjectId.trim(),
        reason: holdReason.trim(),
      });
      setNotice(`Legal hold ${hold.hold_id} placed on ${hold.object_type}:${hold.object_id}.`);
      setHoldObjectId("");
      setHoldReason("");
      await loadCrossTenantHealth();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create legal hold");
    } finally {
      setBusyAction(null);
    }
  };

  const handleToggleSodPolicy = async (policy: IamSodPolicy) => {
    if (!token) return;
    const entity = policy.tenant_id || selectedEntity.trim();
    if (!entity) return;
    setBusyAction(`policy:${policy.policy_id}`);
    setError(null);
    try {
      await api.updateSodPolicy(token, policy.policy_id, {
        entity_id: entity,
        enabled: !policy.enabled,
      });
      await loadCrossTenantHealth();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update SoD policy");
    } finally {
      setBusyAction(null);
    }
  };

  const selectedVerify = selectedEntity.trim()
    ? verifyResults[selectedEntity.trim()]
    : undefined;
  const activeHoldsCount = holds.filter((h) => h.status === "ACTIVE").length;
  const brokenChainsCount = Object.values(verifyResults).filter((r) => !r.valid).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Platform Governance · Audit v2, Anchors &amp; SoD Health
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Daily cryptographic hash-chain anchor verification, cross-tenant Segregation of Duties (SoD) violations, and legal hold compliance (§6.2 &amp; §6.5).
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Link
            href="/governance/flags"
            className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50"
          >
            Dual-Control Flags
          </Link>
          <Link
            href="/governance/audit"
            className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50"
          >
            Legacy Audit Log
          </Link>
          <button
            type="button"
            onClick={() => void loadCrossTenantHealth()}
            disabled={loading}
            className="rounded-lg bg-gray-900 px-3 py-1.5 text-xs font-semibold text-white hover:bg-gray-800 disabled:opacity-50"
          >
            {loading ? "Refreshing..." : "Refresh"}
          </button>
        </div>
      </div>

      {error ? (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800" role="alert">
          <strong>Error:</strong> {error}
        </div>
      ) : null}

      {notice ? (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-xs text-emerald-900" role="status">
          {notice}
        </div>
      ) : null}

      {/* KPI Strip */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="rounded-xl border bg-white p-4 shadow-sm">
          <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
            Daily Chain Anchors ({dayBucket})
          </span>
          <div className="text-2xl font-extrabold text-gray-900 mt-1">{anchors.length}</div>
          <div className="text-xs text-gray-500 mt-1">
            {brokenChainsCount > 0 ? `${brokenChainsCount} broken verified` : "WORM root anchors"}
          </div>
        </div>
        <div className="rounded-xl border bg-white p-4 shadow-sm">
          <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
            Selected Chain Verification
          </span>
          <div className="text-2xl font-extrabold text-gray-900 mt-1">
            {selectedVerify ? (selectedVerify.valid ? "VALID" : "BROKEN") : "PENDING"}
          </div>
          <div className="text-xs font-mono text-gray-500 mt-1 truncate">
            {selectedVerify
              ? `${selectedVerify.event_count} events · ${shortHash(selectedVerify.root_hash)}`
              : selectedEntity || "Select tenant"}
          </div>
        </div>
        <div className="rounded-xl border bg-white p-4 shadow-sm">
          <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
            Cross-Tenant SoD Violations
          </span>
          <div className="text-2xl font-extrabold text-gray-900 mt-1">{sodViolations.length}</div>
          <div className="text-xs text-gray-500 mt-1">
            {sodPolicies.filter((p) => p.enabled).length} / {sodPolicies.length} active SoD rules
          </div>
        </div>
        <div className="rounded-xl border bg-white p-4 shadow-sm">
          <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
            Active Legal Holds
          </span>
          <div className="text-2xl font-extrabold text-gray-900 mt-1">{activeHoldsCount}</div>
          <div className="text-xs text-gray-500 mt-1">{holds.length} total retention holds</div>
        </div>
      </div>

      {/* Daily Chain Anchor Verification Controls */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h2 className="text-sm font-bold text-gray-900">
              Daily Hash-Chain Anchor Verification
            </h2>
            <p className="text-xs text-gray-500">
              Recompute SHA-256 chain links per tenant and day bucket, verify against WORM root anchors, or seal a new anchor.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {supplierTenants.length > 0 ? (
              <select
                aria-label="Supplier Entity"
                value={selectedEntity}
                onChange={(e) => setSelectedEntity(e.target.value)}
                className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
              >
                <option value="">All tenants (cross-tenant)</option>
                {supplierTenants.map((t) => (
                  <option key={t.TenantID} value={t.TenantID}>
                    {t.DisplayName || t.TenantID} ({t.TenantID})
                  </option>
                ))}
              </select>
            ) : (
              <input
                type="text"
                aria-label="Supplier Entity ID"
                placeholder="Supplier entity ID..."
                value={selectedEntity}
                onChange={(e) => setSelectedEntity(e.target.value)}
                className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
              />
            )}
            <input
              type="date"
              aria-label="Day Bucket"
              value={dayBucket}
              onChange={(e) => setDayBucket(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
            />
            <button
              type="button"
              disabled={!token || !selectedEntity.trim() || busyAction === `verify:${selectedEntity.trim()}`}
              onClick={() => void handleVerifyTenant(selectedEntity)}
              className="rounded-lg bg-gray-900 px-3 py-1.5 text-xs font-semibold text-white hover:bg-gray-800 disabled:opacity-50"
            >
              Verify Tenant Chain
            </button>
            <button
              type="button"
              disabled={!token || !selectedEntity.trim() || busyAction === `anchor:${selectedEntity.trim()}`}
              onClick={() => void handleWriteAnchor(selectedEntity)}
              className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            >
              Write Anchor
            </button>
            <button
              type="button"
              disabled={!token || busyAction === "verify-all"}
              onClick={() => void handleVerifyAllTenants()}
              className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            >
              Verify All Tenants
            </button>
          </div>
        </div>

        {/* Anchors Table */}
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left text-xs">
            <thead className="border-b bg-gray-50 font-bold uppercase tracking-wider text-gray-500">
              <tr>
                <th className="px-3 py-2">Tenant</th>
                <th className="px-3 py-2">Day Bucket</th>
                <th className="px-3 py-2 text-right">Events</th>
                <th className="px-3 py-2">Root Hash</th>
                <th className="px-3 py-2">WORM URI</th>
                <th className="px-3 py-2">Verify Status</th>
                <th className="px-3 py-2 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {anchors.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-3 py-6 text-center text-gray-400">
                    No daily chain anchors recorded for this filter.
                  </td>
                </tr>
              ) : (
                anchors.map((a) => {
                  const vr = verifyResults[a.tenant_id];
                  return (
                    <tr key={`${a.tenant_id}:${a.day_bucket}`}>
                      <td className="px-3 py-2 font-mono font-semibold">{a.tenant_id}</td>
                      <td className="px-3 py-2 font-mono">{a.day_bucket}</td>
                      <td className="px-3 py-2 font-mono text-right">{a.event_count}</td>
                      <td className="px-3 py-2 font-mono" title={a.root_hash}>
                        {shortHash(a.root_hash)}
                      </td>
                      <td className="px-3 py-2 font-mono truncate max-w-[200px]" title={a.worm_uri || ""}>
                        {a.worm_uri || "—"}
                      </td>
                      <td className="px-3 py-2 font-mono">
                        {vr ? (
                          <span
                            className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                              vr.valid
                                ? "bg-emerald-100 text-emerald-800"
                                : "bg-rose-100 text-rose-800"
                            }`}
                          >
                            {vr.valid ? "VERIFIED" : "BROKEN"}
                          </span>
                        ) : (
                          <span className="text-gray-400">Not checked</span>
                        )}
                      </td>
                      <td className="px-3 py-2 text-right">
                        <button
                          type="button"
                          onClick={() => void handleVerifyTenant(a.tenant_id)}
                          className="rounded border border-gray-300 px-2 py-1 text-[11px] font-semibold hover:bg-gray-50"
                        >
                          Verify
                        </button>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Cross-Tenant SoD Health & Legal Holds */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* SoD Violations & Policies */}
        <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between border-b pb-3">
            <div>
              <h2 className="text-sm font-bold text-gray-900">
                Cross-Tenant SoD Violations &amp; Policies
              </h2>
              <p className="text-xs text-gray-500">
                Blocked maker-checker and agent approval violations across tenants.
              </p>
            </div>
            <span className="rounded-full bg-gray-100 px-2.5 py-0.5 text-[11px] font-mono font-semibold text-gray-700">
              {sodViolations.length} violations
            </span>
          </div>

          <div className="overflow-x-auto max-h-72">
            <table className="w-full border-collapse text-left text-xs">
              <thead className="border-b bg-gray-50 font-bold uppercase tracking-wider text-gray-500">
                <tr>
                  <th className="px-2.5 py-2">Tenant</th>
                  <th className="px-2.5 py-2">Rule</th>
                  <th className="px-2.5 py-2">Actor / Maker</th>
                  <th className="px-2.5 py-2">Object</th>
                  <th className="px-2.5 py-2">Time</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {sodViolations.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-3 py-6 text-center text-gray-400">
                      Zero SoD violations recorded.
                    </td>
                  </tr>
                ) : (
                  sodViolations.map((v) => (
                    <tr key={v.violation_id}>
                      <td className="px-2.5 py-2 font-mono">{v.tenant_id}</td>
                      <td className="px-2.5 py-2 font-mono font-semibold text-rose-700">
                        {v.rule_code}
                      </td>
                      <td className="px-2.5 py-2 font-mono">
                        {v.actor_id}
                        {v.maker_id ? ` / ${v.maker_id}` : ""}
                      </td>
                      <td className="px-2.5 py-2 font-mono">
                        {v.object_type ? `${v.object_type}:${v.object_id || "—"}` : v.action}
                      </td>
                      <td className="px-2.5 py-2 whitespace-nowrap text-gray-500">
                        {new Date(v.created_at).toLocaleString()}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {sodPolicies.length > 0 ? (
            <div className="border-t pt-3 space-y-2">
              <h3 className="text-xs font-bold uppercase tracking-wider text-gray-500">
                SoD Policies ({selectedEntity || "All"})
              </h3>
              <div className="space-y-1.5 max-h-48 overflow-y-auto">
                {sodPolicies.map((pol) => (
                  <div
                    key={`${pol.tenant_id}:${pol.policy_id}`}
                    className="flex items-center justify-between gap-2 rounded-lg border px-3 py-2 text-xs"
                  >
                    <div className="min-w-0">
                      <div className="font-semibold text-gray-900">
                        {pol.name}{" "}
                        <span className="font-mono text-[11px] text-gray-500">
                          ({pol.tenant_id})
                        </span>
                      </div>
                      <div className="font-mono text-[11px] text-gray-500">
                        {pol.permission_a} ↔ {pol.permission_b}
                      </div>
                    </div>
                    <button
                      type="button"
                      disabled={busyAction === `policy:${pol.policy_id}`}
                      onClick={() => void handleToggleSodPolicy(pol)}
                      className={`rounded px-2 py-1 text-[11px] font-semibold ${
                        pol.enabled
                          ? "bg-emerald-100 text-emerald-800"
                          : "bg-gray-100 text-gray-600"
                      }`}
                    >
                      {pol.enabled ? "Enabled" : "Disabled"}
                    </button>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </div>

        {/* Cross-Tenant Legal Holds */}
        <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between border-b pb-3">
            <div>
              <h2 className="text-sm font-bold text-gray-900">
                Cross-Tenant Legal Holds
              </h2>
              <p className="text-xs text-gray-500">
                Objects locked under statutory or investigation retention hold.
              </p>
            </div>
            <span className="rounded-full bg-gray-100 px-2.5 py-0.5 text-[11px] font-mono font-semibold text-gray-700">
              {activeHoldsCount} active
            </span>
          </div>

          <form onSubmit={(e) => void handleCreateHold(e)} className="grid grid-cols-1 sm:grid-cols-4 gap-2">
            <input
              type="text"
              aria-label="Hold Object Type"
              placeholder="Object type (order)"
              value={holdObjectType}
              onChange={(e) => setHoldObjectType(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs"
              required
            />
            <input
              type="text"
              aria-label="Hold Object ID"
              placeholder="Object ID"
              value={holdObjectId}
              onChange={(e) => setHoldObjectId(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs"
              required
            />
            <input
              type="text"
              aria-label="Hold Reason"
              placeholder="Hold reason..."
              value={holdReason}
              onChange={(e) => setHoldReason(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs"
              required
            />
            <button
              type="submit"
              disabled={!token || !selectedEntity.trim() || busyAction === "create-hold"}
              className="rounded-lg bg-gray-900 px-3 py-1.5 text-xs font-semibold text-white hover:bg-gray-800 disabled:opacity-50"
            >
              Place Legal Hold
            </button>
          </form>

          <div className="overflow-x-auto max-h-72">
            <table className="w-full border-collapse text-left text-xs">
              <thead className="border-b bg-gray-50 font-bold uppercase tracking-wider text-gray-500">
                <tr>
                  <th className="px-2.5 py-2">Tenant</th>
                  <th className="px-2.5 py-2">Object</th>
                  <th className="px-2.5 py-2">Reason</th>
                  <th className="px-2.5 py-2">Status</th>
                  <th className="px-2.5 py-2">Created By</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {holds.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-3 py-6 text-center text-gray-400">
                      No legal holds recorded.
                    </td>
                  </tr>
                ) : (
                  holds.map((h) => (
                    <tr key={h.hold_id}>
                      <td className="px-2.5 py-2 font-mono">{h.tenant_id}</td>
                      <td className="px-2.5 py-2 font-mono">
                        {h.object_type}:{h.object_id}
                      </td>
                      <td className="px-2.5 py-2">{h.reason}</td>
                      <td className="px-2.5 py-2">
                        <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-bold text-amber-800">
                          {h.status}
                        </span>
                      </td>
                      <td className="px-2.5 py-2 font-mono text-gray-600">{h.created_by}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Recent Audit v2 Stream */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-3">
        <div className="flex items-center justify-between border-b pb-3">
          <div>
            <h2 className="text-sm font-bold text-gray-900">
              Recent Hash-Chained Audit Events ({dayBucket})
            </h2>
            <p className="text-xs text-gray-500">
              Universal mutating route &amp; kinetic action provenance across human, agent, connector, and system actors.
            </p>
          </div>
          <span className="text-xs font-mono text-gray-500">{events.length} shown</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left text-xs">
            <thead className="border-b bg-gray-50 font-bold uppercase tracking-wider text-gray-500">
              <tr>
                <th className="px-3 py-2">Seq</th>
                <th className="px-3 py-2">Tenant</th>
                <th className="px-3 py-2">Actor</th>
                <th className="px-3 py-2">Action</th>
                <th className="px-3 py-2">Object</th>
                <th className="px-3 py-2">Trace ID</th>
                <th className="px-3 py-2">Entry Hash</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {events.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-3 py-6 text-center text-gray-400">
                    No audit events for {dayBucket}.
                  </td>
                </tr>
              ) : (
                events.map((ev) => (
                  <tr key={ev.event_id}>
                    <td className="px-3 py-2 font-mono">#{ev.seq_num}</td>
                    <td className="px-3 py-2 font-mono">{ev.tenant_id}</td>
                    <td className="px-3 py-2">
                      <span className="mr-1.5 rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] font-bold uppercase text-gray-700">
                        {ev.actor_type}
                      </span>
                      <span className="font-mono">{ev.actor_id}</span>
                    </td>
                    <td className="px-3 py-2 font-mono font-semibold">{ev.action}</td>
                    <td className="px-3 py-2 font-mono">
                      {ev.object_type ? `${ev.object_type}:${ev.object_id || "—"}` : "—"}
                    </td>
                    <td className="px-3 py-2 font-mono">
                      {ev.trace_id ? <CopyableId id={ev.trace_id} label="Trace" /> : "—"}
                    </td>
                    <td className="px-3 py-2 font-mono text-gray-500" title={ev.entry_hash}>
                      {shortHash(ev.entry_hash)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
