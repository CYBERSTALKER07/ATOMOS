'use client';

import React, { useState, useRef, useEffect, useMemo } from 'react';
import dynamic from 'next/dynamic';
import PageSection from './layout/PageSection';
import { useLanguage } from '../context/LanguageContext';

// Dynamic import with SSR disabled for WebGL canvas
const EcosystemDitherStage = dynamic(
  () => import('./visuals/EcosystemDitherStage'),
  {
    ssr: false,
    loading: () => (
      <div className="w-full h-full flex items-center justify-center bg-[#070709] border border-white/5">
        <span className="font-mono text-xs text-white/30 tracking-widest animate-pulse">
          INITIALIZING_DITHER_PIPELINE...
        </span>
      </div>
    ),
  }
);

interface StepMetadata {
  id: 'supplier' | 'warehouse' | 'retailer' | 'fleet';
  index: number;
  tag: string;
  title: string;
  titleRu: string;
  subtitle: string;
  subtitleRu: string;
  zoneCode: string;
  kpis: {
    label: string;
    labelRu: string;
    val: string;
    sub: string;
  }[];
}

const ECOSYSTEM_STEPS: StepMetadata[] = [
  {
    id: 'supplier',
    index: 0,
    tag: 'PHASE 01 // ORIGIN',
    title: 'Supplier Operations Center',
    titleRu: 'Центр операций поставщика',
    subtitle: 'Factory weighbridge, bulk pallet intake, and automated SKU classification.',
    subtitleRu: 'Заводские весы, паллетная приёмка и автоматическая классификация SKU.',
    zoneCode: 'ZONE_SUPPLIER_WEIGHBRIDGE // LAT: 41.3111 LON: 69.2797',
    kpis: [
      { label: 'Outbound Fill Rate', labelRu: 'Уровень заполнения', val: '99.2%', sub: 'SLA TARGET' },
      { label: 'Daily Dispatch Units', labelRu: 'Отгрузка в день', val: '8,240', sub: '+14% VS PLAN' },
      { label: 'Weighbridge SLA', labelRu: 'SLA весовой рампы', val: '4.2m', sub: 'AVG TURNAROUND' },
    ],
  },
  {
    id: 'warehouse',
    index: 1,
    tag: 'PHASE 02 // NODE',
    title: 'Cross-Dock & Warehouse Control',
    titleRu: 'Диспетчерская склада и кросс-дока',
    subtitle: 'High-bay racking, SSCC-18 pallet tracking, and fulfillment wave orchestration.',
    subtitleRu: 'Высотные стеллажи, трекинг SSCC-18 и волновая сборка заказов.',
    zoneCode: 'ZONE_CENTRAL_DC_BAY_04 // WMS_STATUS: ACTIVE',
    kpis: [
      { label: 'Dock Bay Utilization', labelRu: 'Загрузка рамп', val: '92.4%', sub: '14 ACTIVE GATES' },
      { label: 'Cross-Dock SLA', labelRu: 'Время кросс-дока', val: '38m', sub: '-12m OPTIMIZED' },
      { label: 'Hourly Throughput', labelRu: 'Пропускная способность', val: '840', sub: 'PALLETS/HR' },
    ],
  },
  {
    id: 'retailer',
    index: 2,
    tag: 'PHASE 03 // INTAKE',
    title: 'Retailer Network Hub',
    titleRu: 'Хаб сети ритейлеров',
    subtitle: 'Doorstep reception, Soliq OFD fiscalization, and real-time ePoD signing.',
    subtitleRu: 'Приёмка в торговой точке, фискализация ОФД и цифровые акты ePoD.',
    zoneCode: 'ZONE_RETAIL_STORE_INTAKE // FISCAL: COMPLIANT',
    kpis: [
      { label: 'On-Time Reception', labelRu: 'Приёмка вовремя', val: '98.7%', sub: 'DOORSTEP GEOFENCE' },
      { label: 'Turnaround Time', labelRu: 'Время приёмки', val: '14m', sub: 'PER DELIVERY' },
      { label: 'Match Accuracy', labelRu: 'Точность сверки', val: '99.9%', sub: 'BARCODE VERIFIED' },
    ],
  },
  {
    id: 'fleet',
    index: 3,
    tag: 'PHASE 04 // TRANSIT',
    title: 'Global Fleet Telematics Mesh',
    titleRu: 'Телематическая сеть автопарка',
    subtitle: 'Dynamic shift pairing, CAN-bus telemetry, and mid-shift breakdown hot-swapping.',
    subtitleRu: 'Динамические смены, CAN-телематика и горячая замена ТС на маршруте.',
    zoneCode: 'ZONE_URBAN_CORRIDOR_MESH // DISPATCH: LIVE',
    kpis: [
      { label: 'Active Transit Units', labelRu: 'Активные ТС', val: '450', sub: 'CONNECTED FLEET' },
      { label: 'Route Mileage', labelRu: 'Суточный пробег', val: '85.4K', sub: 'KM DISPATCHED' },
      { label: 'Hot-Swap Readiness', labelRu: 'Готовность резерва', val: '100%', sub: 'ZERO DELAY' },
    ],
  },
];

export default function EcosystemStats() {
  const [activeStep, setActiveStep] = useState<number>(0);
  const [progress, setProgress] = useState<number>(0);
  const trackRef = useRef<HTMLDivElement>(null);
  const { language } = useLanguage();
  const isRu = language === 'ru';

  // Synchronize active step with sticky scroll track
  useEffect(() => {
    const track = trackRef.current;
    if (!track) return;

    let ticking = false;
    const handleScroll = () => {
      if (!ticking) {
        window.requestAnimationFrame(() => {
          if (!track) return;
          const rect = track.getBoundingClientRect();
          const totalScroll = rect.height - window.innerHeight;

          if (totalScroll > 0) {
            // Normalized progress from 0.0 to 1.0
            const currentProgress = Math.min(1, Math.max(0, -rect.top / totalScroll));
            setProgress(currentProgress);

            // 4 steps distributed evenly across track
            const stepIdx = Math.min(3, Math.floor(currentProgress * 4));
            setActiveStep(stepIdx);
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

  // Jump to step on button click
  const scrollToStep = (stepIndex: number) => {
    setActiveStep(stepIndex);
    const track = trackRef.current;
    if (!track) return;

    const rect = track.getBoundingClientRect();
    const totalScroll = rect.height - window.innerHeight;
    if (totalScroll > 0) {
      const targetScroll = rect.top + window.scrollY + (totalScroll * (stepIndex + 0.15)) / 4;
      window.scrollTo({
        top: targetScroll,
        behavior: 'smooth',
      });
    }
  };

  const currentMeta = useMemo(() => ECOSYSTEM_STEPS[activeStep] || ECOSYSTEM_STEPS[0], [activeStep]);

  return (
    <PageSection
      bleed={true}
      className="bg-[#09090B] text-white w-full border-t border-white/10 relative p-0 overflow-visible"
      aria-labelledby="ecosystem-stats-heading"
    >
      {/* 360vh Sticky Scroll Container */}
      <div ref={trackRef} className="relative w-full h-[320vh] sm:h-[360vh]">
        
        {/* Sticky Viewport Shell */}
        <div className="sticky top-0 w-full h-screen flex flex-col justify-between overflow-hidden bg-[#09090B] p-4 sm:p-6 md:p-8 select-none">
          
          {/* Top Tactical Command Bar */}
          <div className="w-full max-w-[1440px] mx-auto flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-white/10 pb-4 z-20">
            <div>
              <div className="flex items-center gap-2 mb-1">
                <span className="w-2 h-2 bg-emerald-400 animate-pulse" />
                <span className="text-[10px] font-mono tracking-widest text-white/50 uppercase">
                  {isRu ? 'ТЕЛЕМЕТРИЯ ЭКОСИСТЕМЫ // V.O.I.D. CORE' : 'SOVEREIGN ECOSYSTEM TELEMETRY // V.O.I.D.'}
                </span>
              </div>
              <h2 id="ecosystem-stats-heading" className="text-xl sm:text-2xl md:text-3xl font-bold tracking-tight text-white uppercase font-mono">
                {isRu ? 'Сквозная оптимизация цепочки' : 'Optimized for the entire logistics chain'}
              </h2>
            </div>

            {/* Step Switcher Navigation */}
            <div className="flex items-center gap-1 border border-white/15 bg-[#121216] p-1">
              {ECOSYSTEM_STEPS.map((step) => {
                const isActive = activeStep === step.index;
                return (
                  <button
                    key={step.id}
                    type="button"
                    onClick={() => scrollToStep(step.index)}
                    className={`px-3 py-1.5 text-[11px] font-mono tracking-wider transition-colors cursor-pointer uppercase ${
                      isActive
                        ? 'bg-white text-black font-bold'
                        : 'text-white/60 hover:text-white hover:bg-white/5'
                    }`}
                  >
                    {`0${step.index + 1} ${step.id}`}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Central 3-Column Control Tower Stage */}
          <div className="w-full max-w-[1440px] mx-auto flex-1 grid grid-cols-1 lg:grid-cols-12 gap-4 py-4 min-h-0 z-10">
            
            {/* Left Column: Interactive Step Cards (4 Cols) */}
            <div className="hidden lg:flex lg:col-span-4 flex-col justify-between gap-2 h-full">
              {ECOSYSTEM_STEPS.map((step) => {
                const isActive = activeStep === step.index;
                return (
                  <button
                    key={step.id}
                    type="button"
                    onClick={() => scrollToStep(step.index)}
                    className={`flex-1 text-left p-4 border transition-all duration-300 flex flex-col justify-between cursor-pointer ${
                      isActive
                        ? 'border-white bg-[#121216] text-white shadow-[inset_0_0_20px_rgba(255,255,255,0.03)]'
                        : 'border-white/10 bg-[#09090B]/60 text-white/40 hover:border-white/25 hover:text-white/70'
                    }`}
                  >
                    <div>
                      <div className="flex items-center justify-between text-[10px] font-mono tracking-wider mb-1">
                        <span className={isActive ? 'text-emerald-400' : 'text-white/30'}>
                          {step.tag}
                        </span>
                        <span className="font-mono">{`[0${step.index + 1}/04]`}</span>
                      </div>
                      <h3 className="text-base font-bold tracking-tight text-white mb-1 font-mono uppercase">
                        {isRu ? step.titleRu : step.title}
                      </h3>
                      <p className="text-xs text-white/60 leading-relaxed font-sans line-clamp-2">
                        {isRu ? step.subtitleRu : step.subtitle}
                      </p>
                    </div>

                    {isActive && (
                      <div className="pt-2 mt-2 border-t border-white/10 flex items-center gap-2 text-[10px] font-mono text-emerald-400">
                        <span className="w-1.5 h-1.5 bg-emerald-400" />
                        <span>CAMERA_TARGET_LOCKED</span>
                      </div>
                    )}
                  </button>
                );
              })}
            </div>

            {/* Center Stage: The 3D Dither Stage (5 Cols on LG, Full on Mobile) */}
            <div className="lg:col-span-5 h-[340px] sm:h-[400px] lg:h-full relative border border-white/20 bg-[#060608] overflow-hidden flex flex-col">
              {/* Corner crosshairs */}
              <span className="absolute top-2 left-2 text-[10px] font-mono text-white/30 z-20 pointer-events-none">+</span>
              <span className="absolute top-2 right-2 text-[10px] font-mono text-white/30 z-20 pointer-events-none">+</span>
              <span className="absolute bottom-2 left-2 text-[10px] font-mono text-white/30 z-20 pointer-events-none">+</span>
              <span className="absolute bottom-2 right-2 text-[10px] font-mono text-white/30 z-20 pointer-events-none">+</span>

              {/* Stage Top Telemetry Overlay */}
              <div className="absolute top-3 left-4 right-4 flex items-center justify-between z-20 pointer-events-none text-[10px] font-mono tracking-widest text-white/60 bg-[#09090B]/80 px-2.5 py-1 border border-white/10 backdrop-blur-sm">
                <span className="truncate">{currentMeta.zoneCode}</span>
                <span className="hidden sm:inline text-white/40">DITHER_AA_SDF // 60FPS</span>
              </div>

              {/* Live WebGL Dither Stage */}
              <div className="w-full h-full flex-1 relative">
                <EcosystemDitherStage activeStep={activeStep} theme="dark" />
              </div>

              {/* Stage Bottom Status */}
              <div className="absolute bottom-3 left-4 right-4 flex items-center justify-between z-20 pointer-events-none text-[10px] font-mono tracking-widest text-white/50 bg-[#09090B]/80 px-2.5 py-1 border border-white/10 backdrop-blur-sm">
                <span>{`RENDER_TARGET: PASS_2_INK_DEPTH`}</span>
                <span className="text-emerald-400 font-bold">{`STEP 0${activeStep + 1} ACTIVE`}</span>
              </div>
            </div>

            {/* Right Column: Dynamic Telemetry Inspector (3 Cols) */}
            <div className="lg:col-span-3 flex flex-col justify-between gap-2 h-full">
              <div className="border border-white/15 bg-[#121216] p-4 flex-1 flex flex-col justify-between">
                <div>
                  <div className="text-[10px] font-mono tracking-wider text-white/40 mb-2 uppercase border-b border-white/10 pb-1">
                    {isRu ? 'ИНСПЕКТОР ПОКАЗАТЕЛЕЙ' : 'REALTIME TELEMETRY'}
                  </div>
                  <div className="text-lg font-bold font-mono text-white mb-4 uppercase">
                    {isRu ? currentMeta.titleRu : currentMeta.title}
                  </div>

                  {/* KPI Cards */}
                  <div className="space-y-4">
                    {currentMeta.kpis.map((kpi, kIdx) => (
                      <div key={kIdx} className="border border-white/10 bg-[#09090B] p-3">
                        <div className="text-[10px] font-mono text-white/50 uppercase mb-1">
                          {isRu ? kpi.labelRu : kpi.label}
                        </div>
                        <div className="text-2xl sm:text-3xl font-mono font-bold text-white tabular-nums">
                          {kpi.val}
                        </div>
                        <div className="text-[9px] font-mono text-emerald-400/80 mt-1 uppercase">
                          {kpi.sub}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="pt-4 border-t border-white/10 text-[10px] font-mono text-white/40 flex items-center justify-between">
                  <span>REFRESH_CYCLE: 1.0s</span>
                  <span className="text-white">STATUS: OK</span>
                </div>
              </div>
            </div>

          </div>

          {/* Bottom Hairline Scroll Progress Track */}
          <div className="w-full max-w-[1440px] mx-auto pt-3 border-t border-white/10 flex items-center justify-between gap-4 z-20">
            <div className="text-[10px] font-mono text-white/40 uppercase tracking-wider flex items-center gap-2">
              <span className="w-1.5 h-1.5 bg-white" />
              <span>{isRu ? 'ПРОКРУТИТЕ ДЛЯ СМЕНЫ ФАЗЫ' : 'SCROLL TO SCRUB PHASES'}</span>
            </div>

            <div className="flex-1 max-w-md h-1.5 bg-white/10 relative overflow-hidden border border-white/10">
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
    </PageSection>
  );
}
