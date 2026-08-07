'use client';

import { useState, useRef, useEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import PageSection from './layout/PageSection';
import SystemLoadWidget from './SystemLoadWidget';
import { useLanguage } from '../context/LanguageContext';

gsap.registerPlugin(ScrollTrigger);

const SIDEBAR_NAV = [
  { id: 'supplier', label: 'Supplier Operations', icon: 'M4 19a2 2 0 1 0 4 0a2 2 0 0 0 -4 0 M3.1 17l1.4 -6.2a2 2 0 0 1 1.9 -1.6h7.2a2 2 0 0 1 1.9 1.6l1.4 6.2 M2 9h10 M17 17a2 2 0 1 0 4 0a2 2 0 0 0 -4 0 M15.1 17l1.4 -6.2a2 2 0 0 1 1.9 -1.6h1.2' },
  { id: 'warehouse', label: 'Warehouse Control', icon: 'M3 21v-14l8 -4l8 4v14 M8 21v-4a2 2 0 0 1 2 -2h4a2 2 0 0 1 2 2v4 M8 10h8 M8 13h8' },
  { id: 'retailer', label: 'Retailer Network', icon: 'M3 21l18 0 M3 7v1a3 3 0 0 0 6 0v-1m0 1a3 3 0 0 0 6 0v-1m0 1a3 3 0 0 0 6 0v-1h-18l2 -4h14l2 4' },
  { id: 'fleet', label: 'Fleet Telemetry', icon: 'M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0 -18 0 M12 12m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0 M12 14l5.5 5.5' },
];

const TAB_DATA = {
  supplier: {
    title: 'Supplier Operations Center',
    subtitle: 'Real-time visibility into outbound fulfillment and dispatch',
    card1: { title: 'Order Volume', subtitle: 'Active outbound processing', value: '8.2K', unit: 'Daily Orders', percent: 92, stat1: '99%', label1: 'FILL RATE', stat2: '2M', label2: 'UNITS', metric: '98.7%' },
    card2: { title: 'Dispatch SLA', subtitle: 'On-time departure monitoring', metric: '99.99%', bars: [45, 75, 35, 90, 55, 100, 40, 65] },
    card3: { title: 'Inventory Health', subtitle: 'Global stock availability', value: '94%', stat1: '12K', label1: 'SKUS', stat2: '45', label2: 'SITES', metric: '8.4M' },
  },
  warehouse: {
    title: 'Warehouse Control Tower',
    subtitle: 'Live gate, dock, and throughput monitoring across DCs',
    card1: { title: 'Dock Utilization', subtitle: 'Live gate processing', value: '14', unit: 'Active Gates', percent: 75, stat1: '85%', label1: 'UTIL', stat2: '1.2K', label2: 'PALLETS', metric: '92.4%' },
    card2: { title: 'Cross-dock Time', subtitle: 'Internal transit SLAs', metric: '42m', bars: [60, 50, 80, 40, 70, 90, 50, 85] },
    card3: { title: 'Throughput', subtitle: 'Hourly volume processed', value: '840', stat1: '150', label1: 'TRUCKS', stat2: '3', label2: 'SHIFTS', metric: '1.2M' },
  },
  retailer: {
    title: 'Retailer Network Hub',
    subtitle: 'Store delivery statuses and unloading turnaround metrics',
    card1: { title: 'Delivery Status', subtitle: 'Live fleet telemetry', value: '142', unit: 'Active Routes', percent: 88, stat1: '96%', label1: 'ON-TIME', stat2: '4K', label2: 'STOPS', metric: '96.2%' },
    card2: { title: 'Unload SLA', subtitle: 'Turnaround time monitoring', metric: '18m', bars: [30, 40, 60, 35, 80, 55, 90, 45] },
    card3: { title: 'Received Volume', subtitle: 'Daily units received', value: '12.5K', stat1: '99%', label1: 'MATCH', stat2: '15', label2: 'DC', metric: '3.4M' },
  },
  fleet: {
    title: 'Global Fleet Telemetry',
    subtitle: 'Live vehicle tracking, fuel consumption, and route efficiency',
    card1: { title: 'Active Vehicles', subtitle: 'Vehicles currently on route', value: '450', unit: 'Trucks', percent: 95, stat1: '1.2K', label1: 'DRIVERS', stat2: '99%', label2: 'UPTIME', metric: '99.9%' },
    card2: { title: 'Fuel Efficiency', subtitle: 'Average MPG performance', metric: '8.4', bars: [50, 60, 55, 80, 65, 95, 75, 85] },
    card3: { title: 'Total Mileage', subtitle: 'Daily distance covered', value: '85K', stat1: '400', label1: 'ROUTES', stat2: '12', label2: 'ZONES', metric: '2.1M' },
  }
} as const;

export default function EcosystemStats() {
  const [activeTab, setActiveTab] = useState<keyof typeof TAB_DATA>('supplier');
  const dashboardRef = useRef<HTMLDivElement>(null);
  const { t, language } = useLanguage();

  const data = TAB_DATA[activeTab];
  const tabDataRu: typeof TAB_DATA | null = language === 'ru' ? {
    supplier: {
      title: 'Центр операций поставщика',
      subtitle: 'Видимость исходящего исполнения и диспетчеризации в реальном времени',
      card1: { title: 'Объём заказов', subtitle: 'Активная исходящая обработка', value: '8.2K', unit: 'Заказов в день', percent: 92, stat1: '99%', label1: 'ЗАПОЛНЕНИЕ', stat2: '2M', label2: 'ЕДИНИЦ', metric: '98.7%' },
      card2: { title: 'SLA диспетчеризации', subtitle: 'Мониторинг выезда вовремя', metric: '99.99%', bars: [45, 75, 35, 90, 55, 100, 40, 65] },
      card3: { title: 'Здоровье запасов', subtitle: 'Глобальная доступность стока', value: '94%', stat1: '12K', label1: 'SKU', stat2: '45', label2: 'ПЛОЩАДОК', metric: '8.4M' },
    },
    warehouse: {
      title: 'Диспетчерская склада',
      subtitle: 'Живой мониторинг ворот, рамп и пропускной способности DC',
      card1: { title: 'Загрузка рамп', subtitle: 'Живая обработка на воротах', value: '14', unit: 'Активных ворот', percent: 75, stat1: '85%', label1: 'ЗАГРУЗКА', stat2: '1.2K', label2: 'ПАЛЛЕТ', metric: '92.4%' },
      card2: { title: 'Время кросс-дока', subtitle: 'Внутренние SLA транзита', metric: '42m', bars: [60, 50, 80, 40, 70, 90, 50, 85] },
      card3: { title: 'Пропускная способность', subtitle: 'Часовой обработанный объём', value: '840', stat1: '150', label1: 'ГРУЗОВИКОВ', stat2: '3', label2: 'СМЕНЫ', metric: '1.2M' },
    },
    retailer: {
      title: 'Хаб сети ритейлеров',
      subtitle: 'Статусы доставки в магазины и метрики разгрузки',
      card1: { title: 'Статус доставки', subtitle: 'Живая телеметрия автопарка', value: '142', unit: 'Активных маршрутов', percent: 88, stat1: '96%', label1: 'ВОВРЕМЯ', stat2: '4K', label2: 'ОСТАНОВОК', metric: '96.2%' },
      card2: { title: 'SLA разгрузки', subtitle: 'Мониторинг времени оборота', metric: '18m', bars: [30, 40, 60, 35, 80, 55, 90, 45] },
      card3: { title: 'Принятый объём', subtitle: 'Единиц принято за день', value: '12.5K', stat1: '99%', label1: 'СОВПАДЕНИЕ', stat2: '15', label2: 'DC', metric: '3.4M' },
    },
    fleet: {
      title: 'Глобальная телеметрия автопарка',
      subtitle: 'Живое отслеживание ТС, расход топлива и эффективность маршрутов',
      card1: { title: 'Активный транспорт', subtitle: 'ТС сейчас на маршруте', value: '450', unit: 'Грузовиков', percent: 95, stat1: '1.2K', label1: 'ВОДИТЕЛЕЙ', stat2: '99%', label2: 'АПТАЙМ', metric: '99.9%' },
      card2: { title: 'Топливная эффективность', subtitle: 'Средний расход', metric: '8.4', bars: [50, 60, 55, 80, 65, 95, 75, 85] },
      card3: { title: 'Общий пробег', subtitle: 'Дневная дистанция', value: '85K', stat1: '400', label1: 'МАРШРУТОВ', stat2: '12', label2: 'ЗОН', metric: '2.1M' },
    },
  } as any : null;
  const localizedData = (tabDataRu?.[activeTab] ?? data);

  useEffect(() => {
    // Advanced staggered entrance animation
    const ctx = gsap.context(() => {
      gsap.timeline({
        scrollTrigger: {
          trigger: dashboardRef.current,
          start: 'top 80%',
          toggleActions: 'play none none reverse',
        }
      })
      .fromTo('.stat-card',
        { opacity: 0, y: 40, scale: 0.95 },
        { opacity: 1, y: 0, scale: 1, duration: 0.8, stagger: 0.15, ease: 'back.out(1.2)', clearProps: 'all' }
      );
    }, dashboardRef);
    return () => ctx.revert();
  }, [activeTab]);

  return (
    <PageSection bleed={true} className="bg-[#020202] w-full border-t border-white/5 relative overflow-hidden !p-0" aria-labelledby="ecosystem-stats-heading">

      {/* Background ambient light */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full h-[500px] bg-white/[0.015] blur-[120px] pointer-events-none rounded-full" />

      <div className="w-full relative z-10">

        {/* Header */}
        <div className="mb-16 px-4 md:px-8">
          <div className="flex items-center gap-3 text-white/40 mb-6">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M4 15l8-8 8 8" />
            </svg>
            <span className="text-[10px] tracking-[0.2em] uppercase font-mono">{t('ecosystem_eyebrow', 'Ecosystem Statistics')}</span>
          </div>
          <h2 id="ecosystem-stats-heading" className="text-5xl md:text-7xl font-medium tracking-tight mb-6 text-white">
            {t('ecosystem_title', 'Optimized for the entire chain')}
          </h2>

        </div>

        {/* Massive Dashboard UI */}
        <div ref={dashboardRef} className="bg-[#050505] border-none  border-white/10 overflow-hidden shadow-[0_20px_60px_-15px_rgba(0,0,0,0.8)] flex flex-col lg:flex-row min-h-[800px] w-full">

          {/* Sidebar */}


          {/* Main Content Area */}
          <div className="flex-1 p-6 md:p-10 dashboard-content flex flex-col bg-[#000000]  relative">
            {/* Grid Pattern Background */}
            <div className="absolute inset-0 bg-[url('/images/grid-pattern.svg')] opacity-[0.03] pointer-events-none" style={{ backgroundSize: '40px 40px' }} />

            {/* Content Header */}


            {/* Metrics Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6 flex-1 relative z-10">

              {/* Feature/Load Widget (takes up 2 columns on wide screens) */}
              <div className="xl:col-span-2 rounded-xl overflow-hidden shadow-2xl stat-card group transition-all duration-500  hover:shadow-[0_30px_60px_-15px_rgba(255,255,255,0.05)]">
                {/* Embedded complex stats */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6 h-full">
                  <div className="h-[380px] md:h-full">
                    <SystemLoadWidget />
                  </div>

                  {/* Secondary large widget */}
                  <div className="bg-[#000000] border border-white/5 p-8 rounded shadow-2xl flex flex-col relative h-[380px] md:h-full transition-colors duration-500 hover:border-white/15">
                    <div className="flex justify-between items-start mb-6">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 border border-white/10 flex items-center justify-center text-white/40 bg-white/5 rounded-sm">
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" /></svg>
                        </div>
                        <span className="text-base font-medium text-white/90">{t('ecosystem_revenue', 'Revenue Impact')}</span>
                      </div>
                      <span className="text-xs font-mono text-green-400">+12.4%</span>
                    </div>
                    <div className="flex-1 flex flex-col justify-end pb-4">
                      <div className="text-5xl font-light text-white mb-8">$2.4M</div>
                      <div className="flex items-end gap-2 h-32">
                        {[30, 45, 20, 60, 40, 80, 55, 90, 70, 100, 85, 110].map((h, i) => (
                          <div key={i} className="flex-1 bg-white/10 hover:bg-white/30 transition-colors rounded-t-sm" style={{ height: `${h}%` }} />
                        ))}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Card 1: Circle Gauge */}
              <div className="bg-[#000000] border border-white/5 p-8 rounded flex flex-col relative h-[380px] md:h-full shadow-2xl stat-card group transition-all duration-500 hover:shadow-[0_30px_60px_-15px_rgba(255,255,255,0.05)] hover:border-white/15">
                <div className="flex justify-between items-start mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 border border-white/10 flex items-center justify-center text-white/40 bg-white/5 rounded-sm">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M18 6L6 18M6 6l12 12" strokeWidth="1.5" strokeLinecap="square" /></svg>
                    </div>
                    <span className="text-base font-medium text-white/90">{localizedData.card1.title}</span>
                  </div>
                  <span className="text-xs font-mono text-white/40">{localizedData.card1.metric}</span>
                </div>
                <div className="text-sm text-white/40 mb-8">{localizedData.card1.subtitle}</div>

                <div className="flex-1 flex items-center justify-center relative">
                  <svg width="220" height="220" className="-rotate-90 drop-shadow-xl">
                    <circle cx="110" cy="110" r="85" fill="none" stroke="rgba(255,255,255,0.03)" strokeWidth="20" />
                    {/* ticks */}
                    <g stroke="rgba(255,255,255,0.15)" strokeWidth="1.5">
                      {[...Array(30)].map((_, i) => (
                        <line key={i} x1="110" y1="15" x2="110" y2="25" transform={`rotate(${i * 12} 110 110)`} />
                      ))}
                    </g>
                    <circle cx="110" cy="110" r="85" fill="none" stroke="white" strokeWidth="20"
                      strokeDasharray={2 * Math.PI * 85}
                      strokeDashoffset={(2 * Math.PI * 85) * (1 - localizedData.card1.percent / 100)}
                      className="transition-all duration-1000 ease-out group-hover:stroke-[#fbff63]"
                    />
                  </svg>
                  <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                    <span className="text-4xl font-light tracking-tight text-white">{localizedData.card1.value}</span>
                    <span className="text-[10px] font-mono text-white/40 mt-1 uppercase">{localizedData.card1.unit}</span>
                  </div>

                  <div className="absolute left-0 bottom-0 flex flex-col gap-4">
                    <div>
                      <div className="text-xs text-white/90">{localizedData.card1.stat1}</div>
                      <div className="text-[9px] tracking-wider text-white/40 font-mono mt-0.5 uppercase">{localizedData.card1.label1}</div>
                    </div>
                    <div>
                      <div className="text-xs text-white/90">{localizedData.card1.stat2}</div>
                      <div className="text-[9px] tracking-wider text-white/40 font-mono mt-0.5 uppercase">{localizedData.card1.label2}</div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Card 2: Bar Chart */}
              <div className="bg-[#000000] border border-white/5 p-8 rounded flex flex-col relative h-[380px] shadow-2xl stat-card group transition-all duration-500 hover:shadow-[0_30px_60px_-15px_rgba(255,255,255,0.05)] hover:border-white/15">
                <div className="flex justify-between items-start mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 border border-white/10 flex items-center justify-center text-white/40 bg-white/5 rounded-sm">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M18 6L6 18M6 6l12 12" strokeWidth="1.5" strokeLinecap="square" /></svg>
                    </div>
                    <span className="text-base font-medium text-white/90">{localizedData.card2.title}</span>
                  </div>
                  <span className="text-xs font-mono text-white/40">{localizedData.card2.metric}</span>
                </div>
                <div className="text-sm text-white/40 mb-10">{localizedData.card2.subtitle}</div>

                <div className="flex-1 flex items-end justify-between relative px-2 pb-6">
                  <div className="absolute top-[35%] left-0 right-0 border-t border-dashed border-white/10" />
                  <div className="absolute top-[35%] left-4 -translate-y-1/2 bg-white text-black text-xs font-mono px-3 py-1 rounded-sm z-10">
                    SLA TARGET
                  </div>

                  {localizedData.card2.bars.map((h, i) => (
                    <div key={i} className="flex flex-col items-center gap-1.5 relative z-0 h-[180px] justify-end">
                      {i === 4 ? (
                        <div className="w-3 h-3 rounded-full bg-gradient-to-tr from-[#FF3366] to-[#33CCFF] absolute -top-5 shadow-[0_0_12px_rgba(255,51,102,0.8)] z-10" />
                      ) : (
                        <div className="w-2 h-2 rounded-full bg-white/80 absolute -top-4" />
                      )}
                      <div className="w-1 bg-white/10 transition-all duration-700 ease-out" style={{ height: `${h * 1.5}px` }} />
                    </div>
                  ))}
                </div>
              </div>

              {/* Card 3: Speedometer */}
              <div className="bg-[#000000] border border-white/5 p-8 rounded flex flex-col relative h-[380px] xl:col-span-2 shadow-2xl stat-card group transition-all duration-500 hover:shadow-[0_30px_60px_-15px_rgba(255,255,255,0.05)] hover:border-white/15">
                <div className="flex justify-between items-start mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 border border-white/10 flex items-center justify-center text-white/40 bg-white/5 rounded-sm">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M18 6L6 18M6 6l12 12" strokeWidth="1.5" strokeLinecap="square" /></svg>
                    </div>
                    <span className="text-base font-medium text-white/90">{localizedData.card3.title}</span>
                  </div>
                  <span className="text-xs font-mono text-white/40">{localizedData.card3.metric}</span>
                </div>
                <div className="text-sm text-white/40 mb-8">{localizedData.card3.subtitle}</div>

                <div className="flex-1 flex flex-col items-center justify-end relative pt-4 pb-4">
                  <svg width="320" height="170" className="overflow-visible drop-shadow-xl absolute top-4">
                    <g className="text-white/15" strokeWidth="1.5">
                      {[...Array(35)].map((_, i) => (
                        <line key={i} x1="160" y1="16" x2="160" y2="24" transform={`rotate(${i * 5 - 85} 160 160)`} stroke="currentColor" />
                      ))}
                    </g>
                    <path d="M 20 160 A 140 140 0 0 1 300 160" fill="none" stroke="rgba(255,255,255,0.03)" strokeWidth="36" />
                    <path d="M 20 160 A 140 140 0 0 1 300 160" fill="none" stroke="white" strokeWidth="36"
                      strokeDasharray={Math.PI * 140} strokeDashoffset={(Math.PI * 140) * 0.3}
                      className="transition-all duration-1000 ease-out group-hover:stroke-[#fbff63]"
                    />
                  </svg>

                  <div className="absolute top-[80px] flex flex-col items-center pointer-events-none">
                    <span className="text-6xl font-light tracking-tight text-white">{localizedData.card3.value}</span>
                  </div>

                  <div className="w-full flex justify-between px-12 mt-32 z-10">
                    <div className="text-center bg-[#000000]/80 backdrop-blur-sm px-4 py-2 rounded border border-white/5">
                      <div className="text-xs text-white/90">{localizedData.card3.stat1}</div>
                      <div className="text-[9px] tracking-wider text-white/40 font-mono mt-0.5 uppercase">{localizedData.card3.label1}</div>
                    </div>
                    <div className="text-center bg-[#000000]/80 backdrop-blur-sm px-4 py-2 rounded border border-white/5">
                      <div className="text-xs text-white/90">{localizedData.card3.stat2}</div>
                      <div className="text-[9px] tracking-wider text-white/40 font-mono mt-0.5 uppercase">{localizedData.card3.label2}</div>
                    </div>
                  </div>
                </div>
              </div>

            </div>
          </div>
        </div>
      </div>
    </PageSection>
  );
}
