'use client';

import React, { useState } from 'react';
import type { InfraLayer, InfraNode } from './types';
import PegasusInfraIsometric from './PegasusInfraIsometric';
import PegasusInfraTabs from './PegasusInfraTabs';
import PegasusInfraDrawer from './PegasusInfraDrawer';
import { INFRA_NODES } from './infraData';
import { useLanguage } from '@/app/context/LanguageContext';

type PegasusInfraSectionProps = {
  id?: string;
  className?: string;
};

export default function PegasusInfraSection({ id, className = '' }: PegasusInfraSectionProps) {
  const { language } = useLanguage();
  const isRu = language === 'ru';

  const [activeLayer, setActiveLayer] = useState<InfraLayer>('all');
  const [selectedNode, setSelectedNode] = useState<InfraNode | null>(null);

  return (
    <div
      id={id}
      aria-label="Pegasus Infrastructure Overview"
      className={`relative w-full bg-black py-8 sm:py-12 overflow-hidden text-white ${className}`}
    >
      <div className="w-full max-w-[1380px] mx-auto space-y-8">
        
        {/* Section Header */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 bg-[#10b981] inline-block" />
            <span className="font-mono text-xs sm:text-[13px] font-bold tracking-[0.22em] uppercase text-zinc-400">
              {isRu ? 'ПОД КАПОТОМ · РАСПРЕДЕЛЕННАЯ ИНФРАСТРУКТУРА' : 'UNDER THE HOOD · DISTRIBUTED ARCHITECTURE'}
            </span>
          </div>

          <div className="flex flex-col lg:flex-row lg:items-end justify-between gap-6">
            <h2 className="text-3xl sm:text-4xl md:text-5xl font-light tracking-tight leading-[1.08] max-w-3xl">
              <span className="font-medium text-white">
                {isRu ? 'Архитектура на базе Spanner, Kafka & Rust' : 'Built on Cloud Spanner, Kafka & Bare-Metal Rust'}
              </span>
            </h2>
            <p className="text-sm sm:text-base text-zinc-400 max-w-md font-light leading-relaxed">
              {isRu
                ? 'Многорегиональная транзакционная согласованность, субсекундный расчет маршрутов и криптографическая верификация на воротах.'
                : 'Multi-region transactional consensus, sub-second routing solvers, and cryptographic edge gate verification mapped to live services.'}
            </p>
          </div>
        </div>

        {/* Modular Layer Switcher */}
        <PegasusInfraTabs
          activeLayer={activeLayer}
          onChangeLayer={setActiveLayer}
        />

        {/* Master Isometric Architecture Topology Canvas */}
        <PegasusInfraIsometric
          activeLayer={activeLayer}
          onSelectNode={(node) => setSelectedNode(node)}
          selectedNodeId={selectedNode?.id}
        />

        {/* Quick Highlights Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4 pt-4">
          <div className="bg-white/[0.02] border border-white/10 rounded-xl p-4">
            <p className="font-mono text-[10px] uppercase tracking-widest text-zinc-500">Transaction Guarantee</p>
            <p className="mt-1 font-mono text-xl sm:text-2xl font-semibold text-white">Strict ACID</p>
            <p className="text-xs text-zinc-400 mt-1">Cloud Spanner TrueTime API</p>
          </div>
          <div className="bg-white/[0.02] border border-white/10 rounded-xl p-4">
            <p className="font-mono text-[10px] uppercase tracking-widest text-zinc-500">Routing Solver</p>
            <p className="mt-1 font-mono text-xl sm:text-2xl font-semibold text-emerald-400">118ms</p>
            <p className="text-xs text-zinc-400 mt-1">Parallel Rayon Rust Core</p>
          </div>
          <div className="bg-white/[0.02] border border-white/10 rounded-xl p-4">
            <p className="font-mono text-[10px] uppercase tracking-widest text-zinc-500">Event Mesh</p>
            <p className="mt-1 font-mono text-xl sm:text-2xl font-semibold text-white">KRaft Mode</p>
            <p className="text-xs text-zinc-400 mt-1">Confluent Kafka Cluster</p>
          </div>
          <div className="bg-white/[0.02] border border-white/10 rounded-xl p-4">
            <p className="font-mono text-[10px] uppercase tracking-widest text-zinc-500">Mobile Edge</p>
            <p className="mt-1 font-mono text-xl sm:text-2xl font-semibold text-white">Offline WAL</p>
            <p className="text-xs text-zinc-400 mt-1">Encrypted Local SQLite</p>
          </div>
        </div>

      </div>

      {/* Detail Node Modal / Drawer */}
      <PegasusInfraDrawer
        node={selectedNode}
        onClose={() => setSelectedNode(null)}
      />
    </div>
  );
}
