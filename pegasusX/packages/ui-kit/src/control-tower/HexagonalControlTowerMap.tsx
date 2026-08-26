"use client";

import React, { useMemo, useState } from "react";
import MapGL from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import DeckGL from "@deck.gl/react";
import { H3HexagonLayer } from "@deck.gl/geo-layers";
import { Map3DViewToggle } from "../desktop/map-chrome";
import "maplibre-gl/dist/maplibre-gl.css";
import { mapInitialViewState, readCachedAuthSession } from "@pegasusx/api-client";
import type { MarketPack } from "@pegasusx/types";

export interface HexagonalControlTowerMapProps {
  data: { hex: string; count: number }[];
  /** Optional extruded hex view — off by default (PX-DESK-3B). */
  show3DViewToggle?: boolean;
  pack?: MarketPack | null;
}

export function HexagonalControlTowerMap({
  data,
  show3DViewToggle = true,
  pack,
}: HexagonalControlTowerMapProps) {
  const [view3D, setView3D] = useState(false);

  const layer = useMemo(
    () =>
      new H3HexagonLayer({
        id: "h3-hexagon-layer",
        data,
        pickable: true,
        wireframe: false,
        filled: true,
        extruded: view3D,
        elevationScale: view3D ? 20 : 0,
        getHexagon: (d) => d.hex,
        getFillColor: (d) => [255, (1 - d.count / 100) * 255, 0, 200],
        getElevation: (d) => (view3D ? d.count : 0),
      }),
    [data, view3D],
  );

  const packCenter = mapInitialViewState(pack ?? readCachedAuthSession()?.pack, 11);
  const initialViewState = {
    ...packCenter,
    maxZoom: 20,
    pitch: view3D ? 30 : 0,
    bearing: 0,
  };

  return (
    <div className="relative h-full min-h-[500px] w-full">
      {show3DViewToggle ? (
        <Map3DViewToggle
          className="absolute left-3 top-3 z-10"
          enabled={view3D}
          onChange={setView3D}
        />
      ) : null}
      <DeckGL
        layers={[layer]}
        initialViewState={initialViewState}
        controller={true}
      >
        <MapGL
          mapLib={maplibregl}
          mapStyle="https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json"
        />
      </DeckGL>
    </div>
  );
}
