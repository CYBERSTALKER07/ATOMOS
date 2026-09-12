'use client';

import { useState, useEffect } from 'react';
import LineSidebar from './LineSidebar';

export const LANDING_SECTIONS = [
  { id: 'section-overview', label: 'Overview' },
  { id: 'section-platform', label: 'Platform' },
  { id: 'section-telemetry', label: 'Telemetry' },
  { id: 'section-workflow', label: 'Workflow' },
  { id: 'section-showcase', label: 'Showcase' },
  { id: 'section-deploy', label: 'Deploy' },
];

export default function LandingPageSidebar() {
  const [activeSectionIndex, setActiveSectionIndex] = useState<number>(0);

  useEffect(() => {
    const handleScroll = () => {
      const scrollPosition = window.scrollY + window.innerHeight / 3;

      for (let i = LANDING_SECTIONS.length - 1; i >= 0; i--) {
        const section = document.getElementById(LANDING_SECTIONS[i].id);
        if (section) {
          const top = section.offsetTop;
          if (scrollPosition >= top) {
            setActiveSectionIndex(i);
            break;
          }
        }
      }
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    handleScroll(); // Initial check

    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const handleItemClick = (index: number) => {
    setActiveSectionIndex(index);
    const targetId = LANDING_SECTIONS[index]?.id;
    if (targetId) {
      const el = document.getElementById(targetId);
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    }
  };

  return (
    <aside className="fixed left-4 xl:left-8 top-1/2 -translate-y-1/2 z-[9999] hidden lg:block pointer-events-auto">
      <div className="bg-black/50 backdrop-blur-xl border border-white/10 rounded-3xl p-6 shadow-[0_0_50px_rgba(0,0,0,0.9)] transition-all hover:border-emerald-500/30">
        <LineSidebar
          items={LANDING_SECTIONS.map((s) => s.label)}
          accentColor="#10B981"
          textColor="#c4c4c4"
          markerColor="#6c6c6c"
          showIndex
          showMarker
          proximityRadius={100}
          maxShift={30}
          falloff="smooth"
          markerLength={60}
          markerGap={0}
          tickScale={0.5}
          scaleTick
          itemGap={20}
          fontSize={1.1}
          smoothing={100}
          defaultActive={activeSectionIndex}
          onItemClick={handleItemClick}
        />
      </div>
    </aside>
  );
}
