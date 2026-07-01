export type PlatformVersion = 'web' | 'mobile' | 'desktop';

export type SubTopic = {
  id: string;
  title: string;
  description: string;
  businessLogic: string;
  edgeCases: string;
};

export type RoleData = {
  id: string;
  name: string;
  description: string;
  platforms: PlatformVersion[];
  subtopics: SubTopic[];
};

export const ROLES_DATA: RoleData[] = [
  {
    id: 'supplier',
    name: 'Supplier',
    description: 'Run your entire network from one place. Complete control over topology, orders, and treasury.',
    platforms: ['web', 'mobile', 'desktop'],
    subtopics: [
      {
        id: 'control-panel',
        title: 'Supplier Control Panel',
        description: 'Vetting, topology, treasury, and dispatch preview.',
        businessLogic: 'Centralized dashboard to view network-wide health, active orders, and pending approvals. Replaces fragmented spreadsheets with live data.',
        edgeCases: 'Handles offline synchronization issues and permission delegation when managers are away.'
      },
      {
        id: 'order-vetting',
        title: 'Supplier Collaboration & Order Vetting',
        description: 'Supplier approves before warehouse dispatch.',
        businessLogic: 'Ensures only validated, high-priority orders are sent to warehouses for dispatch, acting as an early risk-management filter.',
        edgeCases: 'Auto-escalation if a VIP order is stuck in vetting for more than 4 hours.'
      },
      {
        id: 'ai-recommendations',
        title: 'AI/ML Demand & Recommendations',
        description: 'Supplier ops suggestions from live data.',
        businessLogic: 'Predicts inventory shortages and suggests proactive route diversions based on historical demand spikes.',
        edgeCases: 'Fallback to manual routing if AI confidence scores drop below 80% due to anomalous data.'
      }
    ]
  },
  {
    id: 'warehouse',
    name: 'Warehouse',
    description: 'Dispatch with confidence, every morning. Smart execution and inventory management.',
    platforms: ['web', 'mobile', 'desktop'],
    subtopics: [
      {
        id: 'dispatch-assist',
        title: 'Smart Dispatch Assist',
        description: 'Match orders to trucks; warehouse always in control.',
        businessLogic: 'Provides ranked truck suggestions. The floor lead always confirms the load, preventing algorithmic hallucinations from sending empty trucks.',
        edgeCases: 'Partial dispatch commit when a truck is too small, splitting the remaining payload automatically.'
      },
      {
        id: 'inventory-optimization',
        title: 'Multi-Echelon Inventory Optimization',
        description: 'Pre-order hub, stock commitments, and atomic reservations.',
        businessLogic: 'Ensures inventory is held at the correct nodes across the supply chain to minimize stockouts while reducing holding costs.',
        edgeCases: 'Concurrent stock rejects when two warehouses attempt to claim the same pallet of high-velocity goods.'
      }
    ]
  },
  {
    id: 'retailer',
    name: 'Retailer',
    description: 'Order, pay, and track — without phone calls.',
    platforms: ['web', 'mobile', 'desktop'],
    subtopics: [
      {
        id: 'live-tracking',
        title: 'Control Tower & Live Tracking',
        description: 'Order, pay, and track — without phone calls.',
        businessLogic: 'Provides retailers with a consumer-grade tracking experience, reducing inbound support calls and providing accurate ETAs.',
        edgeCases: 'Gracefully handles GPS signal loss from trucks by falling back to the last known zone checkpoint.'
      },
      {
        id: 'assortment-planning',
        title: 'Merchandise & Assortment Planning',
        description: 'Optimize shelf space and ordering.',
        businessLogic: 'Suggests re-order quantities based on localized demand trends and shelf capacity.',
        edgeCases: 'Out-of-season item requests trigger a warning rather than a hard block.'
      }
    ]
  },
  {
    id: 'finance',
    name: 'Finance & Treasury',
    description: 'Close the books without surprises. End-to-end payment confidence.',
    platforms: ['web', 'desktop'],
    subtopics: [
      {
        id: 'post-game-analysis',
        title: 'Post-Game Analysis & Reconciliation',
        description: 'Treasury integrity and cash collection.',
        businessLogic: 'Automatically reconciles driver Cash-on-Delivery collections against expected invoices, highlighting discrepancies immediately.',
        edgeCases: 'Partial payments and disputed deliveries are quarantined in a separate ledger for manual review.'
      },
      {
        id: 's-and-op',
        title: 'Sales & Operations Planning (S&OP)',
        description: 'Future operating model alignment.',
        businessLogic: 'Aligns financial goals with operational realities, enabling scenario planning for peak seasons.',
        edgeCases: 'Handles extreme variance scenarios (e.g., natural disasters) by allowing manual override of constraints.'
      }
    ]
  },
  {
    id: 'driver',
    name: 'Driver',
    description: 'Clear routes. Simple stops. On-time delivery.',
    platforms: ['mobile'],
    subtopics: [
      {
        id: 'execution-app',
        title: 'Driver Execution App',
        description: 'Clear routes and simple stops.',
        businessLogic: 'Guides drivers through their daily manifest with turn-by-turn navigation and seamless proof-of-delivery capture.',
        edgeCases: 'Shop closed at delivery scenarios require a geo-stamped photo before allowing the driver to skip the stop.'
      }
    ]
  },
  {
    id: 'factory',
    name: 'Factory',
    description: 'Keep production and loading in sync.',
    platforms: ['web', 'mobile', 'desktop'],
    subtopics: [
      {
        id: 'factory-loading',
        title: 'Factory Loading & Supply Requests',
        description: 'Supply requests, manifest lifecycle, loading bay.',
        businessLogic: 'Directly maps real-time production output to awaiting trucks, eliminating buffer warehouse delays.',
        edgeCases: 'Handles line stoppages by automatically re-allocating arriving trucks to alternative docks.'
      }
    ]
  },
  {
    id: 'payload-gate',
    name: 'Payload / Gate',
    description: 'Gate control that keeps every load accountable.',
    platforms: ['mobile', 'desktop'],
    subtopics: [
      {
        id: 'gate-control',
        title: 'Returns & Barcode Gate Control',
        description: 'Inbound returns with accountability.',
        businessLogic: 'Scans every payload entering or leaving the facility to ensure the digital manifest matches physical reality.',
        edgeCases: 'Wrong truck sealed scenarios trigger an immediate lockdown of the gate barrier until a supervisor overrides.'
      }
    ]
  }
];
