"use client";

import { useEffect, useState } from "react";
import { CreditCard, Loader2, AlertTriangle } from "lucide-react";
import { apiFetch } from "../lib/auth";

type CreditProfile = {
  retailer_id: string;
  supplier_id: string;
  credit_limit_minor: number;
  current_balance_minor: number;
  available_credit_minor: number;
  risk_tier?: string;
  status: string;
  delinquency_count?: number;
};

export function CreditProfileCard({ className = "" }: { className?: string }) {
  const [profile, setProfile] = useState<CreditProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [missing, setMissing] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      setMissing(false);
      try {
        const res = await apiFetch("/v1/retailer/credit-profile", { method: "GET" });
        if (res.status === 404) {
          if (!cancelled) {
            setMissing(true);
            setProfile(null);
          }
          return;
        }
        if (!res.ok) {
          throw new Error(`credit_${res.status}`);
        }
        const body = (await res.json()) as CreditProfile;
        if (!cancelled) setProfile(body);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : "load_failed");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  if (loading) {
    return (
      <div className={`rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5 ${className}`}>
        <div className="flex items-center gap-2 text-[var(--desk-text-tertiary)] text-sm">
          <Loader2 size={16} className="animate-spin" /> Loading credit…
        </div>
      </div>
    );
  }

  if (missing) {
    return (
      <div className={`rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5 ${className}`}>
        <div className="flex items-center gap-2 mb-1">
          <CreditCard size={18} className="text-[var(--desk-accent)]" />
          <h3 className="md-typescale-title-small font-light">Supplier credit</h3>
        </div>
        <p className="text-sm text-[var(--desk-text-tertiary)]">
          No credit line on file for this supplier relationship.
        </p>
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className={`rounded-2xl border border-red-200 bg-red-50 p-5 ${className}`}>
        <div className="flex items-center gap-2 text-red-700 text-sm">
          <AlertTriangle size={16} />
          {error || "Credit unavailable"}
        </div>
      </div>
    );
  }

  const util =
    profile.credit_limit_minor > 0
      ? ((profile.current_balance_minor * 100) / profile.credit_limit_minor).toFixed(1)
      : "0.0";
  const frozen = profile.status === "FROZEN" || profile.status === "BLACKLISTED";

  return (
    <div
      className={`rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5 ${className} ${
        frozen ? "ring-1 ring-orange-300" : ""
      }`}
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <CreditCard size={18} className="text-[var(--desk-accent)]" />
          <h3 className="md-typescale-title-small font-light">Supplier credit</h3>
        </div>
        <span
          className={`text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded ${
            frozen ? "bg-orange-100 text-orange-800" : "bg-emerald-100 text-emerald-800"
          }`}
        >
          {profile.status}
        </span>
      </div>
      <div className="grid grid-cols-3 gap-3 text-sm">
        <div>
          <div className="text-[10px] uppercase tracking-widest text-[var(--desk-text-tertiary)]">
            Limit
          </div>
          <div className="font-mono font-medium tabular-nums">
            {profile.credit_limit_minor.toLocaleString()}
          </div>
        </div>
        <div>
          <div className="text-[10px] uppercase tracking-widest text-[var(--desk-text-tertiary)]">
            Balance due
          </div>
          <div className="font-mono font-medium tabular-nums">
            {profile.current_balance_minor.toLocaleString()}
          </div>
        </div>
        <div>
          <div className="text-[10px] uppercase tracking-widest text-[var(--desk-text-tertiary)]">
            Available
          </div>
          <div className="font-mono font-medium tabular-nums">
            {profile.available_credit_minor.toLocaleString()}
          </div>
        </div>
      </div>
      <div className="mt-3 flex items-center justify-between text-xs text-[var(--desk-text-tertiary)]">
        <span>
          Utilization {util}%
          {profile.risk_tier ? ` · risk ${profile.risk_tier}` : ""}
        </span>
        {profile.delinquency_count ? (
          <span className="text-red-600">Delinquency {profile.delinquency_count}</span>
        ) : null}
      </div>
    </div>
  );
}
