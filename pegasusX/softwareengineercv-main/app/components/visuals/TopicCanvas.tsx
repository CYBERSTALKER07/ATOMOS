'use client';

import React from 'react';
import Image from 'next/image';

interface TopicImageConfig {
  src: string;
  alt: string;
  badge: string;
  metrics: string;
}

export function getTopicImage(slug: string): TopicImageConfig {
  const s = (slug || '').toLowerCase();

  // Mobile Cockpit / Driver telemetry
  if (
    s.includes('driver') ||
    s === 'mobile-apps' ||
    s.includes('cockpit') ||
    s.includes('telematics')
  ) {
    return {
      src: '/images/topics/mobile_cockpit.jpg',
      alt: 'Autonomous Vehicle Telemetry & Driver Systems',
      badge: 'DRIVER OS // IN-CAB HUD',
      metrics: 'LATENCY: 8ms · GPS L1/L5 · TELEMETRY: ACTIVE',
    };
  }

  // Intermodal, Ports & Global Shipping Logistics
  if (
    s.includes('global') ||
    s.includes('port') ||
    s.includes('intermodal') ||
    s.includes('shipping') ||
    s.includes('freight-visibility') ||
    s.includes('ocean') ||
    s.includes('container')
  ) {
    return {
      src: '/images/topics/port_intermodal.jpg',
      alt: 'Deep-Sea Intermodal Container Port Logistics',
      badge: 'INTERMODAL // PORT AUTOMATION',
      metrics: 'CONTAINER FLOW: OPTIMAL · GANTRY SPEED: SYNC',
    };
  }

  // Fleet & Highway Telemetry Radar
  if (
    s.includes('fleet') ||
    s.includes('tracking') ||
    s.includes('truck') ||
    s.includes('transport') ||
    s.includes('vehicle') ||
    s.includes('telemetry') ||
    s.includes('weather')
  ) {
    return {
      src: '/images/topics/fleet_radar.jpg',
      alt: 'Autonomous Fleet Radar & Highway Guidance',
      badge: 'FLEET RADAR // LIDAR GUIDANCE',
      metrics: 'UNITS ACTIVE: 1,420 · CONVOY MESH: LOCKED',
    };
  }

  // Automated Warehouse & Inventory Robotics
  if (
    s.includes('warehouse') ||
    s.includes('fulfillment') ||
    s.includes('inventory') ||
    s.includes('payload') ||
    s.includes('stock') ||
    s.includes('gate') ||
    s.includes('barcode') ||
    s.includes('factory')
  ) {
    return {
      src: '/images/topics/warehouse_robotics.jpg',
      alt: 'High-Bay Automated Warehouse Robotics',
      badge: 'WAREHOUSE OS // AMR GRID',
      metrics: 'PICK RATE: 420/HR · SORT EFFICIENCY: 99.8%',
    };
  }

  // AI & Vision & Neural Ops
  if (
    s.includes('ai') ||
    s.includes('vision') ||
    s.includes('recommend') ||
    s.includes('assist') ||
    s.includes('predict') ||
    s.includes('neural') ||
    s.includes('future') ||
    s.includes('forecast')
  ) {
    return {
      src: '/images/topics/ai_vision.jpg',
      alt: 'Neural Computer Vision Cargo Inspection',
      badge: 'AI VISION // NEURAL SCAN',
      metrics: 'INSPECTION ACCURACY: 99.97% · SCAN TIME: 120ms',
    };
  }

  // Treasury, Financial, Ledger, Audit, Payments
  if (
    s.includes('payment') ||
    s.includes('treasury') ||
    s.includes('cash') ||
    s.includes('audit') ||
    s.includes('trust') ||
    s.includes('settle') ||
    s.includes('confidence')
  ) {
    return {
      src: '/images/topics/financial_treasury.jpg',
      alt: 'Cryptographic Freight Ledger & Treasury Clearing',
      badge: 'FINANCIAL MESH // ATOMIC SETTLEMENT',
      metrics: 'CONSENSUS: VALIDATED · GAS: ZERO · RECON: 100%',
    };
  }

  // Route Mesh & Network Topology & Dispatch
  if (
    s.includes('route') ||
    s.includes('dispatch') ||
    s.includes('network') ||
    s.includes('topology') ||
    s.includes('zone') ||
    s.includes('map')
  ) {
    return {
      src: '/images/topics/route_mesh.jpg',
      alt: 'Global Satellite Route Optimization Mesh',
      badge: 'ORBITAL MESH // ROUTE GRAPH',
      metrics: 'CORRIDORS: 8,400+ · RE-ROUTE TIME: < 40ms',
    };
  }

  // Default: Control Plane
  return {
    src: '/images/topics/control_plane.jpg',
    alt: 'Pegasus Mission Control Plane',
    badge: 'CONTROL PLANE // COMMAND SYSTEM',
    metrics: 'CORE ENGINE: v4.8 · MULTI-TENANT: STRICT',
  };
}

export default function TopicCanvas({ slug }: { slug: string }) {
  const visual = getTopicImage(slug);

  return (
    <div className="relative w-full h-full min-h-[400px] lg:min-h-[640px] xl:min-h-[700px] bg-[#030303] overflow-hidden flex items-center justify-center group select-none">
      {/* Background Generated Image */}
      <div className="absolute inset-0 transition-transform duration-1000 ease-out group-hover:scale-105">
        <Image
          src={visual.src}
          alt={visual.alt}
          fill
          priority
          sizes="(max-width: 1024px) 100vw, 50vw"
          className="object-cover object-center"
        />
      </div>

      {/* Cinematic Vignettes & Edges */}
      <div className="absolute inset-0 bg-gradient-to-t from-[#000000] via-black/20 to-black/40 pointer-events-none" />
      <div className="absolute inset-0 bg-gradient-to-r from-[#000000] via-transparent to-transparent pointer-events-none" />
      <div className="absolute inset-0 bg-black/20 pointer-events-none" />

      {/* Subtle Grid Lines Overlay for Brutalist Aesthetic */}
      <div
        className="absolute inset-0 opacity-[0.07] pointer-events-none bg-[radial-gradient(#fff_1px,transparent_1px)]"
        style={{ backgroundSize: '24px 24px' }}
      />

      {/* Tech HUD Metadata Badges */}
      <div className="absolute top-6 left-6 z-20 flex items-center space-x-2">
        <div className="w-1.5 h-1.5 bg-white animate-pulse" />
        <span className="font-mono text-[10px] uppercase tracking-[0.25em] text-white/90 bg-black/70 px-2.5 py-1 backdrop-blur-md border border-white/15">
          {visual.badge}
        </span>
      </div>

      <div className="absolute bottom-6 right-6 z-20">
        <span className="font-mono text-[9px] uppercase tracking-[0.2em] text-white/70 bg-black/70 px-2.5 py-1 backdrop-blur-md border border-white/15">
          {visual.metrics}
        </span>
      </div>

      {/* Corner Tech Brackets */}
      <div className="absolute top-4 right-4 w-3 h-3 border-t border-r border-white/30 pointer-events-none" />
      <div className="absolute bottom-4 left-4 w-3 h-3 border-b border-l border-white/30 pointer-events-none" />
    </div>
  );
}
