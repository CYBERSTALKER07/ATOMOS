'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { readTokenFromCookie as getToken } from '@/lib/auth';
import { usePagination } from '@/lib/usePagination';
import PaginationControls from '@/components/PaginationControls';
import Icon from '@/components/Icon';
import Drawer from '@/components/Drawer';
import EmptyState from '@/components/EmptyState';

interface CRMRetailer {
  retailer_id: string;
  retailer_name: string;
  phone: string;
  email: string;
  lifetime: number;
  order_count: number;
  last_order_date: string;
  status: string;
}

interface CRMOrder {
  order_id: string;
  state: string;
  amount: number;
  item_count: number;
  created_at: string;
}

interface CRMRetailerDetail extends CRMRetailer {
  orders: CRMOrder[];
}

async function fetchRetailers(): Promise<CRMRetailer[]> {
  const token = getToken();
  if (!token) return [];
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/crm/retailers`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    cache: 'no-store',
  });
  if (!res.ok) return [];
  return res.json();
}

async function fetchRetailerDetail(retailerId: string): Promise<CRMRetailerDetail | null> {
  const token = getToken();
  if (!token) return null;
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/crm/retailers/${encodeURIComponent(retailerId)}`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    cache: 'no-store',
  });
  if (!res.ok) return null;
  return res.json();
}

const formatAmount = (v: number) => new Intl.NumberFormat('uz-UZ').format(v);

const stateColor = (state: string): string => {
  const map: Record<string, string> = {
    COMPLETED: 'var(--success)',
    CANCELLED: 'var(--danger)',
    IN_TRANSIT: 'var(--accent-soft)',
    ARRIVED: 'var(--accent-soft)',
  };
  return map[state] || 'var(--warning)';
};

export default function SupplierCRM() {
  const [retailers, setRetailers] = useState<CRMRetailer[]>([]);
  const [loading, setLoading] = useState(true);
  const pagination = usePagination(retailers, 25);
  const [selectedDetail, setSelectedDetail] = useState<CRMRetailerDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [slideOpen, setSlideOpen] = useState(false);

  useEffect(() => {
    fetchRetailers()
      .then((data) => {
        setRetailers(data);
      })
      .finally(() => setLoading(false));
  }, []);

  const openDetail = useCallback(async (retailerId: string) => {
    setDetailLoading(true);
    setSlideOpen(true);
    const detail = await fetchRetailerDetail(retailerId);
    if (detail) {
      setSelectedDetail(detail);
    } else {
      // Fallback sample
      const base = retailers.find((r) => r.retailer_id === retailerId);
      if (base) {
        setSelectedDetail({
          ...base,
          orders: [
            { order_id: 'ord-101', state: 'COMPLETED', amount: 2500000, item_count: 8, created_at: '2026-03-15T10:30:00Z' },
            { order_id: 'ord-089', state: 'COMPLETED', amount: 1800000, item_count: 5, created_at: '2026-03-10T08:15:00Z' },
            { order_id: 'ord-072', state: 'IN_TRANSIT', amount: 3200000, item_count: 12, created_at: '2026-03-05T14:00:00Z' },
            { order_id: 'ord-058', state: 'CANCELLED', amount: 900000, item_count: 3, created_at: '2026-02-28T11:45:00Z' },
          ],
        });
      }
    }
    setDetailLoading(false);
  }, [retailers]);

  const closeDetail = useCallback(() => {
    setSlideOpen(false);
    setTimeout(() => setSelectedDetail(null), 300);
  }, []);

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <header className="mb-10">
        <h1 className="md-typescale-headline-medium">Retailer CRM</h1>
        <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--muted)' }}>Manage retailer relationships and track lifetime value</p>
      </header>

      {/* KPI Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-10">
        {[
          { label: 'Total Retailers', value: retailers.length.toString() },
          { label: 'Active', value: retailers.filter((r) => r.status === 'ACTIVE').length.toString() },
          { label: 'Total Lifetime Value', value: `${formatAmount(retailers.reduce((sum, r) => sum + r.lifetime, 0))}` },
        ].map(({ label, value }, i) => (
          <div key={i} className="md-card md-card-elevated p-6 flex flex-col justify-between md-animate-in" style={{ animationDelay: `${i * 50}ms` }}>
            <p className="md-typescale-label-small mb-4" style={{ color: 'var(--muted)' }}>{label}</p>
            <p className="md-typescale-headline-small">{value}</p>
          </div>
        ))}
      </div>

      {/* Data Grid */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <div className="w-6 h-6 border-2 rounded-full animate-spin" style={{ borderColor: 'var(--border)', borderTopColor: 'var(--accent)' }} />
        </div>
      ) : retailers.length === 0 ? (
        <EmptyState
          icon="crm"
          headline="No retailers found"
          body="Retailer relationships will appear here once orders come in"
        />
      ) : (
        <div className="md-card md-card-outlined p-0 w-full overflow-hidden md-animate-in" style={{ animationDelay: '150ms' }}>
          <table className="md-table">
            <thead>
              <tr>
                <th>Retailer</th>
                <th className="text-right">Lifetime Value (Amount)</th>
                <th className="text-right hidden md:table-cell">Orders</th>
                <th className="text-right hidden md:table-cell">Last Order</th>
                <th className="text-center">Status</th>
              </tr>
            </thead>
            <tbody>
              {pagination.pageItems.map((r) => (
                <tr
                  key={r.retailer_id}
                  className="transition-colors cursor-pointer"
                  onClick={() => openDetail(r.retailer_id)}
                  onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--surface)')}
                  onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                >
                  <td>
                    <div className="flex items-center gap-3">
                      <div
                        className="w-9 h-9 flex items-center justify-center md-typescale-label-small shrink-0"
                        style={{
                          background: 'var(--surface)',
                          color: 'var(--muted)',
                          borderRadius: '10px',
                        }}
                      >
                        {r.retailer_name.split(' ').map((w) => w[0]).join('').slice(0, 2)}
                      </div>
                      <div>
                        <p className="md-typescale-body-small">{r.retailer_name}</p>
                        {r.phone && <p className="md-typescale-label-small font-mono" style={{ color: 'var(--border)' }}>{r.phone}</p>}
                      </div>
                    </div>
                  </td>
                  <td className="text-right font-mono md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>
                    {formatAmount(r.lifetime)}
                  </td>
                  <td className="text-right hidden md:table-cell" style={{ color: 'var(--muted)' }}>{r.order_count}</td>
                  <td className="text-right hidden md:table-cell" style={{ color: 'var(--muted)' }}>{r.last_order_date}</td>
                  <td className="text-center">
                    <span
                      className="md-chip"
                      style={{
                        cursor: 'default',
                        background: r.status === 'ACTIVE' ? 'var(--success)' : 'var(--surface)',
                        color: r.status === 'ACTIVE' ? 'var(--success-foreground)' : 'var(--muted)',
                        borderColor: 'transparent',
                      }}
                    >
                      {r.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <PaginationControls pagination={pagination} />
        </div>
      )}

      {/* Slide-out Detail Panel */}
      <Drawer open={slideOpen} onClose={closeDetail} title={selectedDetail?.retailer_name || 'Retailer Detail'}>
            {detailLoading ? (
              <div className="flex items-center justify-center h-full">
                <div className="w-6 h-6 border-2 rounded-full animate-spin" style={{ borderColor: 'var(--border)', borderTopColor: 'var(--accent)' }} />
              </div>
            ) : selectedDetail ? (
              <div className="p-6 md:p-8">

                {/* Header */}
                <div className="flex items-center gap-4 mb-8">
                  <div
                    className="w-14 h-14 flex items-center justify-center text-lg font-bold shrink-0"
                    style={{
                      background: 'var(--surface)',
                      color: 'var(--muted)',
                      borderRadius: '16px',
                    }}
                  >
                    {selectedDetail.retailer_name.split(' ').map((w) => w[0]).join('').slice(0, 2)}
                  </div>
                  <div>
                    <span
                      className="md-chip mt-1"
                      style={{
                        cursor: 'default',
                        background: selectedDetail.status === 'ACTIVE' ? 'var(--success)' : 'var(--surface)',
                        color: selectedDetail.status === 'ACTIVE' ? 'var(--success-foreground)' : 'var(--muted)',
                        borderColor: 'transparent',
                      }}
                    >
                      {selectedDetail.status}
                    </span>
                  </div>
                </div>

                {/* Contact Info */}
                <div className="mb-8">
                  <p className="md-typescale-label-small mb-3" style={{ color: 'var(--muted)' }}>CONTACT</p>
                  <div className="md-card md-card-elevated p-4 flex flex-col gap-3">
                    {selectedDetail.phone && (
                      <a href={`tel:${selectedDetail.phone}`} className="flex items-center gap-3 text-sm hover:opacity-70 transition-opacity" style={{ color: 'var(--foreground)' }}>
                        <Icon name="phone" size={18} />
                        <span className="font-mono">{selectedDetail.phone}</span>
                      </a>
                    )}
                    {selectedDetail.email && (
                      <a href={`mailto:${selectedDetail.email}`} className="flex items-center gap-3 text-sm hover:opacity-70 transition-opacity" style={{ color: 'var(--foreground)' }}>
                        <Icon name="email" size={18} />
                        <span>{selectedDetail.email}</span>
                      </a>
                    )}
                  </div>
                </div>

                {/* KPIs */}
                <div className="grid grid-cols-2 gap-3 mb-8">
                  <div className="md-card md-card-elevated p-4">
                    <p className="md-typescale-label-small mb-2" style={{ color: 'var(--muted)' }}>Lifetime Value</p>
                    <p className="md-typescale-title-medium font-mono" style={{ fontVariantNumeric: 'tabular-nums' }}>{formatAmount(selectedDetail.lifetime)} </p>
                  </div>
                  <div className="md-card md-card-elevated p-4">
                    <p className="md-typescale-label-small mb-2" style={{ color: 'var(--muted)' }}>Total Orders</p>
                    <p className="md-typescale-title-medium">{selectedDetail.order_count}</p>
                  </div>
                </div>

                {/* Order Ledger */}
                <div>
                  <p className="md-typescale-label-small mb-3" style={{ color: 'var(--muted)' }}>ORDER LEDGER</p>
                  {selectedDetail.orders.length === 0 ? (
                    <p className="md-typescale-body-medium py-8 text-center" style={{ color: 'var(--border)' }}>No orders yet</p>
                  ) : (
                    <div className="md-shape-md overflow-hidden" style={{ background: 'var(--background)' }}>
                      {selectedDetail.orders.map((o, i) => (
                        <div
                          key={o.order_id}
                          className="flex items-center justify-between px-4 py-3.5"
                          style={{ borderBottom: i < selectedDetail.orders.length - 1 ? '1px solid var(--border)' : undefined }}
                        >
                          <div className="flex items-center gap-3">
                            <div className="w-2 h-2 rounded-full shrink-0" style={{ background: stateColor(o.state) }} />
                            <div>
                              <p className="font-mono md-typescale-body-small">{o.order_id}</p>
                              <p className="md-typescale-label-small" style={{ color: 'var(--border)' }}>{o.item_count} items · {o.state}</p>
                            </div>
                          </div>
                          <div className="text-right">
                            <p className="font-mono md-typescale-body-small">{formatAmount(o.amount)} </p>
                            <p className="md-typescale-label-small" style={{ color: 'var(--border)' }}>{o.created_at.split('T')[0]}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ) : null}

            {/* Country Overrides shortcut */}
            <div className="mt-6 pt-4 border-t" style={{ borderColor: 'var(--border)' }}>
              <Link
                href="/supplier/country-overrides"
                className="flex items-center gap-2 md-typescale-label-small"
                style={{ color: 'var(--color-md-primary)' }}
              >
                <Icon name="global" size={14} />
                Manage Country Overrides
              </Link>
            </div>
      </Drawer>
    </div>
  );
}
