'use client';

import dynamic from 'next/dynamic';
import React from 'react';

// Lazy-load the Decamodels (Phase 1)
const FleetRadarCanvas = dynamic(() => import('./canvases/algorithms/FleetRadarCanvas'), { ssr: false });
const NodeNetworkCanvas = dynamic(() => import('./canvases/algorithms/NodeNetworkCanvas'), { ssr: false });
const GridSorterCanvas = dynamic(() => import('./canvases/algorithms/GridSorterCanvas'), { ssr: false });
const DataMeshCanvas = dynamic(() => import('./canvases/algorithms/DataMeshCanvas'), { ssr: false });

export default function TopicCanvasRouter({ slug }: { slug: string }) {
  let CanvasEngine;

  // The Decamodel Registry Mapper
  if (slug.includes('fleet') || slug.includes('tracking') || slug === 'driver-app') {
    CanvasEngine = FleetRadarCanvas;
  } else if (slug.includes('route') || slug.includes('dispatch') || slug.includes('network')) {
    CanvasEngine = NodeNetworkCanvas;
  } else if (slug.includes('warehouse') || slug.includes('fulfillment') || slug.includes('inventory')) {
    CanvasEngine = GridSorterCanvas;
  } else {
    // Default fallback to Data Mesh (for analytics, ai-copilot, etc. until their custom engines are built)
    CanvasEngine = DataMeshCanvas;
  }

  return (
    <div className="absolute inset-0 w-full h-full bg-[#030303] flex items-center justify-center">
      {/* Subtle overlay gradient to blend edges */}
      <div className="absolute inset-0 bg-gradient-to-tr from-black via-transparent to-white/5 pointer-events-none z-10" />
      <CanvasEngine />
    </div>
  );
}
