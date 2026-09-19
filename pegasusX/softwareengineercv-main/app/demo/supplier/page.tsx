'use client';

import { MOCK_ORDERS } from '../lib/mockData';
import { DemoPageHeader, KpiCard } from '../components/DemoUi';
import { useLanguage } from '@/app/context/LanguageContext';

const STATUS_MAP: Record<string, string> = {
  Delivered: 'demo_status_delivered',
  Shipped: 'demo_status_shipped',
  Processing: 'demo_status_processing',
  Picking: 'demo_status_picking',
  Packed: 'demo_status_packed',
};

export default function SupplierDashboard() {
  const { t } = useLanguage();

  return (
    <div className="space-y-8">
      <DemoPageHeader
        title={t('demo_supplier_title')}
        subtitle={t('demo_supplier_sub')}
      />

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <KpiCard label={t('demo_total_outbound')} value="12.4k" delta="+14%" />
        <KpiCard label={t('demo_fulfillment_rate')} value="99.2%" delta="Target: 98.0%" />
        <KpiCard label={t('demo_avg_pick_time')} value="42m" delta="+5m vs SLA" deltaPositive={false} />
      </div>

      <div className="overflow-hidden border border-white/10">
        <div className="flex items-center justify-between border-b border-white/10 px-5 py-4">
          <h2 className="text-sm font-medium">{t('demo_recent_orders')}</h2>
          <button
            type="button"
            className="border border-white/15 px-3 py-1.5 text-xs hover:bg-white/5"
          >
            {t('demo_export_csv')}
          </button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-white/[0.02] font-mono text-[10px] uppercase text-white/40">
              <tr>
                <th className="px-5 py-3 font-normal">{t('demo_col_order_id')}</th>
                <th className="px-5 py-3 font-normal">{t('demo_col_item')}</th>
                <th className="px-5 py-3 text-right font-normal">{t('demo_col_qty')}</th>
                <th className="px-5 py-3 font-normal">{t('demo_col_destination')}</th>
                <th className="px-5 py-3 font-normal">{t('demo_col_eta')}</th>
                <th className="px-5 py-3 font-normal">{t('demo_col_status')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {MOCK_ORDERS.map((order) => (
                <tr key={order.id} className="hover:bg-white/[0.02]">
                  <td className="px-5 py-3 font-mono text-white/80">{order.id}</td>
                  <td className="px-5 py-3">{order.item}</td>
                  <td className="px-5 py-3 text-right font-mono text-white/60">{order.qty}</td>
                  <td className="px-5 py-3 text-white/70">{order.destination}</td>
                  <td className="px-5 py-3 font-mono text-white/60">{order.eta}</td>
                  <td className="px-5 py-3">
                    <span className="border border-white/15 px-2 py-0.5 font-mono text-[10px] uppercase">
                      {t(STATUS_MAP[order.status] || order.status as any)}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
