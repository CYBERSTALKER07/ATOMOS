"use client";

import { useMemo } from "react";
import MapGL, { Layer, NavigationControl, Source } from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import { DEMO_FLEET_ROUTES } from "@/lib/mock-data/fleet-routes";

const DEFAULT_CENTER: [number, number] = [69.29, 41.305];

type DemoFleetMapProps = {
  className?: string;
};

export function DemoFleetMap({ className = "" }: DemoFleetMapProps) {
  const lineCollection = useMemo(() => {
    const features = DEMO_FLEET_ROUTES.map((route) => ({
      type: "Feature" as const,
      properties: { opacity: route.opacity },
      geometry: {
        type: "LineString" as const,
        coordinates: route.coordinates.map((p) => [p.lng, p.lat]),
      },
    }));
    return { type: "FeatureCollection" as const, features };
  }, []);

  const pointCollection = useMemo(() => {
    const features = DEMO_FLEET_ROUTES.flatMap((route) => {
      const last = route.coordinates[route.coordinates.length - 1];
      if (!last) return [];
      return [{
        type: "Feature" as const,
        properties: { name: route.driver_name, opacity: route.opacity },
        geometry: { type: "Point" as const, coordinates: [last.lng, last.lat] },
      }];
    });
    return { type: "FeatureCollection" as const, features };
  }, []);

  return (
    <div className={className}>
      <MapGL
        mapLib={maplibregl}
        initialViewState={{ longitude: DEFAULT_CENTER[0], latitude: DEFAULT_CENTER[1], zoom: 12 }}
        mapStyle="https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json"
        style={{ width: "100%", height: "100%" }}
      >
        <NavigationControl position="top-right" showCompass={false} />
        <Source id="demo-routes" type="geojson" data={lineCollection}>
          <Layer
            id="demo-routes-line"
            type="line"
            paint={{
              "line-color": "#ffffff",
              "line-width": 2,
              "line-opacity": ["get", "opacity"],
            }}
          />
        </Source>
        <Source id="demo-drivers" type="geojson" data={pointCollection}>
          <Layer
            id="demo-drivers-circle"
            type="circle"
            paint={{
              "circle-color": "#ffffff",
              "circle-radius": 7,
              "circle-stroke-width": 2,
              "circle-stroke-color": "#000000",
              "circle-opacity": ["get", "opacity"],
            }}
          />
        </Source>
      </MapGL>
    </div>
  );
}
