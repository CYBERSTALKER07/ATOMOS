import { MOCK_INVENTORY, MOCK_GATES } from '../lib/mockData';
import { DemoPageHeader, KpiCard } from '../components/DemoUi';

export default function WarehouseDashboard() {
  return (
    <div className="space-y-8">
      <DemoPageHeader
        title="Warehouse Operations"
        subtitle="Live dock utilization, inbound/outbound queues, and inventory health."
      />

      <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
        <KpiCard label="Live gate util" value="85%" delta="12/14 gates" />
        <KpiCard label="Avg cross-dock" value="42m" delta="-3m vs SLA" />
        <KpiCard label="Throughput (1h)" value="840" delta="Pallets" />
        <KpiCard label="Critical SKUs" value="12" delta="Action required" deltaPositive={false} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Gate Status */}
        <div className="bg-[#0a0a0a] border border-white/5 rounded overflow-hidden">
          <div className="px-5 py-4 border-b border-white/5 bg-white/[0.02]">
            <h2 className="text-sm font-medium text-white/90">Dock Board (Live)</h2>
          </div>
          <div className="p-5 space-y-4">
            {MOCK_GATES.map((gate) => (
              <div key={gate.gateId} className="flex items-center justify-between p-3 border border-white/5 bg-white/[0.01] rounded">
                <div className="flex items-center gap-4">
                  <div className={`w-10 h-10 rounded flex items-center justify-center font-mono text-xs ${
                    gate.status === 'Available' ? 'bg-green-500/10 text-green-400 border border-green-500/20' :
                    gate.status === 'Occupied' ? 'bg-blue-500/10 text-blue-400 border border-blue-500/20' :
                    'bg-white/5 text-white/40 border border-white/10'
                  }`}>
                    {gate.gateId}
                  </div>
                  <div>
                    <div className="text-sm text-white/90 font-medium">{gate.status} <span className="text-white/40 font-normal ml-1">· {gate.type}</span></div>
                    <div className="text-xs text-white/40 font-mono mt-1">{gate.carrier}</div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-xs text-white/60 font-mono">{gate.actualTime}</div>
                  <div className="text-[10px] text-white/30 font-mono mt-0.5">SCHED: {gate.scheduledTime}</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Inventory Heatmap / Status */}
        <div className="bg-[#0a0a0a] border border-white/5 rounded overflow-hidden">
          <div className="px-5 py-4 border-b border-white/5 bg-white/[0.02]">
            <h2 className="text-sm font-medium text-white/90">Critical Inventory</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="text-[10px] text-white/40 font-mono uppercase bg-white/[0.01]">
                <tr>
                  <th className="px-5 py-3 font-normal">SKU</th>
                  <th className="px-5 py-3 font-normal text-right">In Stock</th>
                  <th className="px-5 py-3 font-normal text-right">Rsvd</th>
                  <th className="px-5 py-3 font-normal">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {MOCK_INVENTORY.map((inv) => (
                  <tr key={inv.sku} className="hover:bg-white/[0.02] transition-colors">
                    <td className="px-5 py-3">
                      <div className="font-mono text-white/80">{inv.sku}</div>
                      <div className="text-xs text-white/40 truncate max-w-[150px]">{inv.name}</div>
                    </td>
                    <td className="px-5 py-3 text-right font-mono text-white/90">{inv.inStock}</td>
                    <td className="px-5 py-3 text-right font-mono text-white/60">{inv.reserved}</td>
                    <td className="px-5 py-3">
                      <span className={`px-2 py-0.5 text-[9px] uppercase tracking-wider font-mono rounded-sm border ${
                        inv.status === 'Critical' ? 'border-red-500/30 text-red-400 bg-red-500/10' :
                        inv.status === 'Low Stock' ? 'border-yellow-500/30 text-yellow-400 bg-yellow-500/10' :
                        'border-green-500/30 text-green-400 bg-green-500/10'
                      }`}>
                        {inv.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
