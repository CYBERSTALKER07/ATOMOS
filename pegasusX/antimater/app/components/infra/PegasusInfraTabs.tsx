'use client';

import React from 'react';
import type { InfraLayer } from './types';

type PegasusInfraTabsProps = {
  activeLayer: InfraLayer;
  onChangeLayer: (layer: InfraLayer) => void;
  className?: string;
};

const LAYERS: { id: InfraLayer; label: string; count: number }[] = [
  { id: 'all', label: 'All Layers', count: 6 },
  { id: 'data', label: 'Data & State', count: 1 },
  { id: 'compute', label: 'Compute & Solver', count: 2 },
  { id: 'edge', label: 'Edge Surfaces', count: 2 },
  { id: 'security', label: 'Gate Security', count: 1 },
];

export default function PegasusInfraTabs({
  activeLayer,
  onChangeLayer,
  className = '',
}: PegasusInfraTabsProps) {
  return (
    <div className={`flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-white/10 ${className}`}>
      {/* Layer Filter Buttons */}
      <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
        {LAYERS.map((layer) => {
          const isActive = activeLayer === layer.id;
          return (
            <button
              key={layer.id}
              onClick={() => onChangeLayer(layer.id)}
              className={`inline-flex items-center gap-2 px-3 py-1.5 rounded text-xs font-mono uppercase tracking-wider transition-all duration-150 ${
                isActive
                  ? 'bg-white text-black font-semibold shadow-sm'
                  : 'bg-white/[0.04] text-zinc-400 hover:text-white hover:bg-white/[0.08] border border-white/5'
              }`}
            >
              <span>{layer.label}</span>
              <span
                className={`text-[10px] px-1 py-0.2 rounded ${
                  isActive ? 'bg-black/15 text-black' : 'bg-white/10 text-zinc-400'
                }`}
              >
                {layer.count}
              </span>
            </button>
          );
        })}
      </div>

      {/* Live System Status Indicator */}
      <div className="flex items-center gap-3 font-mono text-[11px] text-zinc-400">
        <span className="flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
          <span className="text-white font-medium">99.998% UPTIME</span>
        </span>
        <span className="text-zinc-600">/</span>
        <span className="text-zinc-300">0.00% DRIFT</span>
      </div>
    </div>
  );
}
