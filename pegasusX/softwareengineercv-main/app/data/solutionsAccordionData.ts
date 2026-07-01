export interface AccordionUseCase {
  title: string;
  href: string;
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
    id: 'integrated-business-planning',
    numberLabel: '[1]',
    title: 'Integrated Business Planning',
    overview:
      'Connect finance, marketing, sales, and supply chain data on a single, live online platform to mitigate risks and capitalize on opportunities.',
    solutionHref: '/solutions/dispatch-the-right-load',
    useCases: [
      { title: 'Annual Operating Plan (AOP) & Budgeting', href: '/solutions/dispatch-the-right-load' },
      { title: 'Digital IBP', href: '/solutions/dispatch-the-right-load' },
      { title: 'Long Range Planning (LRP)', href: '/solutions/dispatch-the-right-load' },
      { title: 'Sales & Operations Planning (S&OP)', href: '/solutions/dispatch-the-right-load' },
    ],
  },
  {
    id: 'demand-planning',
    numberLabel: '[2]',
    title: 'Demand Planning',
    overview:
      'Improve forecast accuracy by capturing real-time market signals and executing agile, responsive demand plans.',
    solutionHref: '/solutions/warehouse-operations',
    useCases: [
      { title: 'AI/ML Forecasting', href: '/solutions/warehouse-operations' },
      { title: 'New Product Introduction (NPI) Planning', href: '/solutions/warehouse-operations' },
      { title: 'Demand Sensing', href: '/solutions/warehouse-operations' },
      { title: 'Promotions Planning', href: '/solutions/warehouse-operations' },
    ],
  },
  {
    id: 'supply-chain-planning',
    numberLabel: '[3]',
    title: 'Supply Chain Planning',
    overview:
      'Model your end-to-end network digitally to evaluate scenarios, constrain plans, and execute with absolute clarity.',
    solutionHref: '/solutions/visual-dispatch-engine',
    useCases: [
      { title: 'Supply Planning', href: '/solutions/visual-dispatch-engine' },
      { title: 'Production Planning & Scheduling', href: '/solutions/factory-loading' },
      { title: 'Inventory Optimization', href: '/solutions/network-coordination' },
    ],
  },
  {
    id: 'supplier-collaboration',
    numberLabel: '[4]',
    title: 'Supplier Collaboration and Risk Management',
    overview:
      'Break down silos with your suppliers. Share live capacity data, commit collaboratively, and handle exceptions in real time.',
    solutionHref: '/platform/supplier-control-plane',
    useCases: [
      { title: 'Supplier Control Tower', href: '/platform/supplier-control-plane' },
      { title: 'Purchase Order Collaboration', href: '/platform/supplier-control-plane' },
      { title: 'Multi-Tier Risk Management', href: '/platform/supplier-control-plane' },
    ],
  },
  {
    id: 'merchandise-planning',
    numberLabel: '[5]',
    title: 'Merchandise Planning',
    overview:
      'Unify assortment, financial planning, and allocation to ensure the right products hit the right locations.',
    solutionHref: '/solutions/fleet-visibility',
    useCases: [
      { title: 'Merchandise Financial Planning', href: '/solutions/fleet-visibility' },
      { title: 'Assortment Planning', href: '/solutions/fleet-visibility' },
      { title: 'Allocation & Replenishment', href: '/solutions/fleet-visibility' },
    ],
  },
  {
    id: 'revenue-growth',
    numberLabel: '[6]',
    title: 'Revenue Growth Management',
    overview:
      'Align commercial strategies across channels, ensuring pricing and promotions deliver optimal P&L results.',
    solutionHref: '/solutions/payment-confidence',
    useCases: [
      { title: 'Trade Promotion Management', href: '/solutions/payment-confidence' },
      { title: 'Pricing Optimization', href: '/solutions/treasury-integrity' },
    ],
  },
];
