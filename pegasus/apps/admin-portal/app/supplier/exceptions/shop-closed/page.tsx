'use client';

import { useState, useEffect } from 'react';
import { apiFetch } from '@/lib/auth';
import { Button } from '@heroui/react';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import { buildSupplierShopClosedResolveIdempotencyKey } from '../../_shared/idempotency';

interface ShopClosedDTO {
  attempt_id: string;
  order_id: string;
  original_route_id: string;
  driver_id: string;
  retailer_id: string;
  resolution: string;
  created_at: string;
}

export default function ShopClosedExceptions() {
  const [data, setData] = useState<ShopClosedDTO[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchExceptions = () => {
    setLoading(true);
    apiFetch('/v1/admin/shop-closed/active')
    .then(res => res.json())
    .then(j => setData(j.data || []))
    .finally(() => setLoading(false));
  };

  useEffect(() => fetchExceptions(), []);

  const resolve = async (attemptId: string, action: string) => {
    const res = await apiFetch('/v1/admin/shop-closed/resolve', {
      method: 'POST',
      headers: {
        'Idempotency-Key': buildSupplierShopClosedResolveIdempotencyKey(attemptId, action),
      },
      body: JSON.stringify({ attempt_id: attemptId, action })
    });
    const body = await res.json().catch(() => ({} as { queued?: boolean }));
    if (body.queued) alert('Resolution queued — it will replay when back online');
    else if (res.ok) fetchExceptions();
    else alert('Failed to resolve');
  };

  if (!loading && data.length === 0) return <EmptyState icon="done_all" headline="No Escalations" body="You are all caught up." />;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-4">
      <div className="flex justify-between items-center mb-6">
        <h1 className="md-typescale-headline-medium">Shop Closed Escalations</h1>
        <Button className="md-btn md-btn-tonal" onPress={fetchExceptions}>
          <Icon name="refresh" /> Refresh
        </Button>
      </div>

      <div className="grid gap-4">
        {data.map(item => (
          <div key={item.attempt_id} className="md-card md-elevation-1 p-4 flex flex-col md:flex-row gap-4 items-start md:items-center">
            <div className="flex-1">
              <p className="md-typescale-title-medium font-bold">Order ID: {item.order_id}</p>
              <p className="text-sm text-(--color-md-on-surface-variant)">Driver: {item.driver_id} | Route: {item.original_route_id}</p>
              <span className={`mt-2 inline-block px-2 py-1 text-xs rounded md-shape-xs bg-yellow-100 text-yellow-800`}>
                {item.resolution}
              </span>
            </div>
            <div className="flex gap-2">
              <Button variant="primary" onPress={() => resolve(item.attempt_id, 'WAIT')}>Wait</Button>
              <Button variant="secondary" onPress={() => resolve(item.attempt_id, 'BYPASS')}>Bypass / Offload</Button>
              <Button variant="danger" onPress={() => resolve(item.attempt_id, 'RETURN_TO_DEPOT')}>Return to Depot</Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
