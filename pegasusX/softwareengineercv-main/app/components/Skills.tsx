'use client';

import React, { useState } from 'react';
import PageSection from './layout/PageSection';
import DigitalCardHover from './DigitalCardHover';
import { usePerfProfile } from '../hooks/useDevice';
import { useLanguage } from '../context/LanguageContext';
import GooeyAgent from './visuals/GooeyAgent';

type CapabilityCard = {
  title: string;
  description: string;
  icon: React.ReactNode;
  className?: string;
};

const capabilityCards: CapabilityCard[] = [
  {
    title: 'Capacity \nAlgorithmic Slotting',
    description: 'Eliminate yard congestion with algorithmic slotting. Synchronize arrival windows with live unload speeds and labor availability.',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <rect x="3" y="3" width="18" height="18" rx="0" strokeWidth="1.2" />
        <path d="M3 9h18M9 21V9" strokeWidth="1.2" />
      </svg>
    )
  },
  {
    title: 'Dynamic \nRate Shopping',
    description: 'Pull spot rates across carrier networks in milliseconds. Lock the optimal payload-to-price ratio before the shipment hits the dock.',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" strokeWidth="1.2" />
      </svg>
    )
  },
  {
    title: 'Cross-Border \nCompliance Engine',
    description: 'Automate customs documentation and hazardous material checks. Never lose a day at the border over a missing HS code.',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" strokeWidth="1.2" />
        <path d="M8 11h8" strokeWidth="1.2" />
      </svg>
    )
  },
  {
    title: 'Real-Time \nException Resolution',
    description: 'Detect climate control failures or route deviations instantly. Re-route and notify stakeholders before the cargo is compromised.',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <circle cx="12" cy="12" r="10" strokeWidth="1.2" strokeDasharray="4 4" />
        <path d="M12 8v4l3 3" strokeWidth="1.2" />
      </svg>
    )
  }
];

function CapabilityCardItem({ card }: { card: CapabilityCard }) {
  const { allowHoverFx } = usePerfProfile();
  const [hovered, setHovered] = useState(false);

  return (
    <div
      className={`relative overflow-hidden bg-black p-8 sm:p-10 md:p-12 lg:p-16 cursor-pointer capability-card group flex flex-col ${card.className || ''}`}
      onMouseEnter={() => allowHoverFx && setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onFocus={() => allowHoverFx && setHovered(true)}
      onBlur={() => setHovered(false)}
      tabIndex={0}
    >
      {allowHoverFx && (
        <div
          className={`absolute inset-0 z-0 transition-opacity duration-500 ${
            hovered ? 'opacity-100' : 'opacity-0'
          }`}
        >
          <DigitalCardHover active={hovered} color="#ffffff" />
        </div>
      )}
      {!allowHoverFx && (
        <div className="absolute inset-0 z-0 opacity-0 group-active:opacity-100 group-focus-within:opacity-100 bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.05),transparent_70%)] transition-opacity duration-300" />
      )}

      <div className="relative z-10 mb-6 sm:mb-8 md:mb-10 opacity-90 group-hover:opacity-100 transition-opacity transform group-hover:-translate-y-1 duration-300 pointer-events-none">
        {card.icon}
      </div>
      <h3 className="relative z-10 font-mono text-base sm:text-lg font-bold text-white mb-4 sm:mb-6 leading-snug whitespace-pre-line pointer-events-none transition-colors duration-300">
        {card.title}
      </h3>
      <p className="relative z-10 text-[13px] md:text-sm text-white/60 leading-relaxed font-sans max-w-md pointer-events-none group-hover:text-white/90 transition-colors duration-300">
        {card.description}
      </p>
    </div>
  );
}

export default function Skills() {
  const { t, language } = useLanguage();
  const localizedCards = capabilityCards.map((card, i) => ({
    ...card,
    title: t(`skills_c${i + 1}_title`, card.title),
    description: t(`skills_c${i + 1}_desc`, card.description),
  }));

  return (
    <PageSection
      id="capabilities"
      bleed
      className="bg-black !p-0 border-t border-white/10"
      innerClassName="w-full max-w-[1600px] mx-auto px-0"
    >
      <div className="flex flex-col md:flex-row items-center justify-between p-8 md:p-16 lg:p-24 border-b border-white/10 relative overflow-hidden">
        <div className="relative z-10">
          <p className="font-mono text-xs uppercase tracking-widest text-white/40 mb-4">Pegasus OS</p>
          <h2 className="text-5xl md:text-7xl font-bold tracking-tighter text-white">
            {language === 'ru' ? 'КЛЮЧЕВЫЕ ВОЗМОЖНОСТИ' : 'KEY CAPABILITIES'}
          </h2>
        </div>
        <div className="relative z-10 mt-10 md:mt-0 opacity-80 mix-blend-screen scale-150 transform origin-right">
          <GooeyAgent color="#ffffff" size={300} />
        </div>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 bg-white/10 gap-px border-b border-white/10">
        {localizedCards.map((card, index) => (
          <CapabilityCardItem key={index} card={card} />
        ))}
      </div>
    </PageSection>
  );
}
