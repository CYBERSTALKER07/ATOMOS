'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';

type ReverseTask = {
  TaskId?: string;
  task_id?: string;
  OrderId?: string;
  order_id?: string;
  Status?: string;
  status?: string;
  ExpectedQtyJson?: string;
  expected_qty_json?: string;
};

const taskId = (t: ReverseTask) => t.TaskId ?? t.task_id ?? '';
const orderId = (t: ReverseTask) => t.OrderId ?? t.order_id ?? '';

export function ReverseLogisticsPanel() {
  const t = usePortalT();
  const [tasks, setTasks] = useState<ReverseTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const wh = warehouseHomeNodeId() || '';
      const q = new URLSearchParams({ status: 'OPEN' });
      if (wh) q.set('warehouse_id', wh);
      const res = await apiFetch(`/v1/warehouse/reverse-logistics?${q.toString()}`);
      if (!res.ok) throw new Error(`load_${res.status}`);
      const body = await res.json();
      setTasks(body.tasks ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function receive(task: ReverseTask) {
    const id = taskId(task);
    if (!id) return;
    const wh = warehouseHomeNodeId() || 'warehouse';
    let receivedQty: Record<string, number> = {};
    try {
      const raw = task.ExpectedQtyJson ?? task.expected_qty_json;
      if (raw) receivedQty = JSON.parse(typeof raw === 'string' ? raw : JSON.stringify(raw));
    } catch {
      receivedQty = {};
    }
    const res = await apiFetch(`/v1/warehouse/reverse-logistics/${encodeURIComponent(id)}/receive`, {
      method: 'POST',
      body: JSON.stringify({ warehouse_id: wh, received_qty: receivedQty }),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      setError((body as { error?: string }).error ?? 'receive_failed');
      return;
    }
    await load();
  }

  if (loading) return <p className="text-sm text-gray-500">{t("warehouse_portal.returns.reverse_logistics_panel.text.loading_reverse_tasks")}</p>;
  if (error) return <p className="text-sm text-red-600">{error}</p>;
  if (tasks.length === 0) return <p className="text-sm text-gray-500">{t("warehouse_portal.returns.reverse_logistics_panel.text.no_open_credit_note_reverse_tasks")}</p>;

  return (
    <ul className="divide-y text-sm">
      {tasks.map((t) => (
        <li key={taskId(t)} className="py-2 flex flex-wrap gap-3 items-center">
          <span className="font-mono">{taskId(t)}</span>
          <span>Order {orderId(t)}</span>
          <button type="button" className="button--primary px-3 py-1 rounded" onClick={() => void receive(t)}>
            Receive
          </button>
        </li>
      ))}
    </ul>
  );
}
