'use client';

import React, { useState, useRef } from 'react';
import Link from 'next/link';
import { useLanguage } from '../context/LanguageContext';
import { useTheme } from '../context/ThemeContext';
import PageSection from './layout/PageSection';
import PixelTerrainCanvas from './six-roles/PixelTerrainCanvas';

interface RoleData {
  id: string;
  name: string;
  nameRu: string;
  title: string;
  titleRu: string;
  scope: string;
  scopeRu: string;
  tags: string[];
  tagsRu: string[];
  devices: string[];
  status: string;
  glyph: number[][]; // 6x6 pixel bitmap
}

// 6x6 Pixel art bitmaps for each role
const PIXEL_GLYPHS: Record<string, number[][]> = {
  supplier: [
    [0, 1, 1, 1, 1, 0],
    [1, 0, 1, 1, 0, 1],
    [1, 1, 1, 1, 1, 1],
    [1, 0, 1, 1, 0, 1],
    [0, 1, 1, 1, 1, 0],
    [0, 0, 1, 1, 0, 0],
  ],
  warehouse: [
    [1, 1, 1, 1, 1, 1],
    [1, 0, 0, 0, 0, 1],
    [1, 1, 1, 1, 1, 1],
    [1, 0, 1, 1, 0, 1],
    [1, 0, 1, 1, 0, 1],
    [1, 1, 1, 1, 1, 1],
  ],
  factory: [
    [0, 1, 0, 0, 1, 0],
    [0, 1, 0, 0, 1, 0],
    [1, 1, 0, 1, 1, 0],
    [1, 1, 1, 1, 1, 1],
    [1, 0, 1, 0, 0, 1],
    [1, 1, 1, 1, 1, 1],
  ],
  driver: [
    [0, 1, 1, 1, 0, 0],
    [1, 1, 1, 1, 1, 0],
    [1, 0, 1, 1, 0, 1],
    [1, 1, 1, 1, 1, 1],
    [0, 1, 0, 0, 1, 0],
    [0, 1, 0, 0, 1, 0],
  ],
  retailer: [
    [0, 1, 1, 1, 1, 0],
    [1, 0, 0, 0, 0, 1],
    [1, 1, 1, 1, 1, 1],
    [1, 0, 1, 1, 0, 1],
    [1, 0, 0, 0, 0, 1],
    [1, 1, 1, 1, 1, 1],
  ],
  payload: [
    [1, 0, 0, 0, 0, 1],
    [0, 1, 0, 0, 1, 0],
    [0, 0, 1, 1, 0, 0],
    [0, 0, 1, 1, 0, 0],
    [0, 1, 0, 0, 1, 0],
    [1, 0, 0, 0, 0, 1],
  ],
};

function PixelGlyph({ bitmap, className = '' }: { bitmap: number[][]; className?: string }) {
  return (
    <svg viewBox="0 0 6 6" className={`w-4 h-4 fill-current ${className}`} aria-hidden="true">
      {bitmap.map((row, r) =>
        row.map((val, c) => (val === 1 ? <rect key={`${r}-${c}`} x={c} y={r} width="1" height="1" /> : null))
      )}
    </svg>
  );
}

// 8-bit Pegasus Wing Pixel Icon for Header (matching reference style)
function PixelPegasusWing() {
  return (
    <svg viewBox="0 0 10 8" className="w-5 h-4 fill-black dark:fill-white" aria-hidden="true">
      <rect x="1" y="1" width="1" height="1" />
      <rect x="2" y="0" width="2" height="1" />
      <rect x="4" y="1" width="2" height="1" />
      <rect x="6" y="0" width="2" height="1" />
      <rect x="8" y="1" width="1" height="1" />
      <rect x="0" y="2" width="10" height="2" />
      <rect x="1" y="4" width="8" height="2" />
      <rect x="3" y="6" width="4" height="1" />
      <rect x="4" y="7" width="2" height="1" />
    </svg>
  );
}

const ROLES_DATA: RoleData[] = [
  {
    id: 'supplier',
    name: 'Supplier',
    nameRu: 'Поставщик',
    title: 'Network Control',
    titleRu: 'Контроль сети',
    scope: 'Network topology, STIR vetting, automated treasury clearing, multi-warehouse dispatch, and SKU catalog.',
    scopeRu: 'Топология сети, проверка ИНН/STIR, клиринг казначейства, мультискладская диспетчеризация и каталог SKU.',
    tags: ['Vetting', 'Topology', 'Treasury', 'Dispatch'],
    tagsRu: ['Проверка', 'Топология', 'Казначейство', 'Диспетчеризация'],
    devices: ['PORTAL', 'MOBILE'],
    status: 'CORE STATE',
    glyph: PIXEL_GLYPHS.supplier,
  },
  {
    id: 'warehouse',
    name: 'Warehouse',
    nameRu: 'Склад',
    title: 'Dispatch Hub',
    titleRu: 'Хаб диспетчеризации',
    scope: 'Real-time stock flow, pick/pack waves, fleet shift pairing, dock bay queues, and DVIR inspection logs.',
    scopeRu: 'Движение остатков, волны сборки, смены водителей, очереди рамп и акты осмотра ТС (DVIR).',
    tags: ['Pre-orders', 'Stock Wave', 'Fleet Pairing', 'Dock Bay'],
    tagsRu: ['Предзаказы', 'Сборка', 'Смены', 'Рампы'],
    devices: ['PORTAL', 'ANDROID'],
    status: 'CORE STATE',
    glyph: PIXEL_GLYPHS.warehouse,
  },
  {
    id: 'factory',
    name: 'Factory',
    nameRu: 'Завод',
    title: 'Loading & Seal',
    titleRu: 'Погрузка и пломба',
    scope: 'Bulk production manifests, loading lane priority, tamper seal verification, and SSCC-18 pallet tracking.',
    scopeRu: 'Производственные манифесты, приоритет полос погрузки, номерные пломбы и паллетный учет SSCC-18.',
    tags: ['Manifests', 'Loading Lanes', 'Seal Logs', 'SSCC-18'],
    tagsRu: ['Манифесты', 'Полосы погрузки', 'Пломбы', 'SSCC-18'],
    devices: ['PORTAL', 'MOBILE'],
    status: 'CORE STATE',
    glyph: PIXEL_GLYPHS.factory,
  },
  {
    id: 'driver',
    name: 'Driver',
    nameRu: 'Водитель',
    title: 'Field Execution',
    titleRu: 'Исполнение в поле',
    scope: 'Daily vehicle check, dynamic CVRP route stops, doorstep geofenced validation, COD cash drawers, and ePoD.',
    scopeRu: 'Предрейсовый чеклист, динамический CVRP-маршрут, геозона вручения (<150м), прием наличных и ePoD.',
    tags: ['Dynamic Routes', 'ePoD', 'Cash COD', 'Hot-Swap'],
    tagsRu: ['Маршруты', 'ePoD', 'Наличные', 'Хот-свап'],
    devices: ['ANDROID', 'IOS'],
    status: 'CORE STATE',
    glyph: PIXEL_GLYPHS.driver,
  },
  {
    id: 'retailer',
    name: 'Retailer',
    nameRu: 'Ритейлер',
    title: 'Commerce & Tracking',
    titleRu: 'Коммерция и трекинг',
    scope: 'Live wholesale catalog, 25M UZS B2B payment gating, real-time doorstep delivery GPS, and E-Factura.',
    scopeRu: 'Оптовый каталог, лимит 25 млн сум по B2B-наличным, живой GPS-трекинг доставки и электронные счета-фактуры.',
    tags: ['Catalog', 'Fast Checkout', 'Live GPS', 'E-Factura'],
    tagsRu: ['Каталог', 'Заказ', 'GPS-трекинг', 'ЭСФ'],
    devices: ['DESKTOP', 'MOBILE'],
    status: 'CORE STATE',
    glyph: PIXEL_GLYPHS.retailer,
  },
  {
    id: 'payload',
    name: 'Payload',
    nameRu: 'Терминал (Payload)',
    title: 'Gate Control',
    titleRu: 'Контроль ворот',
    scope: 'Barcode scanning, automated weight bridge integration, gate seal integrity, and turnaround SLA tracking.',
    scopeRu: 'Сканирование штрихкодов, интеграция автовесов, целостность пломб и контроль SLA простоя у ворот.',
    tags: ['Gate Scan', 'Weight Scales', 'Seal Audit', 'Turnaround'],
    tagsRu: ['Скан ворот', 'Автовесы', 'Аудит пломб', 'Оборот ТС'],
    devices: ['TERMINAL', 'MOBILE'],
    status: 'CORE STATE',
    glyph: PIXEL_GLYPHS.payload,
  },
];

export default function Companies() {
  const { language } = useLanguage();
  const { resolvedTheme } = useTheme();
  const isLight = resolvedTheme === 'light';
  const isRu = language === 'ru';

  const [activeRoleIndex, setActiveRoleIndex] = useState<number | null>(null);
  const [hoveredCellInfo, setHoveredCellInfo] = useState<{ col: number; row: number; type: string } | null>(null);

  const handleCellHover = (col: number, row: number, type: 'light' | 'dark') => {
    setHoveredCellInfo({ col, row, type });
  };

  return (
    <PageSection
      id="companies"
      bleed
      className="overflow-hidden bg-white dark:bg-[#09090B] text-zinc-900 dark:text-white py-12 md:py-16 border-y border-black/10 dark:border-white/10"
    >
      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8">

        {/* Tactical Header Breadcrumb / Meta Bar */}
        <div className="flex flex-wrap items-center justify-between gap-3 pb-4 mb-4 border-b border-black/10 dark:border-white/10 text-xs font-mono">
          <div className="flex items-center gap-2.5">
            <span className="inline-block w-2 h-2 bg-emerald-500 rounded-none animate-pulse" />
            <span className="font-semibold uppercase tracking-wider text-black dark:text-white">
              {isRu ? 'ТОПОЛОГИЯ СЕТИ // ШЕСТЬ РОЛЕЙ' : 'SYSTEM TOPOLOGY // SIX OPERATIONAL ROLES'}
            </span>
            <span className="text-zinc-400 dark:text-zinc-600 hidden sm:inline">|</span>
            <span className="text-zinc-500 dark:text-zinc-400 hidden sm:inline text-[11px]">
              {isRu ? 'ОБЩИЙ РАСПРЕДЕЛЕННЫЙ АВТОМАТ' : 'GOVERNED DISTRIBUTED STATE MACHINE'}
            </span>
          </div>

          <div className="flex items-center gap-4 text-[11px] text-zinc-500 dark:text-zinc-400">
            <span>
              {isRu ? 'УЗЕЛ: ТАШКЕНТ-ЦЕНТРАЛ' : 'NODE: TAS-IX CENTRAL'}
            </span>
            <span>·</span>
            <span className="text-emerald-600 dark:text-emerald-400 font-medium">
              [LATENCY: &lt;4ms]
            </span>
          </div>
        </div>

        {/* Main Pixel Terrain Stage (The exact aesthetic from user's reference screenshot) */}
        <div className="relative border border-black/15 dark:border-white/15 rounded-none overflow-hidden h-[460px] sm:h-[520px] lg:h-[580px] select-none bg-[#EDEDED] shadow-sm">
          
          {/* Interactive Canvas Grid Terrain */}
          <div className="absolute inset-0 z-0">
            <PixelTerrainCanvas
              onCellHover={handleCellHover}
              activeRoleIndex={activeRoleIndex}
              className="w-full h-full"
            />
          </div>

          {/* Foreground Tactical Overlays (Positioned to match the editorial layout of the reference image) */}
          <div className="relative z-10 w-full h-full flex flex-col justify-between p-6 sm:p-8 md:p-10 pointer-events-none">
            
            {/* Top Row: Brand / Section identifier and Role Links */}
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-2.5 bg-white/90 dark:bg-black/80 px-3 py-1.5 border border-black/10 dark:border-white/15 backdrop-blur-sm pointer-events-auto">
                <PixelPegasusWing />
                <span className="text-[11px] font-mono uppercase tracking-widest font-semibold text-black dark:text-white">
                  Pegasus
                </span>
                <span className="text-[10px] font-mono text-zinc-400 dark:text-zinc-500">
                  // OS
                </span>
              </div>

              {/* Minimal Role Tabs in Top-Right */}
              <div className="hidden md:flex items-center gap-2 pointer-events-auto">
                {ROLES_DATA.map((role, idx) => (
                  <button
                    key={role.id}
                    onMouseEnter={() => setActiveRoleIndex(idx)}
                    onMouseLeave={() => setActiveRoleIndex(null)}
                    onClick={() => setActiveRoleIndex(idx === activeRoleIndex ? null : idx)}
                    className={`px-2.5 py-1 text-[10px] font-mono uppercase tracking-wider transition-colors duration-150 border cursor-pointer ${
                      activeRoleIndex === idx
                        ? 'bg-black text-white dark:bg-white dark:text-black border-black dark:border-white font-bold'
                        : 'bg-white/85 dark:bg-black/85 text-zinc-700 dark:text-zinc-300 border-black/10 dark:border-white/15 hover:border-black dark:hover:border-white'
                    }`}
                  >
                    0{idx + 1}. {isRu ? role.nameRu : role.name}
                  </button>
                ))}
              </div>
            </div>

            {/* Middle Left: Large Stark Headline (Matching "Adulthood is hard." in reference) */}
            <div className="max-w-xl pointer-events-auto py-4">
              <h2 className="text-4xl sm:text-6xl md:text-7xl lg:text-8xl font-medium tracking-tight text-black leading-[0.92] select-none drop-shadow-sm">
                {isRu ? (
                  <>
                    Шесть ролей.<br />
                    Одна сеть.
                  </>
                ) : (
                  <>
                    Six roles.<br />
                    One network.
                  </>
                )}
              </h2>
            </div>

            {/* Bottom Row: Mission description in the dark zone and interactive coordinates */}
            <div className="flex flex-col sm:flex-row items-end sm:items-end justify-between gap-4">
              {/* Live Cursor Coordinate Pill */}
              <div className="bg-black/85 text-white/90 px-3 py-1.5 border border-white/20 text-[10px] font-mono uppercase tracking-wider backdrop-blur-sm pointer-events-auto hidden sm:flex items-center gap-2">
                <span className="w-1.5 h-1.5 bg-blue-500" />
                <span>
                  {hoveredCellInfo
                    ? `GRID [C:${hoveredCellInfo.col} R:${hoveredCellInfo.row}] · SECTOR: ${hoveredCellInfo.type.toUpperCase()}`
                    : 'GRID TERRAIN · HOVER TO INSPECT'}
                </span>
              </div>

              {/* Bottom Right Editorial Description (Matching bottom right paragraph in reference) */}
              <div className="max-w-md bg-black/85 text-white p-4 sm:p-5 border border-white/15 backdrop-blur-md pointer-events-auto">
                <p className="text-xs sm:text-[13px] text-zinc-300 font-normal leading-relaxed">
                  {isRu
                    ? 'Синхронизация диспетчеризации, трекинга автопарка, инкассации и передачи грузов в едином конечном автомате. Нулевая фрагментация между шестью участниками.'
                    : 'Synchronizing dispatch, fleet tracking, cash collection, and dock handovers into one governed state machine. Zero blind spots across all six actors.'}
                </p>
                <div className="mt-3 pt-2.5 border-t border-white/15 flex items-center justify-between text-[10px] font-mono text-emerald-400">
                  <span className="flex items-center gap-1.5 uppercase tracking-wider">
                    <span className="w-1.5 h-1.5 bg-emerald-400 animate-ping" />
                    [LIVE SYSTEM OF RECORD]
                  </span>
                  <span className="text-zinc-400">
                    LATENCY: REAL-TIME
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* The Six Rectilinear Role Panels (Strictly NO ROUNDED CARDS - 0px border radius) */}
        <div className="mt-0 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 border-x border-b border-black/15 dark:border-white/15 divide-y sm:divide-y-0 sm:divide-x divide-black/15 dark:divide-white/15 bg-white dark:bg-[#0B0B0E]">
          {ROLES_DATA.map((role, idx) => {
            const isSelected = activeRoleIndex === idx;

            return (
              <div
                key={role.id}
                onMouseEnter={() => setActiveRoleIndex(idx)}
                onMouseLeave={() => setActiveRoleIndex(null)}
                className={`p-5 flex flex-col justify-between transition-all duration-200 cursor-pointer rounded-none group ${
                  isSelected
                    ? 'bg-blue-50/70 dark:bg-blue-950/20 ring-1 ring-inset ring-blue-500'
                    : 'hover:bg-zinc-50 dark:hover:bg-[#121217]'
                }`}
              >
                {/* Header row: Monospace Index + Pixel Glyph */}
                <div>
                  <div className="flex items-center justify-between mb-4 pb-2 border-b border-black/10 dark:border-white/10">
                    <span className="font-mono text-[11px] font-bold tracking-widest text-zinc-400 dark:text-zinc-500 group-hover:text-blue-600 dark:group-hover:text-blue-400">
                      [0{idx + 1}]
                    </span>
                    <div className="p-1 border border-black/15 dark:border-white/15 text-zinc-900 dark:text-white group-hover:border-blue-500 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                      <PixelGlyph bitmap={role.glyph} />
                    </div>
                  </div>

                  {/* Role Name and Title */}
                  <div className="mb-3">
                    <h3 className="text-base font-semibold tracking-tight text-zinc-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                      {isRu ? role.nameRu : role.name}
                    </h3>
                    <p className="text-[11px] font-mono uppercase tracking-wider text-emerald-600 dark:text-emerald-400 mt-0.5">
                      {isRu ? role.titleRu : role.title}
                    </p>
                  </div>

                  {/* Scope Details */}
                  <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed mb-4">
                    {isRu ? role.scopeRu : role.scope}
                  </p>
                </div>

                {/* Footer: Surface chips (Strictly rectangular) */}
                <div className="pt-3 border-t border-black/10 dark:border-white/10 mt-2">
                  <div className="flex flex-wrap gap-1">
                    {role.devices.map((device) => (
                      <span
                        key={device}
                        className="px-1.5 py-0.5 text-[9px] font-mono uppercase tracking-wider border border-black/15 dark:border-white/15 text-zinc-600 dark:text-zinc-400 bg-black/[0.03] dark:bg-white/[0.03] rounded-none"
                      >
                        {device}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

      </div>
    </PageSection>
  );
}
