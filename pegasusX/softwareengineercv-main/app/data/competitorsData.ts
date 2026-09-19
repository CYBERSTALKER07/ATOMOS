export type CompetitorFeatureRating = 'full' | 'partial' | 'none';

export interface CompetitorProfile {
  slug: string;
  name: string;
  category: string;
  tagline: string;
  marketPosition: string;
  pricingModel: string;
  pricingEstimate: string;
  hardwareRequired: boolean;
  idealFor: string[];
  notIdealFor: string[];
  strengths: string[];
  limitations: string[];
  whySwitchToPegasus: string[];
  featureRatings: {
    multiRoleDispatch: { rating: CompetitorFeatureRating; note: string };
    liveTelemetry: { rating: CompetitorFeatureRating; note: string };
    hardwareIndependence: { rating: CompetitorFeatureRating; note: string };
    integratedB2BPayments: { rating: CompetitorFeatureRating; note: string };
    warehouseGateTerminal: { rating: CompetitorFeatureRating; note: string };
    realtimeEventSync: { rating: CompetitorFeatureRating; note: string };
    routeOptimization: { rating: CompetitorFeatureRating; note: string };
    retailerOrderingPortal: { rating: CompetitorFeatureRating; note: string };
  };
  faqs: { question: string; answer: string }[];
  migrationNotes: {
    difficulty: 'Low' | 'Medium' | 'High';
    timeframe: string;
    transferredData: string[];
    migrationSupport: string;
  };
}

export const COMPETITORS_DATA: CompetitorProfile[] = [
  {
    slug: 'samsara',
    name: 'Samsara',
    category: 'Fleet Telematics & IoT',
    tagline: 'Connected Operations Cloud',
    marketPosition: 'Enterprise telematics, dashcams, and IoT sensor ecosystem',
    pricingModel: 'Per-vehicle multi-year hardware contract',
    pricingEstimate: '$30 - $45+ / vehicle / month + hardware upfront',
    hardwareRequired: true,
    idealFor: [
      'Large enterprise motor carriers needing proprietary dashcams and ELDs',
      'Fleets focused primarily on DOT hours-of-service compliance and driver safety scores',
      'Asset tracking across heavy industrial machinery and reefer trailers',
    ],
    notIdealFor: [
      'Suppliers and distributors wanting a unified order-to-cash operating system',
      'Operations that need seamless B2B retailer ordering and warehouse staging',
      'Fleets that want software-first telemetry without proprietary black-box hardware locks',
    ],
    strengths: [
      'Industry-leading AI dashcams and driver safety coaching programs',
      'Vast proprietary IoT hardware and environmental sensor catalog',
      'Mature regulatory ELD and hours-of-service compliance modules',
    ],
    limitations: [
      'High total cost of ownership locked into 36 to 60-month hardware leases',
      'No native B2B retailer ordering portal or wholesale catalog management',
      'Lacks warehouse loading gate terminal and double-entry payment reconciliation',
      'Proprietary black-box gateway dependency creates vendor lock-in',
    ],
    whySwitchToPegasus: [
      'Zero hardware lock-in: runs natively on driver mobile devices and open telematics standards',
      'End-to-end 6-role network: connects suppliers, warehouses, factories, drivers, and retailers',
      'Integrated financial integrity: settlement, driver cash collection, and treasury reconciliation',
      'Real-time event streaming: millisecond state synchronization across all dispatch screens',
    ],
    featureRatings: {
      multiRoleDispatch: {
        rating: 'partial',
        note: 'Driver and fleet manager only; no dedicated supplier-to-retailer network roles',
      },
      liveTelemetry: {
        rating: 'full',
        note: 'Excellent OBD-II sensor and proprietary GPS telemetry',
      },
      hardwareIndependence: {
        rating: 'none',
        note: 'Requires proprietary Samsara VG-series vehicle gateways',
      },
      integratedB2BPayments: {
        rating: 'none',
        note: 'No commercial order vetting, invoice settlement, or cash collection',
      },
      warehouseGateTerminal: {
        rating: 'none',
        note: 'No physical gate seal verification or yard terminal workflow',
      },
      realtimeEventSync: {
        rating: 'partial',
        note: 'Telemetry is near-realtime; business workflows rely on periodic polling',
      },
      routeOptimization: {
        rating: 'partial',
        note: 'Basic route planning; lacks multi-depot CVRP capacity balancing',
      },
      retailerOrderingPortal: {
        rating: 'none',
        note: 'Not supported; external ERP/WMS required',
      },
    },
    faqs: [
      {
        question: 'How does Pegasus compare to Samsara for fleet dispatch?',
        answer:
          'While Samsara specializes in hardware-centric vehicle tracking and dashcam safety, Pegasus is a complete B2B logistics operating system. Pegasus coordinates orders, warehouse staging, loading lanes, gate check-in, driver delivery stops, and B2B payment reconciliation without requiring proprietary vehicle hardware.',
      },
      {
        question: 'Can I use my existing GPS devices with Pegasus?',
        answer:
          'Yes. Pegasus is hardware-agnostic. You can capture telemetry via driver mobile devices (iOS/Android) or integrate existing telematics feeds through our open API.',
      },
      {
        question: 'Why do fleets switch from Samsara to Pegasus?',
        answer:
          'Fleets switch to Pegasus to eliminate expensive multi-year hardware leases and connect their entire supply chain—including retail buyers and warehouse dispatchers—into one unified, real-time control plane.',
      },
    ],
    migrationNotes: {
      difficulty: 'Low',
      timeframe: '1 to 2 weeks',
      transferredData: ['Vehicle rosters', 'Driver profiles', 'Depot coordinates', 'Customer delivery zones'],
      migrationSupport: 'Dedicated onboarding engineering and automated CSV/API vehicle roster importing.',
    },
  },
  {
    slug: 'rose-rocket',
    name: 'Rose Rocket',
    category: 'Freight Broker & Trucking TMS',
    tagline: 'Transportation Management System for Trucking & Brokerage',
    marketPosition: 'Modern freight brokerage and carrier TMS',
    pricingModel: 'Per-user monthly subscription',
    pricingEstimate: '$150 - $250 / user / month',
    hardwareRequired: false,
    idealFor: [
      'Freight brokers matching shippers to third-party spot-market carriers',
      'Long-haul truckload (TL) and less-than-truckload (LTL) carrier companies',
      'Operations requiring EDI integrations with nationwide 3PL brokers',
    ],
    notIdealFor: [
      'Suppliers and manufacturers running their own private distribution fleets',
      'Operations needing factory loading lane coordination and gate seal terminals',
      'High-density local route delivery with daily retail store stops',
    ],
    strengths: [
      'Intuitive, modern web interface tailored for freight dispatchers',
      'Robust freight rate quoting, spot bid management, and EDI connections',
      'Strong driver mobile app for freight bill-of-lading (BOL) document scanning',
    ],
    limitations: [
      'Designed for spot freight and freight brokering, not closed supplier-retailer ecosystems',
      'No factory loading lane orchestration or physical gate inspection terminals',
      'Lacks native automated B2B customer ordering portal and credit limit vetting',
      'Route optimization is freight-lane focused rather than multi-stop urban CVRP',
    ],
    whySwitchToPegasus: [
      'Tailored for private supplier fleets and regional distribution networks',
      'Complete closed-loop ecosystem: retailers order directly, suppliers dispatch, drivers deliver',
      'Google OR-Tools CVRP engine: handles complex multi-stop, multi-capacity urban routing',
      'Dedicated apps across 6 distinct operational roles with real-time reactive UI',
    ],
    featureRatings: {
      multiRoleDispatch: {
        rating: 'partial',
        note: 'Broker, dispatcher, and carrier views; lacks retailer and factory gate roles',
      },
      liveTelemetry: {
        rating: 'partial',
        note: 'Driver mobile ping and third-party ELD integrations',
      },
      hardwareIndependence: {
        rating: 'full',
        note: 'Pure cloud software; supports mobile app telemetry',
      },
      integratedB2BPayments: {
        rating: 'partial',
        note: 'Invoicing and factoring integrations; lacks driver cash-on-delivery settlement',
      },
      warehouseGateTerminal: {
        rating: 'none',
        note: 'No gate terminal or seal integrity workflow',
      },
      realtimeEventSync: {
        rating: 'partial',
        note: 'Standard REST APIs and periodic webhook sync',
      },
      routeOptimization: {
        rating: 'partial',
        note: 'Point-to-point and hub-and-spoke freight routing',
      },
      retailerOrderingPortal: {
        rating: 'none',
        note: 'Customer portal for tracking freight quotes only, not wholesale ordering',
      },
    },
    faqs: [
      {
        question: 'Is Pegasus an alternative to Rose Rocket for private fleets?',
        answer:
          'Yes. Rose Rocket is optimized for freight brokers and for-hire freight carriers. Pegasus is engineered specifically for private supplier networks, regional manufacturers, and distributors managing their own fleets, warehouses, and direct retail buyer accounts.',
      },
      {
        question: 'Does Pegasus support proof of delivery and photo capture?',
        answer:
          'Yes. Pegasus driver apps include instant electronic proof of delivery (ePOD), signature capture, photo verification, and cash-on-delivery reconciliation that updates warehouse treasury in real time.',
      },
    ],
    migrationNotes: {
      difficulty: 'Low',
      timeframe: '1 to 2 weeks',
      transferredData: ['Driver rosters', 'Vehicle fleet specs', 'Customer shipping addresses', 'Route histories'],
      migrationSupport: 'Full assisted migration with bulk import tools for orders, lanes, and customers.',
    },
  },
  {
    slug: 'motive',
    name: 'Motive',
    category: 'Fleet Safety & Compliance',
    tagline: 'Operations management and ELD platform',
    marketPosition: 'ELD compliance, safety dashcams, and asset tracking (formerly KeepTruckin)',
    pricingModel: 'Per-vehicle annual contract',
    pricingEstimate: '$35 - $50 / vehicle / month',
    hardwareRequired: true,
    idealFor: [
      'Heavy commercial vehicle fleets subject to mandatory FMCSA ELD compliance',
      'Safety managers focused on reducing speeding, harsh braking, and cell phone distraction',
      'Asset tracking across remote construction and trailer equipment',
    ],
    notIdealFor: [
      'Distributors looking for multi-stop route optimization and dynamic dispatch boards',
      'Companies that want automated order-to-delivery handoffs across warehouse and gate crews',
      'Businesses seeking an integrated B2B merchant portal and payment collection ledger',
    ],
    strengths: [
      'Industry standard ELD device with seamless DOT inspection compliance',
      'AI dashcam technology with high driver accident reduction ratings',
      'Accurate GPS tracking and geofencing triggers',
    ],
    limitations: [
      'Limited dispatch and routing capabilities—primarily a compliance and safety tool',
      'No warehouse dispatch staging boards, loading lane assignments, or gate check terminals',
      'No customer order management, catalog, or financial settlement workflows',
      'Ongoing hardware lease and installation overhead',
    ],
    whySwitchToPegasus: [
      'Comprehensive B2B logistics OS: integrates dispatch, inventory, routing, and payments',
      'Full warehouse and factory control: staging lanes, gate seal verification, and yard management',
      'Dynamic multi-vehicle route optimization (CVRP) with real-time stop sequencing',
      'No proprietary hardware lock: runs on smartphones, tablets, and modern desktop browsers',
    ],
    featureRatings: {
      multiRoleDispatch: {
        rating: 'none',
        note: 'Fleet safety and compliance focused only',
      },
      liveTelemetry: {
        rating: 'full',
        note: 'Excellent vehicle CAN bus and GPS tracking',
      },
      hardwareIndependence: {
        rating: 'none',
        note: 'Requires proprietary Motive vehicle gateway and dashcams',
      },
      integratedB2BPayments: {
        rating: 'none',
        note: 'Not supported',
      },
      warehouseGateTerminal: {
        rating: 'none',
        note: 'Not supported',
      },
      realtimeEventSync: {
        rating: 'partial',
        note: 'Telemetry syncs regularly; business workflows not present',
      },
      routeOptimization: {
        rating: 'none',
        note: 'Basic route tracking, not multi-stop route optimization',
      },
      retailerOrderingPortal: {
        rating: 'none',
        note: 'Not supported',
      },
    },
    faqs: [
      {
        question: 'Can I replace Motive with Pegasus?',
        answer:
          'If your primary objective is dispatching orders, optimizing delivery routes, coordinating warehouse loading, and managing B2B customer relationships, Pegasus provides the complete operating system. If you require mandatory US FMCSA ELD logbooks, Pegasus integrates seamlessly alongside existing ELD hardware.',
      },
    ],
    migrationNotes: {
      difficulty: 'Low',
      timeframe: '1 week',
      transferredData: ['Vehicles', 'Drivers', 'Geofence landmarks'],
      migrationSupport: 'Guided onboarding and route setup.',
    },
  },
  {
    slug: 'turvo',
    name: 'Turvo',
    category: 'Collaborative Logistics Platform',
    tagline: 'Collaborative TMS for shippers, 3PLs, and brokers',
    marketPosition: 'High-end enterprise collaborative logistics network',
    pricingModel: 'Enterprise annual contracts with high implementation fees',
    pricingEstimate: '$30,000 - $100,000+ / year',
    hardwareRequired: false,
    idealFor: [
      'Large enterprise 3PLs managing multi-party logistics with Fortune 500 shippers',
      'Freight brokerages with large IT teams capable of long implementation cycles',
    ],
    notIdealFor: [
      'Regional distributors and suppliers seeking rapid deployment without seven-figure budgets',
      'Private fleets that need hands-on warehouse loading and driver stop execution',
      'Companies wanting straightforward per-site or usage-based pricing',
    ],
    strengths: [
      'Unified multi-party collaboration feed across disparate logistics entities',
      'Sophisticated enterprise document management and freight visibility',
      'Strong brand recognition among national 3PL networks',
    ],
    limitations: [
      'Excessive implementation complexity (6 to 12 months typical deployment)',
      'Opaque enterprise pricing with substantial upfront professional service costs',
      'Heavyweight overhead for mid-market suppliers who just want fast, reliable dispatch',
    ],
    whySwitchToPegasus: [
      'Instant time-to-value: deploy within days, not quarters',
      'Engineered for speed: sub-second transactional latency on distributed cloud architecture',
      'Purpose-built for operational execution: warehouse boards, driver navigation, and gate checks',
      'Transparent, predictable economics without enterprise lock-in',
    ],
    featureRatings: {
      multiRoleDispatch: {
        rating: 'full',
        note: 'Enterprise multi-party collaboration model',
      },
      liveTelemetry: {
        rating: 'partial',
        note: 'Aggregates third-party ELD feeds; no native hardware',
      },
      hardwareIndependence: {
        rating: 'full',
        note: 'Pure cloud platform',
      },
      integratedB2BPayments: {
        rating: 'partial',
        note: 'Freight audit and billing; lacks local driver cash reconciliation',
      },
      warehouseGateTerminal: {
        rating: 'partial',
        note: 'Basic yard management module on enterprise tiers',
      },
      realtimeEventSync: {
        rating: 'partial',
        note: 'Collaborative activity feed with periodic background sync',
      },
      routeOptimization: {
        rating: 'partial',
        note: 'Freight routing via partner engines',
      },
      retailerOrderingPortal: {
        rating: 'partial',
        note: 'Shipper portal available, but optimized for bulk freight not retail orders',
      },
    },
    faqs: [
      {
        question: 'Why choose Pegasus over Turvo?',
        answer:
          'Pegasus offers the collaborative multi-role visibility of Turvo without the 9-month implementation delay or massive enterprise licensing fees. Pegasus is built for operational execution from day one.',
      },
    ],
    migrationNotes: {
      difficulty: 'Medium',
      timeframe: '2 to 3 weeks',
      transferredData: ['Shippers', 'Carriers', 'Lane histories', 'Rate cards'],
      migrationSupport: 'White-glove migration architecture and API data ingestion.',
    },
  },
  {
    slug: 'onfleet',
    name: 'Onfleet',
    category: 'Last-Mile Delivery Software',
    tagline: 'Last mile delivery management software',
    marketPosition: 'Courier, food, and e-commerce parcel delivery software',
    pricingModel: 'Tiered by monthly delivery task volume',
    pricingEstimate: '$550 - $1,295+ / month (based on task limits)',
    hardwareRequired: false,
    idealFor: [
      'Local couriers, pharmacy delivery, and restaurant meal parcel drops',
      'Direct-to-consumer (D2C) e-commerce businesses needing customer SMS tracking',
    ],
    notIdealFor: [
      'Heavy B2B distribution, palletized wholesale freight, and manufacturer networks',
      'Operations requiring factory loading lane coordination and gate seal inspection',
      'Multi-supplier platforms where retailers browse wholesale catalogs and place bulk orders',
    ],
    strengths: [
      'Beautiful, consumer-friendly tracking links and SMS notifications',
      'Simple, easy-to-use dispatcher interface and driver mobile app',
      'Quick plug-and-play setup for courier delivery businesses',
    ],
    limitations: [
      'Designed strictly for parcel drop-offs, not wholesale freight or industrial distribution',
      'No warehouse pallet staging, loading bay scheduling, or gate security control',
      'No wholesale product catalog, order vetting, or commercial terms management',
      'Punitive pricing tiers based on delivery task overages',
    ],
    whySwitchToPegasus: [
      'Engineered for heavy-duty B2B logistics: pallets, crates, trucks, and multi-depot runs',
      'Complete closed-loop ecosystem from supplier catalog to retailer delivery confirmation',
      'Native warehouse and factory terminal apps for dock workers and gate agents',
      'Integrated treasury: reconciles invoice payments, driver cash, and outstanding balances',
    ],
    featureRatings: {
      multiRoleDispatch: {
        rating: 'partial',
        note: 'Dispatcher and driver only; lacks warehouse, factory, and retailer roles',
      },
      liveTelemetry: {
        rating: 'full',
        note: 'High-accuracy driver mobile GPS tracking',
      },
      hardwareIndependence: {
        rating: 'full',
        note: 'Smartphone app for iOS and Android',
      },
      integratedB2BPayments: {
        rating: 'none',
        note: 'No commercial credit, terms, or cash reconciliation',
      },
      warehouseGateTerminal: {
        rating: 'none',
        note: 'Not supported',
      },
      realtimeEventSync: {
        rating: 'partial',
        note: 'Realtime driver location; standard webhooks for task states',
      },
      routeOptimization: {
        rating: 'full',
        note: 'Fast single-driver and multi-driver parcel routing',
      },
      retailerOrderingPortal: {
        rating: 'none',
        note: 'Not supported',
      },
    },
    faqs: [
      {
        question: 'Can Pegasus replace Onfleet for wholesale deliveries?',
        answer:
          'Yes. Onfleet is designed for consumer parcel couriers. For wholesale distributors delivering pallets and cases to businesses, Pegasus provides route optimization along with warehouse loading lanes, gate check-in, order vetting, and B2B payment collection.',
      },
    ],
    migrationNotes: {
      difficulty: 'Low',
      timeframe: '3 to 5 days',
      transferredData: ['Driver accounts', 'Customer destinations', 'Historical tasks'],
      migrationSupport: 'Simple CSV customer and driver importing.',
    },
  },
  {
    slug: 'bringg',
    name: 'Bringg',
    category: 'Enterprise Delivery Orchestration',
    tagline: 'Delivery management platform for enterprise retailers',
    marketPosition: 'Enterprise retail fulfillment and multi-carrier orchestration',
    pricingModel: 'Enterprise annual licensing + professional services',
    pricingEstimate: '$50,000+ / year',
    hardwareRequired: false,
    idealFor: [
      'Large retail chains orchestrating curbside pickup and third-party courier fleets',
      'Enterprises needing complex multi-carrier parcel aggregator rules',
    ],
    notIdealFor: [
      'Independent suppliers and manufacturers managing dedicated private truck fleets',
      'Mid-market distributors looking for self-contained, turnkey operations',
    ],
    strengths: [
      'Extensive integrations with on-demand delivery networks (DoorDash, Uber, Roadie)',
      'Robust click-and-collect and curbside pickup modules for retail stores',
    ],
    limitations: [
      'High implementation barriers, long customization cycles, and enterprise pricing',
      'Geared toward retail e-commerce parcel routing rather than physical goods distribution',
      'Lacks native factory loading lanes and gate seal integrity verification',
    ],
    whySwitchToPegasus: [
      'Turnkey B2B logistics core built specifically for physical goods and wholesale networks',
      'Dedicated apps across 6 operational roles that work instantly out of the box',
      'Zero carrier markup or expensive enterprise implementation overhead',
    ],
    featureRatings: {
      multiRoleDispatch: {
        rating: 'full',
        note: 'Enterprise retail roles and multi-carrier views',
      },
      liveTelemetry: {
        rating: 'partial',
        note: 'Aggregated from carriers and driver apps',
      },
      hardwareIndependence: {
        rating: 'full',
        note: 'Cloud-based',
      },
      integratedB2BPayments: {
        rating: 'none',
        note: 'Consumer e-commerce payment gateways only',
      },
      warehouseGateTerminal: {
        rating: 'none',
        note: 'Not supported',
      },
      realtimeEventSync: {
        rating: 'partial',
        note: 'Event bus for enterprise retail integrations',
      },
      routeOptimization: {
        rating: 'partial',
        note: 'Carrier allocation engine',
      },
      retailerOrderingPortal: {
        rating: 'none',
        note: 'Consumer retail focus; not B2B merchant wholesale',
      },
    },
    faqs: [
      {
        question: 'How does Pegasus differ from Bringg?',
        answer:
          'Bringg orchestrates retail click-and-collect and third-party parcel carriers. Pegasus is a sovereign operating system for supplier networks moving commercial freight and wholesale orders with their own trucks, warehouses, and factories.',
      },
    ],
    migrationNotes: {
      difficulty: 'Medium',
      timeframe: '2 to 3 weeks',
      transferredData: ['Driver rosters', 'Fleet specs', 'Store/retailer locations'],
      migrationSupport: 'Dedicated transition planning.',
    },
  },
  {
    slug: 'descartes',
    name: 'Descartes / Manhattan',
    category: 'Legacy Enterprise TMS & WMS',
    tagline: 'Global logistics and supply chain execution solutions',
    marketPosition: 'Global legacy logistics and customs compliance infrastructure',
    pricingModel: 'Heavy enterprise licensing, server fees, and multi-year maintenance',
    pricingEstimate: '$100,000 - $1,000,000+ total contract value',
    hardwareRequired: false,
    idealFor: [
      'Multinational conglomerates managing customs clearance and intercontinental shipping',
      'Mega-enterprises with legacy mainframe and SAP/Oracle ERP integration teams',
    ],
    notIdealFor: [
      'Modern agile suppliers, distributors, and fleets that need intuitive, reactive software',
      'Teams that cannot afford multi-year implementations or seven-figure consulting bills',
      'Field crews who require responsive, contemporary mobile and desktop apps',
    ],
    strengths: [
      'Decades of compliance handling for cross-border customs and maritime freight',
      'Broad footprint across global ocean, air, and rail shipping networks',
    ],
    limitations: [
      'Antiquated 1990s user experience requiring extensive employee training',
      'Rigid, batch-driven database architectures with sluggish synchronization',
      'Exorbitant customization costs requiring dedicated external consultants',
    ],
    whySwitchToPegasus: [
      'Modern cloud architecture: Google Cloud Spanner distributed ledger with sub-second sync',
      '60fps tactical UI: built for instant human comprehension and zero training overhead',
      'Transparent total cost of ownership without multi-million dollar consultant lock-in',
      'Native mobile and desktop apps for drivers, warehouse workers, and gate security',
    ],
    featureRatings: {
      multiRoleDispatch: {
        rating: 'partial',
        note: 'Complex legacy modules; poor mobile ergonomics',
      },
      liveTelemetry: {
        rating: 'partial',
        note: 'Batch EDI updates and telematics feeds',
      },
      hardwareIndependence: {
        rating: 'full',
        note: 'Software-based (often legacy on-prem or hosted)',
      },
      integratedB2BPayments: {
        rating: 'partial',
        note: 'ERP accounting integrations; lacks live driver cash reconciliation',
      },
      warehouseGateTerminal: {
        rating: 'partial',
        note: 'Legacy yard management modules',
      },
      realtimeEventSync: {
        rating: 'none',
        note: 'Batch-oriented and EDI-driven; significant sync latency',
      },
      routeOptimization: {
        rating: 'partial',
        note: 'Legacy batch route planning algorithms',
      },
      retailerOrderingPortal: {
        rating: 'none',
        note: 'Separate customer EDI setup required',
      },
    },
    faqs: [
      {
        question: 'Can a mid-market distributor replace legacy TMS/WMS with Pegasus?',
        answer:
          'Yes. Pegasus provides the modern operational alternative to legacy monoliths like Descartes or Manhattan Associates. You get real-time dispatch, route optimization, driver telemetry, warehouse dock staging, and payment reconciliation in an intuitive, cloud-native platform deployed in days.',
      },
    ],
    migrationNotes: {
      difficulty: 'High',
      timeframe: '3 to 4 weeks',
      transferredData: ['Full customer records', 'Item catalogs', 'Depot configurations', 'Carrier rosters'],
      migrationSupport: 'Dedicated solutions architect, historical data migration, and parallel run assistance.',
    },
  },
];

export function getCompetitorBySlug(slug: string): CompetitorProfile | undefined {
  return COMPETITORS_DATA.find((c) => c.slug === slug);
}
