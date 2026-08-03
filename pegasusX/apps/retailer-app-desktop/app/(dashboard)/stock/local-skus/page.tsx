"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Loader2, Plus, RefreshCw } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

type LocalSKU = {
  local_sku_id: string;
  barcode?: string;
  name: string;
  unit?: string;
  default_price_minor: number;
  currency?: string;
  is_active: boolean;
};

export default function LocalSKUsPage() {
  const [items, setItems] = useState<LocalSKU[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [barcode, setBarcode] = useState("");
  const [price, setPrice] = useState("5000");
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch("/v1/retailer/local-skus");
      if (!res.ok) throw new Error(`list_${res.status}`);
      const data = (await res.json()) as { items?: LocalSKU[] };
      setItems(Array.isArray(data.items) ? data.items : []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_failed");
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const create = async () => {
    const n = name.trim();
    if (!n) return;
    setSaving(true);
    try {
      const res = await apiFetch("/v1/retailer/local-skus", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: n,
          barcode: barcode.trim() || undefined,
          default_price_minor: Number(price) || 0,
        }),
      });
      if (!res.ok) {
        const j = (await res.json().catch(() => ({}))) as { error?: string };
        throw new Error(j.error || `create_${res.status}`);
      }
      setName("");
      setBarcode("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "create_failed");
    } finally {
      setSaving(false);
    }
  };

  const toggleActive = async (row: LocalSKU) => {
    const id = encodeURIComponent(row.local_sku_id);
    const res = await apiFetch(`/v1/retailer/local-skus/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ is_active: !row.is_active }),
    });
    if (res.ok) await load();
  };

  return (
    <PageChrome
      icon="cube.box"
      title="Local SKUs"
      description="Non-Pegasus goods for POS. Prefixed local: — never sent to supplier reorder."
      loading={loading}
      skeletonVariant="table"
      actions={
        <div className="flex gap-2">
          <Link
            href="/stock"
            className="portal-btn portal-btn--ghost h-10 px-4 rounded-xl text-sm"
          >
            Store stock
          </Link>
          <button
            type="button"
            onClick={() => void load()}
            className="portal-btn portal-btn--ghost h-10 px-4 rounded-xl"
          >
            <RefreshCw size={16} />
          </button>
        </div>
      }
    >
      <div className="max-w-3xl space-y-6">
        {error && (
          <p className="text-sm text-[var(--desk-warning)]">{error}</p>
        )}

        <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 space-y-3">
          <p className="text-sm font-medium text-[var(--desk-text-primary)]">
            Quick add
          </p>
          <div className="flex flex-wrap gap-2">
            <input
              className="portal-input flex-1 min-w-[140px]"
              placeholder="Name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <input
              className="portal-input w-32"
              placeholder="Barcode"
              value={barcode}
              onChange={(e) => setBarcode(e.target.value)}
            />
            <input
              className="portal-input w-28"
              placeholder="Price minor"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
            />
            <button
              type="button"
              disabled={saving || !name.trim()}
              onClick={() => void create()}
              className="portal-btn portal-btn--primary h-10 px-4 rounded-xl inline-flex items-center gap-2 disabled:opacity-60"
            >
              {saving ? (
                <Loader2 size={16} className="animate-spin" />
              ) : (
                <Plus size={16} />
              )}
              Add
            </button>
          </div>
        </div>

        <div className="rounded-2xl border border-[var(--desk-border)] overflow-hidden">
          <table className="w-full text-sm text-left">
            <thead className="bg-[var(--desk-surface-muted)] text-xs uppercase tracking-wide text-[var(--desk-text-tertiary)]">
              <tr>
                <th className="px-3 py-2">SKU</th>
                <th className="px-3 py-2">Name</th>
                <th className="px-3 py-2">Barcode</th>
                <th className="px-3 py-2">Price</th>
                <th className="px-3 py-2">Active</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-3 py-6 text-[var(--desk-text-secondary)]">
                    No local SKUs yet. Add items sold only at this store.
                  </td>
                </tr>
              ) : (
                items.map((row) => (
                  <tr
                    key={row.local_sku_id}
                    className="border-t border-[var(--desk-border)]"
                  >
                    <td className="px-3 py-2 font-mono text-xs">
                      {row.local_sku_id}
                    </td>
                    <td className="px-3 py-2">{row.name}</td>
                    <td className="px-3 py-2 font-mono text-xs">
                      {row.barcode || "—"}
                    </td>
                    <td className="px-3 py-2">
                      {row.default_price_minor} {row.currency || "UZS"}
                    </td>
                    <td className="px-3 py-2">
                      <button
                        type="button"
                        className="text-xs underline text-[var(--desk-accent)]"
                        onClick={() => void toggleActive(row)}
                      >
                        {row.is_active ? "Active" : "Inactive"}
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </PageChrome>
  );
}
