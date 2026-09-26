'use client';

import React from 'react';
import PegasusInfraIsometric from './PegasusInfraIsometric';

type PegasusInfraSectionProps = {
  id?: string;
  className?: string;
};

export default function PegasusInfraSection({ id, className = '' }: PegasusInfraSectionProps) {
  return (
    <div
      id={id}
      aria-label="Pegasus Infrastructure Overview"
      className={`relative w-full bg-black overflow-hidden text-white ${className}`}
    >
      <PegasusInfraIsometric />
    </div>
  );
}
