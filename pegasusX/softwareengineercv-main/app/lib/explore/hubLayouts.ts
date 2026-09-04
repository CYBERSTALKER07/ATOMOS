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
      body: 'Order lifecycle, payments, topology, and treasury on one shared record — so every role sees the same confirmed status.',
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
      title: 'Reliable core, live sync, instant updates',
      body: 'Every change is checked, saved, then pushed live — so dashboards stay accurate under load.',
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
      title: 'Dispatch, exceptions, and live tracking',
      body: 'Visual boards, Smart Fit overflow, shop-closed handling, freeze-lock, and payment hard-gates — playbooks grounded in the order lifecycle.',
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
      body: 'Smarter dispatch, reliable updates, pay-at-delivery, live fleet tracking, and topology — modules that work the same on portal and native apps.',
    },
  },
  'ai-vision': {
    visual: 'metrics',
    topicGridLayout: 'featured',
    laneLabel: 'Governed AI',
    laneIndex: '05',
    intro: {
      eyebrow: 'AI',
      title: 'Assist with proven fallback',
      body: 'Route assist and recommendations never block the floor — if AI is slow, proven planning rules take over.',
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
      title: 'Portal, mobile, and desktop for every role',
      body: 'Supplier and warehouse portals, driver mobile apps, retailer desktop, gate terminals — same workflows, automatic live refresh.',
    },
  },
  roles: {
    visual: 'flow',
    flowVariant: 'roleJourney',
    topicGridLayout: 'featured',
    intro: {
      eyebrow: 'Roles',
      title: 'Six roles, one order truth',
      body: 'Supplier, warehouse, factory, driver, retailer, and payload/gate — each surface mapped with clear features and role-based access.',
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
