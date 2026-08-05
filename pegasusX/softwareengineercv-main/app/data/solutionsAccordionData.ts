import { SITE_IMAGES, SOLUTIONS_DEFAULT_IMAGE } from '@/app/lib/siteAssets';

export interface AccordionUseCase {
  title: string;
  href: string;
  slug?: string;
  image?: string;
}

export interface AccordionSolution {
  id: string;
  numberLabel: string; // e.g. "[1]"
  title: string;
  overview: string;
  solutionHref: string;
  useCases: AccordionUseCase[];
}

export const SOLUTIONS_ACCORDION_DATA: AccordionSolution[] = [
  {
    id: 'supplier',
    numberLabel: '[1]',
    title: 'Supplier',
    overview:
      'Collaborate seamlessly with your suppliers. Manage orders, predict demand, and maintain total control over your supply network.',
    solutionHref: '/roles/supplier',
    useCases: [
      { 
        title: 'Supplier Control Panel', 
        href: '/solutions/supplier-control-panel',
        slug: 'supplier-control-panel',
        image: SITE_IMAGES.operationsTeam
      },
      { 
        title: 'Supplier Collaboration & Order Vetting', 
        href: '/solutions/supplier-collaboration',
        slug: 'supplier-collaboration',
        image: SITE_IMAGES.multimodalHub,
      },
      { 
        title: 'AI/ML Demand & Recommendations', 
        href: '/solutions/ai-ml-demand',
        slug: 'ai-ml-demand',
        image: SITE_IMAGES.terminalArchitecture,
      },
    ],
  },
  {
    id: 'warehouse',
    numberLabel: '[2]',
    title: 'Warehouse',
    overview:
      'Optimize your dispatch processes and manage inventory efficiently across multiple echelons.',
    solutionHref: '/roles/warehouse',
    useCases: [
      { 
        title: 'Smart Dispatch Assist', 
        href: '/solutions/smart-dispatch-assist',
        slug: 'smart-dispatch-assist',
        image: SITE_IMAGES.warehouseAutomation,
      },
      { 
        title: 'Multi-Echelon Inventory Optimization', 
        href: '/solutions/multi-echelon-inventory',
        slug: 'multi-echelon-inventory',
        image: SITE_IMAGES.warehouseWireframe,
      },
    ],
  },
  {
    id: 'retailer',
    numberLabel: '[3]',
    title: 'Retailer',
    overview:
      'Gain visibility into live tracking and plan your merchandise assortment for maximum retail efficiency.',
    solutionHref: '/roles/retailer',
    useCases: [
      { 
        title: 'Control Tower & Live Tracking', 
        href: '/solutions/control-tower',
        slug: 'control-tower',
        image: SITE_IMAGES.logisticsPlatformUi,
      },
      { 
        title: 'Merchandise & Assortment Planning', 
        href: '/solutions/merchandise-planning',
        slug: 'merchandise-planning',
        image: SITE_IMAGES.truckTerminal,
      },
    ],
  },
  {
    id: 'finance-treasury',
    numberLabel: '[4]',
    title: 'Finance & Treasury',
    overview:
      'Reconcile payments, analyze post-game logistics data, and align sales with operational planning.',
    solutionHref: '/roles/finance',
    useCases: [
      { 
        title: 'Post-Game Analysis & Reconciliation', 
        href: '/solutions/post-game-analysis',
        slug: 'post-game-analysis',
        image: SITE_IMAGES.pegasusContainer,
      },
      { 
        title: 'Sales & Operations Planning (S&OP)', 
        href: '/solutions/sales-operations-planning',
        slug: 'sales-operations-planning',
        image: SITE_IMAGES.portCraneScene,
      },
    ],
  },
  {
    id: 'driver',
    numberLabel: '[5]',
    title: 'Driver',
    overview:
      'Empower drivers with execution apps for seamless navigation, task completion, and proof-of-delivery.',
    solutionHref: '/roles/driver',
    useCases: [
      { 
        title: 'Driver Execution App', 
        href: '/solutions/driver-execution-app',
        slug: 'driver-execution-app',
        image: SITE_IMAGES.truckTerminal,
      },
    ],
  },
  {
    id: 'factory',
    numberLabel: '[6]',
    title: 'Factory',
    overview:
      'Coordinate factory loading and streamline supply requests directly tied to transport availability.',
    solutionHref: '/roles/factory',
    useCases: [
      { 
        title: 'Factory Loading & Supply Requests', 
        href: '/solutions/factory-loading',
        slug: 'factory-loading',
        image: SITE_IMAGES.warehouseAutomation,
      },
    ],
  },
  {
    id: 'payload-gate',
    numberLabel: '[7]',
    title: 'Payload / Gate',
    overview:
      'Secure facility entries and exits with returns handling and seamless barcode gate control.',
    solutionHref: '/roles/payload-gate',
    useCases: [
      { 
        title: 'Returns & Barcode Gate Control', 
        href: '/solutions/returns-barcode-gate',
        slug: 'returns-barcode-gate',
        image: SITE_IMAGES.deliveryDrone,
      },
    ],
  },
];
