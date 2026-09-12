'use client';

import { MOCK_DELIVERIES, MOCK_ORDERS } from '../lib/mockData';
import { DemoPageHeader, KpiCard } from '../components/DemoUi';
import { useLanguage } from '@/app/context/LanguageContext';

const STATUS_KEY: Record<string, string> = {
  'On Route': 'demo_status_on_route',
  Delayed: 'demo_status_delayed',
  Delivered: 'demo_status_delivered',
};

export default function RetailerDashboard() {
  const { t } = useLanguage();

  return (
    <div className="space-y-8">
      <DemoPageHeader
        title={t('demo_retailer_title')}
        subtitle={t('demo_retailer_sub')}
      />

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <KpiCard label={t('demo_rt_in_transit')} value="4.2M" delta={t('demo_rt_units')} />
        <KpiCard label={t('demo_rt_on_time')} value="96.8%" delta="+1.2%" />
        <KpiCard label={t('demo_rt_stockouts')} value="24" delta="Smart routing" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Live Routes Map (Mock visual) */}
        <div className="lg:col-span-2 bg-[#0a0a0a] border border-white/5 rounded overflow-hidden flex flex-col h-[500px]">
          <div className="px-6 py-4 border-b border-white/5 bg-white/[0.02] flex items-center justify-between">
            <h2 className="text-sm font-medium text-white/90">{t('demo_rt_fleet_telemetry')}</h2>
            <div className="flex items-center gap-2 text-xs font-mono text-green-400">
              <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-ping" />
              {t('demo_rt_tracking_active')}
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
                      {t(STATUS_KEY[route.status] || route.status as any)}
                    </span>
                 </div>
                 
                 <div className="space-y-1">
                   <div className="flex justify-between text-[10px] font-mono text-white/50">
                     <span>{t('demo_rt_progress')}</span>
                     <span>{route.stopsCompleted} / {route.totalStops} {t('demo_rt_stops')}</span>
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
            <h2 className="text-sm font-medium text-white/90">{t('demo_rt_incoming_pos')}</h2>
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-3">
            {MOCK_ORDERS.filter(o => o.destination.includes('Retailer')).map(order => (
              <div key={order.id} className="p-4 border border-white/5 bg-white/[0.01] rounded hover:border-white/20 transition-colors cursor-pointer group">
                <div className="flex justify-between items-start mb-2">
                  <div className="text-sm font-medium text-white/90 group-hover:text-white">{order.item}</div>
                  <div className="text-xs font-mono text-white/60">{order.eta}</div>
                </div>
                <div className="flex justify-between items-end">
                  <div className="text-xs text-white/40">{t('demo_rt_from')}: {order.origin}</div>
                  <div className="text-xs font-mono text-white/80">{order.qty} {t('demo_rt_units')}</div>
                </div>
              </div>
            ))}
            
            {/* Empty state filler */}
            <div className="p-4 border border-white/5 border-dashed rounded text-center opacity-50">
               <div className="text-xs text-white/40 py-4">{t('demo_rt_end_manifest')}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
