"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierLoyaltyProgramPatchKey } from "@pegasusx/api-core";
import type { LoyaltyProgram } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { supplierScopeId } from "@/lib/supplier-scope";

const api = createSupplierApi();

export default function SupplierLoyaltyPage() {
  const [program, setProgram] = useState<LoyaltyProgram | null>(null);
  const [earnBps, setEarnBps] = useState("100");
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getSupplierLoyaltyProgram();
      setProgram(resp);
      setEarnBps(String(resp.earn_bps || 100));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load loyalty program");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  async function save() {
    const why = reason.trim();
    if (!why) {
      setStatus("Typed reason required");
      return;
    }
    setBusy(true);
    setStatus(null);
    try {
      const bps = Number.parseInt(earnBps, 10);
      const resp = await api.patchSupplierLoyaltyProgram(
        { supplier_id: supplierScopeId(), earn_bps: Number.isFinite(bps) && bps > 0 ? bps : 100, tiers: program?.tiers ?? [], reason: why },
        supplierLoyaltyProgramPatchKey(supplierScopeId(), why),
      );
      setProgram(resp);
      setReason("");
      setStatus(`Saved (${resp.source || "program"})`);
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Save failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageChrome
      icon="pricing"
      title="Loyalty program"
      description="Earn points on paid orders. Burn is out of scope. Unconfigured retailers see enrolled=false, not a fake Bronze."
      loading={loading}
      error={error}
    >
      <div className="desk-card p-6 space-y-4 max-w-xl">
        <p className="text-sm text-[var(--muted)]">
          Source: {program?.source || "unconfigured"} · default earn 100 bps (1%).
        </p>
        <label className="block space-y-1">
          <span className="md-typescale-label-medium">Earn bps</span>
          <input className="md-input w-full" value={earnBps} onChange={(e) => setEarnBps(e.target.value)} />
        </label>
        <label className="block space-y-1">
          <span className="md-typescale-label-medium">Reason</span>
          <input className="md-input w-full" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Required for PATCH" />
        </label>
        <button type="button" className="md-btn md-btn-filled px-4 py-2" disabled={busy} onClick={() => void save()}>
          {busy ? "Saving…" : "Save program"}
        </button>
        {status ? <p className="text-sm">{status}</p> : null}
        {program ? (
          <ul className="text-sm text-[var(--muted)] list-disc pl-5">
            {(program.tiers ?? []).map((t) => (
              <li key={t.name}>{t.name} from {t.min_points} lifetime points</li>
            ))}
          </ul>
        ) : null}
      </div>
    </PageChrome>
  );
}