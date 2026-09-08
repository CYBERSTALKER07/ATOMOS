export interface IndustrySolution {
  slug: string;
  name: string;
  headline: string;
  summary: string;
  metaTitle: string;
  metaDescription: string;
  keyChallenges: string[];
  howPegasusSolves: { title: string; description: string }[];
  keyFeatures: string[];
  metrics: { label: string; value: string }[];
  workflowSteps: { step: string; detail: string }[];
  faqs: { question: string; answer: string }[];
}

export const INDUSTRIES_DATA: IndustrySolution[] = [
  {
    slug: 'food-and-beverage',
    name: 'Food & Beverage Distribution',
    headline: 'Freshness-First B2B Logistics & Delivery Orchestration',
    summary:
      'Prevent spoilage, meet narrow delivery windows, and automate driver cash collections across wholesale restaurant, grocery, and beverage routes.',
    metaTitle: 'Food & Beverage Logistics & Dispatch Software | Pegasus',
    metaDescription:
      'Optimize perishable delivery schedules, cold-chain compliance, and store replenishment with Pegasus B2B food and beverage logistics software. Book demo.',
    keyChallenges: [
      'Perishable shelf-life constraints requiring strict FIFO and lot-level departure timing',
      'Narrow morning receiving windows at restaurants and retail grocers',
      'High volume of delivery-time invoice adjustments, split crates, and returns',
      'Cash and cheque collection reconciliation at the point of delivery',
    ],
    howPegasusSolves: [
      {
        title: 'Perishable-Aware Route Optimization',
        description:
          'Google OR-Tools solver respects shelf-life priorities, temperature compartments, and store receiving windows to sequence morning drops.',
      },
      {
        title: 'Point-of-Delivery Invoice Adjustments',
        description:
          'Drivers record rejected crates or damaged items instantly in the mobile app, recalculating the final invoice and payment balance on the spot.',
      },
      {
        title: 'Cash-on-Delivery (COD) Treasury Matching',
        description:
          'Drivers log cash and check collections into the driver app, locking ledger entries that warehouse treasury verifies during end-of-day checkout.',
      },
    ],
    keyFeatures: [
      'Multi-compartment temperature status tracking',
      'Proof of delivery with condition photos and signature capture',
      'Real-time retailer order catalog with live inventory availability',
      'Automatic return merchandise authorization (RMA) credit notes',
    ],
    metrics: [
      { label: 'Spoilage Reduction', value: '34%' },
      { label: 'On-Time Morning Drops', value: '98.8%' },
      { label: 'Daily Cash Reconciliation', value: '< 15 min' },
    ],
    workflowSteps: [
      {
        step: '1. Order Cutoff & Stock Allocation',
        detail: 'Retailers submit orders by midnight; stock is reserved and staged by temperature zone.',
      },
      {
        step: '2. Morning Wave Dispatch',
        detail: 'Warehouse loads trucks according to reverse delivery order with verified gate seals.',
      },
      {
        step: '3. Stop Execution & Inspection',
        detail: 'Driver logs arrival, confirms crates, records temperature check, and captures signature.',
      },
      {
        step: '4. Settlement',
        detail: 'Treasury reconciles cash collected against delivery notes in real time.',
      },
    ],
    faqs: [
      {
        question: 'Does Pegasus support catch-weight and split-case items?',
        answer:
          'Yes. Pegasus allows warehouse staging and drivers to adjust catch-weight pricing (e.g., meat, produce, cheeses) and split cases directly at checkout.',
      },
      {
        question: 'Can retailers place recurring weekly replenishment orders?',
        answer:
          'Yes. Retailers have a dedicated desktop or mobile portal with standing orders, automated delivery reminders, and invoice archives.',
      },
    ],
  },
  {
    slug: 'fmcg-consumer-goods',
    name: 'Fast-Moving Consumer Goods (FMCG)',
    headline: 'High-Volume SKU Replenishment & Multi-Depot Distribution',
    summary:
      'Synchronize factory loading lanes, regional warehouse cross-docks, and retail store replenishment with millisecond event visibility.',
    metaTitle: 'FMCG Logistics & Fleet Dispatch Platform | Pegasus TMS',
    metaDescription:
      'Scale high-velocity consumer goods distribution. Connect factory docks, warehouse staging, and retail deliveries into one real-time logistics operating system.',
    keyChallenges: [
      'Massive daily SKU turn rates leading to dock congestion and staging bottlenecks',
      'Coordination disconnects between factory production lines and warehouse dispatch bays',
      'Penalty fines (chargebacks) from major retail chains for late or incomplete shipments',
    ],
    howPegasusSolves: [
      {
        title: 'Factory-to-Warehouse Staging Sync',
        description:
          'Factory production schedules link directly to warehouse dispatch bays, preventing stockouts before dispatch trucks are loaded.',
      },
      {
        title: 'Cross-Dock Yard & Gate Management',
        description:
          'Inbound trailers are sealed and checked through the gate terminal, transferring directly to outbound delivery routes with zero lost time.',
      },
      {
        title: 'Full Audit-Trail OTIF Compliance',
        description:
          'On-Time In-Full (OTIF) timestamps eliminate dispute friction with retail buyers by recording gate, dock, and delivery timestamps.',
      },
    ],
    keyFeatures: [
      'Multi-depot inventory visibility across regional hubs',
      'High-speed barcode and QR manifest scanning at factory gates',
      'Automated load balancing across private trucks and third-party haulers',
      'Automated electronic proof of delivery with instant billing triggers',
    ],
    metrics: [
      { label: 'OTIF Compliance Score', value: '99.2%' },
      { label: 'Dock Turnaround Time', value: '-45%' },
      { label: 'Dispatch Capacity Utilization', value: '+28%' },
    ],
    workflowSteps: [
      {
        step: '1. Factory Manifest Generation',
        detail: 'Finished goods roll off production and are assigned to cross-dock dispatch lanes.',
      },
      {
        step: '2. Fleet Routing & Vehicle Assignment',
        detail: 'Routes are calculated using vehicle capacity constraints and store receiving hours.',
      },
      {
        step: '3. Gate Inspection & Seal Locking',
        detail: 'Security terminals record truck departures with tamper-evident digital seal numbers.',
      },
      {
        step: '4. Store Confirmation',
        detail: 'Receiving clerk scans pallet barcodes, confirming immediate inventory release.',
      },
    ],
    faqs: [
      {
        question: 'Can Pegasus handle hundreds of daily delivery trucks simultaneously?',
        answer:
          'Yes. Built on Google Cloud Spanner and Go distributed microservices, Pegasus processes hundreds of concurrent dispatch operations with sub-10ms database latency.',
      },
    ],
  },
  {
    slug: 'cold-chain-pharma',
    name: 'Pharmaceutical & Cold Chain Logistics',
    headline: 'Regulatory-Compliant Temperature Telemetry & Chain of Custody',
    summary:
      'Maintain flawless temperature compliance, unbroken digital audit trails, and secure gate control for pharmaceuticals, vaccines, and sensitive biologics.',
    metaTitle: 'Pharma Cold Chain Logistics Software | Pegasus TMS',
    metaDescription:
      'Ensure regulatory cold-chain compliance and tamper-proof chain of custody. Real-time temperature telemetry, gate verification, and digital audit trails.',
    keyChallenges: [
      'Strict GDP / FDA regulatory compliance requiring unbroken temperature logs',
      'High value of cargo with zero tolerance for thermal excursions or transit delays',
      'Mandatory chain-of-custody verification at every physical handover',
    ],
    howPegasusSolves: [
      {
        title: 'Continuous Telemetry & Excursion Alerts',
        description:
          'Temperature and humidity sensors stream telemetry alongside vehicle GPS, triggering instant alerts upon any thermal excursion threshold.',
      },
      {
        title: 'Digital Seal & Gate Verification',
        description:
          'Every departure and arrival requires verified digital seal authentication at the payload terminal before driver handoff is allowed.',
      },
      {
        title: 'Cryptographic Audit Ledger',
        description:
          'Immutable event logs record who loaded, vetted, inspected, and signed for each shipment, producing export-ready compliance reports.',
      },
    ],
    keyFeatures: [
      'GDP compliant digital chain of custody records',
      'Two-factor authorized handoffs between drivers and receiving pharmacists',
      'Automated quarantine protocol upon temperature breach',
      '21 CFR Part 11 compliant digital signatures',
    ],
    metrics: [
      { label: 'Temperature Compliance', value: '99.99%' },
      { label: 'Audit Preparation Time', value: 'Zero (Instant)' },
      { label: 'Excursion Incident Rate', value: '-82%' },
    ],
    workflowSteps: [
      {
        step: '1. Pre-Trip Thermal Conditioning',
        detail: 'Refrigerated vehicle precooling verified prior to dispatch allocation.',
      },
      {
        step: '2. Sealed Loading',
        detail: 'Warehouse crew verifies lot serials and applies digital security seal.',
      },
      {
        step: '3. Monitored Transit',
        detail: 'Continuous sensor pings track location and compartment temperatures every 10 seconds.',
      },
      {
        step: '4. Pharmacist Handoff',
        detail: 'Receiver inspects seal integrity and signs electronic chain of custody.',
      },
    ],
    faqs: [
      {
        question: 'Can regulatory auditors export temperature logs from Pegasus?',
        answer:
          'Yes. Pegasus generates comprehensive PDF and CSV compliance audit reports including timestamped sensor telemetry, GPS breadcrumbs, and authorized digital signatures.',
      },
    ],
  },
  {
    slug: 'building-materials',
    name: 'Building Materials & Heavy Freight',
    headline: 'Heavy Haul Dispatch, Job-Site Staging & Axle Weight Optimization',
    summary:
      'Coordinate flatbeds, cranes, cement, and bulk materials with job-site delivery windows, weight limit calculations, and proof of offload.',
    metaTitle: 'Building Materials Dispatch & Fleet Software | Pegasus',
    metaDescription:
      'Streamline heavy construction logistics. Calculate axle weight limits, optimize job-site delivery sequences, and capture digital proof of offload with Pegasus.',
    keyChallenges: [
      'Strict gross vehicle weight (GVW) and axle weight limit regulations',
      'Job-site access restrictions, unpaved surfaces, and required crane/forklift staging',
      'High demurrage costs when delivery trucks are delayed waiting for contractor equipment',
    ],
    howPegasusSolves: [
      {
        title: 'Weight & Axle-Aware Dispatch',
        description:
          'Pegasus calculates cumulative payload weights against truck capacities, preventing overweight fines and uneven cargo distribution.',
      },
      {
        title: 'Live Contractor ETA Tracking',
        description:
          'Job-site foremen receive real-time SMS and map links with precision ETAs so offloading equipment is staged before the truck arrives.',
      },
      {
        title: 'Photo Proof of Offload',
        description:
          'Drivers photograph offloaded materials and drop locations on job sites, attaching high-res images to delivery receipts to eliminate claims.',
      },
    ],
    keyFeatures: [
      'Gross and axle weight calculation on the dispatch board',
      'Job-site pin-drop geofencing for remote construction locations',
      'Driver offload photo documentation with timestamp and GPS stamp',
      'Demurrage and detention time tracking',
    ],
    metrics: [
      { label: 'Job-Site Wait Times', value: '-40%' },
      { label: 'Weight Violation Citations', value: '0' },
      { label: 'Disputed Delivery Claims', value: '-91%' },
    ],
    workflowSteps: [
      {
        step: '1. Staging & Weight Validation',
        detail: 'Orders are weighed at the yard scale; Pegasus checks truck payload limits.',
      },
      {
        step: '2. Route Clearance',
        detail: 'Routes account for commercial truck height, weight, and bridge restrictions.',
      },
      {
        step: '3. Contractor Notification',
        detail: 'Automated alert sent when truck is 20 minutes from job site for crane prep.',
      },
      {
        step: '4. Offload & Photo Confirmation',
        detail: 'Driver photographs placed pallets and captures supervisor digital signature.',
      },
    ],
    faqs: [
      {
        question: 'Does Pegasus support split deliveries across multiple job sites?',
        answer:
          'Yes. Our vehicle routing solver sequences multi-stop drops in reverse load order so materials at the back of the flatbed are accessible first.',
      },
    ],
  },
  {
    slug: 'wholesale-distribution',
    name: 'Wholesale & Regional B2B Distribution',
    headline: 'End-to-End B2B Logistics: Order Vetting, Dispatch & Cash Settlement',
    summary:
      'Empower regional distributors to manage commercial trade credit, warehouse dispatch waves, driver delivery stops, and instant payment reconciliation.',
    metaTitle: 'B2B Wholesale Distribution Logistics Software | Pegasus',
    metaDescription:
      'The complete operating system for B2B wholesale distributors. Connect retail merchant ordering, warehouse dispatch, fleet tracking, and treasury cash collection.',
    keyChallenges: [
      'Manual phone, email, and paper orders causing high fulfillment error rates',
      'Customer credit limit overruns and risky unvetted deliveries',
      'Inefficient manual dispatching taking hours of spreadsheet work each morning',
      'Disconnected accounting systems delaying payment reconciliation by days',
    ],
    howPegasusSolves: [
      {
        title: 'Self-Service Retailer Ordering',
        description:
          'Store owners browse catalogs, view custom wholesale pricing, and place orders directly from desktop or mobile devices.',
      },
      {
        title: 'Automated Order Vetting & Credit Gates',
        description:
          'Orders are automatically checked against merchant credit limits, outstanding balances, and inventory reserves before release to dispatch.',
      },
      {
        title: 'One-Click Wave Dispatching',
        description:
          'Pegasus groups approved orders into optimized truck routes in seconds, generating loading manifests and digital driver manifests.',
      },
    ],
    keyFeatures: [
      'Custom wholesale price books and customer credit management',
      'Automated order vetting pipeline with manual ops override',
      'Live driver telemetry and automated customer delivery notifications',
      'Integrated treasury reconciliation for cash, cheque, and bank transfers',
    ],
    metrics: [
      { label: 'Order Processing Time', value: '-75%' },
      { label: 'Dispatch Planning Time', value: '5 min' },
      { label: 'Bad Debt / Credit Overruns', value: '-60%' },
    ],
    workflowSteps: [
      {
        step: '1. Retailer Checkout',
        detail: 'Store owner places order via digital catalog under assigned credit terms.',
      },
      {
        step: '2. Automated Vetting',
        detail: 'System validates credit limit and stock availability; notifies sales rep if approval is needed.',
      },
      {
        step: '3. Optimized Dispatch',
        detail: 'Warehouse assigns orders to trucks; loading manifests generated automatically.',
      },
      {
        step: '4. Delivery & Settlement',
        detail: 'Driver confirms delivery and collects payment; treasury ledger updates instantly.',
      },
    ],
    faqs: [
      {
        question: 'Can our sales reps place orders on behalf of retail customers?',
        answer:
          'Yes. Field sales representatives have full portal access to place orders, check inventory, and review customer payment history in real time.',
      },
    ],
  },
];

export function getIndustryBySlug(slug: string): IndustrySolution | undefined {
  return INDUSTRIES_DATA.find((i) => i.slug === slug);
}
