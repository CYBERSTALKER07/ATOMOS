'use client';

import PageSection from './layout/PageSection';

const capabilityCards = [
  {
    title: 'Visual Dispatch\nEngine',
    description: 'Map out multi-step routing behaviors on a high-precision grid. Drag and drop triggers, logic gates, and actions to craft custom paths.',
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
    description: 'Run complex decision trees without manual intervention. Our engine handles conditional branching and error recovery automatically.',
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
    description: 'Every vehicle and data transfer is shielded by industrial-grade security. Maintain total control over your organizational data flow.',
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
    description: 'Connect core business platforms and internal services through secure, ready integrations that scale with your volume.',
    icon: (
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.2" strokeLinejoin="round">
        <polygon points="12 4 20 8 12 12 4 8" fill="white" />
        <path d="M4 11 L12 15 L20 11" />
        <path d="M4 14 L12 18 L20 14" />
        <line x1="8" y1="6" x2="16" y2="10" stroke="black" strokeWidth="1" />
        <line x1="8" y1="10" x2="16" y2="6" stroke="black" strokeWidth="1" />
      </svg>
    )
  }
];

export default function Skills() {
  return (
    <PageSection
      id="capabilities"
      bleed
      className="bg-[#050505] !p-0 border-t border-white/10"
      innerClassName="w-full max-w-[1600px] mx-auto px-0"
    >
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 bg-white/10 gap-px border-b border-white/10">
        {capabilityCards.map((card, index) => (
          <div 
            key={index} 
            className="bg-[#050505] p-12 md:p-16 hover:bg-[#080808] transition-colors cursor-pointer group flex flex-col"
          >
            <div className="mb-10 opacity-80 group-hover:opacity-100 transition-opacity transform group-hover:-translate-y-1 duration-300">
              {card.icon}
            </div>
            <h3 className="font-mono text-lg font-bold text-white mb-6 leading-snug whitespace-pre-line">
              {card.title}
            </h3>
            <p className="text-[13px] md:text-sm text-white/50 leading-relaxed font-sans">
              {card.description}
            </p>
          </div>
        ))}
      </div>
    </PageSection>
  );
}
