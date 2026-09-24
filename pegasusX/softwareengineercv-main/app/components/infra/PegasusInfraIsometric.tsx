'use client';

import React, { useState, useId } from 'react';
import type { InfraLayer, InfraNode } from './types';
import { INFRA_NODES, HUD_CARDS } from './infraData';

type PegasusInfraIsometricProps = {
  activeLayer: InfraLayer;
  onSelectNode: (node: InfraNode) => void;
  selectedNodeId?: string;
  className?: string;
};

export default function PegasusInfraIsometric({
  activeLayer,
  onSelectNode,
  selectedNodeId,
  className = '',
}: PegasusInfraIsometricProps) {
  const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null);
  const glowFilterId = useId();

  // Helper to determine if a node matches current active layer
  const isNodeVisible = (node: InfraNode) => {
    if (activeLayer === 'all') return true;
    return node.category === activeLayer;
  };

  return (
    <div className={`relative w-full aspect-[16/10] min-h-[460px] md:min-h-[560px] lg:min-h-[640px] bg-[#0d0d0d] rounded-2xl overflow-hidden border border-white/10 select-none ${className}`}>
      
      {/* Dynamic Background Grid Texture */}
      <div 
        className="absolute inset-0 pointer-events-none opacity-40"
        style={{
          backgroundImage: `
            radial-gradient(circle at 50% 50%, rgba(16, 185, 129, 0.08) 0%, transparent 70%),
            linear-gradient(to right, rgba(255, 255, 255, 0.02) 1px, transparent 1px),
            linear-gradient(to bottom, rgba(255, 255, 255, 0.02) 1px, transparent 1px)
          `,
          backgroundSize: '100% 100%, 40px 40px, 40px 40px',
        }}
      />

      {/* SVG Canvas for Isometric Architecture Topology */}
      <svg
        viewBox="0 0 1000 680"
        className="w-full h-full block"
        preserveAspectRatio="xMidYMid meet"
      >
        <defs>
          {/* Emerald Circuit Glow Filter */}
          <filter id={glowFilterId} x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>

          {/* Linear Gradients for Isometric Building Faces */}
          <linearGradient id="roofGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#2c2c2c" />
            <stop offset="100%" stopColor="#1e1e1e" />
          </linearGradient>
          <linearGradient id="wallLeftGrad" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#1c1c1c" />
            <stop offset="100%" stopColor="#121212" />
          </linearGradient>
          <linearGradient id="wallRightGrad" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#242424" />
            <stop offset="100%" stopColor="#161616" />
          </linearGradient>

          {/* Active Highlight Gradients */}
          <linearGradient id="activeRoofGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#064e3b" />
            <stop offset="100%" stopColor="#022c22" />
          </linearGradient>
        </defs>

        {/* ========================================================
            1. ISOMETRIC FLOOR GRID PLANE
        ======================================================== */}
        <g stroke="rgba(255, 255, 255, 0.04)" strokeWidth="1" fill="none">
          {/* Isometric diamond floor mesh */}
          {[-4, -2, 0, 2, 4, 6].map((offset, i) => (
            <path
              key={`h-${i}`}
              d={`M ${100 + offset * 30} ${380 + offset * 18} L ${500 + offset * 30} ${140 + offset * 18} L ${900 + offset * 30} ${380 + offset * 18} L ${500 + offset * 30} ${620 + offset * 18} Z`}
            />
          ))}
        </g>

        {/* ========================================================
            2. LUMINOUS EMERALD CIRCUIT BUS TRACES
        ======================================================== */}
        <g stroke="#10b981" strokeWidth="1.5" fill="none" opacity="0.85">
          {/* Main bus: Center Core to Backend Tower */}
          <path
            d="M 480 470 L 400 520 L 320 470 L 320 420"
            strokeDasharray="4 2"
          />
          {/* Central to East Tower (Optimizer) */}
          <path
            d="M 520 470 L 620 530 L 720 470 L 720 400"
            filter={`url(#${glowFilterId})`}
          />
          {/* East to South-East Mobile Node */}
          <path
            d="M 720 470 L 640 520 L 620 540"
            strokeDasharray="6 3"
          />
          {/* Central to West Tower (Portals) */}
          <path
            d="M 450 420 L 320 340 L 320 300"
            filter={`url(#${glowFilterId})`}
          />
          {/* West to Gate Security */}
          <path
            d="M 280 420 L 240 440 L 240 490"
            strokeDasharray="4 2"
          />
          {/* Interconnect Cross-line */}
          <path
            d="M 320 470 L 480 570 L 640 520"
            stroke="#10b981"
            strokeWidth="1.2"
          />

          {/* Micro-Controller Connector Chips along circuit lines */}
          <g fill="#0e1713" stroke="#10b981" strokeWidth="1">
            {/* Chip West */}
            <path d="M 335 440 L 350 430 L 365 440 L 350 450 Z" />
            <text x="350" y="443" fill="#10b981" fontSize="6" fontFamily="monospace" textAnchor="middle">SPNR</text>

            {/* Chip Center-South */}
            <path d="M 465 540 L 480 530 L 495 540 L 480 550 Z" />
            <text x="480" y="543" fill="#10b981" fontSize="6" fontFamily="monospace" textAnchor="middle">KAFKA</text>

            {/* Chip East */}
            <path d="M 765 445 L 780 435 L 795 445 L 780 455 Z" />
            <text x="780" y="448" fill="#10b981" fontSize="6" fontFamily="monospace" textAnchor="middle">RUST</text>
          </g>

          {/* Glowing Animated Signal Packets */}
          <circle cx="480" cy="570" r="3" fill="#34d399">
            <animate attributeName="opacity" values="0.3;1;0.3" dur="1.8s" repeatCount="indefinite" />
          </circle>
          <circle cx="320" cy="470" r="2.5" fill="#34d399">
            <animate attributeName="opacity" values="0.2;1;0.2" dur="2.2s" repeatCount="indefinite" />
          </circle>
          <circle cx="640" cy="520" r="2.5" fill="#34d399">
            <animate attributeName="opacity" values="1;0.2;1" dur="1.5s" repeatCount="indefinite" />
          </circle>
        </g>

        {/* ========================================================
            3. ISOMETRIC ARCHITECTURE NODES & CLUSTERS
        ======================================================== */}

        {/* --- NODE A: WEST TOWER (Unified Control Plane & Portals) --- */}
        <g
          className="cursor-pointer transition-transform duration-200 hover:scale-[1.01]"
          onClick={() => onSelectNode(INFRA_NODES[3])}
          onMouseEnter={() => setHoveredNodeId('edge-portals')}
          onMouseLeave={() => setHoveredNodeId(null)}
          opacity={isNodeVisible(INFRA_NODES[3]) ? 1 : 0.25}
        >
          {/* Base shadow */}
          <polygon points="320,440 370,410 320,380 270,410" fill="rgba(0,0,0,0.6)" />

          {/* Building Left Wall */}
          <polygon
            points="270,410 320,440 320,290 270,260"
            fill="url(#wallLeftGrad)"
            stroke={hoveredNodeId === 'edge-portals' || selectedNodeId === 'edge-portals' ? '#10b981' : '#ffffff'}
            strokeWidth="1"
          />
          {/* Horizontal floor slats (Left) */}
          {[310, 330, 350, 370, 390].map((y) => (
            <line key={`slat-l-${y}`} x1="270" y1={y - 130} x2="320" y2={y - 100} stroke="rgba(255,255,255,0.25)" strokeWidth="0.8" />
          ))}

          {/* Building Right Wall */}
          <polygon
            points="320,440 370,410 370,260 320,290"
            fill="url(#wallRightGrad)"
            stroke={hoveredNodeId === 'edge-portals' || selectedNodeId === 'edge-portals' ? '#10b981' : '#ffffff'}
            strokeWidth="1"
          />
          {/* Horizontal floor slats (Right) */}
          {[310, 330, 350, 370, 390].map((y) => (
            <line key={`slat-r-${y}`} x1="320" y1={y - 100} x2="370" y2={y - 130} stroke="rgba(255,255,255,0.25)" strokeWidth="0.8" />
          ))}

          {/* Building Roof */}
          <polygon
            points="320,290 370,260 320,230 270,260"
            fill={hoveredNodeId === 'edge-portals' || selectedNodeId === 'edge-portals' ? 'url(#activeRoofGrad)' : 'url(#roofGrad)'}
            stroke={hoveredNodeId === 'edge-portals' || selectedNodeId === 'edge-portals' ? '#34d399' : '#ffffff'}
            strokeWidth="1"
          />
          {/* Roof Spire Antenna */}
          <line x1="320" y1="230" x2="320" y2="190" stroke="#ffffff" strokeWidth="1.2" />
          <circle cx="320" cy="190" r="2" fill="#10b981" />
        </g>


        {/* --- NODE B: CENTRAL DATA CORE (Cloud Spanner & Server Discs) --- */}
        <g
          className="cursor-pointer transition-transform duration-200 hover:scale-[1.01]"
          onClick={() => onSelectNode(INFRA_NODES[0])}
          onMouseEnter={() => setHoveredNodeId('data-core')}
          onMouseLeave={() => setHoveredNodeId(null)}
          opacity={isNodeVisible(INFRA_NODES[0]) ? 1 : 0.25}
        >
          {/* Stacked Server Disc 1 (Bottom) */}
          <g>
            <polygon points="410,470 500,520 590,470 500,420" fill="url(#wallLeftGrad)" stroke="#ffffff" strokeWidth="1" />
            <polygon points="410,470 500,520 500,545 410,495" fill="#161616" stroke="#ffffff" strokeWidth="1" />
            <polygon points="500,520 590,470 590,495 500,545" fill="#1c1c1c" stroke="#ffffff" strokeWidth="1" />
            {/* Server LED indicators */}
            <circle cx="495" cy="530" r="1.8" fill="#10b981" />
            <circle cx="505" cy="530" r="1.8" fill="#10b981" />
            <circle cx="515" cy="530" r="1.8" fill="#34d399" />
          </g>

          {/* Stacked Server Disc 2 (Top) */}
          <g>
            <polygon points="410,445 500,495 590,445 500,395" fill="url(#wallLeftGrad)" stroke="#ffffff" strokeWidth="1" />
            <polygon points="410,445 500,495 500,470 410,420" fill="#181818" stroke="#ffffff" strokeWidth="1" />
            <polygon points="500,495 590,445 590,420 500,470" fill="#202020" stroke="#ffffff" strokeWidth="1" />
            {/* Disc Top Face Ring Pattern */}
            <ellipse cx="500" cy="445" rx="35" ry="18" fill="none" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
            <circle cx="495" cy="482" r="1.8" fill="#10b981" />
            <circle cx="505" cy="482" r="1.8" fill="#34d399" />
            <circle cx="515" cy="482" r="1.8" fill="#10b981" />
          </g>

          {/* Floating 3D Cloud Glyph over Central Spanner Stack */}
          <g transform="translate(425, 300)">
            {/* Cloud extruded back/shadow */}
            <path
              d="M 35,65 C 20,65 10,52 10,38 C 10,25 20,16 32,15 C 38,5 50,0 65,0 C 82,0 95,8 100,22 C 105,20 110,18 118,18 C 132,18 145,28 145,42 C 145,55 135,65 120,65 Z"
              fill="#181818"
              stroke={hoveredNodeId === 'data-core' || selectedNodeId === 'data-core' ? '#10b981' : '#ffffff'}
              strokeWidth="1.4"
            />
            {/* Front Cloud Face highlight */}
            <path
              d="M 30,60 C 15,60 5,47 5,33 C 5,20 15,11 27,10 C 33,0 45,-5 60,-5 C 77,-5 90,3 95,17 C 100,15 105,13 113,13 C 127,13 140,23 140,37 C 140,50 130,60 115,60 Z"
              fill="#222222"
              stroke={hoveredNodeId === 'data-core' || selectedNodeId === 'data-core' ? '#34d399' : '#ffffff'}
              strokeWidth="1.4"
            />
            {/* Stylized Internal Circuit Mark on Cloud */}
            <path d="M 50,30 L 70,30 L 80,42 L 105,42" fill="none" stroke="#10b981" strokeWidth="1.5" />
            <circle cx="50" cy="30" r="2" fill="#34d399" />
            <circle cx="105" cy="42" r="2" fill="#34d399" />
          </g>
        </g>


        {/* --- NODE C: EAST TOWERS (Go Backend & Rust Optimizer Clusters) --- */}
        <g
          className="cursor-pointer transition-transform duration-200 hover:scale-[1.01]"
          onClick={() => onSelectNode(INFRA_NODES[1])}
          onMouseEnter={() => setHoveredNodeId('backend-go')}
          onMouseLeave={() => setHoveredNodeId(null)}
          opacity={isNodeVisible(INFRA_NODES[1]) ? 1 : 0.25}
        >
          {/* East Tower 1: Go State Machine (Tall) */}
          <polygon
            points="610,460 670,490 670,290 610,260"
            fill="url(#wallLeftGrad)"
            stroke={hoveredNodeId === 'backend-go' || selectedNodeId === 'backend-go' ? '#10b981' : '#ffffff'}
            strokeWidth="1"
          />
          <polygon
            points="670,490 730,460 730,260 670,290"
            fill="url(#wallRightGrad)"
            stroke={hoveredNodeId === 'backend-go' || selectedNodeId === 'backend-go' ? '#10b981' : '#ffffff'}
            strokeWidth="1"
          />
          <polygon
            points="670,290 730,260 670,230 610,260"
            fill={hoveredNodeId === 'backend-go' || selectedNodeId === 'backend-go' ? 'url(#activeRoofGrad)' : 'url(#roofGrad)'}
            stroke={hoveredNodeId === 'backend-go' || selectedNodeId === 'backend-go' ? '#34d399' : '#ffffff'}
            strokeWidth="1"
          />
          {/* Slats */}
          {[300, 325, 350, 375, 400, 425].map((y) => (
            <React.Fragment key={`go-slat-${y}`}>
              <line x1="610" y1={y - 30} x2="670" y2={y} stroke="rgba(255,255,255,0.22)" strokeWidth="0.8" />
              <line x1="670" y1={y} x2="730" y2={y - 30} stroke="rgba(255,255,255,0.22)" strokeWidth="0.8" />
            </React.Fragment>
          ))}
          {/* Roof Beacon */}
          <line x1="670" y1="230" x2="670" y2="200" stroke="#ffffff" strokeWidth="1.2" />
          <circle cx="670" cy="200" r="2.5" fill="#a78bfa" />
        </g>

        {/* East Tower 2: Rust VRP Solver Core */}
        <g
          className="cursor-pointer transition-transform duration-200 hover:scale-[1.01]"
          onClick={() => onSelectNode(INFRA_NODES[2])}
          onMouseEnter={() => setHoveredNodeId('optimizer-rust')}
          onMouseLeave={() => setHoveredNodeId(null)}
          opacity={isNodeVisible(INFRA_NODES[2]) ? 1 : 0.25}
        >
          <polygon
            points="710,430 760,455 760,300 710,275"
            fill="url(#wallLeftGrad)"
            stroke={hoveredNodeId === 'optimizer-rust' || selectedNodeId === 'optimizer-rust' ? '#60a5fa' : '#ffffff'}
            strokeWidth="1"
          />
          <polygon
            points="760,455 810,430 810,275 760,300"
            fill="url(#wallRightGrad)"
            stroke={hoveredNodeId === 'optimizer-rust' || selectedNodeId === 'optimizer-rust' ? '#60a5fa' : '#ffffff'}
            strokeWidth="1"
          />
          <polygon
            points="760,300 810,275 760,250 710,275"
            fill={hoveredNodeId === 'optimizer-rust' || selectedNodeId === 'optimizer-rust' ? '#1e293b' : 'url(#roofGrad)'}
            stroke={hoveredNodeId === 'optimizer-rust' || selectedNodeId === 'optimizer-rust' ? '#60a5fa' : '#ffffff'}
            strokeWidth="1"
          />
          {/* Slats */}
          {[310, 335, 360, 385, 410].map((y) => (
            <React.Fragment key={`rust-slat-${y}`}>
              <line x1="710" y1={y - 25} x2="760" y2={y} stroke="rgba(255,255,255,0.2)" strokeWidth="0.8" />
              <line x1="760" y1={y} x2="810" y2={y - 25} stroke="rgba(255,255,255,0.2)" strokeWidth="0.8" />
            </React.Fragment>
          ))}
          <line x1="760" y1="250" x2="760" y2="225" stroke="#ffffff" strokeWidth="1" />
          <circle cx="760" cy="225" r="2" fill="#60a5fa" />
        </g>


        {/* --- NODE D: GATEWAY BLOCK (Cryptographic Gate Terminal) --- */}
        <g
          className="cursor-pointer transition-transform duration-200 hover:scale-[1.01]"
          onClick={() => onSelectNode(INFRA_NODES[4])}
          onMouseEnter={() => setHoveredNodeId('gate-security')}
          onMouseLeave={() => setHoveredNodeId(null)}
          opacity={isNodeVisible(INFRA_NODES[4]) ? 1 : 0.25}
        >
          {/* Stepped Low-Poly Gate Architecture */}
          <polygon
            points="180,480 230,510 230,450 180,420"
            fill="url(#wallLeftGrad)"
            stroke={hoveredNodeId === 'gate-security' || selectedNodeId === 'gate-security' ? '#10b981' : '#ffffff'}
            strokeWidth="1"
          />
          <polygon
            points="230,510 260,495 260,435 230,450"
            fill="url(#wallRightGrad)"
            stroke={hoveredNodeId === 'gate-security' || selectedNodeId === 'gate-security' ? '#10b981' : '#ffffff'}
            strokeWidth="1"
          />
          <polygon
            points="230,450 260,435 210,405 180,420"
            fill="url(#roofGrad)"
            stroke="#ffffff"
            strokeWidth="1"
          />
          {/* Internal Gate Cutout Notch */}
          <polygon
            points="200,490 230,510 230,480 200,460"
            fill="#090909"
            stroke="#10b981"
            strokeWidth="0.8"
          />
        </g>


        {/* --- NODE E: SOUTH-EAST STEP BLOCK (Mobile & Handheld Offline Mesh) --- */}
        <g
          className="cursor-pointer transition-transform duration-200 hover:scale-[1.01]"
          onClick={() => onSelectNode(INFRA_NODES[5])}
          onMouseEnter={() => setHoveredNodeId('mobile-fleet')}
          onMouseLeave={() => setHoveredNodeId(null)}
          opacity={isNodeVisible(INFRA_NODES[5]) ? 1 : 0.25}
        >
          <polygon
            points="590,560 630,580 630,540 590,520"
            fill="url(#wallLeftGrad)"
            stroke={hoveredNodeId === 'mobile-fleet' || selectedNodeId === 'mobile-fleet' ? '#10b981' : '#ffffff'}
            strokeWidth="1"
          />
          <polygon
            points="630,580 670,560 670,520 630,540"
            fill="url(#wallRightGrad)"
            stroke={hoveredNodeId === 'mobile-fleet' || selectedNodeId === 'mobile-fleet' ? '#10b981' : '#ffffff'}
            strokeWidth="1"
          />
          <polygon
            points="630,540 670,520 630,500 590,520"
            fill="url(#roofGrad)"
            stroke="#ffffff"
            strokeWidth="1"
          />
        </g>


        {/* ========================================================
            4. HUD LEADER LINES WITH GLOWING ANCHOR POINTS
        ======================================================== */}
        <g stroke="rgba(255, 255, 255, 0.35)" strokeWidth="1" fill="none">
          {HUD_CARDS.map((card) => {
            const isTargetNodeActive = activeLayer === 'all' || INFRA_NODES.find(n => n.id === card.anchorNodeId)?.category === activeLayer;
            if (!isTargetNodeActive) return null;

            return (
              <g key={`leader-${card.id}`}>
                <line
                  x1={`${card.leaderLine.startX * 10}`}
                  y1={`${card.leaderLine.startY * 6.8}`}
                  x2={`${card.leaderLine.endX * 10}`}
                  y2={`${card.leaderLine.endY * 6.8}`}
                  strokeDasharray="3 3"
                />
                {/* Glowing anchor point */}
                <circle
                  cx={`${card.leaderLine.endX * 10}`}
                  cy={`${card.leaderLine.endY * 6.8}`}
                  r="3.5"
                  fill="#10b981"
                  stroke="#ffffff"
                  strokeWidth="1"
                />
              </g>
            );
          })}
        </g>
      </svg>

      {/* ========================================================
          5. FLOATING GLASSMORPHIC TELEMETRY HUD BADGES
      ======================================================== */}
      {HUD_CARDS.map((card) => {
        const isTargetNodeActive = activeLayer === 'all' || INFRA_NODES.find(n => n.id === card.anchorNodeId)?.category === activeLayer;
        if (!isTargetNodeActive) return null;

        const targetNode = INFRA_NODES.find(n => n.id === card.anchorNodeId);

        return (
          <div
            key={card.id}
            onClick={() => targetNode && onSelectNode(targetNode)}
            className="absolute z-20 cursor-pointer transform -translate-x-1/2 -translate-y-1/2 transition-all duration-200 hover:scale-105"
            style={{
              left: `${card.cardPos.x}%`,
              top: `${card.cardPos.y}%`,
            }}
          >
            <div className="bg-[#121212]/85 backdrop-blur-md border border-white/20 hover:border-[#10b981] rounded-lg p-2.5 sm:p-3 shadow-[0_8px_24px_rgba(0,0,0,0.7)] text-left min-w-[140px] sm:min-w-[180px]">
              {/* Header */}
              <div className="flex items-center justify-between gap-2 pb-1.5 border-b border-white/10">
                <span className="font-mono text-[9px] sm:text-[10px] uppercase tracking-wider text-zinc-300 font-semibold truncate">
                  {card.title}
                </span>
                <span
                  className="font-mono text-[8px] sm:text-[9px] uppercase px-1 py-0.5 rounded tracking-widest font-bold"
                  style={{
                    backgroundColor: 'rgba(16, 185, 129, 0.15)',
                    color: card.statusColor || '#10b981',
                  }}
                >
                  {card.status}
                </span>
              </div>

              {/* Metric rows */}
              <div className="mt-2 space-y-1 font-mono text-[9px] sm:text-[11px]">
                {card.metrics.map((m, idx) => (
                  <div key={idx} className="flex items-center justify-between gap-3 text-zinc-400">
                    <span className="flex items-center gap-1.5 truncate">
                      {m.icon && <span className="text-emerald-400 text-[10px]">{m.icon}</span>}
                      <span>{m.label}</span>
                    </span>
                    <span
                      className="font-semibold shrink-0"
                      style={{ color: m.color || '#ffffff' }}
                    >
                      {m.value}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        );
      })}

      {/* Interactive Helper Legend */}
      <div className="absolute bottom-3 left-4 z-20 pointer-events-none flex items-center gap-3 font-mono text-[10px] sm:text-[11px] text-zinc-400 bg-black/60 backdrop-blur-sm px-3 py-1.5 rounded-md border border-white/10">
        <span className="w-2 h-2 rounded-full bg-[#10b981] animate-pulse" />
        <span>Click any node or badge to inspect service architecture</span>
      </div>

    </div>
  );
}
