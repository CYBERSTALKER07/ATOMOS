'use client';

import PageSection from './layout/PageSection';

const FEATURES = [
  {
    title: 'Secure Guard',
    description: 'We fortify your logistics deployments with robust security protocols. Our system ensures every order adheres to strict data privacy standards.',
    icon: 'lock',
  },
  {
    title: 'Agent Build',
    description: 'Tailored automation agents designed for your specific needs. We develop custom logic and workflows that integrate deeply with your existing tools.',
    icon: 'nodes',
  },
  {
    title: 'Cloud Scale',
    description: 'Infrastructure optimization for high-traffic networks. We ensure your systems remain fast, responsive, and ready for any level of demand.',
    icon: 'server',
  },
  {
    title: 'Data Mining',
    description: 'Transform raw fleet information into actionable intelligence. We build the pipelines and vector stores that power your organization\'s future.',
    icon: 'data',
  },
];

function FeatureIcon({ type }: { type: string }) {
  if (type === 'lock') {
    return (
      <svg width="80" height="80" viewBox="0 0 100 100" fill="none" stroke="currentColor" strokeWidth="1" strokeLinejoin="round" aria-hidden="true" className="text-white drop-shadow-[0_0_15px_rgba(255,255,255,0.1)]">
        {/* Lock Body */}
        <rect x="25" y="45" width="40" height="35" rx="4" />
        {/* Lock Shackle */}
        <path d="M32 45V30c0-7.18 5.82-13 13-13s13 5.82 13 13v15" />
        {/* Keyhole */}
        <circle cx="45" cy="60" r="3" />
        <path d="M43.5 62h3l-1 8h-1l-1-8z" />
        {/* Gear */}
        <g transform="translate(65, 55)">
          <circle cx="0" cy="0" r="12" />
          <circle cx="0" cy="0" r="4" fill="currentColor" />
          {[0, 45, 90, 135, 180, 225, 270, 315].map((angle) => (
            <rect key={angle} x="-3" y="-16" width="6" height="4" transform={`rotate(${angle})`} />
          ))}
        </g>
      </svg>
    );
  }
  if (type === 'nodes') {
    return (
      <svg width="80" height="80" viewBox="0 0 100 100" fill="none" stroke="currentColor" strokeWidth="1" strokeLinejoin="round" aria-hidden="true" className="text-white drop-shadow-[0_0_15px_rgba(255,255,255,0.1)]">
        {/* Top Node */}
        <path d="M30 20 L60 20 A5 5 0 0 1 65 25 L65 35 A5 5 0 0 1 60 40 L30 40 A5 5 0 0 1 25 35 L25 25 A5 5 0 0 1 30 20 Z" />
        <rect x="40" y="27" width="10" height="6" />
        {/* Bottom Node */}
        <path d="M40 60 L70 60 A5 5 0 0 1 75 65 L75 75 A5 5 0 0 1 70 80 L40 80 A5 5 0 0 1 35 75 L35 65 A5 5 0 0 1 40 60 Z" />
        <rect x="50" y="67" width="10" height="6" />
        {/* Connection Cable */}
        <path d="M45 40 v8" strokeDasharray="2 2" />
        <path d="M45 48 c 0 6, -15 6, -15 12" strokeDasharray="2 2" />
        <path d="M55 60 v-8" strokeDasharray="2 2" />
        <path d="M55 52 c 0 -6, 15 -6, 15 -12" strokeDasharray="2 2" />
      </svg>
    );
  }
  if (type === 'server') {
    return (
      <svg width="80" height="80" viewBox="0 0 100 100" fill="none" stroke="currentColor" strokeWidth="1" strokeLinejoin="round" aria-hidden="true" className="text-white drop-shadow-[0_0_15px_rgba(255,255,255,0.1)]">
        {/* Isometric server box */}
        <path d="M25 35 L50 20 L75 35 L75 75 L50 90 L25 75 Z" />
        <path d="M25 35 L50 50 L75 35" />
        <path d="M50 50 L50 90" />
        {/* Fan circle inside left face */}
        <ellipse cx="37.5" cy="55" rx="8" ry="12" transform="rotate(-30 37.5 55)" />
        <path d="M37.5 55 L32 45" />
        <path d="M37.5 55 L43 65" />
        <path d="M37.5 55 L30 60" />
        <path d="M37.5 55 L45 50" />
        {/* Ports on top face */}
        <path d="M55 30 L60 27 L62 30 L57 33 Z" />
        <path d="M45 25 L50 22 L52 25 L47 28 Z" />
      </svg>
    );
  }
  if (type === 'data') {
    return (
      <svg width="80" height="80" viewBox="0 0 100 100" fill="none" stroke="currentColor" strokeWidth="1" strokeLinejoin="round" aria-hidden="true" className="text-white drop-shadow-[0_0_15px_rgba(255,255,255,0.1)]">
        {/* Servers */}
        <ellipse cx="35" cy="40" rx="15" ry="6" />
        <path d="M20 40 v10 a 15 6 0 0 0 30 0 v-10" />
        <path d="M20 50 v10 a 15 6 0 0 0 30 0 v-10" />
        <path d="M20 60 v10 a 15 6 0 0 0 30 0 v-10" />
        {/* Server lights */}
        <circle cx="28" cy="45" r="1.5" />
        <circle cx="33" cy="45" r="1.5" />
        <circle cx="28" cy="55" r="1.5" />
        <circle cx="33" cy="55" r="1.5" />
        <circle cx="28" cy="65" r="1.5" />
        <circle cx="33" cy="65" r="1.5" />
        {/* Folder */}
        <path d="M45 25 L55 25 L60 30 L80 30 L80 70 L45 70 Z" />
        <path d="M45 35 L80 35" />
      </svg>
    );
  }
  return null;
}

export default function PlatformFeatures() {
  return (
    <section className="bg-black text-white relative border-t border-white/5 overflow-hidden">
      <div className="max-w-[1600px] mx-auto">
        {/* Header Section */}


        {/* 4-Column Grid Section */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 border-t border-white/5 relative z-10">
          {FEATURES.map((feature, index) => (
            <div
              key={feature.title}
              className={`p-8 md:p-12 relative flex flex-col items-center text-center border-white/5
                ${index < FEATURES.length - 1 ? 'border-b lg:border-b-0 lg:border-r' : ''}
                ${index === 1 ? 'md:border-r-0 lg:border-r' : ''}
                ${index === 0 || index === 2 ? 'md:border-r' : ''}
                ${index < 2 ? 'md:border-b lg:border-b-0' : ''}
              `}
            >
              {/* Dotted Background */}
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,255,255,0.15)_1px,transparent_1px)] bg-[size:24px_24px] opacity-20 pointer-events-none" />

              {/* Icon Container */}
              <div className="h-48 w-full flex items-center justify-center relative z-10 mb-8">
                <FeatureIcon type={feature.icon} />
              </div>

              {/* Text Content */}
              <h3 className="text-white font-mono text-sm tracking-widest uppercase mb-4 relative z-10">
                {feature.title}
              </h3>
              <p className="text-white/60 text-sm leading-relaxed relative z-10">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
