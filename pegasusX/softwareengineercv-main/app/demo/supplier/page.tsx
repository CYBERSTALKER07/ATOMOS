import { MOCK_ORDERS } from '../lib/mockData';
import { DemoPageHeader, KpiCard } from '../components/DemoUi';

export default function SupplierDashboard() {
  return (
    <div className="space-y-8">
      <DemoPageHeader
        title="Supplier Overview"
        subtitle="Monitor outbound fulfillment, order volume, and SLAs."
      />

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <KpiCard label="Total outbound" value="12.4k" delta="+14% vs last week" />
        <KpiCard label="Fulfillment rate" value="99.2%" delta="Target: 98.0%" />
        <KpiCard label="Avg pick time" value="42m" delta="+5m vs SLA" deltaPositive={false} />
      </div>

      <div className="overflow-hidden border border-white/10">
        <div className="flex items-center justify-between border-b border-white/10 px-5 py-4">
          <h2 className="text-sm font-medium">Recent wholesale orders</h2>
          <button
            type="button"
            className="border border-white/15 px-3 py-1.5 text-xs hover:bg-white/5"
          >
            Export CSV
          </button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-white/[0.02] font-mono text-[10px] uppercase text-white/40">
              <tr>
                <th className="px-5 py-3 font-normal">Order ID</th>
                <th className="px-5 py-3 font-normal">Item</th>
                <th className="px-5 py-3 text-right font-normal">Qty</th>
                <th className="px-5 py-3 font-normal">Destination</th>
                <th className="px-5 py-3 font-normal">ETA</th>
                <th className="px-5 py-3 font-normal">Status</th>
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
                      {order.status}
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
