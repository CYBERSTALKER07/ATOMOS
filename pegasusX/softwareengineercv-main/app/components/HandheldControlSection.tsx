'use client';

import React from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useLanguage } from '@/app/context/LanguageContext';

export default function HandheldControlSection() {
  const { language } = useLanguage();
  const isRu = language === 'ru';

  return (
    <section className="w-full bg-[#000000] overflow-hidden relative select-none">
      <div className="w-full relative z-10">
        <div className="relative grid grid-cols-1 lg:grid-cols-2 bg-[#000000] w-full min-h-[640px] lg:min-h-[85vh] xl:min-h-[90vh]">
          
          {/* LEFT 1/2: Text Only (Just like Hero Section) */}
          <div className="flex flex-col justify-center p-8 sm:p-14 lg:p-20 xl:p-28 py-16 lg:py-24 relative z-10">
            
            {/* Headline */}
            <h2 className="text-4xl sm:text-5xl md:text-6xl lg:text-[4.5rem] xl:text-[5rem] font-light md:font-normal tracking-tight text-white leading-[1.05] mb-8">
              {isRu
                ? 'Управление логистикой на ладони'
                : 'Logistics control in your palm'}
            </h2>

            {/* Subtitle */}
            <p className="text-base sm:text-lg md:text-xl font-light text-white/60 leading-relaxed max-w-xl mb-10">
              {isRu
                ? 'Синхронизированные роли, мониторинг флота и WMS в реальном времени с единого сенсорного экрана.'
                : 'Real-time fleet tracking, warehouse operations, and freight execution synchronized across every role in your network.'}
            </p>

            {/* CTA Button */}
            <div>
              <Link
                href="/join"
                className="inline-flex items-center justify-center gap-2 px-8 py-4 text-sm sm:text-base font-semibold bg-white text-black hover:bg-zinc-200 transition-colors rounded-none"
              >
                <span>{isRu ? 'Запросить демо' : 'Request Demo'}</span>
                <span className="text-lg leading-none mt-[-2px]">›</span>
              </Link>
            </div>
          </div>

          {/* RIGHT 1/2: Image Only (Just like Hero Section) */}
          <div className="relative w-full h-[520px] lg:h-full min-h-[520px] lg:min-h-full bg-[#000000] overflow-hidden">
            <Image
              src="/images/pegasus_handheld_os.jpg"
              alt="Pegasus Logistics Mobile Handheld OS"
              fill
              priority
              sizes="(max-width: 1024px) 100vw, 50vw"
              className="object-cover object-center"
            />
            {/* Subtle soft edge blend into the dark layout */}
            <div className="absolute inset-0 bg-gradient-to-r from-[#000000] via-transparent to-transparent pointer-events-none w-24 sm:w-40" />
            <div className="absolute inset-x-0 bottom-0 h-20 bg-gradient-to-t from-[#000000] to-transparent pointer-events-none" />
            <div className="absolute inset-x-0 top-0 h-20 bg-gradient-to-b from-[#000000] to-transparent pointer-events-none" />
          </div>
          
        </div>
      </div>
    </section>
  );
}
