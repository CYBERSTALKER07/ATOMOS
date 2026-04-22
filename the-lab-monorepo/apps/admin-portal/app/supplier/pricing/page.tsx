"use client";

import { useState, useEffect } from "react";
import { Button } from '@heroui/react';
import { getAdminToken } from "@/lib/auth";
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

type PricingRule = {
  tier_id: string;
  sku_id: string;
  min_pallets: number;
  discount_percent: number;
  target_retailer_tier: string;
  valid_until: string;
  is_active?: boolean;
};

type ProductSku = {
  sku_id: string;
  name: string;
};

const RETAILER_TIERS = ["ALL", "BRONZE", "SILVER", "GOLD"] as const;

export default function SupplierPricingPage() {
  const [rules, setRules] = useState<PricingRule[]>([]);
  const [products, setProducts] = useState<ProductSku[]>([]);
  const [loadingRules, setLoadingRules] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [deactivating, setDeactivating] = useState<string | null>(null);
  const { toast } = useToast();
  const [form, setForm] = useState<PricingRule>({
    tier_id: "",
    sku_id: "",
    min_pallets: 10,
    discount_percent: 5,
    target_retailer_tier: "ALL",
    valid_until: "",
  });

  const fetchRulesAndProducts = async () => {
    try {
      const token = await getAdminToken();
      const headers = { Authorization: `Bearer ${token}` };
      const [rulesRes, productsRes] = await Promise.all([
        fetch(`${API}/v1/supplier/pricing/rules`, { headers }),
        fetch(`${API}/v1/supplier/products`, { headers }),
      ]);
      if (rulesRes.ok) {
        const data = await rulesRes.json();
        setRules(data);
      }
      if (productsRes.ok) {
        const pData = await productsRes.json();
        setProducts((pData.data || []).map((p: { sku_id: string; name: string }) => ({ sku_id: p.sku_id, name: p.name })));
      }
    } catch {} finally {
      setLoadingRules(false);
    }
  };

  useEffect(() => { fetchRulesAndProducts(); }, []);

  const handleDeactivateRule = async (tierId: string) => {
    setDeactivating(tierId);
    try {
      const token = await getAdminToken();
      const res = await fetch(`${API}/v1/supplier/pricing/rules/${tierId}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) {
        const msg = await res.text();
        throw new Error(msg || 'Deactivation failed');
      }
      toast('Pricing rule deactivated', 'success');
      fetchRulesAndProducts();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : String(err), 'error');
    } finally {
      setDeactivating(null);
    }
  };

  const skuNameMap = new Map(products.map(p => [p.sku_id, p.name]));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);

    const tierId = form.tier_id || crypto.randomUUID();

    try {
      const token = await getAdminToken();

      const res = await fetch(`${API}/v1/supplier/pricing/rules`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token.trim()}`,
        },
        body: JSON.stringify({
          ...form,
          tier_id: tierId,
          valid_until: form.valid_until ? new Date(form.valid_until).toISOString() : undefined,
        }),
      });

      if (!res.ok) {
        const msg = await res.text();
        throw new Error(msg || "Rule rejected");
      }

      await res.json();
      toast('Pricing rule locked successfully' , 'success');
      fetchRulesAndProducts();
      setForm({
        tier_id: "",
        sku_id: "",
        min_pallets: 10,
        discount_percent: 5,
        target_retailer_tier: "ALL",
        valid_until: "",
      });
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : String(err) , 'error');
    } finally {
      setSubmitting(false);
    }
  };

  const inputClass =
    "md-input-outlined w-full font-mono";
  const labelClass = "md-typescale-label-small block mb-2";

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Header */}
      <header className="mb-10">
        <h1 className="md-typescale-headline-medium">Pricing Engine</h1>
        <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--muted)' }}>B2B Volume Discount Rules — Upsert & Manage</p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
        {/* Form Panel — M3 Filled Card */}
        <div className="lg:col-span-2">
          <div className="md-card md-card-elevated p-6 md-animate-in">
            <div className="flex items-center justify-between mb-6">
              <h2 className="md-typescale-title-small">New Pricing Rule</h2>
            </div>

            <form onSubmit={handleSubmit} className="space-y-5">
              <div>
                <label className={labelClass} style={{ color: 'var(--muted)' }}>SKU</label>
                {products.length > 0 ? (
                  <select
                    required
                    className={inputClass}
                    value={form.sku_id}
                    onChange={e => setForm({ ...form, sku_id: e.target.value })}
                  >
                    <option value="">Select a product…</option>
                    {products.map(p => (
                      <option key={p.sku_id} value={p.sku_id}>{p.name} ({p.sku_id.slice(0, 8)}…)</option>
                    ))}
                  </select>
                ) : (
                  <input
                    required
                    type="text"
                    placeholder="e.g. SKU-COKE-001"
                    className={inputClass}
                    value={form.sku_id}
                    onChange={e => setForm({ ...form, sku_id: e.target.value })}
                  />
                )}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass} style={{ color: 'var(--muted)' }}>Min Pallets</label>
                  <input
                    required
                    type="number"
                    min={1}
                    className={inputClass}
                    value={form.min_pallets}
                    onChange={e => setForm({ ...form, min_pallets: parseInt(e.target.value) || 1 })}
                  />
                </div>
                <div>
                  <label className={labelClass} style={{ color: 'var(--muted)' }}>Discount %</label>
                  <input
                    required
                    type="number"
                    min={1}
                    max={40}
                    className={inputClass}
                    value={form.discount_percent}
                    onChange={e => setForm({ ...form, discount_percent: Math.min(40, Math.max(1, parseInt(e.target.value) || 1)) })}
                  />
                  <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>Max 40% — hard cap</p>
                </div>
              </div>

              <div>
                <label className={labelClass} style={{ color: 'var(--muted)' }}>Target Retailer Tier</label>
                <div className="flex gap-2">
                  {RETAILER_TIERS.map(tier => (
                    <button
                      key={tier}
                      type="button"
                      onClick={() => setForm({ ...form, target_retailer_tier: tier })}
                      className={form.target_retailer_tier === tier ? "md-chip md-chip-selected" : "md-chip"}
                    >
                      {tier}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className={labelClass} style={{ color: 'var(--muted)' }}>Valid Until (optional)</label>
                <input
                  type="datetime-local"
                  className={inputClass}
                  value={form.valid_until}
                  onChange={e => setForm({ ...form, valid_until: e.target.value })}
                />
              </div>

              <div>
                <label className={labelClass} style={{ color: 'var(--muted)' }}>Tier ID (optional — auto-generated if blank)</label>
                <input
                  type="text"
                  placeholder="UUID for idempotent upserts"
                  className={inputClass}
                  value={form.tier_id}
                  onChange={e => setForm({ ...form, tier_id: e.target.value })}
                />
              </div>

              {/* M3 Filled Button */}
              <Button
                type="submit"
                variant="primary"
                isDisabled={submitting}
                fullWidth
              >
                {submitting ? "Locking Rule..." : "Lock Pricing Rule"}
              </Button>
            </form>
          </div>
        </div>

        {/* Active Rules Table — M3 Data Table */}
        <div className="lg:col-span-3 md-animate-in" style={{ animationDelay: "100ms" }}>
          <div className="md-card md-card-outlined p-0 overflow-hidden">
            <div className="px-6 py-5" style={{ borderBottom: '1px solid var(--border)' }}>
              <h2 className="md-typescale-title-small">Session Rules</h2>
              <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>Rules created this session — {rules.length} active</p>
            </div>

            {loadingRules ? (
              <div className="p-16 text-center">
                <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>Loading pricing rules…</p>
              </div>
            ) : rules.length === 0 ? (
              <EmptyState
                icon="pricing"
                headline="No rules created yet"
                body="Submit a pricing rule to see it here"
              />
            ) : (
              <table className="md-table">
                <thead>
                  <tr>
                    <th>SKU</th>
                    <th className="text-right">Min Pallets</th>
                    <th className="text-right">Discount</th>
                    <th>Tier Target</th>
                    <th>Expires</th>
                    <th>Status</th>
                    <th className="text-center">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {rules.map((rule, i) => (
                    <tr key={rule.tier_id || i} className="transition-colors">
                      <td>
                        <div className="font-mono md-typescale-body-small">{skuNameMap.get(rule.sku_id) || rule.sku_id}</div>
                        {skuNameMap.has(rule.sku_id) && (
                          <div className="md-typescale-label-small" style={{ color: 'var(--border)' }}>{rule.sku_id.slice(0, 8)}…</div>
                        )}
                      </td>
                      <td className="text-right" style={{ color: 'var(--muted)', fontVariantNumeric: 'tabular-nums' }}>{rule.min_pallets}</td>
                      <td className="text-right">
                        <span className="md-chip" style={{ cursor: 'default', background: 'var(--accent-soft)', color: 'var(--accent-soft-foreground)', borderColor: 'transparent' }}>
                          {rule.discount_percent}%
                        </span>
                      </td>
                      <td className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>{rule.target_retailer_tier}</td>
                      <td className="md-typescale-body-small" style={{ color: 'var(--border)' }}>
                        {rule.valid_until ? new Date(rule.valid_until).toLocaleDateString() : "∞"}
                      </td>
                      <td>
                        <span className="text-xs font-semibold px-2 py-0.5 rounded-full" style={{
                          background: rule.is_active !== false
                            ? 'color-mix(in srgb, var(--success) 15%, transparent)'
                            : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                          color: rule.is_active !== false ? 'var(--success)' : 'var(--danger)',
                        }}>
                          {rule.is_active !== false ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="text-center">
                        {rule.is_active !== false && (
                          <button
                            type="button"
                            disabled={deactivating === rule.tier_id}
                            onClick={() => handleDeactivateRule(rule.tier_id)}
                            className="text-xs font-medium px-3 py-1 rounded-lg transition-colors disabled:opacity-38"
                            style={{ background: 'color-mix(in srgb, var(--danger) 12%, transparent)', color: 'var(--danger)' }}
                          >
                            {deactivating === rule.tier_id ? '...' : 'Deactivate'}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
