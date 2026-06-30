export const SKETCHFAB_SEMI_EMBED =
  'https://sketchfab.com/models/cc31b3a7141e4cfcb2bfbcfaf0a56f47/embed?autostart=1&internal=1&tracking=0&ui_ar=0&ui_infos=0&ui_snapshots=1&ui_stop=0&ui_theatre=1&ui_watermark=0';

export const SKETCHFAB_SEMI_PAGE =
  'https://sketchfab.com/3d-models/tesla-semi-truck-cc31b3a7141e4cfcb2bfbcfaf0a56f47';

export const FLEET_TRUCK_IMAGES = [
  {
    src: '/electric_semi_truck_tesla_with_trailer_rigged_362-1.jpg',
    alt: 'Electric semi truck with trailer — rigged 3D render',
    caption: 'Rigged fleet asset',
  },
  {
    src: '/electricsemitruckteslawithtrailerrigged3dmodel000-3.jpg',
    alt: 'Electric semi truck 3D model studio render',
    caption: 'Studio lighting pass',
  },
  {
    src: '/tesla_semi_trucks_and_trailers_collection_023.jpg',
    alt: 'Tesla semi trucks and trailers collection',
    caption: 'Multi-trailer yard view',
  },
  {
    src: '/teslasemitrucksandtrailerscollectionvray3dmodel005-2.jpg',
    alt: 'Tesla semi fleet V-Ray render — angle two',
    caption: 'Fleet lineup',
  },
  {
    src: '/teslasemitrucksandtrailerscollectionvray3dmodel026.jpg',
    alt: 'Tesla semi fleet V-Ray render — depot',
    caption: 'Depot staging',
  },
  {
    src: '/teslasemitrucksandtrailerscollectionvray3dmodel005-3.jpg',
    alt: 'Tesla semi fleet V-Ray render — angle three',
    caption: 'Load-ready configuration',
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
