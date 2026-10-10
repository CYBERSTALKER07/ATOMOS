'use client';

import React from 'react';
import type { InfraNode } from './types';

type PegasusInfraDrawerProps = {
  node: InfraNode | null;
  onClose: () => void;
};

export default function PegasusInfraDrawer({ node, onClose }: PegasusInfraDrawerProps) {
  if (!node) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in duration-200">
      <div 
        className="relative w-full max-w-2xl bg-[#141414] border border-white/20 rounded-2xl p-6 sm:p-8 shadow-2xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Subtle decorative background gradient */}
        <div className="absolute top-0 right-0 w-80 h-80 bg-emerald-500/10 rounded-full blur-3xl pointer-events-none" />

        {/* Top Header */}
        <div className="flex items-start justify-between gap-4 pb-5 border-b border-white/10">
          <div>
            <div className="flex items-center gap-2 mb-1.5">
              <span className="w-2 h-2 rounded-full bg-emerald-400" />
              <span className="font-mono text-[11px] uppercase tracking-widest text-emerald-400 font-semibold">
                {node.category.toUpperCase()} NODE · {node.status.toUpperCase()}
              </span>
            </div>
            <h3 className="text-2xl sm:text-3xl font-medium tracking-tight text-white">
              {node.name}
            </h3>
            <p className="text-sm text-zinc-400 mt-1 font-light">
              {node.subtitle}
            </p>
          </div>

          <button
            onClick={onClose}
            className="p-2 text-zinc-400 hover:text-white bg-white/5 hover:bg-white/10 rounded-lg transition-colors font-mono text-sm"
            aria-label="Close details"
          >
            ✕
          </button>
        </div>

        {/* Metrics Grid */}
        <div className="grid grid-cols-3 gap-3 my-6">
          {node.metrics.map((m, idx) => (
            <div key={idx} className="bg-white/[0.03] border border-white/10 rounded-lg p-3 text-center">
              <p className="font-mono text-[10px] uppercase tracking-wider text-zinc-400 truncate">
                {m.label}
              </p>
              <p className="mt-1 font-mono text-lg sm:text-2xl font-bold text-white tracking-tight">
                {m.value}
                {m.unit && <span className="text-xs font-normal text-zinc-400 ml-1">{m.unit}</span>}
              </p>
            </div>
          ))}
        </div>

        {/* Description */}
        <div className="space-y-4">
          <div>
            <h4 className="font-mono text-xs uppercase tracking-wider text-zinc-400 mb-2">
              Architecture & Invariants
            </h4>
            <p className="text-sm sm:text-base text-zinc-300 leading-relaxed font-light">
              {node.description}
            </p>
          </div>

          {/* Tech Stack Chips */}
          <div>
            <h4 className="font-mono text-xs uppercase tracking-wider text-zinc-400 mb-2">
              Core Technologies
            </h4>
            <div className="flex flex-wrap gap-2">
              {node.tech.map((t, idx) => (
                <span
                  key={idx}
                  className="font-mono text-xs px-2.5 py-1 rounded bg-white/5 border border-white/10 text-zinc-200"
                >
                  {t}
                </span>
              ))}
            </div>
          </div>

          {/* Repo Implementation Path */}
          <div className="pt-2">
            <h4 className="font-mono text-xs uppercase tracking-wider text-zinc-400 mb-1.5">
              Pegasus Repository Location
            </h4>
            <code className="block font-mono text-xs px-3 py-2 bg-black/60 border border-white/10 rounded text-emerald-300 overflow-x-auto">
              {node.repoPath}
            </code>
          </div>
        </div>

        {/* Footer Action */}
        <div className="mt-8 pt-4 border-t border-white/10 flex items-center justify-end gap-3">
          <button
            onClick={onClose}
            className="px-5 py-2.5 bg-white text-black font-mono text-xs uppercase tracking-wider font-semibold rounded hover:bg-zinc-200 transition-colors"
          >
            Close Inspector
          </button>
        </div>
      </div>
    </div>
  );
}
