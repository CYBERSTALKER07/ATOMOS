import { MOCK_DELIVERIES, MOCK_ORDERS } from '../lib/mockData';

export default function RetailerDashboard() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-medium tracking-tight mb-2">Retailer Command Center</h1>
        <p className="text-white/50 text-sm">Inbound shipment tracking, route ETAs, and receiving workflows.</p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-[#0a0a0a] border border-white/5 p-6 rounded relative overflow-hidden">
          <div className="absolute top-0 right-0 p-4 opacity-10">
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
          </div>
          <div className="text-xs font-mono text-white/40 uppercase mb-4 relative z-10">In-Transit Volume</div>
          <div className="text-4xl font-light mb-1 relative z-10">4.2M</div>
          <div className="text-xs text-white/40 font-mono relative z-10">UNITS ARRIVING TODAY</div>
        </div>
        <div className="bg-[#0a0a0a] border border-white/5 p-6 rounded">
          <div className="text-xs font-mono text-white/40 uppercase mb-4">On-Time Delivery</div>
          <div className="text-4xl font-light mb-1">96.8%</div>
          <div className="text-xs text-green-400 font-mono">+1.2% vs trailing 30d</div>
        </div>
        <div className="bg-[#0a0a0a] border border-white/5 p-6 rounded">
          <div className="text-xs font-mono text-white/40 uppercase mb-4">Stockouts Prevented</div>
          <div className="text-4xl font-light mb-1">24</div>
          <div className="text-xs text-white/40 font-mono">VIA SMART ROUTING</div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Live Routes Map (Mock visual) */}
        <div className="lg:col-span-2 bg-[#0a0a0a] border border-white/5 rounded overflow-hidden flex flex-col h-[500px]">
          <div className="px-6 py-4 border-b border-white/5 bg-white/[0.02] flex items-center justify-between">
            <h2 className="text-sm font-medium text-white/90">Live Fleet Telemetry</h2>
            <div className="flex items-center gap-2 text-xs font-mono text-green-400">
              <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-ping" />
              TRACKING ACTIVE
            </div>
          </div>
          
          <div className="flex-1 relative bg-[#050505] overflow-hidden p-6 flex flex-col gap-4">
             {/* Map Grid Pattern */}
             <div className="absolute inset-0 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:40px_40px]" />
             
             {MOCK_DELIVERIES.map((route) => (
               <div key={route.routeId} className="relative z-10 bg-[#0a0a0a] border border-white/10 p-4 rounded shadow-2xl flex flex-col gap-3">
                 <div className="flex items-center justify-between">
                   <div className="flex items-center gap-3">
                     <div className="w-8 h-8 rounded bg-white/5 flex items-center justify-center">
                       <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/></svg>
                     </div>
                     <div>
                       <div className="text-sm font-medium text-white/90">{route.routeId}</div>
                       <div className="text-xs font-mono text-white/40">{route.driver}</div>
                     </div>
                   </div>
                   <span className={`px-2 py-1 text-[10px] uppercase font-mono rounded-sm border ${
                      route.status === 'Delayed' ? 'border-red-500/30 text-red-400 bg-red-500/10' :
                      'border-green-500/30 text-green-400 bg-green-500/10'
                    }`}>
                      {route.status}
                    </span>
                 </div>
                 
                 <div className="space-y-1">
                   <div className="flex justify-between text-[10px] font-mono text-white/50">
                     <span>Progress</span>
                     <span>{route.stopsCompleted} / {route.totalStops} STOPS</span>
                   </div>
                   <div className="w-full h-1.5 bg-white/5 rounded-full overflow-hidden">
                     <div 
                       className={`h-full rounded-full transition-all duration-1000 ${route.status === 'Delayed' ? 'bg-red-500' : 'bg-green-500'}`}
                       style={{ width: `${route.progress}%` }} 
                     />
                   </div>
                 </div>
               </div>
             ))}
          </div>
        </div>

        {/* Incoming Shipments */}
        <div className="bg-[#0a0a0a] border border-white/5 rounded overflow-hidden flex flex-col h-[500px]">
          <div className="px-6 py-4 border-b border-white/5 bg-white/[0.02]">
            <h2 className="text-sm font-medium text-white/90">Incoming POs</h2>
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-3">
            {MOCK_ORDERS.filter(o => o.destination.includes('Retailer')).map(order => (
              <div key={order.id} className="p-4 border border-white/5 bg-white/[0.01] rounded hover:border-white/20 transition-colors cursor-pointer group">
                <div className="flex justify-between items-start mb-2">
                  <div className="text-sm font-medium text-white/90 group-hover:text-white">{order.item}</div>
                  <div className="text-xs font-mono text-white/60">{order.eta}</div>
                </div>
                <div className="flex justify-between items-end">
                  <div className="text-xs text-white/40">From: {order.origin}</div>
                  <div className="text-xs font-mono text-white/80">{order.qty} Units</div>
                </div>
              </div>
            ))}
            
            {/* Empty state filler */}
            <div className="p-4 border border-white/5 border-dashed rounded text-center opacity-50">
               <div className="text-xs text-white/40 py-4">End of incoming manifest</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
