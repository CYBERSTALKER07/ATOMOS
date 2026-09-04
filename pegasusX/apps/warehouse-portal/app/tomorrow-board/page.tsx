'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import { PageChrome } from '@/components/PageChrome';
import type { DeliveryExpectation } from '@pegasusx/types';

type BoardOrder = {
  order_id: string;
  status: string;
  retailer_id?: string;
  total_minor: number;
  delivery_expectation?: DeliveryExpectation;
};

type BoardResponse = {
  date: string;
  preorders: BoardOrder[];
  deliver_before: BoardOrder[];
  draft_manifests: Array<{ manifest_id: string; state: string; stop_count: number }>;
  loading_manifests: Array<{ manifest_id: string; state: string; stop_count: number }>;
};

export default function TomorrowBoardPage() {
  const t = usePortalT();
  const [board, setBoard] = useState<BoardResponse | null>(null);
  const [date, setDate] = useState(() => {
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    return tomorrow.toISOString().slice(0, 10);
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    void apiFetch(`/v1/warehouse/ops/board?date=${encodeURIComponent(date)}`)
      .then(async (res) => {
        const data = (await res.json()) as BoardResponse;
        setBoard(data);
      })
      .catch(() => setBoard(null))
      .finally(() => setLoading(false));
  }, [date]);

  const rows = useMemo(() => {
    if (!board) return [];
    return [
      ...board.preorders.map((o) => ({ ...o, lane: 'Pre-order' })),
      ...board.deliver_before.map((o) => ({ ...o, lane: 'Deliver by' })),
    ];
  }, [board]);

  return (
    <PageChrome title={t("portal.nav.tomorrow_board")} description={t("warehouse_portal.residual.text.orders_and_manifests_grouped_by_delivery_date")} loading={loading}>
      <div className="flex flex-col gap-4">
        <label className="text-sm">
          Date{' '}
          <input type="date" value={date} onChange={(e) => setDate(e.target.value)} className="ml-2 border rounded px-2 py-1" />
        </label>
        <div className="grid gap-3">
          {rows.length === 0 ? (
            <p className="text-sm text-[var(--muted)]">{t("warehouse_portal.tomorrow_board.text.no_orders_scheduled_for_this_date")}</p>
          ) : (
            rows.map((row) => (
              <div key={row.order_id} className="border border-[var(--border)] rounded-xl p-4">
                <div className="flex justify-between gap-3">
                  <div>
                    <div className="font-semibold">{row.lane}</div>
                    <Link href={`/orders/${row.order_id}`} className="text-sm underline">
                      {row.order_id}
                    </Link>
                  </div>
                  <div className="text-sm text-right">
                    {row.delivery_expectation?.target_label ?? row.status}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </PageChrome>
  );
}
