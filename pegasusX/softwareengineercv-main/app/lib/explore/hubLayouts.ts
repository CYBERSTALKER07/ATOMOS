import type { FlowVariant } from '@/app/data/topicTypes';

export type HubVisualType = 'kpi' | 'flow' | 'fleet' | 'metrics' | 'devices' | 'none';

export type HubLayoutConfig = {
  visual: HubVisualType;
  flowVariant?: FlowVariant;
  visualTitle?: string;
  visualSubtitle?: string;
  topicGridLayout?: 'uniform' | 'masonry' | 'featured';
  showPromoInHero?: boolean;
  hidePromoBody?: boolean;
  showFleetBand?: boolean;
  heroVisual?: boolean;
  laneLabel?: string;
  laneIndex?: string;
  intro?: { eyebrow: string; title: string; body: string };
};

export const hubLayoutConfigs: Record<string, HubLayoutConfig> = {
  platform: {
    visual: 'kpi',
    topicGridLayout: 'featured',
    showPromoInHero: true,
    laneLabel: 'Control plane',
    laneIndex: '01',
    intro: {
      eyebrow: 'Platform',
      title: 'One control plane across six roles',
      body: 'Order lifecycle, payments, topology, and treasury on Cloud Spanner — with outbox fanout so every role sees the same commit.',
    },
  },
  technology: {
    visual: 'flow',
    flowVariant: 'mutatingHandler',
    topicGridLayout: 'uniform',
    laneLabel: 'Engineering',
    laneIndex: '02',
    intro: {
      eyebrow: 'Stack',
      title: 'Spanner, outbox, Redis, Kafka, WebSocket',
      body: 'Mutating handlers follow verify → validate → save → refresh → notify. Cache invalidation and Kafka stay in the same write transaction as domain state.',
    },
  },
  operations: {
    visual: 'flow',
    flowVariant: 'dispatchBoard',
    topicGridLayout: 'masonry',
    laneLabel: 'Live ops',
    laneIndex: '03',
    intro: {
      eyebrow: 'Floor',
      title: 'Dispatch, exceptions, and live telemetry',
      body: 'Visual boards, Smart Fit overflow, SHOP_CLOSED_PENDING, freeze-lock, and fiscal hard-gate — playbooks grounded in the order state machine.',
    },
  },
  capabilities: {
    visual: 'kpi',
    topicGridLayout: 'masonry',
    laneLabel: 'Features',
    laneIndex: '04',
    intro: {
      eyebrow: 'Capabilities',
      title: 'What the network runs on',
      body: 'Smarter dispatch, reliable updates, pay-at-delivery, live fleet tracking, and topology — modules that share contracts across portal and native apps.',
    },
  },
  'ai-vision': {
    visual: 'metrics',
    topicGridLayout: 'featured',
    laneLabel: 'Governed AI',
    laneIndex: '05',
    intro: {
      eyebrow: 'AI',
      title: 'Assist with deterministic fallback',
      body: 'ai-worker VRP, pre-order assist, and supplier recommendations never block the floor — every path degrades to pure deterministic engines.',
    },
  },
  'apps-deploy': {
    visual: 'devices',
    topicGridLayout: 'uniform',
    showFleetBand: true,
    showPromoInHero: true,
    laneLabel: 'Surfaces',
    laneIndex: '06',
    intro: {
      eyebrow: 'Deploy',
      title: 'Portal, mobile, and desktop per role row',
      body: 'Supplier and warehouse portals, driver Kotlin/Swift apps, retailer Tauri desktop, payload terminals — same API contracts, WS silent refresh.',
    },
  },
  roles: {
    visual: 'flow',
    flowVariant: 'roleJourney',
    topicGridLayout: 'featured',
    intro: {
      eyebrow: 'Roles',
      title: 'Six roles, one order truth',
      body: 'Supplier, warehouse, factory, driver, retailer, and payload/gate — each surface mapped from FEATURES_BY_APP_ROLE and the JWT role matrix.',
    },
  },
};

export function getHubLayoutConfig(hubId: string): HubLayoutConfig {
  return (
    hubLayoutConfigs[hubId] ?? {
      visual: 'kpi',
      topicGridLayout: 'uniform',
    }
  );
}
