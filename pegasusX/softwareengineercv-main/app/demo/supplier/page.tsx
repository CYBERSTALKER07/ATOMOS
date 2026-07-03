import { MOCK_ORDERS } from '../lib/mockData';

export default function SupplierDashboard() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-medium tracking-tight mb-2">Supplier Overview</h1>
        <p className="text-white/50 text-sm">Monitor outbound fulfillment, order volume, and SLAs.</p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-[#0a0a0a] border border-white/5 p-6 rounded flex flex-col gap-4">
          <div className="text-xs font-mono text-white/40 uppercase">Total Outbound</div>
          <div className="text-4xl font-light">12.4k</div>
          <div className="text-xs text-green-400 font-mono">+14% vs last week</div>
        </div>
        <div className="bg-[#0a0a0a] border border-white/5 p-6 rounded flex flex-col gap-4">
          <div className="text-xs font-mono text-white/40 uppercase">Fulfillment Rate</div>
          <div className="text-4xl font-light">99.2%</div>
          <div className="text-xs text-white/40 font-mono">Target: 98.0%</div>
        </div>
        <div className="bg-[#0a0a0a] border border-white/5 p-6 rounded flex flex-col gap-4">
          <div className="text-xs font-mono text-white/40 uppercase">Avg Pick Time</div>
          <div className="text-4xl font-light">42m</div>
          <div className="text-xs text-red-400 font-mono">+5m vs SLA</div>
        </div>
      </div>

      {/* Recent Orders */}
      <div className="bg-[#0a0a0a] border border-white/5 rounded overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5 bg-white/[0.02] flex items-center justify-between">
          <h2 className="text-sm font-medium text-white/90">Recent Wholesale Orders</h2>
          <button className="text-xs bg-white/5 border border-white/10 px-3 py-1.5 rounded-sm hover:bg-white/10 transition-colors">
            Export CSV
          </button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="text-xs text-white/40 font-mono uppercase bg-white/[0.01]">
              <tr>
                <th className="px-6 py-4 font-normal">Order ID</th>
                <th className="px-6 py-4 font-normal">Item</th>
                <th className="px-6 py-4 font-normal text-right">Qty</th>
                <th className="px-6 py-4 font-normal">Destination</th>
                <th className="px-6 py-4 font-normal">ETA</th>
                <th className="px-6 py-4 font-normal">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {MOCK_ORDERS.map((order) => (
                <tr key={order.id} className="hover:bg-white/[0.02] transition-colors">
                  <td className="px-6 py-4 font-mono text-white/80">{order.id}</td>
                  <td className="px-6 py-4 text-white/90">{order.item}</td>
                  <td className="px-6 py-4 text-right font-mono text-white/60">{order.qty}</td>
                  <td className="px-6 py-4 text-white/70">{order.destination}</td>
                  <td className="px-6 py-4 font-mono text-white/60">{order.eta}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2.5 py-1 text-[10px] uppercase tracking-wider font-mono rounded-sm border ${
                      order.status === 'Delivered' ? 'border-green-500/30 text-green-400 bg-green-500/10' :
                      order.status === 'Shipped' ? 'border-blue-500/30 text-blue-400 bg-blue-500/10' :
                      order.status === 'Processing' ? 'border-yellow-500/30 text-yellow-400 bg-yellow-500/10' :
                      'border-white/10 text-white/70 bg-white/5'
                    }`}>
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
