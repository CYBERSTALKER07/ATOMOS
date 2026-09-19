'use client';

import React from 'react';
import PageSection from './layout/PageSection';
import GooeyAgent from '@/app/components/visuals/GooeyAgent';

const checkIcon = (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-white shrink-0 mt-1">
    <path d="M5 12h14M12 5l7 7-7 7" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);
// Standard check icon is more appropriate:
const actualCheck = (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-white shrink-0 mt-1">
    <path d="M20 6L9 17l-5-5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

export default function Skills() {
  return (
    <PageSection
      id="capabilities"
      className="bg-[#050505] !p-0 border-y border-white/10"
      innerClassName="w-full max-w-[1600px] mx-auto px-4 sm:px-8 lg:px-16"
    >
      <div className="flex flex-col lg:flex-row items-center justify-between py-24 lg:py-40 gap-16 lg:gap-24">
        {/* Left Side: Content */}
        <div className="flex-1 w-full relative z-10">
          <h2 className="text-5xl sm:text-6xl lg:text-8xl font-medium tracking-tight text-white mb-16 leading-[1.05]">
            Key Capabilities
          </h2>
          
          <ul className="space-y-8 lg:space-y-12">
            <li className="flex gap-6 items-start">
              {actualCheck}
              <span className="text-2xl sm:text-3xl lg:text-4xl font-light text-white/90 leading-tight">
                Real-time visibility into supply chain operations
              </span>
            </li>
            <li className="flex gap-6 items-start">
              {actualCheck}
              <span className="text-2xl sm:text-3xl lg:text-4xl font-light text-white/90 leading-tight">
                Predictive analytics powered by AI/ML
              </span>
            </li>
            <li className="flex gap-6 items-start">
              {actualCheck}
              <span className="text-2xl sm:text-3xl lg:text-4xl font-light text-white/90 leading-tight">
                Seamless integration with existing ERP systems
              </span>
            </li>
          </ul>
        </div>

        {/* Right Side: GooeyAgent */}
        <div className="flex-1 w-full flex justify-center items-center lg:justify-end relative z-10 scale-[1.5] lg:scale-[2] transform origin-center lg:origin-right opacity-90">
          <GooeyAgent color="#ffffff" size={300} />
        </div>
      </div>
    </PageSection>
  );
}
