export type FleetRoutePoint = { lat: number; lng: number };

export type DemoFleetRoute = {
  id: string;
  driver_name: string;
  driver_id: string;
  coordinates: FleetRoutePoint[];
  opacity: number;
};

export const DEMO_FLEET_ROUTES: DemoFleetRoute[] = [
  {
    id: "route-1",
    driver_name: "A. Karimov",
    driver_id: "drv-01",
    opacity: 1,
    coordinates: [
      { lat: 41.311, lng: 69.279 },
      { lat: 41.305, lng: 69.288 },
      { lat: 41.298, lng: 69.301 },
      { lat: 41.292, lng: 69.315 },
    ],
  },
  {
    id: "route-2",
    driver_name: "S. Yusupova",
    driver_id: "drv-02",
    opacity: 0.7,
    coordinates: [
      { lat: 41.318, lng: 69.265 },
      { lat: 41.312, lng: 69.272 },
      { lat: 41.307, lng: 69.285 },
      { lat: 41.301, lng: 69.298 },
    ],
  },
  {
    id: "route-3",
    driver_name: "Planned corridor",
    driver_id: "planned",
    opacity: 0.4,
    coordinates: [
      { lat: 41.315, lng: 69.27 },
      { lat: 41.31, lng: 69.282 },
      { lat: 41.304, lng: 69.295 },
    ],
  },
];
