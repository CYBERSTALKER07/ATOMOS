'use client';

import React from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useLanguage } from '@/app/context/LanguageContext';

export default function HandheldControlSection() {
  const { language } = useLanguage();
  const isRu = language === 'ru';

  return (
    <section className="w-full bg-[#000000] border-t border-white/10 overflow-hidden relative select-none">
      <div className="w-full relative z-10">
        <div className="relative grid grid-cols-1 lg:grid-cols-2 bg-[#000000] w-full min-h-[640px] lg:min-h-[760px]">
          
          {/* LEFT 1/2: Editorial Headline, Subtitle, Telemetry Grid & CTAs */}
          <div className="flex flex-col justify-center p-8 sm:p-12 lg:p-16 xl:p-24 relative z-10">
            
            {/* Eyebrow */}
            <div className="mb-6 flex items-center">
              <div className="h-1.5 w-1.5 bg-white mr-3 animate-pulse" />
              <span className="font-mono text-[10px] uppercase tracking-[0.25em] text-white/50">
                {isRu ? 'МОБИЛЬНАЯ ОС PEGASUS // ПОЛЕВОЙ КОНТРОЛЬ' : 'PEGASUS OS // NATIVE FIELD SYSTEMS'}
              </span>
            </div>

            {/* Headline */}
            <h2 className="text-3xl sm:text-4xl md:text-5xl lg:text-[3.75rem] font-medium tracking-tight text-white leading-[1.08] mb-6">
              {isRu
                ? 'Вся цепочка поставок на ладони вашей руки'
                : 'Mission control in the palm of your hand'}
            </h2>

            {/* Subtitle */}
            <p className="text-base sm:text-lg md:text-xl font-light text-white/60 leading-relaxed max-w-xl mb-10">
              {isRu
                ? 'Единый сенсорный интерфейс с минимальной задержкой для диспетчеров, складов и экспедиторов. Управление автопарком, WMS, авиаперевозки и таможенная очистка без переключения контекста.'
                : 'Every operational tier unified into an ultra-low-latency mobile interface. Real-time fleet tracking, barcode-gated WMS, air cargo, and instant customs duty clearance—connected directly to the core transactional ledger.'}
            </p>

            {/* High-Tech Telemetry Stats Grid */}
            <div className="grid grid-cols-2 gap-6 max-w-lg mb-10 border-t border-white/10 pt-8">
              <div>
                <div className="font-mono text-[10px] uppercase tracking-wider text-white/40 mb-1">
                  {isRu ? 'МОНИТОРИНГ ФЛОТА' : 'FLEET TRACKING'}
                </div>
                <div className="font-mono text-xs sm:text-sm text-white font-medium">
                  {isRu ? '65% АКТИВНО В ПУТИ' : '65% ACTIVE IN-TRANSIT'}
                </div>
              </div>
              <div>
                <div className="font-mono text-[10px] uppercase tracking-wider text-white/40 mb-1">
                  {isRu ? 'СКЛАДСКАЯ СИСТЕМА WMS' : 'WAREHOUSE WMS'}
                </div>
                <div className="font-mono text-xs sm:text-sm text-white font-medium">
                  10,000+ MANAGED SKUS
                </div>
              </div>
              <div>
                <div className="font-mono text-[10px] uppercase tracking-wider text-white/40 mb-1">
                  {isRu ? 'ГЛОБАЛЬНАЯ ERP' : 'GLOBAL ERP'}
                </div>
                <div className="font-mono text-xs sm:text-sm text-white font-medium">
                  {isRu ? '100% СИНХРОНИЗАЦИЯ' : '100% SYNCHRONIZED'}
                </div>
              </div>
              <div>
                <div className="font-mono text-[10px] uppercase tracking-wider text-white/40 mb-1">
                  {isRu ? 'ТАМОЖНЯ И ПОШЛИНЫ' : 'DUTY & CUSTOMS'}
                </div>
                <div className="font-mono text-xs sm:text-sm text-white font-medium">
                  {isRu ? 'МГНОВЕННЫЙ РАСЧЕТ' : 'SUB-SECOND CLEARANCE'}
                </div>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="flex flex-wrap items-center gap-4">
              <Link
                href="/join"
                className="inline-flex items-center justify-center gap-2 px-8 py-3.5 text-sm sm:text-base font-medium bg-white text-black hover:bg-white/90 transition-all rounded-none"
              >
                <span>{isRu ? 'Запросить мобильное демо' : 'Request Field Demo'}</span>
                <span className="text-lg leading-none mt-[-2px]">›</span>
              </Link>
              <Link
                href="/apps-deploy/mobile-apps"
                className="inline-flex items-center justify-center gap-2 px-8 py-3.5 text-sm sm:text-base font-medium bg-white/5 text-white hover:bg-white/10 border border-white/10 transition-all rounded-none"
              >
                <span>{isRu ? 'Мобильная архитектура' : 'Mobile Architecture'}</span>
                <span className="text-lg leading-none mt-[-2px]">›</span>
              </Link>
            </div>
          </div>

          {/* RIGHT 1/2: Handheld OS Visual Showcase */}
          <div className="flex flex-col justify-center items-center relative overflow-hidden bg-[#050505] min-h-[500px] lg:min-h-[760px] p-6 sm:p-10 lg:p-16 border-t lg:border-t-0 lg:border-l border-white/10 group">
            
            {/* Ambient Background Glow & Tech Grid */}
            <div className="absolute inset-0 bg-gradient-to-br from-white/[0.03] via-transparent to-black pointer-events-none" />
            <div
              className="absolute inset-0 opacity-[0.05] pointer-events-none bg-[radial-gradient(#fff_1px,transparent_1px)]"
              style={{ backgroundSize: '24px 24px' }}
            />

            {/* Floating Device Container */}
            <div className="relative w-full max-w-[420px] sm:max-w-[460px] lg:max-w-[480px] aspect-[1888/2218] rounded-[24px] overflow-hidden shadow-[0_20px_60px_rgba(0,0,0,0.8)] border border-white/15 transition-transform duration-700 ease-out group-hover:scale-[1.02]">
              <Image
                src="/images/pegasus_handheld_os.jpg"
                alt="Pegasus Logistics Mobile Handheld OS"
                fill
                priority
                sizes="(max-width: 1024px) 100vw, 50vw"
                className="object-cover object-center"
              />

              {/* Edge Vignette */}
              <div className="absolute inset-0 bg-gradient-to-t from-black/40 via-transparent to-black/20 pointer-events-none" />

              {/* Tech HUD Metadata Overlays */}
              <div className="absolute top-4 left-4 z-20">
                <span className="font-mono text-[9px] uppercase tracking-[0.2em] text-white/90 bg-black/75 px-2 py-0.5 backdrop-blur-md border border-white/15">
                  PEGASUS TOUCH // v4.2
                </span>
              </div>

              <div className="absolute bottom-4 right-4 z-20">
                <span className="font-mono text-[9px] uppercase tracking-[0.2em] text-white/80 bg-black/75 px-2 py-0.5 backdrop-blur-md border border-white/15">
                  10 ROLE-SCOPED TILES · LIVE
                </span>
              </div>

              {/* Corner Tech Brackets */}
              <div className="absolute top-3 right-3 w-3 h-3 border-t border-r border-white/40 pointer-events-none" />
              <div className="absolute bottom-3 left-3 w-3 h-3 border-b border-l border-white/40 pointer-events-none" />
            </div>

            {/* Bottom Caption Bar */}
            <div className="mt-6 flex items-center space-x-3 text-white/40 font-mono text-[10px] uppercase tracking-[0.2em]">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-ping" />
              <span>LIVE FIELD TELEMETRY FEED ACTIVE</span>
            </div>

          </div>
          
        </div>
      </div>
    </section>
  );
}
