'use client';

import React from 'react';
import { motion } from 'framer-motion';

export default function PegasusInfraIsometric() {
  return (
    <div className="relative w-full h-[800px] bg-black overflow-hidden flex items-center justify-center font-mono">
      {/* Title / Description Overlay (like the Runlayer example left side) */}
      <div className="absolute top-20 left-8 md:left-16 z-20 max-w-lg pointer-events-none">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-emerald-400 text-[10px] tracking-widest uppercase mb-6">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
          Infrastructure Overview
        </div>
        <h2 className="text-4xl md:text-5xl font-light text-white tracking-tight mb-4">
          The Simpler, Safer Way<br />
          <span className="font-medium">to Orchestrate Logistics</span>
        </h2>
        <p className="text-zinc-400 text-sm md:text-base leading-relaxed">
          Pegasus securely connects edge fleet devices to the enterprise stack with strict ACID transactions, sub-second routing solvers, and complete observability for autonomous deployments.
        </p>
      </div>

      {/* 3D Isometric Canvas */}
      <div 
        className="absolute inset-0 flex items-center justify-center"
        style={{ 
          perspective: '2000px',
        }}
      >
        <div 
          className="relative w-full h-full flex items-center justify-center transform-gpu"
          style={{ 
            transform: 'rotateX(60deg) rotateZ(-45deg) scale(0.9)',
            transformStyle: 'preserve-3d' 
          }}
        >
          {/* Base Grid */}
          <div 
            className="absolute w-[2000px] h-[2000px] border border-white/5"
            style={{ 
              backgroundImage: `
                linear-gradient(to right, rgba(255,255,255,0.05) 1px, transparent 1px),
                linear-gradient(to bottom, rgba(255,255,255,0.05) 1px, transparent 1px)
              `,
              backgroundSize: '50px 50px',
              transform: 'translateZ(-100px)'
            }} 
          />

          {/* CENTRAL STACK: Data Plane (Spanner & Redis) */}
          <div className="absolute w-72 h-72" style={{ transform: 'translateZ(0px)', transformStyle: 'preserve-3d' }}>
            {/* Layer 1 (Bottom) */}
            <div className="absolute inset-0 rounded-2xl border border-white/10 bg-black/80 backdrop-blur-sm" style={{ transform: 'translateZ(0px)' }} />
            {/* Layer 2 (Middle) */}
            <div className="absolute inset-4 rounded-2xl border border-white/10 bg-black/80 backdrop-blur-sm" style={{ transform: 'translateZ(30px)' }} />
            {/* Layer 3 (Top) */}
            <div className="absolute inset-8 rounded-2xl border border-emerald-500/40 bg-emerald-950/20 backdrop-blur-md flex items-center justify-center shadow-[0_0_50px_rgba(16,185,129,0.1)]" style={{ transform: 'translateZ(60px)' }}>
              {/* Core Icon/Logo */}
              <svg className="w-16 h-16 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1">
                <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
              </svg>
            </div>
            
            {/* Vertical connector lines for central stack */}
            <div className="absolute top-0 left-0 w-px h-[60px] bg-white/20" style={{ transform: 'rotateX(90deg) translateZ(30px) translateY(-30px)' }} />
            <div className="absolute bottom-0 right-0 w-px h-[60px] bg-emerald-500/40" style={{ transform: 'rotateX(90deg) translateZ(30px) translateY(-30px)' }} />
          </div>

          {/* TOP RIGHT: Server Blades (Service Plane - Kafka & Go) */}
          <div className="absolute" style={{ transform: 'translate(250px, -200px)', transformStyle: 'preserve-3d' }}>
            {/* Blade 1 */}
            <div className="absolute w-40 h-24 border border-white/10 bg-black/90 rounded-lg flex items-center justify-center" style={{ transform: 'translateZ(0px)' }}>
              <span className="text-white/40 text-xs tracking-widest">KAFKA MESH</span>
            </div>
            {/* Blade 2 */}
            <div className="absolute w-40 h-24 border border-white/10 bg-black/90 rounded-lg flex items-center justify-center" style={{ transform: 'translateZ(40px)' }}>
              <span className="text-white/40 text-xs tracking-widest">GO BACKEND</span>
            </div>
            {/* Blade 3 */}
            <div className="absolute w-40 h-24 border border-purple-500/30 bg-black/90 rounded-lg flex items-center justify-center shadow-[0_0_30px_rgba(168,85,247,0.1)]" style={{ transform: 'translateZ(80px)' }}>
              <span className="text-purple-400 text-xs tracking-widest">RUST CORE</span>
            </div>
            {/* Connection line to center */}
            <div className="absolute left-[-150px] top-12 w-[150px] h-px bg-gradient-to-r from-emerald-500/40 to-purple-500/40" style={{ transform: 'translateZ(40px)' }} />
          </div>

          {/* BOTTOM LEFT: Floating Pills (Mobile Fleet) */}
          <div className="absolute" style={{ transform: 'translate(-300px, 150px)', transformStyle: 'preserve-3d' }}>
            {/* Pill 1 */}
            <div className="absolute top-0 left-0 w-32 h-10 border border-white/20 bg-black/80 rounded-full flex items-center justify-center" style={{ transform: 'translateZ(20px)' }}>
              <span className="text-white/60 text-[10px] tracking-wider">DRIVER APP</span>
            </div>
            {/* Pill 2 */}
            <div className="absolute top-16 left-12 w-32 h-10 border border-white/20 bg-black/80 rounded-full flex items-center justify-center" style={{ transform: 'translateZ(20px)' }}>
              <span className="text-white/60 text-[10px] tracking-wider">WAREHOUSE</span>
            </div>
            {/* Pill 3 */}
            <div className="absolute top-32 left-24 w-32 h-10 border border-white/20 bg-black/80 rounded-full flex items-center justify-center" style={{ transform: 'translateZ(20px)' }}>
              <span className="text-white/60 text-[10px] tracking-wider">RETAILER</span>
            </div>
            {/* Connection lines dropping down to grid */}
            <div className="absolute top-5 left-16 w-px h-[120px] bg-white/20" style={{ transform: 'rotateX(90deg) translateZ(-40px) translateY(60px)' }} />
            <div className="absolute top-[84px] left-[112px] w-px h-[120px] bg-white/20" style={{ transform: 'rotateX(90deg) translateZ(-40px) translateY(60px)' }} />
            {/* Route line to center */}
            <div className="absolute left-[160px] top-16 w-[200px] h-px border-t border-dashed border-emerald-500/30" style={{ transform: 'translateZ(-100px)' }} />
          </div>

          {/* BOTTOM RIGHT: Enterprise Portals */}
          <div className="absolute" style={{ transform: 'translate(200px, 250px)', transformStyle: 'preserve-3d' }}>
            {/* Pill 1 */}
            <div className="absolute top-0 left-0 w-40 h-12 border border-blue-500/30 bg-blue-950/20 rounded-full flex items-center justify-center shadow-[0_0_20px_rgba(59,130,246,0.1)]" style={{ transform: 'translateZ(40px)' }}>
              <span className="text-blue-400 text-[10px] tracking-wider">ADMIN PORTAL</span>
            </div>
            {/* Pill 2 */}
            <div className="absolute top-20 left-16 w-40 h-12 border border-blue-500/30 bg-blue-950/20 rounded-full flex items-center justify-center shadow-[0_0_20px_rgba(59,130,246,0.1)]" style={{ transform: 'translateZ(40px)' }}>
              <span className="text-blue-400 text-[10px] tracking-wider">SUPPLIER PORTAL</span>
            </div>
            {/* Drop lines */}
            <div className="absolute top-6 left-20 w-px h-[140px] bg-blue-500/30" style={{ transform: 'rotateX(90deg) translateZ(-30px) translateY(70px)' }} />
            {/* Route line to center */}
            <div className="absolute left-[-150px] top-6 w-[150px] h-px border-t border-blue-500/30" style={{ transform: 'translateZ(-100px)' }} />
          </div>

          {/* TOP LEFT: Edge Services (Firebase) */}
          <div className="absolute" style={{ transform: 'translate(-200px, -250px)', transformStyle: 'preserve-3d' }}>
            <div className="absolute w-48 h-32 border border-white/10 bg-black/80 rounded-xl" style={{ transform: 'translateZ(0px)' }}>
               {/* Internal Nodes */}
               <div className="absolute top-4 left-4 w-10 h-10 border border-emerald-500/50 rounded-md flex items-center justify-center">
                 <div className="w-2 h-2 bg-emerald-400 rounded-full" />
               </div>
               <div className="absolute bottom-4 right-4 w-10 h-10 border border-emerald-500/50 rounded-md flex items-center justify-center">
                 <div className="w-2 h-2 bg-emerald-400 rounded-full" />
               </div>
               <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 text-white/50 text-[10px] tracking-widest">FIREBASE EDGE</div>
            </div>
            {/* Connection */}
            <div className="absolute left-[192px] top-[64px] w-[150px] h-px border-t border-emerald-500/20" style={{ transform: 'translateZ(0px)' }} />
          </div>

        </div>
      </div>
    </div>
  );
}
