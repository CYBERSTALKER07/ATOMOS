'use client';

import React from 'react';
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

// 28-column exact crenellation pattern extracted from Spur design
// [top_is_black, bot_is_black]
const CRENELLATION_PATTERN: [number, number][] = [
  [1, 0], [0, 1], [0, 0], [1, 1], [0, 1], [0, 1], [1, 0],
  [0, 1], [0, 1], [0, 1], [1, 1], [0, 1], [1, 0], [0, 1],
  [1, 1], [1, 1], [1, 1], [1, 0], [0, 1], [0, 1], [0, 1],
  [1, 0], [0, 1], [0, 1], [1, 1], [0, 0], [0, 1], [1, 0],
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

  const finalEyebrow = eyebrow ?? (isRu ? 'НАЧАТЬ' : 'GET STARTED');
  const finalH1 = headlineLine1 ?? (isRu ? 'Почувствуйте разницу между' : 'See the Difference Between');
  const finalH2 = headlineLine2 ?? (isRu ? 'Сырыми данными и интеллектом' : 'Raw Data & Real Intelligence');
  const finalDesc =
    description ??
    (isRu
      ? 'Обогащайте сетевую телеметрию в реальном времени, выявляя аномалии, прокси и ботов, скрытых на виду.'
      : 'Start enriching IPs with Spur to reveal the residential proxies, VPNs, and bots hiding in plain sight.');
  const finalDemo = demoLabel ?? (isRu ? 'ЗАПРОСИТЬ ДЕМО' : 'REQUEST A DEMO');
  const finalTrial = trialLabel ?? (isRu ? 'НАЧАТЬ ТЕСТ' : 'START FREE TRIAL');

  const colWidth = 1024 / CRENELLATION_PATTERN.length;

  return (
    <section
      aria-label="Intelligence Overview"
      className={`relative w-full bg-[#CEFF00] text-black overflow-hidden select-none ${className}`}
    >
      {/* Top Content Area */}
      <div className="w-full max-w-[1440px] mx-auto px-6 sm:px-10 md:px-16 lg:px-20 pt-20 sm:pt-28 md:pt-36 pb-16 sm:pb-24 md:pb-28">
        {/* Eyebrow badge */}
        <div className="flex items-center gap-2.5 mb-6 sm:mb-8">
          <span className="w-2.5 h-2.5 bg-black inline-block flex-shrink-0" aria-hidden="true" />
          <span className="font-mono text-xs sm:text-[13px] font-bold tracking-[0.22em] uppercase text-black">
            {finalEyebrow}
          </span>
        </div>

        {/* Main Display Headline */}
        <h2 className="font-title text-4xl sm:text-5xl md:text-6xl lg:text-7xl xl:text-[82px] 2xl:text-[92px] font-normal tracking-[-0.035em] leading-[1.04] text-black max-w-6xl">
          <span className="block">{finalH1}</span>
          <span className="block">{finalH2}</span>
        </h2>

        {/* Offset Grid: Paragraph & CTAs positioned on the right */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-12 items-start mt-8 sm:mt-12 md:mt-16">
          <div className="lg:col-span-6 lg:col-start-7 flex flex-col gap-8 md:gap-10">
            <p className="text-base sm:text-lg md:text-xl text-black font-normal leading-[1.45] max-w-xl">
              {finalDesc}
            </p>

            {/* Action Buttons Row */}
            <div className="flex flex-wrap items-center gap-6 sm:gap-10 md:gap-12 pt-2">
              {/* White Box CTA: REQUEST A DEMO ■ */}
              <Link
                href={demoHref}
                className="group inline-flex items-center justify-between gap-4 bg-white text-black font-mono text-xs sm:text-[13px] uppercase tracking-[0.18em] font-semibold px-6 sm:px-7 py-3.5 sm:py-4 transition-all duration-200 hover:bg-black hover:text-white shadow-sm"
              >
                <span>{finalDemo}</span>
                <span
                  className="w-2 h-2 bg-black transition-colors duration-200 group-hover:bg-white inline-block flex-shrink-0"
                  aria-hidden="true"
                />
              </Link>

              {/* Monospace Link CTA with dual-segment underline: START FREE TRIAL → */}
              <div className="group inline-flex flex-col items-start cursor-pointer">
                <Link
                  href={trialHref}
                  className="inline-flex items-center gap-3 font-mono text-xs sm:text-[13px] uppercase tracking-[0.18em] font-semibold text-black transition-opacity hover:opacity-85"
                >
                  <span>{finalTrial}</span>
                  <span
                    className="text-base leading-none transition-transform duration-200 group-hover:translate-x-1"
                    aria-hidden="true"
                  >
                    →
                  </span>
                </Link>
                {/* Stepped progress-line indicator: 35% bold segment, remainder subtle base */}
                <div className="w-full min-w-[155px] h-[3px] mt-2 relative bg-black/25 overflow-hidden">
                  <div className="absolute left-0 top-0 h-full w-[35%] bg-black transition-all duration-300 ease-out group-hover:w-full" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Signature Stepped Pixel Crenellation Transition Border */}
      <div
        className="w-full relative overflow-hidden bg-[#CEFF00] leading-none select-none pointer-events-none"
        aria-hidden="true"
      >
        <svg
          viewBox="0 0 1024 80"
          preserveAspectRatio="none"
          className="w-full h-14 sm:h-20 md:h-24 lg:h-28 block"
        >
          {/* Base bottom solid fill matching dark section background (#09090B) */}
          <rect x="0" y="20" width="1024" height="60" fill="#09090B" />

          {/* Top row (y=0..20): black blocks cutting up into lime canvas */}
          {/* Mid row (y=20..40): lime blocks protruding down into dark base */}
          {CRENELLATION_PATTERN.map(([top, bot], i) => {
            const x = i * colWidth;
            const w = colWidth + 0.15; // prevent subpixel gaps on varied screen resolutions
            return (
              <g key={i}>
                {top === 1 && (
                  <rect x={x.toFixed(2)} y="0" width={w.toFixed(2)} height="20" fill="#09090B" />
                )}
                {bot === 0 && (
                  <rect x={x.toFixed(2)} y="20" width={w.toFixed(2)} height="20" fill="#CEFF00" />
                )}
              </g>
            );
          })}
        </svg>
      </div>
    </section>
  );
}
