"use client";

import { useEffect, useState } from "react";
import { api, type FlagEval, type FlagOverride } from "@/lib/api";

// Mirrors featureflags.MoneyAffectingFlags on the backend.
const MONEY_FLAGS = [
  "AR_INVOICES_ENABLED",
  "AR_DUNNING_ENABLED",
  "AUTO_ORDER_PLACE_ENABLED",
  "AUTO_ORDER_SHADOW_ENABLED",
  "AUTO_ORDER_SOAK_GATE_DISABLED",
  "FISCAL_PROVIDER",
];

export default function FlagsPanel({ token }: { token: string }) {
  const [tenantType, setTenantType] = useState("SUPPLIER");
  const [tenantId, setTenantId] = useState("");
  const [flagKey, setFlagKey] = useState(MONEY_FLAGS[0]);
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<FlagEval | null>(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState<FlagOverride[] | null>(null);
  const isMoney = MONEY_FLAGS.includes(flagKey);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const r = await api.listPendingFlags(token);
        if (!cancelled) setPending(r.items || []);
      } catch {
        if (!cancelled) setPending(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token, message]);

  const evaluate = async () => {
    setError("");
    setMessage("");
    try {
      setResult(await api.evalFlag(token, flagKey, tenantType, tenantId));
    } catch (e) {
      setError(e instanceof Error ? e.message : "evaluate failed");
    }
  };

  const setFlag = async (enabled: boolean) => {
    setError("");
    setMessage("");
    try {
      const r = await api.setFlag(token, flagKey, tenantType, tenantId, enabled, reason);
      setMessage(
        r.status === "PENDING"
          ? "Override recorded as PENDING — a different PLATFORM_ADMIN must approve before it takes effect."
          : "Override active."
      );
      await evaluate();
    } catch (e) {
      setError(e instanceof Error ? e.message : "set failed");
    }
  };

  const approve = async () => {
    setError("");
    setMessage("");
    try {
      await api.approveFlag(token, flagKey, tenantType, tenantId);
      setMessage("Override approved and now ACTIVE.");
      await evaluate();
    } catch (e) {
      setError(e instanceof Error ? e.message : "approve failed");
    }
  };

  return (
    <section className="max-w-2xl space-y-4">
      <div className="rounded border p-3" data-testid="gs-u-admin-pending-flags">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500">Pending dual-control</h2>
        {pending == null ? (
          <p className="mt-2 text-sm text-gray-600">unavailable</p>
        ) : pending.length === 0 ? (
          <p className="mt-2 text-sm text-gray-600">empty</p>
        ) : (
          <ul className="mt-2 space-y-1 text-xs">
            {pending.map((f) => (
              <li key={`${f.FlagKey}|${f.TenantType}|${f.TenantID}`}>
                <span className="font-mono">{f.FlagKey}</span> · {f.TenantID}
              </li>
            ))}
          </ul>
        )}
      </div>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Tenant type">
          <select value={tenantType} onChange={(e) => setTenantType(e.target.value)} className="w-full rounded border px-2 py-1.5 text-sm">
            <option>SUPPLIER</option>
            <option>RETAILER</option>
          </select>
        </Field>
        <Field label="Tenant ID">
          <input value={tenantId} onChange={(e) => setTenantId(e.target.value)} placeholder="sup_…" className="w-full rounded border px-2 py-1.5 text-sm" />
        </Field>
      </div>
      <Field label="Flag">
        <select value={flagKey} onChange={(e) => setFlagKey(e.target.value)} className="w-full rounded border px-2 py-1.5 text-sm">
          {MONEY_FLAGS.map((f) => (
            <option key={f}>{f}</option>
          ))}
        </select>
      </Field>
      {isMoney && (
        <Field label="Reason (required for money-affecting flags)">
          <input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Why this change" className="w-full rounded border px-2 py-1.5 text-sm" />
        </Field>
      )}

      <div className="flex flex-wrap gap-2">
        <button onClick={() => void evaluate()} className="rounded border px-3 py-1.5 text-sm hover:bg-gray-100">
          Evaluate
        </button>
        <button onClick={() => void setFlag(true)} className="rounded bg-green-600 px-3 py-1.5 text-sm text-white hover:bg-green-700">
          Enable
        </button>
        <button onClick={() => void setFlag(false)} className="rounded bg-red-600 px-3 py-1.5 text-sm text-white hover:bg-red-700">
          Disable
        </button>
        {isMoney && (
          <button onClick={() => void approve()} className="rounded bg-indigo-600 px-3 py-1.5 text-sm text-white hover:bg-indigo-700">
            Approve (2nd admin)
          </button>
        )}
      </div>

      {isMoney && (
        <p className="rounded bg-amber-50 px-3 py-2 text-xs text-amber-800">
          Dual control: money-affecting overrides stay PENDING until a different PLATFORM_ADMIN approves. The same actor cannot
          request and approve.
        </p>
      )}
      {message && <p className="rounded bg-green-50 px-3 py-2 text-sm text-green-700">{message}</p>}
      {error && <p className="rounded bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}

      {result && (
        <div className="rounded border bg-white p-4 text-sm">
          <Row k="Flag" v={result.flag_key} />
          <Row k="Enabled" v={String(result.enabled)} />
          <Row k="Source" v={result.source} />
          <Row k="Money-affecting" v={String(result.money_affecting)} />
        </div>
      )}
    </section>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-gray-600">{label}</span>
      {children}
    </label>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex justify-between border-b py-1 last:border-0">
      <span className="text-gray-500">{k}</span>
      <span className="font-mono">{v}</span>
    </div>
  );
}
