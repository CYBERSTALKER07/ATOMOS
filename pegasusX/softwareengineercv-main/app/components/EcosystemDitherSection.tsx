'use client';

import React, { useState, useRef, useEffect } from 'react';
import dynamic from 'next/dynamic';
import PageSection from './layout/PageSection';
import { useLanguage } from '../context/LanguageContext';

// Dynamically import the 3D WebGL Dither Stage with SSR disabled
const EcosystemDitherStage = dynamic(
  () => import('./visuals/EcosystemDitherStage'),
  {
    ssr: false,
    loading: () => (
      <div className="w-full h-full flex items-center justify-center bg-[#060608] border border-white/10">
        <span className="font-mono text-xs text-white/40 tracking-widest uppercase animate-pulse">
          INITIALIZING_DITHER_RENDERER...
        </span>
      </div>
    ),
  }
);

interface StepItem {
  id: string;
  n: number;
  tag: string;
  title: string;
  titleRu: string;
  body: string;
  bodyRu: string;
  dockTitle: string;
  dockTitleRu: string;
  dockDesc: string;
  dockDescRu: string;
  stats: {
    label: string;
    labelRu: string;
    val: string;
  }[];
}

const STEPS: StepItem[] = [
  {
    id: 'demand',
    n: 0,
    tag: '01 / INPUT',
    title: 'Demand',
    titleRu: 'Спрос',
    body: 'Store orders, consumer consumption patterns, and weather signals feed continuous real-time demand modeling.',
    bodyRu: 'Заказы торговых точек, паттерны потребления и погодные сигналы непрерывно обучают модель спроса.',
    dockTitle: 'Real-time Demand Sensing',
    dockTitleRu: 'Сенсорика спроса в реальном времени',
    dockDesc: 'Ingests point-of-sale telemetry and seasonal variances to forecast SKU-level volume before orders are even placed.',
    dockDescRu: 'Анализирует чеки продаж и сезонные колебания, прогнозируя объёмы до формирования заказов.',
    stats: [
      { label: 'Forecast Accuracy', labelRu: 'Точность прогноза', val: '99.4%' },
      { label: 'Daily Intents', labelRu: 'Потоков спроса', val: '2.4M' },
      { label: 'Telemetry Lag', labelRu: 'Задержка данных', val: '< 12ms' },
    ],
  },
  {
    id: 'labor',
    n: 1,
    tag: '02 / ROSTER',
    title: 'Labor',
    titleRu: 'Смены',
    body: 'Driver shift pairing, loading dock rosters, and pre-trip inspections sync directly with planned inbound waves.',
    bodyRu: 'Назначение водителей на рейсы, графики рамп и чек-листы осмотра синхронизированы с входящими волнами.',
    dockTitle: 'Dynamic Shift & Driver Pairing',
    dockTitleRu: 'Динамическое распределение смен и водителей',
    dockDesc: 'Automates daily clock-in, vehicle assignment, and mid-shift hot-swapping to eliminate dock bottleneck stalls.',
    dockDescRu: 'Автоматизирует выход на смену, привязку ТС и горячую замену при поломках без остановки поставок.',
    stats: [
      { label: 'Shift Fulfillment', labelRu: 'Укомплектованность', val: '99.8%' },
      { label: 'Active Crews', labelRu: 'Активных бригад', val: '142' },
      { label: 'Turnaround SLA', labelRu: 'Оборот экипажа', val: '4.2m' },
    ],
  },
  {
    id: 'inventory',
    n: 2,
    tag: '03 / BUFFER',
    title: 'Inventory',
    titleRu: 'Запасы',
    body: 'High-bay racking, SSCC-18 pallet tracking, and cross-dock wave generation prevent overstock and stockouts.',
    bodyRu: 'Высотные стеллажи, SSCC-18 трекинг паллет и волновая сортировка исключают дефицит и затоваривание.',
    dockTitle: 'Autonomous Cross-Dock Orchestration',
    dockTitleRu: 'Автономная кросс-док оркестрация',
    dockDesc: 'Directs pallet transfers straight from inbound factory trailers to outbound city delivery manifests.',
    dockDescRu: 'Направляет паллеты с заводских полуприцепов прямо на рампы городской доставки без промежуточного хранения.',
    stats: [
      { label: 'Stockout Prevention', labelRu: 'Предотвращение дефицита', val: '99.9%' },
      { label: 'Cross-Dock Time', labelRu: 'Время кросс-дока', val: '38m' },
      { label: 'Throughput', labelRu: 'Пропускная способность', val: '840/h' },
    ],
  },
  {
    id: 'locations',
    n: 3,
    tag: '04 / TRANSIT',
    title: 'Locations',
    titleRu: 'Локации',
    body: 'Multi-modal transit grid, regional distribution centers, and doorstep delivery geofences run on autopilot.',
    bodyRu: 'Мультимодальная сеть, региональные распредцентры и геозоны доставки работают на автопилоте.',
    dockTitle: 'End-to-End Fleet Telematics Mesh',
    dockTitleRu: 'Сквозная телематическая сеть флота',
    dockDesc: 'Monitors CAN-bus GPS telemetry, validates 150m delivery geofences, and triggers instant fiscalization receipts.',
    dockDescRu: 'Контролирует GPS по CAN-шине, подтверждает геозону доставки <150м и выбивает фискальный чек в ОФД.',
    stats: [
      { label: 'Connected Vehicles', labelRu: 'Подключено ТС', val: '450' },
      { label: 'On-Time Deliveries', labelRu: 'Доставка вовремя', val: '98.9%' },
      { label: 'Active Corridors', labelRu: 'Активных коридоров', val: '18' },
    ],
  },
];

export default function EcosystemDitherSection() {
  const [activeStep, setActiveStep] = useState<number>(0);
  const [progress, setProgress] = useState<number>(0);
  const sectionRef = useRef<HTMLDivElement>(null);
  const { language } = useLanguage();
  const isRu = language === 'ru';

  // Scroll listener synchronizing normalized progress across the 360vh track
  useEffect(() => {
    const el = sectionRef.current;
    if (!el) return;

    let ticking = false;
    const handleScroll = () => {
      if (!ticking) {
        window.requestAnimationFrame(() => {
          if (!el) return;
          const rect = el.getBoundingClientRect();
          const maxScroll = rect.height - window.innerHeight;

          if (maxScroll > 0) {
            const p = Math.min(1, Math.max(0, -rect.top / maxScroll));
            setProgress(p);

            const step = Math.min(3, Math.floor(p * 4));
            setActiveStep(step);
          }
          ticking = false;
        });
        ticking = true;
      }
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    handleScroll();

    return () => {
      window.removeEventListener('scroll', handleScroll);
    };
  }, []);

  const scrollToStep = (idx: number) => {
    setActiveStep(idx);
    const el = sectionRef.current;
    if (!el) return;

    const rect = el.getBoundingClientRect();
    const maxScroll = rect.height - window.innerHeight;
    if (maxScroll > 0) {
      const targetTop = rect.top + window.scrollY + (maxScroll * (idx + 0.15)) / 4;
      window.scrollTo({
        top: targetTop,
        behavior: 'smooth',
      });
    }
  };

  const currentStep = STEPS[activeStep] || STEPS[0];

  return (
    <PageSection
      id="how-it-works"
      bleed={true}
      className="bg-[#09090B] text-white w-full border-t border-white/10 relative p-0 overflow-visible"
    >
      {/* 360vh Scroll Track */}
      <div ref={sectionRef} className="relative w-full h-[320vh] sm:h-[360vh]">
        
        {/* Sticky Viewport Stage */}
        <div className="sticky top-0 w-full h-screen flex flex-col justify-between overflow-hidden bg-[#09090B] p-4 sm:p-6 md:p-8 select-none">
          
          {/* Framed Control Box (Meuze Architecture) */}
          <div className="w-full max-w-[1520px] mx-auto flex-1 flex flex-col justify-between border border-white/15 bg-[#09090B] p-4 sm:p-6 md:p-8 relative">
            
            {/* Top Corner Markers */}
            <span className="absolute top-2 left-2 font-mono text-[10px] text-white/30 pointer-events-none">+</span>
            <span className="absolute top-2 right-2 font-mono text-[10px] text-white/30 pointer-events-none">+</span>
            <span className="absolute bottom-2 left-2 font-mono text-[10px] text-white/30 pointer-events-none">+</span>
            <span className="absolute bottom-2 right-2 font-mono text-[10px] text-white/30 pointer-events-none">+</span>

            {/* Header: Eyebrow + Claim + Deck */}
            <div className="border-b border-white/10 pb-6 mb-4">
              <p className="text-[11px] font-mono tracking-widest text-white/50 uppercase mb-2">
                {isRu ? 'КАК ЭТО РАБОТАЕТ // ОПЕРАЦИОННОЕ ЯДРО' : 'HOW IT WORKS // SOVEREIGN ENGINE'}
              </p>
              <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 items-baseline">
                <h2 className="lg:col-span-6 text-2xl sm:text-3xl md:text-4xl font-bold tracking-tight text-white uppercase font-mono">
                  {isRu ? 'Мы изучаем ваши процессы, затем управляем ими.' : 'We learn your operation, then we run it.'}
                </h2>
                <p className="lg:col-span-6 text-xs sm:text-sm text-white/60 leading-relaxed font-sans">
                  {isRu
                    ? 'Pegasus подключается к вашей инфраструктуре и строит математическую модель всей цепочки поставок. Данные о спросе, складе и флоте управляют диспетчеризацией с прозрачным обоснованием каждого числа.'
                    : 'Pegasus plugs into your systems and builds models tailored to your supply chain. Demand, inventory, staging, and fleet telemetry then drive replenishment and dispatch from that model, with the reason behind every number.'}
                </p>
              </div>
            </div>

            {/* Center Stage Layout: 4 Step Buttons + 3D WebGL Dither Stage */}
            <div className="flex-1 grid grid-cols-1 lg:grid-cols-12 gap-6 min-h-0 items-stretch">
              
              {/* Left Column: Interactive Step Selector */}
              <div className="lg:col-span-4 flex flex-col justify-between gap-2.5">
                {STEPS.map((step) => {
                  const isActive = activeStep === step.n;
                  return (
                    <button
                      key={step.id}
                      type="button"
                      onClick={() => scrollToStep(step.n)}
                      className={`flex-1 text-left p-4 sm:p-5 border rounded-none transition-all duration-300 flex flex-col justify-between cursor-pointer ${
                        isActive
                          ? 'border-white bg-[#141419] text-white shadow-[inset_0_0_25px_rgba(255,255,255,0.04)]'
                          : 'border-white/10 bg-[#09090B]/60 text-white/40 hover:border-white/25 hover:text-white/70'
                      }`}
                    >
                      <div className="flex items-center justify-between text-[10px] font-mono tracking-wider mb-1">
                        <span className={isActive ? 'text-white font-bold' : 'text-white/30'}>
                          {step.tag}
                        </span>
                        <span className="font-mono">{`[0${step.n + 1}/04]`}</span>
                      </div>
                      <div>
                        <h3 className="text-base sm:text-lg font-bold tracking-tight text-white uppercase font-mono mb-1">
                          {isRu ? step.titleRu : step.title}
                        </h3>
                        <p className="text-xs text-white/60 leading-relaxed font-sans line-clamp-2">
                          {isRu ? step.bodyRu : step.body}
                        </p>
                      </div>

                      {isActive && (
                        <div className="mt-2 pt-2 border-t border-white/10 flex items-center gap-2 text-[10px] font-mono text-white/80">
                          <span className="w-1.5 h-1.5 bg-white animate-pulse" />
                          <span>SIMULATION_TARGET_FOCUSED</span>
                        </div>
                      )}
                    </button>
                  );
                })}
              </div>

              {/* Right Column: The 3D Halftone Dither Stage (Meuze Dual-Pass Pipeline) */}
              <div className="lg:col-span-8 relative border border-white/20 bg-[#060608] min-h-[320px] sm:min-h-[380px] lg:min-h-0 flex flex-col overflow-hidden">
                {/* Stage Corner Marks */}
                <span className="absolute top-2 left-2 font-mono text-[9px] text-white/25 z-20 pointer-events-none">+</span>
                <span className="absolute top-2 right-2 font-mono text-[9px] text-white/25 z-20 pointer-events-none">+</span>
                <span className="absolute bottom-2 left-2 font-mono text-[9px] text-white/25 z-20 pointer-events-none">+</span>
                <span className="absolute bottom-2 right-2 font-mono text-[9px] text-white/25 z-20 pointer-events-none">+</span>

                {/* Telemetry Header Pill */}
                <div className="absolute top-3 left-3 right-3 flex items-center justify-between z-20 pointer-events-none text-[10px] font-mono tracking-widest text-white/70 bg-[#09090B]/85 px-3 py-1.5 border border-white/10 backdrop-blur-sm">
                  <div className="flex items-center gap-2">
                    <span className="w-1.5 h-1.5 bg-white" />
                    <span className="uppercase">{`PHASE_0${activeStep + 1} // ${currentStep.title.toUpperCase()}`}</span>
                  </div>
                  <span className="hidden sm:inline text-white/40">DUAL_PASS_DITHER // CEL_EDGE_AA</span>
                </div>

                {/* Real-time 3D WebGL Canvas */}
                <div className="w-full h-full flex-1 relative">
                  <EcosystemDitherStage activeStep={activeStep} theme="dark" />
                </div>

                {/* Telemetry Bottom Pill */}
                <div className="absolute bottom-3 left-3 right-3 flex items-center justify-between z-20 pointer-events-none text-[10px] font-mono tracking-widest text-white/50 bg-[#09090B]/85 px-3 py-1.5 border border-white/10 backdrop-blur-sm">
                  <span>RENDER_PASS: U_INK_DEPTH_SDF</span>
                  <span className="text-white font-bold">{`ACTIVE: ${currentStep.id.toUpperCase()}`}</span>
                </div>
              </div>

            </div>

            {/* Bottom Telemetry Dock (Meuze Dock Panel) */}
            <div className="mt-4 pt-4 border-t border-white/10 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
              <div className="max-w-xl">
                <div className="text-[10px] font-mono tracking-widest text-white/40 uppercase mb-1">
                  {isRu ? 'ОПЕРАЦИОННЫЙ СТАТУС' : 'ACTIVE SUBSYSTEM TELEMETRY'}
                </div>
                <div className="text-sm font-bold text-white uppercase font-mono mb-0.5">
                  {isRu ? currentStep.dockTitleRu : currentStep.dockTitle}
                </div>
                <p className="text-xs text-white/60 font-sans line-clamp-1">
                  {isRu ? currentStep.dockDescRu : currentStep.dockDesc}
                </p>
              </div>

              {/* Real-time Metric Badges */}
              <div className="flex items-center gap-3 w-full md:w-auto overflow-x-auto no-scrollbar">
                {currentStep.stats.map((stat, sIdx) => (
                  <div key={sIdx} className="border border-white/10 bg-[#121216] px-3.5 py-2 flex flex-col min-w-[120px]">
                    <span className="text-[9px] font-mono text-white/40 uppercase">
                      {isRu ? stat.labelRu : stat.label}
                    </span>
                    <span className="text-base font-mono font-bold text-white tabular-nums">
                      {stat.val}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {/* Bottom Progress Bar */}
            <div className="mt-4 pt-3 border-t border-white/10 flex items-center justify-between gap-4">
              <div className="text-[10px] font-mono text-white/40 uppercase tracking-wider flex items-center gap-2">
                <span className="w-1.5 h-1.5 bg-white" />
                <span>{isRu ? 'ПРОКРУТИТЕ ДЛЯ СМЕНЫ ФАЗЫ' : 'SCROLL TO SCRUB PHASES'}</span>
              </div>

              <div className="flex-1 max-w-md h-1 bg-white/10 relative overflow-hidden border border-white/10">
                <div
                  className="h-full bg-white transition-all duration-75"
                  style={{ width: `${Math.min(100, Math.max(0, progress * 100))}%` }}
                />
              </div>

              <div className="text-[10px] font-mono text-white font-bold tabular-nums">
                {`${Math.round(progress * 100)}% // [0${activeStep + 1}/04]`}
              </div>
            </div>

          </div>
        </div>
      </div>
    </PageSection>
  );
}
