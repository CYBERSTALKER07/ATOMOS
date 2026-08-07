'use client';

import React from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useLanguage } from '../context/LanguageContext';

export function PegasusTestimonialsSection() {
  const { t } = useLanguage();

  return (
    <section className="w-full bg-black text-white py-16 px-4 flex justify-center items-center font-sans antialiased">
      <div className="max-w-4xl w-full mx-auto flex flex-col items-center">
        
        {/* Single Featured Card: CTO of Pegasus */}
        <div className="flex flex-col gap-6 p-6 md:p-8 rounded-2xl bg-zinc-950/60 border border-zinc-800/60 hover:border-zinc-700/80 transition-all w-full">
          <div className="flex flex-col sm:flex-row gap-6 sm:gap-8 items-start sm:items-center">
            <div className="w-32 h-32 flex-shrink-0 rounded-xl overflow-hidden bg-zinc-950 relative border border-zinc-800">
              <Image 
                src="/Gemini_Generated_Image_e86uare86uare86u.png" 
                alt={t('cto_role')}
                width={128}
                height={128}
                className="w-full h-full object-cover filter contrast-105"
              />
            </div>
            <blockquote className="font-serif text-2xl md:text-3xl leading-snug text-zinc-100 tracking-tight">
              {t('cto_quote')}
            </blockquote>
          </div>
          
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pt-4 border-t border-zinc-800/60 sm:pl-[160px]">
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1.5 text-xs font-bold tracking-widest uppercase text-white">
                <svg className="h-5 fill-current" viewBox="0 0 24 24"><path d="M12 2L2 19h20L12 2zm0 4l6.5 11h-13L12 6z"/></svg>
                PEGASUS
              </div>
              <div className="flex flex-col">
                <span className="text-base font-semibold text-zinc-100 leading-tight">{t('cto_name')}</span>
                <span className="text-xs text-zinc-400 leading-tight">{t('cto_role')}</span>
              </div>
            </div>
            
            <Link href="/contact" className="inline-flex items-center gap-1.5 text-xs font-medium text-zinc-400 hover:text-white transition-colors group">
              {t('read_case_study')} <span className="group-hover:translate-x-1 transition-transform">→</span>
            </Link>
          </div>
        </div>

      </div>
    </section>
  );
}
