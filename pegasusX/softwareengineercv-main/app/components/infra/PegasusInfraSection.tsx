'use client';

import React, { useState } from 'react';
import type { InfraNode } from './types';
import PegasusInfraIsometric from './PegasusInfraIsometric';
import PegasusInfraDrawer from './PegasusInfraDrawer';

type PegasusInfraSectionProps = {
  id?: string;
  className?: string;
};

export default function PegasusInfraSection({ id, className = '' }: PegasusInfraSectionProps) {
  const [selectedNode, setSelectedNode] = useState<InfraNode | null>(null);

  return (
    <div
      id={id}
      aria-label="Pegasus Infrastructure Overview"
      className={`relative w-full bg-black py-4 sm:py-6 overflow-hidden text-white ${className}`}
    >
      <div className="w-full max-w-[1380px] mx-auto">
        {/* Master Isometric Architecture Topology Canvas */}
        <PegasusInfraIsometric
          activeLayer="all"
          onSelectNode={(node) => setSelectedNode(node)}
          selectedNodeId={selectedNode?.id}
        />
      </div>

      {/* Detail Node Modal / Drawer */}
      <PegasusInfraDrawer
        node={selectedNode}
        onClose={() => setSelectedNode(null)}
      />
    </div>
  );
}
