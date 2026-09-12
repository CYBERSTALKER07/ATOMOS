'use client';

import { useState, type ReactNode } from 'react';
import PageSection from './layout/PageSection';
import DigitalCardHover from './DigitalCardHover';
import { usePerfProfile } from '../hooks/useDevice';
import { useLanguage } from '../context/LanguageContext';

type CapabilityCard = {
  title: string;
  description: string;
  className?: string;
  icon: ReactNode;
};

const capabilityCards: CapabilityCard[] = [
  {
    title: 'Visual Dispatch\nEngine',
    description: 'Map out multi-step routing behaviors on a high-precision grid. Drag and drop triggers, logic gates, and actions to craft custom paths with complete flexibility and control over your entire operation.',
    className: 'lg:col-span-2 lg:row-span-2',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <polygon points="12 4 20 8 12 12 4 8" fill="white" />
        <path d="M4 11 L12 15 L20 11" />
        <path d="M4 14 L12 18 L20 14" />
        <circle cx="12" cy="8" r="1.5" fill="black" stroke="none" />
        <circle cx="16" cy="6" r="1" fill="black" stroke="none" />
        <path d="M12 8 L16 6" stroke="black" strokeWidth="0.5" />
      </svg>
    )
  },
  {
    title: 'Autonomous\nExecution',
    description: 'Run complex decision trees without manual intervention. Our engine handles conditional branching automatically.',
    className: 'lg:col-span-1 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <polygon points="12 4 20 8 12 12 4 8" fill="white" />
        <path d="M4 11 L12 15 L20 11" />
        <path d="M4 14 L12 18 L20 14" />
        <rect x="11" y="6" width="2" height="4" fill="black" stroke="none" transform="rotate(45 12 8)" />
      </svg>
    )
  },
  {
    title: 'End-to-End\nVisibility',
    description: 'Every vehicle and data transfer is shielded by industrial-grade security. Maintain total control over data flow.',
    className: 'lg:col-span-1 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <polygon points="12 4 20 8 12 12 4 8" fill="white" />
        <path d="M4 11 L12 15 L20 11" />
        <path d="M12 2 L15 5 L12 8 L9 5 Z" fill="black" stroke="none" />
        <path d="M12 2 L12 8" stroke="white" strokeWidth="0.5" />
      </svg>
    )
  },
  {
    title: 'Production-\nReady Stack',
    description: 'Connect core business platforms and internal services through secure, ready integrations that scale.',
    className: 'lg:col-span-1 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <polygon points="12 4 20 8 12 12 4 8" fill="white" />
        <path d="M4 11 L12 15 L20 11" />
        <path d="M4 14 L12 18 L20 14" />
        <line x1="8" y1="6" x2="16" y2="10" stroke="black" strokeWidth="1" />
        <line x1="8" y1="10" x2="16" y2="6" stroke="black" strokeWidth="1" />
      </svg>
    )
  },
  {
    title: 'Real-time\nTelemetry',
    description: 'Monitor fleet vitals, driver status, and delivery ETA with millisecond precision globally.',
    className: 'lg:col-span-1 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <polygon points="12 4 20 8 12 12 4 8" fill="white" />
        <path d="M4 11 L12 15 L20 11" />
        <circle cx="12" cy="6" r="1.5" fill="black" stroke="none" />
        <path d="M12 6 L14 8" stroke="black" strokeWidth="0.5" />
      </svg>
    )
  },
  {
    title: 'Predictive AI\nRouting',
    description: 'Leverage machine learning to anticipate traffic patterns, optimize dispatch routes before delays occur, and intelligently re-route resources on the fly to maximize efficiency across your entire supply chain network.',
    className: 'lg:col-span-2 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <polygon points="12 4 20 8 12 12 4 8" fill="white" />
        <path d="M4 11 L12 15 L20 11" />
        <path d="M4 14 L12 18 L20 14" />
        <path d="M10 6 L14 6 M12 4 L12 8" stroke="black" strokeWidth="1" />
      </svg>
    )
  },
  {
    title: 'Multi-Tenant\nArchitecture',
    description: 'Isolate data streams per client while managing a unified global network of carriers and suppliers with strict access controls.',
    className: 'lg:col-span-2 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <polygon points="12 4 20 8 12 12 4 8" fill="white" />
        <path d="M4 11 L12 15 L20 11" />
        <rect x="9" y="5" width="2" height="2" fill="black" />
        <rect x="13" y="5" width="2" height="2" fill="black" />
        <rect x="11" y="7" width="2" height="2" fill="black" />
      </svg>
    )
  },
  {
    title: 'Automated\nSettlement',
    description: 'Reconcile invoices against delivery proofs instantly. Trigger smart contract payments upon successful gate exit or drop-off.',
    className: 'lg:col-span-1 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" strokeWidth="1.2" />
      </svg>
    )
  },
  {
    title: 'Capacity\nForecasting',
    description: 'Predict warehouse and fleet load constraints before they happen using historical volume analysis and market signals.',
    className: 'lg:col-span-1 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <path d="M3 3v18h18" strokeWidth="1.2" />
        <path d="M18 9l-5 5-4-4-5 5" strokeWidth="1.2" />
      </svg>
    )
  },
  {
    title: 'Dock\nScheduling',
    description: 'Eliminate yard congestion with algorithmic slotting. Synchronize arrival windows with live unload speeds and labor availability.',
    className: 'lg:col-span-1 lg:row-span-1',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <rect x="3" y="4" width="18" height="18" rx="2" ry="2" strokeWidth="1.2" />
        <line x1="16" y1="2" x2="16" y2="6" strokeWidth="1.2" />
        <line x1="8" y1="2" x2="8" y2="6" strokeWidth="1.2" />
        <line x1="3" y1="10" x2="21" y2="10" strokeWidth="1.2" />
      </svg>
    )
  },
  {
    title: 'Geofence\nTriggers',
    description: 'Automate status updates, notify receivers, and prepare staging areas precisely when a vehicle breaches virtual perimeter boundaries.',
    className: 'lg:col-span-1 lg:row-span-1',
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
      className={`relative overflow-hidden bg-[#050505] p-8 sm:p-10 md:p-12 lg:p-16 cursor-pointer group flex flex-col ${card.className || ''}`}
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
          <DigitalCardHover active={hovered} color="#b8b4b3" />
        </div>
      )}
      {!allowHoverFx && (
        <div className="absolute inset-0 z-0 opacity-0 group-active:opacity-100 group-focus-within:opacity-100 bg-[radial-gradient(ellipse_at_center,rgba(180,180,180,0.12),transparent_70%)] transition-opacity duration-300" />
      )}

      <div className="relative z-10 mb-6 sm:mb-8 md:mb-10 opacity-90 group-hover:opacity-100 transition-opacity transform group-hover:-translate-y-1 duration-300 pointer-events-none drop-shadow-[0_2px_12px_rgba(0,0,0,0.85)]">
        {card.icon}
      </div>
      <h3 className="relative z-10 font-mono text-base sm:text-lg font-bold text-white mb-4 sm:mb-6 leading-snug whitespace-pre-line pointer-events-none transition-colors duration-300 group-hover:text-white drop-shadow-[0_2px_10px_rgba(0,0,0,0.9)]">
        {card.title}
      </h3>
      <p className="relative z-10 text-[13px] md:text-sm text-white/60 leading-relaxed font-sans max-w-md pointer-events-none group-hover:text-white/90 transition-colors duration-300 drop-shadow-[0_1px_8px_rgba(0,0,0,0.85)]">
        {card.description}
      </p>
    </div>
  );
}

export default function Skills() {
  const { t } = useLanguage();
  const localizedCards = capabilityCards.map((card, i) => ({
    ...card,
    title: t(`skills_c${i + 1}_title`, card.title),
    description: t(`skills_c${i + 1}_desc`, card.description),
  }));
  return (
    <PageSection
      id="capabilities"
      bleed
      className="bg-[#050505] !p-0 border-t border-white/10"
      innerClassName="w-full max-w-[1600px] mx-auto px-0"
    >
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 bg-white/10 gap-px border-b border-white/10">
        {localizedCards.map((card, index) => (
          <CapabilityCardItem key={index} card={card} />
        ))}
      </div>
    </PageSection>
  );
}
