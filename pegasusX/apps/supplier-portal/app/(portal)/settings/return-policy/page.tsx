"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierReturnPolicyPutKey } from "@pegasusx/api-client/idempotency";
import type { SupplierReturnPolicy } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";

const api = createSupplierApi();
const PRESETS = [8, 24, 48, 72] as const;

export default function ReturnPolicySettingsPage() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [hours, setHours] = useState(48);
  const [concealed, setConcealed] = useState<string>("");
  const [requirePhoto, setRequirePhoto] = useState(true);
  const [allowExpired, setAllowExpired] = useState(false);
  const [sourceHint, setSourceHint] = useState<string>("");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const p = await api.getSupplierReturnPolicy();
      setHours(p.default_window_hours || 48);
      setConcealed(
        p.concealed_damage_window_hours != null
          ? String(p.concealed_damage_window_hours)
          : "",
      );
      setRequirePhoto(p.require_photo !== false);
      setAllowExpired(Boolean(p.allow_expired_claims));
      setSourceHint(p.policy_source_hint || "");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load return policy");
    } finally {
      setLoading(false);
    }
  }, []);

  useSupplierSessionReconcile(load);
  useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    if (hours < 1 || hours > 168) {
      setError("Default window must be between 1 and 168 hours");
      return;
    }
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const body: SupplierReturnPolicy = {
        default_window_hours: hours,
        require_photo: requirePhoto,
        allow_expired_claims: allowExpired,
      };
      const concealedNum = concealed.trim() ? Number(concealed) : NaN;
      if (!Number.isNaN(concealedNum) && concealedNum > 0) {
        body.concealed_damage_window_hours = concealedNum;
      }
      const savedPolicy = await api.putSupplierReturnPolicy(
        body,
        supplierReturnPolicyPutKey(supplierScopeId(), hours),
      );
      setSourceHint(savedPolicy.policy_source_hint || "SUPPLIER");
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageChrome
      title="Return policy"
      subtitle="Claim filing windows applied when orders complete. Retailers see the countdown from this policy."
    >
      {loading ? (
        <p className="text-sm text-[var(--desk-text-secondary)]">Loading…</p>
      ) : (
        <div className="mx-auto max-w-xl space-y-6">
          {error && (
            <p className="text-sm font-semibold text-red-600">{error}</p>
          )}
          {saved && (
            <p className="text-sm text-emerald-700">Return policy saved.</p>
          )}
          {sourceHint && (
            <p className="text-xs text-[var(--desk-text-secondary)]">
              Source hint: {sourceHint}
            </p>
          )}
          <div>
            <label className="mb-2 block text-sm font-medium">
              Default claim window (hours)
            </label>
            <div className="mb-2 flex flex-wrap gap-2">
              {PRESETS.map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setHours(p)}
                  className={`rounded-lg border px-3 py-1.5 text-sm ${
                    hours === p
                      ? "border-[var(--desk-accent)] bg-[var(--desk-accent)]/10"
                      : "border-[var(--desk-border)]"
                  }`}
                >
                  {p}h
                </button>
              ))}
            </div>
            <input
              type="number"
              min={1}
              max={168}
              value={hours}
              onChange={(e) => setHours(Number(e.target.value) || 0)}
              className="w-full rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-3 py-2"
            />
            <p className="mt-1 text-xs text-[var(--desk-text-secondary)]">
              Custom 1–168 hours. Preview: retailers may file claims for {hours}h
              after delivery.
            </p>
          </div>
          <div>
            <label className="mb-2 block text-sm font-medium">
              Concealed damage window (hours, optional)
            </label>
            <input
              type="number"
              min={1}
              max={168}
              value={concealed}
              onChange={(e) => setConcealed(e.target.value)}
              placeholder="Same as default"
              className="w-full rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-3 py-2"
            />
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={requirePhoto}
              onChange={(e) => setRequirePhoto(e.target.checked)}
            />
            Require photo evidence on claims
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={allowExpired}
              onChange={(e) => setAllowExpired(e.target.checked)}
            />
            Allow filing after window expires (ops override)
          </label>
          <button
            type="button"
            disabled={saving}
            onClick={() => void save()}
            className="portal-btn portal-btn--primary h-11 rounded-xl px-4 disabled:opacity-60"
          >
            {saving ? "Saving…" : "Save return policy"}
          </button>
        </div>
      )}
    </PageChrome>
  );
}
