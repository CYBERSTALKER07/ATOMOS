'use client';

import React, { useState, useRef, useEffect, useMemo } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import SiteNav from '@/app/components/explore/SiteNav';
import {
  getSolutionsAccordionData,
  AccordionSolution,
} from '../data/solutionsAccordionData';
import { useLanguage } from '@/app/context/LanguageContext';

const AccordionItem = ({
  item,
  isOpen,
  onToggle,
  t,
}: {
  item: AccordionSolution;
  isOpen: boolean;
  onToggle: () => void;
  t: (key: string, fallback?: string) => string;
}) => {
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!contentRef.current) return;

    if (isOpen) {
      gsap.to(contentRef.current, {
        height: 'auto',
        opacity: 1,
        duration: 0.5,
        ease: 'power3.inOut',
      });
    } else {
      gsap.to(contentRef.current, {
        height: 0,
        opacity: 0,
        duration: 0.5,
        ease: 'power3.inOut',
      });
    }
  }, [isOpen]);

  return (
    <div className="border-b border-white/20">
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between py-8 md:py-12 group text-left focus:outline-none"
      >
        <div className="flex items-center gap-6 md:gap-12">
          <span className="text-sm md:text-base font-mono text-white/60 group-hover:text-white transition-colors">
            {item.numberLabel}
          </span>
          <h2 className="text-3xl md:text-5xl font-normal tracking-tight text-white group-hover:text-[#FBFF63] transition-colors">
            {item.title}
          </h2>
        </div>
        <div className="text-4xl font-light text-white/80 group-hover:text-white transition-colors">
          {isOpen ? '—' : '+'}
        </div>
      </button>

      <div ref={contentRef} className="h-0 opacity-0 overflow-hidden">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 lg:gap-24 pb-12 pt-4">
          <div className="flex flex-col">
            <h3 className="text-xs tracking-widest font-mono text-white/50 mb-8 uppercase">
              {t('sol_overview', 'Overview')}
            </h3>
            <p className="text-lg md:text-xl leading-relaxed text-white/90 mb-12 max-w-lg">
              {item.overview}
            </p>
            <div className="mt-auto">
              <Link
                href={item.solutionHref}
                className="inline-flex items-center justify-center bg-white text-black px-6 py-3 text-sm font-bold tracking-wider uppercase hover:bg-gray-200 transition-colors"
              >
                {t('sol_go_to', 'Go to Solution')}
              </Link>
            </div>
          </div>

          <div className="flex flex-col">
            <div className="flex items-center justify-between mb-8">
              <h3 className="text-xs tracking-widest font-mono text-white/50 uppercase">
                {t('sol_use_cases', 'Use Cases')}
              </h3>
              <span className="text-xs tracking-widest font-mono text-white/50 uppercase">
                {item.useCases.length} {t('sol_items', 'Items')}
              </span>
            </div>

            <div className="flex flex-col">
              {item.useCases.map((useCase, idx) => (
                <Link
                  key={idx}
                  href={useCase.href}
                  className="group/item flex items-center justify-between py-6 border-t border-white/20 hover:border-white/60 transition-colors"
                >
                  <span className="text-base md:text-lg text-white/90 group-hover/item:text-white transition-colors">
                    {useCase.title}
                  </span>
                  <div className="flex items-center gap-6">
                    <span className="text-xs font-mono text-white/40 group-hover/item:text-white/80 transition-colors hidden md:block">
                      [{t('sol_docs', 'DOCS')}]
                    </span>
                    <span className="text-white/50 group-hover/item:text-white group-hover/item:-translate-y-1 group-hover/item:translate-x-1 transition-transform">
                      ↗
                    </span>
                  </div>
                </Link>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default function SolutionsAccordion() {
  const [openIndex, setOpenIndex] = useState<number>(0);
  const { t, language } = useLanguage();
  const data = useMemo(() => getSolutionsAccordionData(language), [language]);

  return (
    <div className="min-h-screen bg-black text-white pb-24 selection:bg-white/30">
      <SiteNav activeHref="/solutions" />
      <div className="pt-32">
        <div className="max-w-7xl mx-auto px-6 md:px-12">
          <div className="flex items-center gap-2 text-xs font-mono tracking-widest text-white/50 mb-12 uppercase">
            <Link href="/" className="hover:text-white transition-colors">
              ⌂
            </Link>
            <span>/</span>
            <span>{t('nav_solutions', 'Solutions')}</span>
          </div>

          <div className="max-w-4xl mb-24">
            <h1 className="text-6xl md:text-8xl font-normal tracking-tight mb-8">
              {t('sol_page_title', 'Our Solutions')}
            </h1>
            <p className="text-lg md:text-xl leading-relaxed text-white/70 max-w-3xl">
              {t(
                'sol_page_desc',
                'PegasusX is helping global supply chains transform their logistics, warehouse management, and operational decision-making for digital age volatility and complexity. Find more information about the various solutions that our platform provides across every key role.'
              )}
            </p>
          </div>

          <div className="border-t border-white/20">
            {data.map((item, idx) => (
              <AccordionItem
                key={item.id}
                item={item}
                isOpen={openIndex === idx}
                onToggle={() => setOpenIndex(openIndex === idx ? -1 : idx)}
                t={t}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
