'use client';

import { useEffect, useState } from 'react';
import BranchedMenu from './visuals/BranchedMenu';

export default function GlobalSectionTracker() {
  const [activeSection, setActiveSection] = useState('section-overview');

  useEffect(() => {
    const sectionIds = [
      'section-overview',
      'section-platform',
      'section-last-mile',
      'section-analytics',
      'section-workflow',
      'section-showcase',
      'section-intelligence',
      'section-deploy',
    ];

    const observer = new IntersectionObserver(
      (entries) => {
        // Find the section most visible in the viewport
        entries.forEach((entry) => {
          if (entry.isIntersecting && entry.intersectionRatio >= 0.2) {
            setActiveSection(entry.target.id);
          }
        });
      },
      {
        root: null,
        rootMargin: '-20% 0px -60% 0px',
        threshold: [0.2, 0.5, 0.8],
      }
    );

    sectionIds.forEach((id) => {
      const el = document.getElementById(id);
      if (el) observer.observe(el);
    });

    return () => observer.disconnect();
  }, []);

  const handleSelect = (value: string) => {
    setActiveSection(value);
    const el = document.getElementById(value);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth' });
    }
  };

  return (
    <div className="fixed left-6 sm:left-10 top-1/2 -translate-y-1/2 z-[9999] opacity-30 hover:opacity-100 transition-opacity duration-300 hidden lg:block mix-blend-difference pointer-events-auto">
      <BranchedMenu
        items={[
          {
            label: 'Introduction',
            children: [
              { value: 'section-overview', label: 'Overview' },
              { value: 'section-platform', label: 'Architecture & Dispatch' }
            ]
          },
          {
            label: 'Delivery & Analytics',
            children: [
              { value: 'section-last-mile', label: 'Live Fleet Tracking' },
              { value: 'section-analytics', label: 'AI Operations' }
            ]
          },
          {
            label: 'Logistics Core',
            children: [
              { value: 'section-workflow', label: 'Key Capabilities' },
              { value: 'section-intelligence', label: 'Spur Intelligence' }
            ]
          },
          {
            label: 'Ecosystem',
            children: [
              { value: 'section-showcase', label: 'Projects & Partners' },
              { value: 'section-deploy', label: 'Licensing & Deploy' }
            ]
          }
        ]}
        defaultOpen={[0, 1, 2, 3]}
        defaultActive={activeSection}
        onSelect={(value) => handleSelect(value)}
        color="#ffffff"
        accentColor="#ffffff"
        lineColor="#ffffff"
                width={340}
                              />
    </div>
  );
}
