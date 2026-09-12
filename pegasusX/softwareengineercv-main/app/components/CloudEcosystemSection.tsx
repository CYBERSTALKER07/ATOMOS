'use client';

import React, { useState } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';
import ChamferButton from './ChamferButton';
import { useLanguage } from '../context/LanguageContext';
import {
  SiGooglecloud,
  SiApachekafka,
  SiRedis,
  SiKubernetes,
  SiGooglebigquery,
} from 'react-icons/si';

type TechLayer = 'all' | 'spanner' | 'gke' | 'kafka' | 'redis';

interface LayerMeta {
  id: TechLayer;
  name: string;
  badge: string;
  metric: string;
  metricLabel: string;
  description: string;
  color: string;
  icon: React.ReactNode;
}

const LAYERS: LayerMeta[] = [
  {
    id: 'spanner',
    name: 'Cloud Spanner',
    badge: 'GLOBAL DISTRIBUTED SQL',
    metric: '99.999%',
    metricLabel: 'SLA Availability',
    description: 'Multi-tenant database with strict serializable transactions, external consistency, and zero maintenance windows.',
    color: '#3B82F6',
    icon: <SiGooglecloud className="text-xl" />,
  },
  {
    id: 'gke',
    name: 'GKE Enterprise',
    badge: 'KUBERNETES AUTOPILOT',
    metric: '< 150ms',
    metricLabel: 'Pod Auto-Scaling',
    description: 'Multi-region container orchestration running stateless Go backends with zero-downtime canary rollouts.',
    color: '#10B981',
    icon: <SiKubernetes className="text-xl" />,
  },
  {
    id: 'kafka',
    name: 'Kafka + Outbox',
    badge: 'EVENT STREAMING BUS',
    metric: '0 Msg',
    metricLabel: 'Dropped Event Rate',
    description: 'Spanner transactional outbox tables paired with Apache Kafka workers for guaranteed at-least-once cross-role delivery.',
    color: '#FF7A1A',
    icon: <SiApachekafka className="text-xl" />,
  },
  {
    id: 'redis',
    name: 'Redis 7 Cluster',
    badge: 'IN-MEMORY PUB/SUB',
    metric: '< 1ms',
    metricLabel: 'Telemetry Read Latency',
    description: 'High-throughput cache for live vehicle GPS coordinates, active WebSocket connection presence, and fast rate limiting.',
    color: '#DC2626',
    icon: <SiRedis className="text-xl" />,
  },
];

export default function CloudEcosystemSection() {
  const { t } = useLanguage();
  const [activeLayer, setActiveLayer] = useState<TechLayer>('all');

  return (
    <PageSection id="cloud-ecosystem" className="border-t border-white/10 bg-[#09090B] text-white overflow-hidden relative">
      {/* ── Background Subtle Isometric Grid ── */}
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.07]"
        style={{
          backgroundImage: `
            linear-gradient(30deg, #10B981 12%, transparent 12.5%, transparent 87%, #10B981 87.5%, #10B981),
            linear-gradient(150deg, #10B981 12%, transparent 12.5%, transparent 87%, #10B981 87.5%, #10B981),
            linear-gradient(30deg, #10B981 12%, transparent 12.5%, transparent 87%, #10B981 87.5%, #10B981),
            linear-gradient(150deg, #10B981 12%, transparent 12.5%, transparent 87%, #10B981 87.5%, #10B981),
            linear-gradient(60deg, #10B98177 25%, transparent 25.5%, transparent 75%, #10B98177 75%, #10B98177),
            linear-gradient(60deg, #10B98177 25%, transparent 25.5%, transparent 75%, #10B98177 75%, #10B98177)
          `,
          backgroundSize: '80px 140px',
          backgroundPosition: '0 0, 0 0, 40px 70px, 40px 70px, 0 0, 40px 70px',
        }}
      />

      {/* ── Section Header ── */}
      <div className="relative z-10 mb-10 flex flex-col items-center text-center max-w-3xl mx-auto">
        <SectionHeader
          align="center"
          eyebrow={t('cloud_eco_eyebrow', 'Cloud ecosystem')}
          title={t('cloud_eco_title', 'Fully backed by the best of Google Cloud')}
          description={t(
            'cloud_eco_desc',
            'Databases, servers, messaging, and delivery — Spanner, Kafka, Redis, GKE, and the GCP services that keep every role online.',
          )}
          className="mb-6"
        />

        {/* Tactical Layer Filter Chips */}
        <div className="flex flex-wrap items-center justify-center gap-2 font-mono text-xs">
          <button
            type="button"
            onClick={() => setActiveLayer('all')}
            className={`px-3.5 py-1.5 rounded-full border transition-all ${
              activeLayer === 'all'
                ? 'border-emerald-500 bg-emerald-500/15 text-emerald-400 font-semibold'
                : 'border-white/10 bg-white/5 text-white/60 hover:text-white hover:border-white/20'
            }`}
          >
            ● ALL SYSTEMS
          </button>
          {LAYERS.map((layer) => (
            <button
              key={layer.id}
              type="button"
              onClick={() => setActiveLayer(layer.id)}
              className={`flex items-center space-x-2 px-3.5 py-1.5 rounded-full border transition-all ${
                activeLayer === layer.id
                  ? 'border-white/40 bg-white/15 text-white font-semibold shadow-lg'
                  : 'border-white/10 bg-white/5 text-white/60 hover:text-white hover:border-white/20'
              }`}
            >
              <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: layer.color }} />
              <span>{layer.name}</span>
            </button>
          ))}
        </div>
      </div>

      {/* ── The Centerpiece: Isometric Architecture Control Stage ── */}
      <div className="relative z-10 mb-12 max-w-5xl mx-auto">
        <div className="relative overflow-hidden rounded-3xl border border-white/15 bg-[#09090B] p-4 sm:p-8 md:p-12 shadow-2xl">
          {/* Radial Emerald & Cobalt Backdrop Glow */}
          <div className="absolute inset-0 pointer-events-none opacity-60 bg-[radial-gradient(circle_at_center,_rgba(16,185,129,0.18)_0%,_rgba(59,130,246,0.08)_45%,_transparent_75%)]" />

          {/* Top Stage Telemetry Bar */}
          <div className="relative z-20 flex flex-wrap items-center justify-between border-b border-white/10 pb-4 mb-6 font-mono text-[11px] text-white/60">
            <div className="flex items-center space-x-3">
              <span className="flex h-2.5 w-2.5 rounded-full bg-emerald-400 animate-pulse" />
              <span className="font-bold text-white uppercase tracking-wider">
                GCP REGION: ASIA-NORTHEAST1 / GLOBAL MULTI-REGION
              </span>
            </div>
            <div className="flex items-center space-x-4">
              <span className="hidden sm:inline text-emerald-400 font-semibold">
                ACTIVE TENANTS: 1,420+
              </span>
              <span className="px-2 py-0.5 rounded bg-emerald-500/10 border border-emerald-500/30 text-emerald-300">
                ENCRYPTED STORAGE: ACTIVE
              </span>
            </div>
          </div>

          {/* Central Isometric Graphic Container */}
          <div className="relative z-10 flex items-center justify-center my-2 sm:my-6">
            <div className="relative w-full max-w-[700px] aspect-square flex items-center justify-center">
              {/* High-Resolution Isometric Dark Artwork */}
              <Image
                src="/google_cloud_isometric_dark.png"
                alt="Google Cloud Platform Isometric Architecture"
                width={736}
                height={736}
                priority
                className="w-full h-full object-contain filter contrast-105 brightness-105 select-none"
              />

              {/* Interactive Hotspot 1: Center Cloud Core */}
              <div
                className={`absolute top-[46%] left-[48%] -translate-x-1/2 -translate-y-1/2 transition-transform duration-300 ${
                  activeLayer === 'all' || activeLayer === 'spanner' ? 'scale-100 opacity-100' : 'scale-90 opacity-40'
                }`}
              >
                <div className="relative group cursor-pointer">
                  <span className="absolute -inset-2 rounded-full bg-blue-500/30 blur-sm animate-ping" />
                  <span className="relative flex h-4 w-4 rounded-full border-2 border-white bg-blue-500 shadow-[0_0_12px_#3b82f6]" />
                  {/* Tooltip */}
                  <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:flex flex-col items-center pointer-events-none whitespace-nowrap z-30">
                    <div className="px-2.5 py-1 rounded bg-black/90 border border-blue-500/40 text-[10px] font-mono text-blue-300 shadow-xl">
                      CLOUD SPANNER · MULTI-REGION CORE
                    </div>
                  </div>
                </div>
              </div>

              {/* Interactive Hotspot 2: Left Client Monolith (GKE) */}
              <div
                className={`absolute top-[38%] left-[32%] -translate-x-1/2 -translate-y-1/2 transition-transform duration-300 ${
                  activeLayer === 'all' || activeLayer === 'gke' ? 'scale-100 opacity-100' : 'scale-90 opacity-40'
                }`}
              >
                <div className="relative group cursor-pointer">
                  <span className="relative flex h-3.5 w-3.5 rounded-full border-2 border-white bg-emerald-500 shadow-[0_0_10px_#10b981]" />
                  <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:flex flex-col items-center pointer-events-none whitespace-nowrap z-30">
                    <div className="px-2.5 py-1 rounded bg-black/90 border border-emerald-500/40 text-[10px] font-mono text-emerald-300 shadow-xl">
                      GKE AUTOPILOT · INGRESS TOWER
                    </div>
                  </div>
                </div>
              </div>

              {/* Interactive Hotspot 3: Right Client Monolith (Kafka/Redis) */}
              <div
                className={`absolute top-[44%] right-[28%] -translate-x-1/2 -translate-y-1/2 transition-transform duration-300 ${
                  activeLayer === 'all' || activeLayer === 'kafka' || activeLayer === 'redis' ? 'scale-100 opacity-100' : 'scale-90 opacity-40'
                }`}
              >
                <div className="relative group cursor-pointer">
                  <span className="relative flex h-3.5 w-3.5 rounded-full border-2 border-white bg-orange-500 shadow-[0_0_10px_#f97316]" />
                  <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:flex flex-col items-center pointer-events-none whitespace-nowrap z-30">
                    <div className="px-2.5 py-1 rounded bg-black/90 border border-orange-500/40 text-[10px] font-mono text-orange-300 shadow-xl">
                      KAFKA OUTBOX & REDIS 7 CLUSTER
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Bottom Stage Status Row */}
          <div className="relative z-20 pt-4 border-t border-white/10 flex flex-wrap items-center justify-between gap-4 font-mono text-xs">
            <div className="flex items-center space-x-4">
              <span className="text-white/60">DATA INTEGRITY:</span>
              <span className="text-[#E2FD52] font-semibold">100% EXTERNAL CONSISTENCY</span>
            </div>
            <div className="flex items-center space-x-6 text-[11px] text-white/50">
              <span>● SPANNER RW ATOMIC</span>
              <span>● ZERO-DATA-LOSS OUTBOX</span>
              <span>● TLS 1.3 ENCRYPTED</span>
            </div>
          </div>
        </div>
      </div>

      {/* ── 4 Core Pillars Strip ── */}
      <div className="relative z-10 mb-12 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 max-w-5xl mx-auto">
        {LAYERS.map((layer) => {
          const isSelected = activeLayer === layer.id;
          return (
            <div
              key={layer.id}
              onClick={() => setActiveLayer(layer.id)}
              className={`cursor-pointer rounded-2xl border p-5 transition-all duration-200 ${
                isSelected
                  ? 'border-white/40 bg-[#16161F] shadow-lg'
                  : 'border-white/10 bg-[#111116] hover:border-white/25 hover:bg-[#14141A]'
              }`}
            >
              <div className="flex items-center justify-between mb-3">
                <div
                  className="w-9 h-9 rounded-xl flex items-center justify-center bg-white/5 border border-white/10"
                  style={{ color: layer.color }}
                >
                  {layer.icon}
                </div>
                <span className="font-mono text-[10px] uppercase tracking-wider text-white/50">
                  {layer.badge}
                </span>
              </div>
              <h4 className="text-base font-semibold text-white mb-1">{layer.name}</h4>
              <div className="flex items-baseline space-x-2 mb-2 font-mono">
                <span className="text-lg font-bold text-white tabular-nums">{layer.metric}</span>
                <span className="text-[10px] text-white/40 uppercase">{layer.metricLabel}</span>
              </div>
              <p className="text-xs text-white/60 leading-relaxed font-sans">{layer.description}</p>
            </div>
          );
        })}
      </div>

      {/* ── Bottom CTAs ── */}
      <div className="relative z-10 flex flex-col sm:flex-row items-center justify-center gap-3">
        <ChamferButton href="/cloud-ecosystem" variant="fill">
          {t('cloud_eco_cta_page', 'Explore full stack')}
        </ChamferButton>
        <ChamferButton href="/technology" variant="ghost">
          {t('cloud_eco_cta_tech', 'Technology hub')}
        </ChamferButton>
      </div>
    </PageSection>
  );
}
