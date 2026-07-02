"use client";

import React, { useState } from "react";
import Map from "react-map-gl";
import DeckGL from "@deck.gl/react";
import { H3HexagonLayer } from "@deck.gl/geo-layers";
import "mapbox-gl/dist/mapbox-gl.css";

const INITIAL_VIEW_STATE = {
  longitude: -122.4,
  latitude: 37.74,
  zoom: 11,
  maxZoom: 20,
  pitch: 30,
  bearing: 0,
};

// Mapbox token should ideally be passed in or loaded from env
const MAPBOX_ACCESS_TOKEN = process.env.NEXT_PUBLIC_MAPBOX_TOKEN || "pk.eyJ1IjoiZGVmYXVsdCIsImEiOiJjbHg1bW14Mm0wMTI2MmpxaXV3eWY2bmM2In0.default";

export interface HexagonalControlTowerMapProps {
  data: { hex: string; count: number }[];
}

export function HexagonalControlTowerMap({ data }: HexagonalControlTowerMapProps) {
  const [viewState, setViewState] = useState(INITIAL_VIEW_STATE);

  const layer = new H3HexagonLayer({
    id: "h3-hexagon-layer",
    data,
    pickable: true,
    wireframe: false,
    filled: true,
    extruded: true,
    elevationScale: 20,
    getHexagon: (d) => d.hex,
    getFillColor: (d) => [255, (1 - d.count / 100) * 255, 0, 200], // Example color scale
    getElevation: (d) => d.count,
  });

  return (
    <div className="relative w-full h-full min-h-[500px]">
      <DeckGL
        layers={[layer]}
        initialViewState={INITIAL_VIEW_STATE}
        controller={true}
        onViewStateChange={(e) => setViewState(e.viewState as any)}
      >
        <Map
          mapboxAccessToken={MAPBOX_ACCESS_TOKEN}
          mapStyle="mapbox://styles/mapbox/dark-v11"
        />
      </DeckGL>
    </div>
  );
}
