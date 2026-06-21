'use client';

import { useCallback, useEffect, useState } from 'react';
import { warehouseApi } from '@/lib/warehouse-api';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';

interface StockRow {
  sku_id: string;
  name?: string;
  image_url?: string;
  on_hand: number;
  available_qty: number;
  reserved_asap: number;
  reserved_scheduled: number;
  deficit_qty: number;
}

export default function StockCommitmentsPage() {
  const [items, setItems] = useState<StockRow[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await warehouseApi.getWarehouseStockCommitments();
      const rows = (data.items ?? data.skus ?? []) as StockRow[];
      setItems(rows.filter((r) => r.deficit_qty > 0 || r.reserved_asap > 0 || r.reserved_scheduled > 0));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <PageTransition>
      <PageChrome
        icon="inventory"
        title="Stock commitments"
        description="SKU-level ASAP + scheduled demand vs on-hand"
        loading={loading}
        empty={!loading && items.length === 0}
        emptyMessage="Active orders have not reserved stock yet."
      >
        {items.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {items.map((row) => (
              <article key={row.sku_id} className="rounded-2xl border border-[var(--desk-border)] p-4">
                {row.image_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={row.image_url} alt="" className="h-16 w-16 rounded-lg object-cover mb-3" />
                ) : null}
                <h3 className="font-semibold">{row.name || row.sku_id}</h3>
                <p className="text-xs text-[var(--desk-text-tertiary)] mt-1">
                  Available {row.available_qty ?? Math.max(0, row.on_hand - row.reserved_asap - row.reserved_scheduled)} · ASAP {row.reserved_asap} · Scheduled {row.reserved_scheduled}
                </p>
                <p className="text-xs text-[var(--desk-text-tertiary)]">On hand {row.on_hand}</p>
                {row.deficit_qty > 0 ? (
                  <span className="inline-block mt-2 rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-800">
                    Short {row.deficit_qty}
                  </span>
                ) : null}
              </article>
            ))}
          </div>
        ) : !loading ? (
          <EmptyState headline="No commitments" body="Active orders have not reserved stock yet." />
        ) : null}
      </PageChrome>
    </PageTransition>
  );
}
