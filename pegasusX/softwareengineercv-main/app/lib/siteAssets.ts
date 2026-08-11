/** Public marketing art — stippled Pegasus logistics illustrations (2026 refresh). */
export const SITE_IMAGES = {
  pegasusContainer: '/Unknown-10.jpg',
  truckTerminal: '/Unknown-5.jpg',
  multimodalHub: '/Unknown-6.jpg',
  warehouseAutomation: '/Unknown-7.jpg',
  logisticsPlatformUi: '/Unknown-8.jpg',
  deliveryDrone: '/Unknown-9.jpg',
  containerShip: '/EbszSCwA.jpeg',
  terminalArchitecture: '/Gemini_Generated_Image_1y7rbo1y7rbo1y7r.png',
  portCraneScene: '/Gemini_Generated_Image_ngsos5ngsos5ngso.png',
  operationsTeam: '/Gemini_Generated_Image_xvlgisxvlgisxvlg.png',
  warehouseWireframe: '/Gemini_Generated_Image_y7jkmqy7jkmqy7jk.png',
  fleekHeroNew: '/Gemini_Generated_Image_un3te4un3te4un3t.png',
  /** Driver → storefront handoff (stipple). */
  lastMileDelivery: '/Unknown-11.jpg',
} as const;

/** Rotating editorial cards — supplier, warehouse, retailer, fleet, finance, etc. */
export const EDITORIAL_IMAGES = [
  SITE_IMAGES.logisticsPlatformUi,
  SITE_IMAGES.deliveryDrone,
  SITE_IMAGES.pegasusContainer,
  SITE_IMAGES.terminalArchitecture,
  SITE_IMAGES.fleekHeroNew,
  SITE_IMAGES.operationsTeam,
  SITE_IMAGES.warehouseWireframe,
] as const;

/** Fleet showcase carousel — trucks, ships, and intermodal yards. */
export const FLEET_TRUCK_IMAGES = [
  {
    src: SITE_IMAGES.truckTerminal,
    alt: 'Pegasus semi truck at a lit terminal gate',
    caption: 'Gate-ready fleet',
  },
  {
    src: SITE_IMAGES.operationsTeam,
    alt: 'Dispatch, driver, and ops leads in front of a Pegasus trailer',
    caption: 'Role-aligned execution',
  },
  {
    src: SITE_IMAGES.multimodalHub,
    alt: 'Air, sea, road, and rail logistics around a Pegasus container',
    caption: 'Multi-modal network',
  },
  {
    src: SITE_IMAGES.containerShip,
    alt: 'Container ship bow with stippled cargo stacks',
    caption: 'Ocean lane capacity',
  },
  {
    src: SITE_IMAGES.portCraneScene,
    alt: 'Port cranes, rail, and docked container ship',
    caption: 'Terminal throughput',
  },
  {
    src: SITE_IMAGES.pegasusContainer,
    alt: 'Pegasus container lifted by crane',
    caption: 'Sealed manifest handoff',
  },
] as const;

export type FleetTruckImage = (typeof FLEET_TRUCK_IMAGES)[number];

export const FLEET_SHOWCASE_CAPTIONS = [
  'Plan the load before wheels roll',
  'Seal at the gate — every manifest verified',
  'Live telemetry on planned vs actual routes',
  'Deviation alerts before retailers call',
  'Peak dispatch — visual boards under pressure',
  'One fleet picture across six roles',
] as const;

export const HERO_VIDEO_POSTER = SITE_IMAGES.truckTerminal;
export const ORDER_LIFECYCLE_POSTER = SITE_IMAGES.truckTerminal;
/** Brand mark — nav, footer, and social link previews. */
export const BRAND_LOGO = '/pegasus.jpg';
export const OG_IMAGE = BRAND_LOGO;
export const SOLUTIONS_DEFAULT_IMAGE = SITE_IMAGES.warehouseWireframe;
export const DISPATCH_ARCADE_IMAGE = SITE_IMAGES.warehouseAutomation;
