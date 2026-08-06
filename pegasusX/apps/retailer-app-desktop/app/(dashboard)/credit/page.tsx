"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { getRetailerToken } from "@/lib/auth";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

type Relationship = {
  supplier_id: string;
  credit_enabled: boolean;
  terms_days: number;
  grace_period_days: number;
  credit_limit_minor: number;
  available_credit_minor?: number;
  current_balance_minor?: number;
  profile_status?: string;
  on_hold?: boolean;
};

type Invoice = {
  invoice_id: string;
  supplier_id: string;
  order_id: string;
  status: string;
  balance_minor: number;
  due_at: string;
  terms_days: number;
  aging_bucket?: string;
};

export default function RetailerCreditPartnersPage() {
  const t = usePortalT();
  const [rels, setRels] = useState<Relationship[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = await getRetailerToken();
      const headers = { Authorization: `Bearer ${token}` };
      const [rRes, iRes] = await Promise.all([
        fetch(`${API}/v1/retailer/credit-relationships`, { headers, credentials: "include" }),
        fetch(`${API}/v1/retailer/ar/invoices?status=OPEN`, { headers, credentials: "include" }),
      ]);
      if (!rRes.ok) throw new Error(`relationships_${rRes.status}`);
      const rBody = (await rRes.json()) as { relationships?: Relationship[] };
      setRels(rBody.relationships ?? []);
      if (iRes.ok) {
        const iBody = (await iRes.json()) as { invoices?: Invoice[] };
        setInvoices(iBody.invoices ?? []);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.load_failed"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h1 className="text-2xl font-semibold tracking-tight">{t("portal.nav.credit_partners")}</h1>
      <p className="mt-1 text-sm text-[var(--muted, #666)]">
        Suppliers that have enabled trade credit with your store. You cannot enable or disable credit here.
      </p>
      {loading ? <p className="mt-4 text-sm">{t("retailer_desktop.credit.text.loading")}</p> : null}
      {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}

      <section className="mt-8">
        <h2 className="text-base font-medium mb-3">{t("retailer_desktop.credit.text.relationships")}</h2>
        <ul className="space-y-3">
          {rels.map((r) => (
            <li key={r.supplier_id} className="border-b border-[var(--border,#ddd)] pb-3">
              <div className="font-mono text-sm">Credit with Supplier {r.supplier_id}</div>
              <div className="text-sm mt-1">
                Net {r.terms_days} · Available {r.available_credit_minor ?? 0} · Balance{" "}
                {r.current_balance_minor ?? 0}
                {r.on_hold || r.profile_status === "FROZEN" ? (
                  <span className="ml-2 text-amber-700">{t("retailer_desktop.credit.text.on_hold")}</span>
                ) : null}
              </div>
            </li>
          ))}
          {!loading && rels.length === 0 ? (
            <li className="text-sm text-[var(--muted,#666)]">{t("retailer_desktop.credit.text.no_credit_partners_yet")}</li>
          ) : null}
        </ul>
      </section>

      <section className="mt-10">
        <h2 className="text-base font-medium mb-3">{t("retailer_desktop.credit.text.open_invoices")}</h2>
        <ul className="space-y-3">
          {invoices.map((inv) => (
            <li key={inv.invoice_id} className="border-b border-[var(--border,#ddd)] pb-3 text-sm">
              <div>
                Order {inv.order_id} · due {new Date(inv.due_at).toLocaleDateString()} ·{" "}
                {inv.balance_minor} due
              </div>
              <div className="text-[var(--muted,#666)]">
                {inv.aging_bucket || inv.status} · Net {inv.terms_days}
              </div>
            </li>
          ))}
          {!loading && invoices.length === 0 ? (
            <li className="text-sm text-[var(--muted,#666)]">{t("retailer_desktop.credit.text.no_open_invoices")}</li>
          ) : null}
        </ul>
      </section>
    </div>
  );
}
