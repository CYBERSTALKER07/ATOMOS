"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { canPatchOrderStatus } from "@/lib/admin-scope";
import { supplierScopeId } from "@/lib/supplier-scope";
import { ApiError } from "@pegasusx/api-client";
import {
  supplierNetworkModePutKey,
  supplierPlanningKillSwitchKey,
  supplierPlanningPullMatrixKey,
} from "@pegasusx/api-client/idempotency";
import type { NetworkModeResponse, PullMatrixResponse, KillSwitchResponse } from "@pegasusx/types";

const api = createSupplierApi();

const MODES = ["SPEED", "ECONOMY", "BALANCED", "LOW_CARBON", "MANUAL_ONLY"] as const;

function planningDisabledMessage(err: unknown): string | null {
  if (!(err instanceof ApiError) || err.status !== 409) return null;
  const payload = err.payload as { error?: string; code?: string } | null;
  const code = payload?.error || payload?.code || err.message;
  if (String(code).includes("factory_planning_disabled")) {
    return "Planning engines disabled (FACTORY_PLANNING_ENABLED off). Mode and kill-switch still work; pull-matrix will 409 until the env flag is on.";
  }
  return err.message;
}

export default function FactoryPlanningOpsPanel() {
  const [mode, setMode] = useState<NetworkModeResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [killReason, setKillReason] = useState("");
  const isAdmin = canPatchOrderStatus();

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getNetworkMode();
      setMode(resp);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load network mode");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function putMode(next: string) {
    setBusy(true);
    setStatus(null);
    try {
      const scope = supplierScopeId();
      const resp = await api.putNetworkMode(
        { mode: next },
        supplierNetworkModePutKey(scope, next),
      );
      setStatus(`Mode ${resp.old_mode} → ${resp.new_mode}`);
      await load();
    } catch (err) {
      setStatus(planningDisabledMessage(err) || (err instanceof Error ? err.message : "Mode update failed"));
    } finally {
      setBusy(false);
    }
  }

  async function runPullMatrix() {
    setBusy(true);
    setStatus(null);
    try {
      const scope = supplierScopeId();
      const resp: PullMatrixResponse = await api.postPlanningPullMatrix(
        supplierPlanningPullMatrixKey(scope),
      );
      setStatus(`Pull-matrix ${resp.status}: ${resp.transfers} transfers, ${resp.skus} SKUs (${resp.source})`);
    } catch (err) {
      setStatus(planningDisabledMessage(err) || (err instanceof Error ? err.message : "Pull-matrix failed"));
    } finally {
      setBusy(false);
    }
  }

  async function runKillSwitch() {
    const reason = killReason.trim();
    if (!reason) {
      setStatus("Typed reason required");
      return;
    }
    setBusy(true);
    setStatus(null);
    try {
      const scope = supplierScopeId();
      const resp: KillSwitchResponse = await api.postPlanningKillSwitch(
        { reason },
        supplierPlanningKillSwitchKey(scope, reason),
      );
      setStatus(`Kill-switch ${resp.status}: cancelled ${resp.cancelled_transfers}, mode ${resp.mode}`);
      setKillReason("");
      await load();
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Kill-switch failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="desk-card p-6 mt-6">
      <h2 className="bento-card-title">Factory network ops</h2>
      <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
        Network mode, pull-matrix, and ADMIN kill-switch. Pull-matrix creates SYSTEM_THRESHOLD transfers only when FACTORY_PLANNING_ENABLED is on. Kill-switch cancels system drafts. Does not change S&amp;OP on /planning.
      </p>

      {error ? (
        <p className="md-typescale-body-small mt-3" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      ) : loading && !mode ? (
        <p className="md-typescale-body-small mt-3" style={{ color: "var(--desk-text-secondary)" }}>
          Loading network mode…
        </p>
      ) : (
        <div className="mt-4 space-y-4">
          <div>
            <div className="text-xs uppercase tracking-wide mb-2" style={{ color: "var(--desk-text-secondary)" }}>
              Mode {mode?.planning_enabled === false ? "(engines off)" : mode?.planning_enabled ? "(engines on)" : ""}
            </div>
            <div className="flex flex-wrap gap-2">
              {MODES.map((m) => (
                <button
                  key={m}
                  type="button"
                  disabled={busy}
                  onClick={() => void putMode(m)}
                  className="portal-btn portal-btn--ghost text-xs"
                  style={
                    mode?.mode === m
                      ? { background: "var(--color-md-primary)", color: "var(--color-md-on-primary, #fff)" }
                      : undefined
                  }
                >
                  {m.replace(/_/g, " ")}
                </button>
              ))}
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <button type="button" className="portal-btn portal-btn--primary text-xs" disabled={busy} onClick={() => void runPullMatrix()}>
              Run pull-matrix
            </button>
          </div>

          {isAdmin ? (
            <div className="space-y-2">
              <label className="text-xs uppercase tracking-wide" style={{ color: "var(--desk-text-secondary)" }}>
                Kill-switch reason (ADMIN)
              </label>
              <input
                className="w-full rounded-lg border px-3 py-2 text-sm"
                style={{ borderColor: "var(--color-md-outline-variant)" }}
                value={killReason}
                onChange={(e) => setKillReason(e.target.value)}
                placeholder="Typed reason required"
              />
              <button type="button" className="portal-btn portal-btn--ghost text-xs" disabled={busy} onClick={() => void runKillSwitch()}>
                Kill-switch (cancel system drafts)
              </button>
            </div>
          ) : (
            <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
              Kill-switch is ADMIN only.
            </p>
          )}

          {status ? (
            <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
              {status}
            </p>
          ) : null}
        </div>
      )}
    </section>
  );
}
