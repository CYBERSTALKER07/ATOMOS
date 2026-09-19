'use client';

import React from 'react';
import Image from 'next/image';

interface TopicImageConfig {
  src: string;
  alt: string;
  badge: string;
  metrics: string;
}

const VISUAL_LIBRARY: Record<string, TopicImageConfig> = {
  control_plane: {
    src: '/images/topics/control_plane.jpg',
    alt: 'Pegasus Mission Control Plane',
    badge: 'CONTROL PLANE // COMMAND CORE',
    metrics: 'CORE ENGINE: v4.8 · MULTI-TENANT: STRICT',
  },
  cloud_spanner: {
    src: '/images/topics/cloud_spanner.jpg',
    alt: 'Distributed Cloud Database & Shard Ledger',
    badge: 'DATABASE CLUSTER // CLOUD SPANNER',
    metrics: 'GLOBAL REPLICATION · ACID SLA: 99.999%',
  },
  event_pipeline: {
    src: '/images/topics/event_pipeline.jpg',
    alt: 'High-Throughput Realtime Event Stream Pipeline',
    badge: 'STREAM PIPELINE // KAFKA & REDIS',
    metrics: 'THROUGHPUT: 4.2M MSG/SEC · LATENCY: 3ms',
  },
  order_lifecycle: {
    src: '/images/topics/order_lifecycle.jpg',
    alt: 'State Machine Order Lifecycle Milestones',
    badge: 'STATE MACHINE // ORDER FLOW',
    metrics: 'RETRY KEYS: ACTIVE · TRANSITIONS: ATOMIC',
  },
  enterprise_desktop: {
    src: '/images/topics/enterprise_desktop.jpg',
    alt: 'Multi-Monitor Operations Command Workstation',
    badge: 'ENTERPRISE APPS // ROLE PORTAL',
    metrics: 'ROLE SCOPE: DERIVED · WEBSOCKET: CONNECTED',
  },
  fleet_radar: {
    src: '/images/topics/fleet_radar.jpg',
    alt: 'Autonomous Commercial Fleet Telemetry Radar',
    badge: 'FLEET RADAR // LIDAR GUIDANCE',
    metrics: 'UNITS ACTIVE: 1,420 · CONVOY MESH: LOCKED',
  },
  mobile_cockpit: {
    src: '/images/topics/mobile_cockpit.jpg',
    alt: 'Autonomous Driver In-Cab HUD Telematics',
    badge: 'DRIVER OS // IN-CAB HUD',
    metrics: 'LATENCY: 8ms · GPS L1/L5 · SENSORS: 60Hz',
  },
  warehouse_robotics: {
    src: '/images/topics/warehouse_robotics.jpg',
    alt: 'High-Bay Automated Robotic Fulfillment Matrix',
    badge: 'WAREHOUSE OS // AMR GRID',
    metrics: 'PICK RATE: 420/HR · SORT EFFICIENCY: 99.8%',
  },
  ai_vision: {
    src: '/images/topics/ai_vision.jpg',
    alt: 'Neural Computer Vision Cargo Inspection',
    badge: 'AI VISION // NEURAL SCAN',
    metrics: 'INSPECTION ACCURACY: 99.97% · SCAN: 120ms',
  },
  port_intermodal: {
    src: '/images/topics/port_intermodal.jpg',
    alt: 'Deep-Sea Intermodal Container Port Logistics',
    badge: 'INTERMODAL // PORT TERMINAL',
    metrics: 'CONTAINER FLOW: OPTIMAL · SYNC: 100%',
  },
  route_mesh: {
    src: '/images/topics/route_mesh.jpg',
    alt: 'Global Satellite Route Optimization Mesh',
    badge: 'ORBITAL MESH // ROUTE GRAPH',
    metrics: 'CORRIDORS: 8,400+ · RE-ROUTE TIME: < 40ms',
  },
  financial_treasury: {
    src: '/images/topics/financial_treasury.jpg',
    alt: 'Cryptographic Freight Ledger & Treasury Settlement',
    badge: 'FINANCIAL MESH // ATOMIC SETTLE',
    metrics: 'CONSENSUS: VALIDATED · AUDIT TRAIL: IMMUTABLE',
  },
};

const VISUAL_KEYS = Object.keys(VISUAL_LIBRARY);

// Explicit 1-to-1 slug mappings for every page on Pegasus
const SLUG_ROUTING: Record<string, keyof typeof VISUAL_LIBRARY> = {
  // Platform Hub & Topics
  platform: 'control_plane',
  'atomos-control-plane': 'control_plane',
  'supplier-control-plane': 'control_plane',
  'how-pegasus-works': 'enterprise_desktop',
  'order-lifecycle': 'order_lifecycle',
  'mutating-handler-contract': 'event_pipeline',
  'reliable-updates': 'event_pipeline',
  'network-topology': 'route_mesh',
  'trust-reliability': 'financial_treasury',

  // Technology Hub & Topics
  technology: 'cloud_spanner',
  'go-backend-platform': 'event_pipeline',
  'cloud-spanner': 'cloud_spanner',
  'redis-kafka': 'event_pipeline',
  'websocket-hubs': 'event_pipeline',
  'osrm-routing': 'route_mesh',
  'firebase-otp': 'mobile_cockpit',
  'next-js-surfaces': 'enterprise_desktop',
  'native-mobile-desktop': 'enterprise_desktop',

  // Apps & Deploy Hub & Topics
  'apps-deploy': 'enterprise_desktop',
  'mobile-apps': 'mobile_cockpit',
  'desktop-apps': 'enterprise_desktop',
  'web-apps': 'enterprise_desktop',
  'dispatch-fleet': 'fleet_radar',
  'payments-treasury': 'financial_treasury',
  'realtime-coordination': 'event_pipeline',
  'enterprise-rollout': 'control_plane',
  'request-demo': 'enterprise_desktop',

  // AI & Vision Hub & Topics
  'ai-vision': 'ai_vision',
  'smart-dispatch-assist': 'ai_vision',
  'ai-recommendations': 'ai_vision',
  'pulse-timeline': 'order_lifecycle',
  'explain-status-banners': 'ai_vision',
  'exception-weather-map': 'route_mesh',
  'override-impact-preview': 'control_plane',
  'future-operating-model': 'ai_vision',

  // Operations Hub & Exceptions
  operations: 'route_mesh',
  'zone-miss-handling': 'route_mesh',
  'concurrent-stock-reject': 'warehouse_robotics',
  'truck-too-small': 'fleet_radar',
  'partial-dispatch-commit': 'warehouse_robotics',
  'wrong-truck-sealed': 'fleet_radar',
  'driver-reassignment': 'mobile_cockpit',
  'shop-closed-at-delivery': 'mobile_cockpit',
  'cash-at-door-cod': 'financial_treasury',
  'returns-wrong-barcode': 'warehouse_robotics',
  'live-tracking-expectations': 'fleet_radar',

  // Capabilities Hub & Topics
  capabilities: 'route_mesh',
  'smarter-dispatch': 'route_mesh',
  'payment-confidence': 'financial_treasury',
  'live-fleet-tracking': 'fleet_radar',
  'instant-coordination': 'event_pipeline',
  'connected-network': 'route_mesh',
  'returns-barcode-gate': 'warehouse_robotics',
  'dispatch-preview': 'enterprise_desktop',

  // Roles Hub & Topics
  roles: 'enterprise_desktop',
  supplier: 'control_plane',
  warehouse: 'warehouse_robotics',
  driver: 'mobile_cockpit',
  retailer: 'enterprise_desktop',
  factory: 'warehouse_robotics',
  payload: 'warehouse_robotics',
  'payload-gate': 'warehouse_robotics',
  'order-vetting': 'financial_treasury',
  'cash-collection': 'financial_treasury',
  'role-parity-matrix': 'enterprise_desktop',

  // Solutions Hub & Topics
  solutions: 'port_intermodal',
  'dispatch-the-right-load': 'route_mesh',
  'visual-dispatch-engine': 'enterprise_desktop',
  'fleet-visibility': 'fleet_radar',
  'treasury-integrity': 'financial_treasury',
  'network-coordination': 'route_mesh',
  'warehouse-operations': 'warehouse_robotics',
  'factory-loading': 'warehouse_robotics',

  // Standalone Marketing Pages
  'global-logistics': 'port_intermodal',
  'supply-chain-software': 'control_plane',
  'logistics-automation': 'warehouse_robotics',
  markets: 'route_mesh',
  compare: 'route_mesh',
  alternatives: 'route_mesh',
  'cookie-policy': 'financial_treasury',
  'cloud-ecosystem': 'cloud_spanner',
};

export function getTopicImage(slug: string): TopicImageConfig {
  const s = (slug || '').toLowerCase().trim();

  // 1. Direct Slug Lookup
  if (SLUG_ROUTING[s]) {
    return VISUAL_LIBRARY[SLUG_ROUTING[s]];
  }

  // 2. Keyword-based Semantic Matching
  if (s.includes('driver') || s.includes('cockpit') || s.includes('telematics')) {
    return VISUAL_LIBRARY.mobile_cockpit;
  }
  if (s.includes('global') || s.includes('port') || s.includes('intermodal') || s.includes('shipping') || s.includes('ocean')) {
    return VISUAL_LIBRARY.port_intermodal;
  }
  if (s.includes('fleet') || s.includes('truck') || s.includes('transport') || s.includes('vehicle')) {
    return VISUAL_LIBRARY.fleet_radar;
  }
  if (s.includes('warehouse') || s.includes('inventory') || s.includes('fulfillment') || s.includes('barcode') || s.includes('gate')) {
    return VISUAL_LIBRARY.warehouse_robotics;
  }
  if (s.includes('ai') || s.includes('vision') || s.includes('assist') || s.includes('recommend') || s.includes('neural')) {
    return VISUAL_LIBRARY.ai_vision;
  }
  if (s.includes('payment') || s.includes('treasury') || s.includes('cash') || s.includes('settle') || s.includes('audit')) {
    return VISUAL_LIBRARY.financial_treasury;
  }
  if (s.includes('spanner') || s.includes('database') || s.includes('cloud') || s.includes('storage')) {
    return VISUAL_LIBRARY.cloud_spanner;
  }
  if (s.includes('kafka') || s.includes('stream') || s.includes('pipeline') || s.includes('websocket') || s.includes('realtime')) {
    return VISUAL_LIBRARY.event_pipeline;
  }
  if (s.includes('order') || s.includes('lifecycle') || s.includes('timeline')) {
    return VISUAL_LIBRARY.order_lifecycle;
  }
  if (s.includes('desktop') || s.includes('portal') || s.includes('app') || s.includes('surface')) {
    return VISUAL_LIBRARY.enterprise_desktop;
  }
  if (s.includes('route') || s.includes('dispatch') || s.includes('network') || s.includes('topology') || s.includes('market')) {
    return VISUAL_LIBRARY.route_mesh;
  }

  // 3. Deterministic Hash Rotation Fallback (Never show same fallback consecutively)
  let hash = 0;
  for (let i = 0; i < s.length; i++) {
    hash = (hash << 5) - hash + s.charCodeAt(i);
    hash |= 0;
  }
  const keyIndex = Math.abs(hash) % VISUAL_KEYS.length;
  const fallbackKey = VISUAL_KEYS[keyIndex];

  return VISUAL_LIBRARY[fallbackKey] || VISUAL_LIBRARY.control_plane;
}

export default function TopicCanvas({ slug }: { slug: string }) {
  const visual = getTopicImage(slug);

  return (
    <div className="relative w-full h-full min-h-[420px] lg:min-h-[640px] xl:min-h-[700px] bg-[#030303] overflow-hidden flex items-center justify-center group select-none">
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
