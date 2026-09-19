'use client';

import dynamic from 'next/dynamic';
import React from 'react';

// Lazy-load the Decamodels (Phase 1 & 2)
const FleetRadarCanvas = dynamic(() => import('./canvases/algorithms/FleetRadarCanvas'), { ssr: false });
const NodeNetworkCanvas = dynamic(() => import('./canvases/algorithms/NodeNetworkCanvas'), { ssr: false });
const GridSorterCanvas = dynamic(() => import('./canvases/algorithms/GridSorterCanvas'), { ssr: false });
const DataMeshCanvas = dynamic(() => import('./canvases/algorithms/DataMeshCanvas'), { ssr: false });
const WaveOscillatorCanvas = dynamic(() => import('./canvases/algorithms/WaveOscillatorCanvas'), { ssr: false });

export default function TopicCanvasRouter({ slug }: { slug: string }) {
  let CanvasEngine;
  
  const s = slug.toLowerCase();

  // Phase 3: The Decamodel Registry Mapper - Routing every slug to a physics engine
  if (
    s.includes('fleet') || 
    s.includes('tracking') || 
    s.includes('driver') ||
    s.includes('telemetry') ||
    s.includes('weather') ||
    s.includes('vehicle') ||
    s.includes('transport')
  ) {
    CanvasEngine = FleetRadarCanvas;
  } else if (
    s.includes('route') || 
    s.includes('dispatch') || 
    s.includes('network') ||
    s.includes('topology') ||
    s.includes('zone') ||
    s.includes('node') ||
    s.includes('supplier')
  ) {
    CanvasEngine = NodeNetworkCanvas;
  } else if (
    s.includes('warehouse') || 
    s.includes('fulfillment') || 
    s.includes('inventory') ||
    s.includes('payload') ||
    s.includes('stock') ||
    s.includes('gate') ||
    s.includes('barcode')
  ) {
    CanvasEngine = GridSorterCanvas;
  } else if (
    s.includes('demand') || 
    s.includes('planning') || 
    s.includes('forecast') ||
    s.includes('pulse') ||
    s.includes('timeline') ||
    s.includes('future') ||
    s.includes('analytics')
  ) {
    CanvasEngine = WaveOscillatorCanvas;
  } else {
    // Default fallback (DataMesh is used for finance, security, integration, etc.)
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
