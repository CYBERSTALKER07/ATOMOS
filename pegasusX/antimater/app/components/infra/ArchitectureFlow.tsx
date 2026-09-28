'use client';

import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { INFRA_NODES } from './infraData';
import type { InfraNode } from './types';
import { 
  Database, 
  Server, 
  Cpu, 
  Smartphone, 
  Globe, 
  ShieldCheck,
  Activity,
  ArrowRight,
  Code
} from 'lucide-react';

const iconMap: Record<string, React.ReactNode> = {
  'data-core': <Database className="w-5 h-5 text-emerald-400" />,
  'backend-go': <Server className="w-5 h-5 text-emerald-400" />,
  'optimizer-rust': <Cpu className="w-5 h-5 text-purple-400" />,
  'edge-portals': <Globe className="w-5 h-5 text-blue-400" />,
  'gate-security': <ShieldCheck className="w-5 h-5 text-amber-400" />,
  'mobile-fleet': <Smartphone className="w-5 h-5 text-emerald-400" />
};

const ArchitectureFlow: React.FC = () => {
  const [activeNode, setActiveNode] = useState<InfraNode | null>(null);

  // Group nodes logically
  const fleetNode = INFRA_NODES.find(n => n.id === 'mobile-fleet');
  const gateNode = INFRA_NODES.find(n => n.id === 'gate-security');
  
  const portalNode = INFRA_NODES.find(n => n.id === 'edge-portals');
  
  const rustNode = INFRA_NODES.find(n => n.id === 'optimizer-rust');
  const goNode = INFRA_NODES.find(n => n.id === 'backend-go');
  
  const dataNode = INFRA_NODES.find(n => n.id === 'data-core');

  const NodeCard = ({ node }: { node?: InfraNode }) => {
    if (!node) return null;
    const isActive = activeNode?.id === node.id;
    return (
      <div 
        onClick={() => setActiveNode(isActive ? null : node)}
        className={`relative z-10 p-5 rounded-2xl border transition-all duration-300 cursor-pointer backdrop-blur-md overflow-hidden group
          ${isActive 
            ? 'bg-emerald-950/40 border-emerald-500/50 shadow-[0_0_30px_rgba(16,185,129,0.15)]' 
            : 'bg-black/60 border-white/10 hover:border-emerald-500/30 hover:bg-black/80'
          }`}
      >
        <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-emerald-500/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
        
        <div className="flex items-start gap-4">
          <div className={`p-3 rounded-xl border ${isActive ? 'bg-emerald-900/50 border-emerald-500/30' : 'bg-white/5 border-white/10'}`}>
            {iconMap[node.id] || <Activity className="w-5 h-5 text-zinc-400" />}
          </div>
          <div>
            <h4 className="text-white font-medium tracking-wide text-sm">{node.name}</h4>
            <p className="text-zinc-500 text-xs mt-1">{node.subtitle}</p>
          </div>
        </div>

        {/* Expanded Details */}
        <motion.div 
          initial={false}
          animate={{ height: isActive ? 'auto' : 0, opacity: isActive ? 1 : 0 }}
          className="overflow-hidden"
        >
          <div className="pt-5 mt-5 border-t border-white/10">
            <p className="text-zinc-300 text-sm leading-relaxed mb-4">{node.description}</p>
            
            <div className="grid grid-cols-2 gap-3 mb-4">
              {node.metrics?.slice(0, 2).map((m, idx) => (
                <div key={idx} className="bg-white/5 rounded-lg p-3 border border-white/5">
                  <div className="text-emerald-400 font-mono text-lg">{m.value}<span className="text-xs text-emerald-600 ml-1">{m.unit}</span></div>
                  <div className="text-[10px] text-zinc-500 uppercase tracking-wider mt-1">{m.label}</div>
                </div>
              ))}
            </div>

            <div className="flex flex-wrap gap-2">
              {node.tech.map((t, idx) => (
                <span key={idx} className="px-2.5 py-1 rounded-full bg-white/5 border border-white/10 text-[10px] text-zinc-300 font-mono">
                  {t}
                </span>
              ))}
            </div>
          </div>
        </motion.div>
      </div>
    );
  };

  return (
    <div className="w-full min-h-screen bg-black relative flex items-center justify-center py-24 overflow-hidden">
      {/* Generated Cyberpunk Background */}
      <div 
        className="absolute inset-0 z-0 opacity-40 bg-cover bg-center"
        style={{ backgroundImage: 'url(/infra_bg.jpg)' }}
      />
      
      {/* Fade Gradients */}
      <div className="absolute inset-0 z-0 bg-gradient-to-b from-black via-transparent to-black" />
      <div className="absolute inset-0 z-0 bg-gradient-to-r from-black via-transparent to-black" />

      <div className="relative z-10 w-full max-w-[1600px] mx-auto px-6 lg:px-12">
        
        {/* Title */}
        <div className="mb-16 text-center max-w-2xl mx-auto">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-[10px] font-mono tracking-widest uppercase mb-6">
            <Code className="w-3 h-3" /> System Architecture
          </div>
          <h2 className="text-4xl md:text-5xl font-light text-white tracking-tight mb-4">
            Distributed <span className="font-medium">Topology</span>
          </h2>
          <p className="text-zinc-400">
            Interactive overview of our multi-region event mesh, edge portals, and strict ACID transaction nodes. Click any module to inspect real-time telemetry and tech stacks.
          </p>
        </div>

        {/* Diagram Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8 relative">
          
          {/* Connector Lines (Desktop Only) */}
          <div className="hidden lg:block absolute top-1/2 left-0 w-full h-0.5 bg-gradient-to-r from-transparent via-emerald-500/20 to-transparent -translate-y-1/2 z-0" />

          {/* Column 1: Fleet & Edge */}
          <div className="lg:col-span-3 flex flex-col gap-6 justify-center">
            <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest mb-2 px-2 border-l border-emerald-500/50">Edge Surfaces</div>
            <NodeCard node={fleetNode} />
            <NodeCard node={gateNode} />
          </div>

          {/* Column 2: Portals */}
          <div className="lg:col-span-3 flex flex-col gap-6 justify-center">
            <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest mb-2 px-2 border-l border-blue-500/50">Web Portals</div>
            <NodeCard node={portalNode} />
          </div>

          {/* Column 3: Compute */}
          <div className="lg:col-span-3 flex flex-col gap-6 justify-center">
            <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest mb-2 px-2 border-l border-purple-500/50">Service Plane</div>
            <NodeCard node={goNode} />
            <NodeCard node={rustNode} />
          </div>

          {/* Column 4: Data Core */}
          <div className="lg:col-span-3 flex flex-col gap-6 justify-center">
            <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest mb-2 px-2 border-l border-emerald-500/50">Data Plane</div>
            <NodeCard node={dataNode} />
          </div>

        </div>
      </div>
    </div>
  );
};

export default ArchitectureFlow;
