'use client';

import React from 'react';
import type { LucideIcon } from 'lucide-react';

export interface IsometricTopNode {
  id: string;
  label: string;
  sublabel?: string;
  icon?: LucideIcon;
  iconSvg?: React.ReactNode;
}

export interface DossierIsometricStackProps {
  topNodes: [IsometricTopNode, IsometricTopNode, IsometricTopNode, IsometricTopNode];
  midTierLabel?: string;
  baseTierLabel?: string;
  accentColor?: string; // default subtle lime/cyan/white
  className?: string;
}

/**
 * Precision 3D Layered Isometric Technical Wireframe Visual
 * Replicating the architectural axonometric diagram in the reference design:
 * - Top Tier: 4 floating isometric cubes with domain icons and labels
 * - Mid Tier: Floating isometric circuit/network platform with interconnected nodes
 * - Base Tier: Solid foundation pedestal with isometric grid and signature core glyph
 */
export default function DossierIsometricStack({
  topNodes,
  midTierLabel = 'HIGH-FREQUENCY CONSENSUS MESH',
  baseTierLabel = 'PEGASUS CORE INFRASTRUCTURE',
  accentColor = '#CEFF00',
  className = '',
}: DossierIsometricStackProps) {
  // 4 Top cubes in isometric diamond layout:
  // 0: Top/Back (X: 250, Y: 90)
  // 1: Left (X: 180, Y: 130)
  // 2: Right (X: 320, Y: 130)
  // 3: Bottom/Front (X: 250, Y: 170)
  const cubeCoords = [
    { cx: 250, cy: 92, node: topNodes[0] }, // Back
    { cx: 175, cy: 135, node: topNodes[1] }, // Left
    { cx: 325, cy: 135, node: topNodes[2] }, // Right
    { cx: 250, cy: 178, node: topNodes[3] }, // Front
  ];

  // Dimensions of each isometric cube
  const w = 42; // half width
  const h = 24; // half height of top rhombus
  const depth = 32; // height of cube side

  return (
    <div className={`relative w-full max-w-[480px] mx-auto aspect-[5/6] flex items-center justify-center select-none ${className}`}>
      {/* Ambient background glow behind the isometric stack */}
      <div 
        className="absolute inset-x-12 inset-y-16 rounded-full blur-[70px] opacity-15 pointer-events-none"
        style={{ background: `radial-gradient(circle, ${accentColor} 0%, rgba(255,255,255,0.1) 70%, transparent 100%)` }}
      />

      <svg
        viewBox="0 0 500 580"
        className="w-full h-full overflow-visible drop-shadow-[0_12px_32px_rgba(0,0,0,0.8)]"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <defs>
          {/* Linear gradients for isometric cube facets */}
          <linearGradient id="cubeTopGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#1C1C22" stopOpacity="0.95" />
            <stop offset="100%" stopColor="#141418" stopOpacity="0.95" />
          </linearGradient>

          <linearGradient id="cubeLeftGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#101014" stopOpacity="0.95" />
            <stop offset="100%" stopColor="#08080A" stopOpacity="0.95" />
          </linearGradient>

          <linearGradient id="cubeRightGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#15151A" stopOpacity="0.95" />
            <stop offset="100%" stopColor="#0C0C0E" stopOpacity="0.95" />
          </linearGradient>

          {/* Platform slabs gradients */}
          <linearGradient id="platformTopGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#181820" stopOpacity="0.85" />
            <stop offset="100%" stopColor="#0E0E12" stopOpacity="0.85" />
          </linearGradient>

          <linearGradient id="baseSlabGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#14141A" stopOpacity="0.95" />
            <stop offset="100%" stopColor="#09090C" stopOpacity="0.95" />
          </linearGradient>

          <filter id="subtleGlow" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="2.5" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
        </defs>

        {/* =================================================================== */}
        {/* TIER 3: BASE FOUNDATION PEDESTAL (Bottom Slab)                      */}
        {/* =================================================================== */}
        <g className="transition-transform duration-500 hover:translate-y-1">
          {/* Base slab extrusion bottom thickness */}
          {/* Center around X: 250, Y: 430, Width radius: 180, Height radius: 100, Depth: 45 */}
          <path
            d="M 70 430 L 70 475 L 250 575 L 430 475 L 430 430 L 250 530 Z"
            fill="url(#cubeLeftGrad)"
            stroke="#2A2A35"
            strokeWidth="1.2"
          />
          <path
            d="M 250 430 L 250 575 L 430 475 L 430 430 Z"
            fill="url(#cubeRightGrad)"
            stroke="#2A2A35"
            strokeWidth="1.2"
          />

          {/* Base slab top isometric surface */}
          <polygon
            points="250,330 430,430 250,530 70,430"
            fill="url(#baseSlabGrad)"
            stroke="#3B3B4A"
            strokeWidth="1.4"
          />

          {/* Double inner contour wireframe outline on base */}
          <polygon
            points="250,346 414,430 250,514 86,430"
            fill="none"
            stroke="rgba(255,255,255,0.18)"
            strokeWidth="0.9"
            strokeDasharray="4 3"
          />

          {/* Base Grid Crosshairs & Coordinate Lines */}
          <line x1="160" y1="380" x2="340" y2="480" stroke="rgba(255,255,255,0.1)" strokeWidth="0.8" />
          <line x1="340" y1="380" x2="160" y2="480" stroke="rgba(255,255,255,0.1)" strokeWidth="0.8" />
          <line x1="250" y1="330" x2="250" y2="530" stroke="rgba(255,255,255,0.12)" strokeWidth="0.8" strokeDasharray="3 3" />

          {/* Core Foundation Signature Glyph (Double-lined X / Pegasus Vector Emblem) */}
          <g transform="translate(250, 430)">
            {/* Isometric projection transformed glyph */}
            <g transform="scale(1, 0.58) rotate(45)">
              {/* Outer rotated square */}
              <rect
                x="-36"
                y="-36"
                width="72"
                height="72"
                fill="none"
                stroke="rgba(255,255,255,0.3)"
                strokeWidth="1.2"
              />
              {/* Dynamic technical X glyph */}
              <path
                d="M -24 -24 L 24 24 M 24 -24 L -24 24"
                stroke="#EDE8DF"
                strokeWidth="3.5"
                strokeLinecap="round"
              />
              <path
                d="M -24 -24 L 24 24 M 24 -24 L -24 24"
                stroke={accentColor}
                strokeWidth="1"
                strokeLinecap="round"
                filter="url(#subtleGlow)"
              />
              {/* Inner core dot */}
              <circle cx="0" cy="0" r="4.5" fill="#EDE8DF" />
              <circle cx="0" cy="0" r="2" fill="#09090B" />
            </g>
          </g>

          {/* Base Micro-Label */}
          <text
            x="250"
            y="556"
            textAnchor="middle"
            fill="rgba(255,255,255,0.4)"
            fontSize="9"
            fontFamily="monospace"
            letterSpacing="0.25em"
            className="uppercase font-semibold tracking-widest"
          >
            {baseTierLabel}
          </text>
        </g>

        {/* =================================================================== */}
        {/* TIER 2: MID NETWORK / CIRCUIT PLATFORM                              */}
        {/* =================================================================== */}
        <g className="transition-transform duration-500 hover:-translate-y-1">
          {/* Vertical connection dashed pylons linking Base to Mid */}
          <line x1="120" y1="400" x2="120" y2="285" stroke="rgba(255,255,255,0.2)" strokeWidth="1" strokeDasharray="3 4" />
          <line x1="380" y1="400" x2="380" y2="285" stroke="rgba(255,255,255,0.2)" strokeWidth="1" strokeDasharray="3 4" />
          <line x1="250" y1="330" x2="250" y2="215" stroke="rgba(255,255,255,0.15)" strokeWidth="0.8" strokeDasharray="2 3" />

          {/* Mid platform extrusion thickness */}
          {/* Center around X: 250, Y: 275, Width radius: 155, Height radius: 86, Depth: 24 */}
          <path
            d="M 95 275 L 95 298 L 250 384 L 405 298 L 405 275 L 250 361 Z"
            fill="url(#cubeLeftGrad)"
            stroke="#2E2E3C"
            strokeWidth="1.2"
          />
          <path
            d="M 250 275 L 250 384 L 405 298 L 405 275 Z"
            fill="url(#cubeRightGrad)"
            stroke="#2E2E3C"
            strokeWidth="1.2"
          />

          {/* Mid platform top isometric surface */}
          <polygon
            points="250,190 405,275 250,361 95,275"
            fill="url(#platformTopGrad)"
            stroke="#4A4A5E"
            strokeWidth="1.3"
          />

          {/* Circuit network topology with illuminated node clusters */}
          <g className="circuit-network" stroke="#EDE8DF" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            {/* Hexagonal / Mesh Node Pattern */}
            {/* Center Node */}
            <circle cx="250" cy="275" r="7" fill="#181822" stroke="#EDE8DF" strokeWidth="2" />
            <circle cx="250" cy="275" r="3" fill={accentColor} />

            {/* Orbiting Satellite Nodes */}
            {/* North-West */}
            <line x1="250" y1="275" x2="200" y2="248" stroke="#EDE8DF" strokeWidth="1.5" />
            <circle cx="200" cy="248" r="5" fill="#181822" stroke="#EDE8DF" strokeWidth="1.8" />
            <circle cx="200" cy="248" r="2" fill="#EDE8DF" />

            {/* North-East */}
            <line x1="250" y1="275" x2="300" y2="248" stroke="#EDE8DF" strokeWidth="1.5" />
            <circle cx="300" cy="248" r="5" fill="#181822" stroke="#EDE8DF" strokeWidth="1.8" />
            <circle cx="300" cy="248" r="2" fill="#EDE8DF" />

            {/* South-West */}
            <line x1="250" y1="275" x2="180" y2="295" stroke="#EDE8DF" strokeWidth="1.5" />
            <circle cx="180" cy="295" r="5.5" fill="#181822" stroke="#EDE8DF" strokeWidth="1.8" />
            <circle cx="180" cy="295" r="2" fill={accentColor} />

            {/* South-East */}
            <line x1="250" y1="275" x2="320" y2="295" stroke="#EDE8DF" strokeWidth="1.5" />
            <circle cx="320" cy="295" r="5.5" fill="#181822" stroke="#EDE8DF" strokeWidth="1.8" />
            <circle cx="320" cy="295" r="2" fill={accentColor} />

            {/* Outer interconnecting loop */}
            <path
              d="M 200 248 L 300 248 L 320 295 L 250 330 L 180 295 Z"
              fill="none"
              stroke="rgba(237, 232, 223, 0.45)"
              strokeWidth="1.2"
              strokeDasharray="3 3"
            />
            {/* South anchor node */}
            <circle cx="250" cy="330" r="4" fill="#181822" stroke="#EDE8DF" strokeWidth="1.5" />
          </g>

          {/* Mid Micro-Label */}
          <text
            x="250"
            y="376"
            textAnchor="middle"
            fill="rgba(255,255,255,0.4)"
            fontSize="8.5"
            fontFamily="monospace"
            letterSpacing="0.22em"
            className="uppercase font-semibold tracking-widest"
          >
            {midTierLabel}
          </text>
        </g>

        {/* =================================================================== */}
        {/* TIER 1: 4 FLOATING ISOMETRIC CUBES (Top Tier)                       */}
        {/* =================================================================== */}
        <g className="cubes-tier">
          {/* Vertical dashed light rays ascending from Mid Platform to Cubes */}
          {cubeCoords.map((c, i) => (
            <line
              key={`ray-${i}`}
              x1={c.cx}
              y1={c.cy + depth + 10}
              x2={c.cx}
              y2={c.cy + 110}
              stroke="rgba(255,255,255,0.16)"
              strokeWidth="0.8"
              strokeDasharray="2 3"
            />
          ))}

          {/* Render 4 Isometric Cubes (Render back cube first for proper painter depth) */}
          {cubeCoords.map((c, idx) => {
            const IconCmp = c.node.icon;
            return (
              <g
                key={`cube-${idx}`}
                className="group/cube cursor-pointer transition-transform duration-300 hover:-translate-y-2"
              >
                {/* Cube Left Face */}
                <path
                  d={`M ${c.cx - w} ${c.cy} L ${c.cx} ${c.cy + h} L ${c.cx} ${c.cy + h + depth} L ${c.cx - w} ${c.cy + depth} Z`}
                  fill="url(#cubeLeftGrad)"
                  stroke="#323240"
                  strokeWidth="1.2"
                />

                {/* Cube Right Face */}
                <path
                  d={`M ${c.cx} ${c.cy + h} L ${c.cx + w} ${c.cy} L ${c.cx + w} ${c.cy + depth} L ${c.cx} ${c.cy + h + depth} Z`}
                  fill="url(#cubeRightGrad)"
                  stroke="#323240"
                  strokeWidth="1.2"
                />

                {/* Cube Top Face */}
                <polygon
                  points={`${c.cx},${c.cy - h} ${c.cx + w},${c.cy} ${c.cx},${c.cy + h} ${c.cx - w},${c.cy}`}
                  fill="url(#cubeTopGrad)"
                  stroke="#4E4E62"
                  strokeWidth="1.4"
                  className="group-hover/cube:stroke-[#EDE8DF] transition-colors"
                />

                {/* Inner Bevel Wireframe on Cube Top */}
                <polygon
                  points={`${c.cx},${c.cy - h + 4} ${c.cx + w - 7},${c.cy} ${c.cx},${c.cy + h - 4} ${c.cx - w + 7},${c.cy}`}
                  fill="none"
                  stroke="rgba(255,255,255,0.12)"
                  strokeWidth="0.8"
                />

                {/* Domain Icon / Glyph on Top Face (Simulating isometric tilt) */}
                <g transform={`translate(${c.cx}, ${c.cy})`}>
                  {c.node.iconSvg ? (
                    <g transform="scale(0.85, 0.52) rotate(-15)">{c.node.iconSvg}</g>
                  ) : IconCmp ? (
                    <foreignObject x="-14" y="-12" width="28" height="24" className="overflow-visible pointer-events-none">
                      <div className="w-full h-full flex items-center justify-center transform scale-y-[0.62] rotate-[-5deg] text-[#EDE8DF] group-hover/cube:text-white transition-colors">
                        <IconCmp className="w-4 h-4" strokeWidth={2.2} />
                      </div>
                    </foreignObject>
                  ) : (
                    /* Default Tactical User / Block Icon */
                    <g transform="scale(1, 0.58)">
                      <circle cx="0" cy="-4" r="5" fill="#EDE8DF" />
                      <path
                        d="M -8 8 C -8 3, 8 3, 8 8 Z"
                        fill="#EDE8DF"
                      />
                    </g>
                  )}
                </g>

                {/* Monospace Micro-Label underneath each cube */}
                <text
                  x={c.cx}
                  y={c.cy + depth + 16}
                  textAnchor="middle"
                  fill="#EDE8DF"
                  fontSize="8.5"
                  fontFamily="monospace"
                  fontWeight="bold"
                  letterSpacing="0.1em"
                  className="uppercase select-none opacity-85 group-hover/cube:opacity-100 transition-opacity"
                >
                  {c.node.label}
                </text>
              </g>
            );
          })}
        </g>
      </svg>
    </div>
  );
}
