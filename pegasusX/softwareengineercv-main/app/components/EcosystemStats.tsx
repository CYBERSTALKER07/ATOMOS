'use client';

import { useState } from 'react';
import PageSection from './layout/PageSection';

const TAB_DATA = {
  Supplier: {
    card1: { title: 'Order Volume', subtitle: 'Active outbound processing', value: '8.2K', unit: 'Daily Orders', percent: 92, stat1: '99%', label1: 'FILL RATE', stat2: '2M', label2: 'UNITS', metric: '98.7%' },
    card2: { title: 'Dispatch SLA', subtitle: 'On-time departure monitoring', metric: '99.99%', bars: [45, 75, 35, 90, 55, 100, 40, 65] },
    card3: { title: 'Inventory Health', subtitle: 'Global stock availability', value: '94%', stat1: '12K', label1: 'SKUS', stat2: '45', label2: 'SITES', metric: '8.4M' },
  },
  Warehouse: {
    card1: { title: 'Dock Utilization', subtitle: 'Live gate processing', value: '14', unit: 'Active Gates', percent: 75, stat1: '85%', label1: 'UTIL', stat2: '1.2K', label2: 'PALLETS', metric: '92.4%' },
    card2: { title: 'Cross-dock Time', subtitle: 'Internal transit SLAs', metric: '42m', bars: [60, 50, 80, 40, 70, 90, 50, 85] },
    card3: { title: 'Throughput', subtitle: 'Hourly volume processed', value: '840', stat1: '150', label1: 'TRUCKS', stat2: '3', label2: 'SHIFTS', metric: '1.2M' },
  },
  Retailer: {
    card1: { title: 'Delivery Status', subtitle: 'Live fleet telemetry', value: '142', unit: 'Active Routes', percent: 88, stat1: '96%', label1: 'ON-TIME', stat2: '4K', label2: 'STOPS', metric: '96.2%' },
    card2: { title: 'Unload SLA', subtitle: 'Turnaround time monitoring', metric: '18m', bars: [30, 40, 60, 35, 80, 55, 90, 45] },
    card3: { title: 'Received Volume', subtitle: 'Daily units received', value: '12.5K', stat1: '99%', label1: 'MATCH', stat2: '15', label2: 'DC', metric: '3.4M' },
  }
} as const;

export default function EcosystemStats() {
  const [activeTab, setActiveTab] = useState<keyof typeof TAB_DATA>('Supplier');

  const data = TAB_DATA[activeTab];

  return (
    <PageSection className="bg-[#050505] w-full !py-32 md:!py-48 border-t border-white/5 relative overflow-hidden" aria-labelledby="ecosystem-stats-heading">

      {/* Background ambient light */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full max-w-[1000px] h-[400px] bg-white/[0.02] blur-[100px] pointer-events-none rounded-full" />

      <div className="max-w-[1200px] mx-auto px-4 md:px-8 relative z-10">

        {/* Header */}
        <div className="mb-12">
          <div className="flex items-center gap-3 text-white/40 mb-6">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path d="M4 15l8-8 8 8" />
            </svg>
            <span className="text-[10px] tracking-[0.2em] uppercase font-mono">Ecosystem Statistics</span>
          </div>
          <h2 id="ecosystem-stats-heading" className="text-5xl md:text-6xl font-medium tracking-tight mb-6 text-white">
            Optimized for the entire chain
          </h2>
          <p className="text-white/50 max-w-2xl text-sm md:text-base leading-relaxed">
            Monitor every network pulse in real-time. Pegasus provides deep telemetry into supplier fulfillment, warehouse operations, and retailer deliveries.
          </p>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-10 border-b border-white/10 pb-0 overflow-x-auto no-scrollbar">
          {(Object.keys(TAB_DATA) as Array<keyof typeof TAB_DATA>).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-6 py-4 text-sm font-medium tracking-wide transition-colors relative whitespace-nowrap ${activeTab === tab ? 'text-white' : 'text-white/40 hover:text-white/80'
                }`}
            >
              {tab}
              {activeTab === tab && (
                <div className="absolute bottom-0 left-0 w-full h-[2px] bg-white shadow-[0_0_10px_rgba(255,255,255,0.5)]" />
              )}
            </button>
          ))}
        </div>

        {/* 3-Card Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

          {/* Card 1: Circle Gauge */}
          <div className="bg-[#0a0a0a] border border-white/5 p-8 rounded flex flex-col relative h-[460px] shadow-2xl">
            <div className="flex justify-between items-start mb-2">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 border border-white/10 flex items-center justify-center text-white/40 bg-white/5 rounded-sm">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M18 6L6 18M6 6l12 12" strokeWidth="1.5" strokeLinecap="square" /></svg>
                </div>
                <span className="text-base font-medium text-white/90">{data.card1.title}</span>
              </div>
              <span className="text-xs font-mono text-white/40">{data.card1.metric}</span>
            </div>
            <div className="text-sm text-white/40 mb-8">{data.card1.subtitle}</div>

            <div className="flex-1 flex items-center justify-center relative">
              <svg width="260" height="260" className="-rotate-90 drop-shadow-xl">
                <circle cx="130" cy="130" r="100" fill="none" stroke="rgba(255,255,255,0.03)" strokeWidth="24" />
                {/* ticks */}
                <g stroke="rgba(255,255,255,0.15)" strokeWidth="1.5">
                  {[...Array(40)].map((_, i) => (
                    <line key={i} x1="130" y1="16" x2="130" y2="24" transform={`rotate(${i * 9} 130 130)`} />
                  ))}
                </g>
                <circle cx="130" cy="130" r="100" fill="none" stroke="white" strokeWidth="24"
                  strokeDasharray={2 * Math.PI * 100}
                  strokeDashoffset={(2 * Math.PI * 100) * (1 - data.card1.percent / 100)}
                  className="transition-all duration-1000 ease-out"
                />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                <span className="text-5xl font-light tracking-tight text-white">{data.card1.value}</span>
                <span className="text-xs font-mono text-white/40 mt-1 uppercase">{data.card1.unit}</span>
              </div>

              <div className="absolute left-0 bottom-4 flex flex-col gap-4">
                <div>
                  <div className="text-xs text-white/90">{data.card1.stat1}</div>
                  <div className="text-[9px] tracking-wider text-white/40 font-mono mt-0.5 uppercase">{data.card1.label1}</div>
                </div>
                <div>
                  <div className="text-xs text-white/90">{data.card1.stat2}</div>
                  <div className="text-[9px] tracking-wider text-white/40 font-mono mt-0.5 uppercase">{data.card1.label2}</div>
                </div>
              </div>
            </div>
          </div>

          {/* Card 2: Bar Chart */}
          <div className="bg-[#0a0a0a] border border-white/5 p-8 rounded flex flex-col relative h-[460px] shadow-2xl">
            <div className="flex justify-between items-start mb-2">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 border border-white/10 flex items-center justify-center text-white/40 bg-white/5 rounded-sm">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M18 6L6 18M6 6l12 12" strokeWidth="1.5" strokeLinecap="square" /></svg>
                </div>
                <span className="text-base font-medium text-white/90">{data.card2.title}</span>
              </div>
              <span className="text-xs font-mono text-white/40">{data.card2.metric}</span>
            </div>
            <div className="text-sm text-white/40 mb-10">{data.card2.subtitle}</div>

            <div className="flex-1 flex items-end justify-between relative px-2 pb-6">
              <div className="absolute top-[35%] left-0 right-0 border-t border-dashed border-white/10" />
              <div className="absolute top-[35%] left-4 -translate-y-1/2 bg-white text-black text-xs font-mono px-3 py-1 rounded-sm z-10">
                SLA TARGET
              </div>

              {data.card2.bars.map((h, i) => (
                <div key={i} className="flex flex-col items-center gap-1.5 relative z-0 h-[220px] justify-end">
                  {i === 4 ? (
                    <div className="w-3 h-3 rounded-full bg-gradient-to-tr from-[#FF3366] to-[#33CCFF] absolute -top-5 shadow-[0_0_12px_rgba(255,51,102,0.8)] z-10" />
                  ) : (
                    <div className="w-2 h-2 rounded-full bg-white/80 absolute -top-4" />
                  )}
                  <div className="w-1 bg-white/10 transition-all duration-700 ease-out" style={{ height: `${h * 1.8}px` }} />
                </div>
              ))}
            </div>
          </div>

          {/* Card 3: Speedometer */}
          <div className="bg-[#0a0a0a] border border-white/5 p-8 rounded flex flex-col relative h-[460px] shadow-2xl">
            <div className="flex justify-between items-start mb-2">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 border border-white/10 flex items-center justify-center text-white/40 bg-white/5 rounded-sm">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M18 6L6 18M6 6l12 12" strokeWidth="1.5" strokeLinecap="square" /></svg>
                </div>
                <span className="text-base font-medium text-white/90">{data.card3.title}</span>
              </div>
              <span className="text-xs font-mono text-white/40">{data.card3.metric}</span>
            </div>
            <div className="text-sm text-white/40 mb-8">{data.card3.subtitle}</div>

            <div className="flex-1 flex flex-col items-center justify-end relative pt-12 pb-4">
              <svg width="320" height="170" className="overflow-visible drop-shadow-xl">
                <g className="text-white/15" strokeWidth="1.5">
                  {[...Array(35)].map((_, i) => (
                    <line key={i} x1="160" y1="16" x2="160" y2="24" transform={`rotate(${i * 5 - 85} 160 160)`} stroke="currentColor" />
                  ))}
                </g>
                <path d="M 20 160 A 140 140 0 0 1 300 160" fill="none" stroke="rgba(255,255,255,0.03)" strokeWidth="36" />
                <path d="M 20 160 A 140 140 0 0 1 300 160" fill="none" stroke="white" strokeWidth="36"
                  strokeDasharray={Math.PI * 140} strokeDashoffset={(Math.PI * 140) * 0.3}
                  className="transition-all duration-1000 ease-out"
                />
              </svg>

              <div className="absolute top-[120px] flex flex-col items-center pointer-events-none">
                <span className="text-6xl font-light tracking-tight text-white">{data.card3.value}</span>
              </div>

              <div className="w-full flex justify-between px-8 mt-8">
                <div className="text-center">
                  <div className="text-xs text-white/90">{data.card3.stat1}</div>
                  <div className="text-[9px] tracking-wider text-white/40 font-mono mt-0.5 uppercase">{data.card3.label1}</div>
                </div>
                <div className="text-center">
                  <div className="text-xs text-white/90">{data.card3.stat2}</div>
                  <div className="text-[9px] tracking-wider text-white/40 font-mono mt-0.5 uppercase">{data.card3.label2}</div>
                </div>
              </div>
            </div>
          </div>

        </div>
      </div>
    </PageSection>
  );
}
