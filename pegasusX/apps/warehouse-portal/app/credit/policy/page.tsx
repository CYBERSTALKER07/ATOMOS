"use client";

import { useCallback, useEffect, useState } from "react";

/**
 * Warehouse finance parity: same supplier-scoped credit policy APIs as supplier portal.
 * Uses warehouse session cookie / bearer via relative /api proxy when present.
 */
async function whFetch(path: string, init?: RequestInit): Promise<Response> {
  const base = process.env.NEXT_PUBLIC_API_URL || "";
  return fetch(`${base}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
  });
}

type Program = {
  program_enabled: boolean;
  global_terms_days: number;
  global_default_limit_minor: number;
};

type Relationship = {
  retailer_id: string;
  credit_enabled: boolean;
  terms_days: number;
  credit_limit_minor: number;
  profile_status?: string;
  current_balance_minor?: number;
};

export default function WarehouseCreditPolicyPage() {
  const [program, setProgram] = useState<Program | null>(null);
  const [rels, setRels] = useState<Relationship[]>([]);
  const [invoices, setInvoices] = useState<Array<{ invoice_id: string; due_at: string; balance_minor: number; aging_bucket?: string }>>([]);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const [p, r, i] = await Promise.all([
        whFetch("/v1/supplier/credit-program"),
        whFetch("/v1/supplier/credit-relationships"),
        whFetch("/v1/supplier/ar/invoices?status=OPEN"),
      ]);
      if (p.status === 403) {
        setError("finance_permission_required");
        return;
      }
      if (!p.ok) throw new Error(`program_${p.status}`);
      setProgram(await p.json());
      if (r.ok) {
        const body = await r.json();
        setRels(body.relationships ?? []);
      }
      if (i.ok) {
        const body = await i.json();
        setInvoices(body.invoices ?? []);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "load_failed");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <main className="p-6 max-w-4xl mx-auto">
      <h1 className="text-xl font-semibold">Credit policy (supplier-scoped)</h1>
      <p className="text-sm mt-1 opacity-70">
        Same ledger as supplier finance — not a second credit book. Permanent disable requires Pegaus support.
      </p>
      {error ? <p className="mt-3 text-sm text-red-600">{error}</p> : null}
      <section className="mt-6">
        <h2 className="font-medium">Program</h2>
        <p className="text-sm mt-1">
          {program?.program_enabled ? `ON · Net ${program.global_terms_days}` : "OFF"}
        </p>
      </section>
      <section className="mt-6">
        <h2 className="font-medium">Relationships</h2>
        <ul className="mt-2 space-y-2 text-sm">
          {rels.map((r) => (
            <li key={r.retailer_id}>
              Retailer {r.retailer_id}: Net {r.terms_days}, bal {r.current_balance_minor ?? 0},{" "}
              {r.profile_status || (r.credit_enabled ? "ON" : "OFF")}
            </li>
          ))}
        </ul>
      </section>
      <section className="mt-6">
        <h2 className="font-medium">Open AR (dock ops)</h2>
        <ul className="mt-2 space-y-2 text-sm">
          {invoices.map((inv) => (
            <li key={inv.invoice_id}>
              {inv.invoice_id}: due {inv.due_at} · {inv.balance_minor} · {inv.aging_bucket}
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
