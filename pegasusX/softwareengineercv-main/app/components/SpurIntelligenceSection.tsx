'use client';

import React, { useState, useRef, useCallback } from 'react';
import Link from 'next/link';
import { useLanguage } from '@/app/context/LanguageContext';

type SpurIntelligenceSectionProps = {
  eyebrow?: string;
  headlineLine1?: string;
  headlineLine2?: string;
  description?: string;
  demoLabel?: string;
  demoHref?: string;
  trialLabel?: string;
  trialHref?: string;
  className?: string;
};

// 32 cols x 18 rows pixel frontier with Minesweeper neighbor clues [row, col, is_dark, clue_num]
const GRID_CELLS: [number, number, number, number][] = [
  [0,0,0,0], [0,1,0,0], [0,2,0,0], [0,3,0,0], [0,4,0,0], [0,5,0,0], [0,6,0,0], [0,7,0,0],
  [0,8,0,0], [0,9,0,0], [0,10,0,0], [0,11,0,0], [0,12,0,0], [0,13,0,2], [0,14,1,0], [0,15,1,0],
  [0,16,0,2], [0,17,0,0], [0,18,0,0], [0,19,0,2], [0,20,1,0], [0,21,1,0], [0,22,1,0], [0,23,1,0],
  [0,24,1,0], [0,25,1,0], [0,26,1,0], [0,27,1,0], [0,28,1,0], [0,29,1,0], [0,30,1,0], [0,31,1,0],
  [1,0,0,0], [1,1,0,0], [1,2,0,0], [1,3,0,0], [1,4,0,0], [1,5,0,0], [1,6,0,0], [1,7,0,0],
  [1,8,0,0], [1,9,0,0], [1,10,0,0], [1,11,0,0], [1,12,0,0], [1,13,0,2], [1,14,1,0], [1,15,1,0],
  [1,16,0,2], [1,17,0,0], [1,18,0,1], [1,19,0,3], [1,20,1,0], [1,21,1,0], [1,22,1,0], [1,23,1,0],
  [1,24,1,0], [1,25,1,0], [1,26,1,0], [1,27,1,0], [1,28,1,0], [1,29,1,0], [1,30,1,0], [1,31,1,0],
  [2,0,0,0], [2,1,0,0], [2,2,0,0], [2,3,0,0], [2,4,0,0], [2,5,0,0], [2,6,0,0], [2,7,0,0],
  [2,8,0,0], [2,9,0,1], [2,10,0,1], [2,11,0,1], [2,12,0,0], [2,13,0,1], [2,14,0,2], [2,15,0,2],
  [2,16,0,1], [2,17,0,1], [2,18,0,3], [2,19,1,0], [2,20,1,0], [2,21,1,0], [2,22,1,0], [2,23,1,0],
  [2,24,1,0], [2,25,1,0], [2,26,1,0], [2,27,1,0], [2,28,1,0], [2,29,1,0], [2,30,1,0], [2,31,1,0],
  [3,0,0,0], [3,1,0,0], [3,2,0,0], [3,3,0,0], [3,4,0,0], [3,5,0,0], [3,6,0,0], [3,7,0,0],
  [3,8,0,0], [3,9,0,2], [3,10,1,0], [3,11,0,2], [3,12,0,0], [3,13,0,0], [3,14,0,0], [3,15,0,0],
  [3,16,0,1], [3,17,0,3], [3,18,1,0], [3,19,1,0], [3,20,1,0], [3,21,1,0], [3,22,1,0], [3,23,1,0],
  [3,24,1,0], [3,25,1,0], [3,26,1,0], [3,27,1,0], [3,28,1,0], [3,29,1,0], [3,30,1,0], [3,31,1,0],
  [4,0,0,0], [4,1,0,0], [4,2,0,0], [4,3,0,0], [4,4,0,0], [4,5,0,0], [4,6,0,0], [4,7,0,0],
  [4,8,0,0], [4,9,0,2], [4,10,1,0], [4,11,0,2], [4,12,0,0], [4,13,0,0], [4,14,0,1], [4,15,0,2],
  [4,16,0,3], [4,17,1,0], [4,18,1,0], [4,19,1,0], [4,20,1,0], [4,21,1,0], [4,22,1,0], [4,23,1,0],
  [4,24,1,0], [4,25,1,0], [4,26,1,0], [4,27,1,0], [4,28,1,0], [4,29,1,0], [4,30,1,0], [4,31,1,0],
  [5,0,0,0], [5,1,0,0], [5,2,0,0], [5,3,0,0], [5,4,0,0], [5,5,0,0], [5,6,0,0], [5,7,0,0],
  [5,8,0,0], [5,9,0,1], [5,10,0,1], [5,11,0,1], [5,12,0,0], [5,13,0,1], [5,14,0,3], [5,15,1,0],
  [5,16,1,0], [5,17,1,0], [5,18,1,0], [5,19,1,0], [5,20,1,0], [5,21,1,0], [5,22,1,1], [5,23,1,1],
  [5,24,1,1], [5,25,1,0], [5,26,1,0], [5,27,1,0], [5,28,1,0], [5,29,1,0], [5,30,1,0], [5,31,1,0],
  [6,0,0,0], [6,1,0,0], [6,2,0,0], [6,3,0,0], [6,4,0,0], [6,5,0,1], [6,6,0,1], [6,7,0,1],
  [6,8,0,0], [6,9,0,0], [6,10,0,0], [6,11,0,0], [6,12,0,1], [6,13,0,3], [6,14,1,0], [6,15,1,0],
  [6,16,1,0], [6,17,1,0], [6,18,1,0], [6,19,1,0], [6,20,1,0], [6,21,1,0], [6,22,1,2], [6,23,0,3],
  [6,24,1,0], [6,25,1,1], [6,26,1,0], [6,27,1,0], [6,28,1,0], [6,29,1,0], [6,30,1,0], [6,31,1,0],
  [7,0,0,0], [7,1,0,0], [7,2,0,0], [7,3,0,0], [7,4,0,0], [7,5,0,2], [7,6,1,0], [7,7,0,2],
  [7,8,0,0], [7,9,0,0], [7,10,0,1], [7,11,0,2], [7,12,0,3], [7,13,1,0], [7,14,1,0], [7,15,1,0],
  [7,16,1,0], [7,17,1,0], [7,18,1,0], [7,19,1,0], [7,20,1,0], [7,21,1,0], [7,22,1,2], [7,23,0,3],
  [7,24,0,3], [7,25,1,1], [7,26,1,0], [7,27,1,0], [7,28,1,0], [7,29,1,0], [7,30,1,0], [7,31,1,0],
  [8,0,0,0], [8,1,0,0], [8,2,0,0], [8,3,0,0], [8,4,0,0], [8,5,0,2], [8,6,1,0], [8,7,0,2],
  [8,8,0,0], [8,9,0,0], [8,10,0,2], [8,11,1,0], [8,12,1,0], [8,13,1,0], [8,14,1,0], [8,15,1,0],
  [8,16,1,0], [8,17,1,0], [8,18,1,0], [8,19,1,0], [8,20,1,0], [8,21,1,0], [8,22,1,1], [8,23,1,2],
  [8,24,1,2], [8,25,1,1], [8,26,1,0], [8,27,1,0], [8,28,1,0], [8,29,1,0], [8,30,1,0], [8,31,1,0],
  [9,0,0,0], [9,1,0,0], [9,2,0,0], [9,3,0,0], [9,4,0,0], [9,5,0,1], [9,6,0,1], [9,7,0,1],
  [9,8,0,0], [9,9,0,0], [9,10,0,3], [9,11,1,0], [9,12,1,0], [9,13,1,0], [9,14,1,0], [9,15,1,0],
  [9,16,1,0], [9,17,1,0], [9,18,1,0], [9,19,1,0], [9,20,1,0], [9,21,1,0], [9,22,1,0], [9,23,1,0],
  [9,24,1,0], [9,25,1,0], [9,26,1,1], [9,27,1,1], [9,28,1,1], [9,29,1,0], [9,30,1,0], [9,31,1,0],
  [10,0,0,0], [10,1,0,0], [10,2,0,1], [10,3,0,1], [10,4,0,1], [10,5,0,0], [10,6,0,0], [10,7,0,0],
  [10,8,0,0], [10,9,0,1], [10,10,0,3], [10,11,1,0], [10,12,1,0], [10,13,1,0], [10,14,1,0], [10,15,1,0],
  [10,16,1,0], [10,17,1,0], [10,18,1,0], [10,19,1,0], [10,20,1,0], [10,21,1,0], [10,22,1,0], [10,23,1,0],
  [10,24,1,0], [10,25,1,0], [10,26,1,2], [10,27,0,3], [10,28,1,2], [10,29,1,0], [10,30,1,0], [10,31,1,0],
  [11,0,0,0], [11,1,0,0], [11,2,0,2], [11,3,1,0], [11,4,0,2], [11,5,0,0], [11,6,0,0], [11,7,0,0],
  [11,8,0,1], [11,9,0,3], [11,10,1,0], [11,11,1,0], [11,12,1,0], [11,13,1,0], [11,14,1,0], [11,15,1,0],
  [11,16,1,0], [11,17,1,0], [11,18,1,0], [11,19,0,3], [11,20,1,0], [11,21,1,1], [11,22,1,0], [11,23,1,0],
  [11,24,1,0], [11,25,1,0], [11,26,1,2], [11,27,0,3], [11,28,1,2], [11,29,1,0], [11,30,1,0], [11,31,1,0],
  [12,0,0,0], [12,1,0,0], [12,2,0,2], [12,3,1,0], [12,4,0,2], [12,5,0,0], [12,6,0,0], [12,7,0,1],
  [12,8,0,3], [12,9,1,0], [12,10,1,0], [12,11,1,0], [12,12,1,0], [12,13,1,0], [12,14,1,0], [12,15,1,0],
  [12,16,1,0], [12,17,1,0], [12,18,1,0], [12,19,0,3], [12,20,0,3], [12,21,1,1], [12,22,1,0], [12,23,1,0],
  [12,24,1,0], [12,25,1,0], [12,26,1,1], [12,27,1,1], [12,28,1,1], [12,29,1,0], [12,30,1,0], [12,31,1,0],
  [13,0,0,0], [13,1,0,0], [13,2,0,1], [13,3,0,1], [13,4,0,1], [13,5,0,0], [13,6,0,1], [13,7,0,3],
  [13,8,1,0], [13,9,1,0], [13,10,1,0], [13,11,1,0], [13,12,1,0], [13,13,1,0], [13,14,1,0], [13,15,1,0],
  [13,16,1,0], [13,17,1,0], [13,18,1,0], [13,19,1,0], [13,20,1,0], [13,21,1,2], [13,22,1,1], [13,23,1,1],
  [13,24,1,0], [13,25,1,0], [13,26,1,0], [13,27,1,0], [13,28,1,0], [13,29,1,0], [13,30,1,0], [13,31,1,0],
  [14,0,0,0], [14,1,0,0], [14,2,0,0], [14,3,0,0], [14,4,0,0], [14,5,0,1], [14,6,0,3], [14,7,1,0],
  [14,8,1,0], [14,9,1,0], [14,10,1,0], [14,11,1,0], [14,12,1,0], [14,13,1,0], [14,14,1,0], [14,15,1,0],
  [14,16,0,3], [14,17,1,0], [14,18,1,0], [14,19,1,0], [14,20,1,0], [14,21,1,1], [14,22,0,3], [14,23,1,1],
  [14,24,1,0], [14,25,1,0], [14,26,1,0], [14,27,1,0], [14,28,1,0], [14,29,1,0], [14,30,1,0], [14,31,1,0],
  [15,0,0,0], [15,1,0,0], [15,2,0,0], [15,3,0,0], [15,4,0,1], [15,5,0,3], [15,6,1,0], [15,7,1,0],
  [15,8,1,0], [15,9,1,0], [15,10,1,0], [15,11,1,0], [15,12,1,0], [15,13,1,0], [15,14,1,0], [15,15,1,0],
  [15,16,0,3], [15,17,0,3], [15,18,1,0], [15,19,1,0], [15,20,1,0], [15,21,1,1], [15,22,1,1], [15,23,1,1],
  [15,24,1,0], [15,25,1,0], [15,26,1,0], [15,27,1,0], [15,28,1,0], [15,29,1,0], [15,30,1,0], [15,31,1,0],
  [16,0,0,0], [16,1,0,0], [16,2,0,0], [16,3,0,1], [16,4,0,3], [16,5,1,0], [16,6,1,0], [16,7,1,0],
  [16,8,1,0], [16,9,1,0], [16,10,1,0], [16,11,1,0], [16,12,1,0], [16,13,1,0], [16,14,1,0], [16,15,1,0],
  [16,16,1,0], [16,17,1,0], [16,18,1,0], [16,19,1,0], [16,20,1,0], [16,21,1,0], [16,22,1,0], [16,23,1,0],
  [16,24,1,0], [16,25,1,0], [16,26,1,0], [16,27,1,0], [16,28,1,0], [16,29,1,0], [16,30,1,0], [16,31,1,0],
  [17,0,0,0], [17,1,0,0], [17,2,0,0], [17,3,0,1], [17,4,1,0], [17,5,1,0], [17,6,1,0], [17,7,1,0],
  [17,8,1,0], [17,9,1,0], [17,10,1,0], [17,11,1,0], [17,12,1,0], [17,13,1,0], [17,14,1,0], [17,15,1,0],
  [17,16,1,0], [17,17,1,0], [17,18,1,0], [17,19,1,0], [17,20,1,0], [17,21,1,0], [17,22,1,0], [17,23,1,0],
  [17,24,1,0], [17,25,1,0], [17,26,1,0], [17,27,1,0], [17,28,1,0], [17,29,1,0], [17,30,1,0], [17,31,1,0],
];

export default function SpurIntelligenceSection({
  eyebrow,
  headlineLine1,
  headlineLine2,
  description,
  demoLabel,
  demoHref = '/join',
  trialLabel,
  trialHref = '/demo',
  className = '',
}: SpurIntelligenceSectionProps) {
  const { language } = useLanguage();
  const isRu = language === 'ru';

  const finalEyebrow = eyebrow ?? (isRu ? 'АГЕНТСТВО АВТОНОМНЫХ ОПЕРАЦИЙ' : 'AUTONOMOUS OPERATIONS AGENCY');
  const finalH1 = headlineLine1 ?? (isRu ? 'Таблицы — это сложно.' : 'Spreadsheets are hard.');
  const finalH2 = headlineLine2 ?? (isRu ? 'Pegasus — нет.' : 'Pegasus is not.');
  const finalDesc =
    description ??
    (isRu
      ? 'Помогаем диспетчерам, складам и транспортным сетям перейти от ручных таблиц к автоматизированной системе на базе ИИ.'
      : 'Helping modern enterprises (regional fleets or global 3PLs) transition from manual spreadsheets to real-time autonomous logistics.');
  const finalDemo = demoLabel ?? (isRu ? 'ЗАПРОСИТЬ ДЕМО' : 'REQUEST A DEMO');
  const finalTrial = trialLabel ?? (isRu ? 'НАЧАТЬ ТЕСТ' : 'START FREE TRIAL');

  // Interactive hovered cell state [row, col]
  const [hoveredCell, setHoveredCell] = useState<{ r: number; c: number } | null>({ r: 6, c: 20 });
  const svgRef = useRef<SVGSVGElement | null>(null);

  const handlePointerMove = useCallback((e: React.PointerEvent<SVGSVGElement>) => {
    if (!svgRef.current) return;
    const rect = svgRef.current.getBoundingClientRect();
    const x = Math.max(0, Math.min(rect.width - 1, e.clientX - rect.left));
    const y = Math.max(0, Math.min(rect.height - 1, e.clientY - rect.top));
    const c = Math.floor((x / rect.width) * 32);
    const r = Math.floor((y / rect.height) * 18);
    setHoveredCell({ r: Math.max(0, Math.min(17, r)), c: Math.max(0, Math.min(31, c)) });
  }, []);

  const handlePointerLeave = useCallback(() => {
    // Return to iconic spotlight position matching reference image
    setHoveredCell({ r: 6, c: 20 });
  }, []);

  return (
    <section
      aria-label="Intelligence Overview"
      className={`relative w-full bg-black py-10 sm:py-16 md:py-20 px-3 sm:px-6 md:px-10 lg:px-16 overflow-hidden select-none ${className}`}
    >
      <div className="w-full max-w-[1380px] mx-auto">
        {/* Main Aesthetic Card with Minesweeper Binary Pixel Grid */}
        <div className="relative w-full rounded-2xl md:rounded-3xl overflow-hidden border border-white/10 shadow-2xl bg-[#1c1c1c]">
          
          {/* Background SVG Grid Canvas */}
          <div className="absolute inset-0 w-full h-full pointer-events-auto">
            <svg
              ref={svgRef}
              viewBox="0 0 1024 576"
              preserveAspectRatio="none"
              className="w-full h-full block cursor-crosshair"
              shapeRendering="crispEdges"
              onPointerMove={handlePointerMove}
              onPointerLeave={handlePointerLeave}
            >
              {/* Default Light Background */}
              <rect x="0" y="0" width="1024" height="576" fill="#dedede" />

              {/* Grid cells */}
              {GRID_CELLS.map(([r, c, dark, num], i) => {
                const x = c * 32;
                const y = r * 32;
                const isHovered = hoveredCell?.r === r && hoveredCell?.c === c;

                return (
                  <g key={i}>
                    {/* Dark filled block */}
                    {dark === 1 && (
                      <rect
                        x={x}
                        y={y}
                        width={32}
                        height={32}
                        fill="#1c1c1c"
                      />
                    )}

                    {/* Faint border to accentuate pixel grid texture */}
                    <rect
                      x={x}
                      y={y}
                      width={32}
                      height={32}
                      fill="none"
                      stroke={dark === 1 ? 'rgba(255,255,255,0.025)' : 'rgba(0,0,0,0.035)'}
                      strokeWidth="1"
                    />

                    {/* Numeric Clue (Minesweeper numbers) */}
                    {num > 0 && !isHovered && (
                      <text
                        x={x + 16}
                        y={y + 20}
                        textAnchor="middle"
                        fill={dark === 1 ? '#555555' : '#8c8c8c'}
                        fontSize="9.5"
                        fontFamily="ui-monospace, SFMono-Regular, Menlo, monospace"
                        fontWeight="500"
                        className="pointer-events-none select-none"
                      >
                        {num}
                      </text>
                    )}

                    {/* Active Hover Glow / Focus Indicator */}
                    {isHovered && (
                      <g className="pointer-events-none">
                        <rect
                          x={x}
                          y={y}
                          width={32}
                          height={32}
                          fill="#202866"
                          stroke="#485fc7"
                          strokeWidth="1.2"
                        />
                      </g>
                    )}
                  </g>
                );
              })}

              {/* Glowing Interactive Cursor Indicator over active cell */}
              {hoveredCell && (
                <g
                  className="pointer-events-none transition-all duration-75 ease-out"
                  transform={`translate(${hoveredCell.c * 32 + 14}, ${hoveredCell.r * 32 + 12})`}
                >
                  <path
                    d="M0,0 L0,15 L4,11 L7,17 L9.5,15.8 L6.5,10 L11.5,10 Z"
                    fill="#ffffff"
                    stroke="#141414"
                    strokeWidth="1.2"
                    strokeLinejoin="round"
                  />
                </g>
              )}
            </svg>
          </div>

          {/* Foreground UI Layer */}
          <div className="relative z-10 pointer-events-none flex flex-col justify-between min-h-[580px] sm:min-h-[640px] md:min-h-[700px] p-6 sm:p-10 md:p-14 lg:p-16">
            
            {/* Top Navigation / Brand Bar */}
            <div className="w-full flex items-center justify-between pb-8 pointer-events-auto">
              {/* Brand: 8-bit Pixel Heart + Title */}
              <div className="flex items-center gap-3">
                <svg className="w-6 h-6 text-black shrink-0" viewBox="0 0 16 16" fill="currentColor">
                  {/* 8-bit Pixel Heart */}
                  <path d="M3 2h3v2H3zM10 2h3v2h-3zM2 4h5v2H2zM9 4h5v2H9zM1 6h14v3H1zM2 9h12v2H2zM4 11h8v2H4zM6 13h4v1H6zM7 14h2v1H7z" />
                </svg>
                <div className="flex flex-col">
                  <span className="font-sans text-xs sm:text-sm font-semibold tracking-tight text-black leading-tight">
                    Pegasus Autonomous
                  </span>
                  <span className="font-mono text-[10px] sm:text-[11px] font-normal uppercase tracking-wider text-black/60">
                    {finalEyebrow}
                  </span>
                </div>
              </div>

              {/* Minimalist Top Links */}
              <nav className="hidden sm:flex items-center gap-6 md:gap-8 font-mono text-xs uppercase tracking-widest text-black/80 font-medium">
                <Link href="/platform" className="hover:text-black transition-colors">
                  {isRu ? 'СЕРВИСЫ' : 'SERVICES'}
                </Link>
                <Link href="/technology" className="hover:text-black transition-colors">
                  {isRu ? 'АРХИТЕКТУРА' : 'ABOUT'}
                </Link>
                <Link href="/contact" className="hover:text-black transition-colors">
                  {isRu ? 'КОНТАКТЫ' : 'CONTACT'}
                </Link>
              </nav>
            </div>

            {/* Middle & Bottom Layout Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-12 items-end pt-4 pb-2">
              
              {/* Left Column (Light Zone): Stark Editorial Headline */}
              <div className="lg:col-span-6 pointer-events-auto">
                <h2 className="text-3xl sm:text-5xl md:text-6xl lg:text-[72px] xl:text-[80px] font-normal tracking-[-0.04em] leading-[0.98] text-[#111111] max-w-xl">
                  <span className="block font-medium">{finalH1}</span>
                  <span className="block text-[#111111] font-light mt-1 sm:mt-2">{finalH2}</span>
                </h2>
              </div>

              {/* Right Column (Dark Zone): Concise narrative & High-Contrast CTAs */}
              <div className="lg:col-span-6 lg:col-start-7 flex flex-col gap-6 md:gap-8 pointer-events-auto">
                <p className="text-sm sm:text-base md:text-lg text-[#d8d8d8] font-normal leading-[1.5] max-w-lg">
                  {finalDesc}
                </p>

                {/* Action Buttons Row */}
                <div className="flex flex-wrap items-center gap-6 sm:gap-8 pt-1">
                  {/* High-contrast solid button */}
                  <Link
                    href={demoHref}
                    className="group inline-flex items-center justify-between gap-4 bg-white text-black font-mono text-xs sm:text-[13px] uppercase tracking-[0.16em] font-semibold px-6 sm:px-7 py-3.5 sm:py-4 transition-all duration-200 hover:bg-black hover:text-white border border-white"
                  >
                    <span>{finalDemo}</span>
                    <span
                      className="w-2 h-2 bg-black transition-colors duration-200 group-hover:bg-white inline-block flex-shrink-0"
                      aria-hidden="true"
                    />
                  </Link>

                  {/* Monospace Link with Arrow */}
                  <Link
                    href={trialHref}
                    className="group inline-flex items-center gap-2 font-mono text-xs sm:text-[13px] uppercase tracking-[0.16em] font-semibold text-white/90 hover:text-white transition-colors py-2"
                  >
                    <span>{finalTrial}</span>
                    <span
                      className="text-base leading-none transition-transform duration-200 group-hover:translate-x-1"
                      aria-hidden="true"
                    >
                      →
                    </span>
                  </Link>
                </div>
              </div>

            </div>

          </div>

        </div>
      </div>
    </section>
  );
}
