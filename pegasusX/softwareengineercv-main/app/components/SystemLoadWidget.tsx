'use client';
import React, { useEffect, useRef } from 'react';
import gsap from 'gsap';

export default function SystemLoadWidget() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const ctx = gsap.context(() => {
      // Add a cool loading animation for the circles
      gsap.fromTo('.system-load-circle', 
        { strokeDashoffset: 1000 },
        { strokeDashoffset: 130, duration: 2, ease: 'power3.out', delay: 0.5 }
      );
      gsap.fromTo('.system-load-value',
        { opacity: 0, y: 20 },
        { opacity: 1, y: 0, duration: 1, ease: 'power2.out', delay: 1, stagger: 0.2 }
      );
    }, containerRef);
    return () => ctx.revert();
  }, []);

  return (
    <div ref={containerRef} className="bg-[#0a0a0a] border border-white/5 p-6 rounded flex flex-col relative shadow-2xl overflow-hidden h-full">
      {/* Background ambient glow */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[200px] h-[200px] bg-white/[0.02] blur-[50px] rounded-full pointer-events-none" />

      {/* Header */}
      <div className="flex justify-between items-start mb-8 relative z-10">
        <div className="flex gap-3 items-center">
          <div className="w-8 h-8 rounded bg-white/5 flex items-center justify-center text-white border border-white/10">
            <svg width="16" height="16" viewBox="0 0 256 256" fill="currentColor">
              <path d="M224,48H32A16,16,0,0,0,16,64V192a16,16,0,0,0,16,16H224a16,16,0,0,0,16-16V64A16,16,0,0,0,224,48Zm0,144H32V64H224V192ZM176,88v80a8,8,0,0,1-16,0V88a8,8,0,0,1,16,0Zm-32,32v48a8,8,0,0,1-16,0V120a8,8,0,0,1,16,0Zm-32-16v64a8,8,0,0,1-16,0V104a8,8,0,0,1,16,0ZM80,136v32a8,8,0,0,1-16,0V136a8,8,0,0,1,16,0Z" />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-medium text-white">System Load</h3>
            <p className="text-[11px] text-white/40 uppercase tracking-wider font-mono mt-0.5">Active neural processing</p>
          </div>
        </div>
        <div className="text-white/40">
          <svg width="20" height="20" viewBox="0 0 256 256" fill="currentColor">
            <path d="M128,24A104,104,0,1,0,232,128,104.11,104.11,0,0,0,128,24Zm0,192a88,88,0,1,1,88-88A88.1,88.1,0,0,1,128,216Zm16-40a8,8,0,0,1-8,8,16,16,0,0,1-16-16V128a8,8,0,0,1,0-16,16,16,0,0,1,16,16v40A8,8,0,0,1,144,176ZM112,84a12,12,0,1,1,12,12A12,12,0,0,1,112,84Z" />
          </svg>
        </div>
      </div>

      {/* Main Radial Chart Area */}
      <div className="flex-1 flex flex-col items-center justify-center relative z-10 py-4">
        <div className="relative w-[180px] h-[180px] flex items-center justify-center">
          <svg width="180" height="180" viewBox="0 0 180 180" className="-rotate-90">
            {/* Background track */}
            <circle cx="90" cy="90" r="80" fill="none" stroke="rgba(255,255,255,0.05)" strokeWidth="6" />
            {/* Outer dotted ring */}
            <circle cx="90" cy="90" r="70" fill="none" stroke="rgba(255,255,255,0.1)" strokeWidth="1" strokeDasharray="4 4" />
            {/* Progress track */}
            <circle cx="90" cy="90" r="80" fill="none" stroke="white" strokeWidth="6" 
              strokeLinecap="round"
              strokeDasharray={2 * Math.PI * 80}
              strokeDashoffset={2 * Math.PI * 80 * (1 - 0.987)} // 98.7%
              className="system-load-circle"
            />
          </svg>
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <span className="text-4xl font-light text-white tracking-tighter system-load-value">
              98.7<span className="text-xl text-white/50">%</span>
            </span>
          </div>
        </div>
      </div>

      {/* Footer Stats */}
      <div className="grid grid-cols-2 gap-4 mt-6 relative z-10 pt-4 border-t border-white/5">
        <div className="flex flex-col gap-1 system-load-value">
          <div className="w-8 h-[2px] bg-white/20 mb-1" />
          <span className="text-xs text-white/40 uppercase font-mono tracking-wider">Core Systems</span>
          <span className="text-lg font-medium text-white">62%</span>
        </div>
        <div className="flex flex-col gap-1 system-load-value">
          <div className="w-8 h-[2px] bg-white/20 mb-1" />
          <span className="text-xs text-white/40 uppercase font-mono tracking-wider">Memory Allocation</span>
          <span className="text-lg font-medium text-white">15%</span>
        </div>
      </div>
    </div>
  );
}
