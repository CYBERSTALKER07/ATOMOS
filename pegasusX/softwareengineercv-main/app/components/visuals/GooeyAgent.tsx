'use client';

import React from 'react';
import Image from 'next/image';

interface GooeyAgentProps {
  className?: string;
  size?: number;
}

export default function GooeyAgent({
  className = '',
  size = 120,
}: GooeyAgentProps) {
  return (
    <div className={`relative flex items-center justify-center cursor-pointer ${className}`} style={{ width: size, height: size }}>
      <Image
        src="/bloub-default-cycle.gif"
        alt="Animated Grok Bot"
        width={size}
        height={size}
        unoptimized // Important for GIFs so Next.js doesn't freeze them into static WebP
        className="w-full h-full object-contain mix-blend-screen"
        priority
      />
    </div>
  );
}
