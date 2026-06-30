'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import type { FlowConfig } from '@/app/data/topicTypes';
import { useReducedMotion } from '@/app/hooks/useDevice';
import { FlowShell } from './FlowShell';

gsap.registerPlugin(ScrollTrigger);

type Props = { config?: FlowConfig };

export default function TopologyMapFlow({ config }: Props) {
  const ref = useRef<SVGSVGElement>(null);
  const reduced = useReducedMotion();
  const highlight = config?.highlightStep ?? 0;

  useEffect(() => {
    if (reduced || !ref.current) return;
    gsap.fromTo(
      ref.current.querySelectorAll('.topo-node'),
      { opacity: 0.2 },
      { opacity: 1, stagger: 0.12, scrollTrigger: { trigger: ref.current, start: 'top 78%' } }
    );
  }, [reduced]);

  const nodes = [
    { id: 'supplier', label: 'Supplier', x: 120, y: 60 },
    { id: 'wh', label: 'Warehouse', x: 320, y: 40 },
    { id: 'factory', label: 'Factory', x: 520, y: 60 },
    { id: 'zone', label: 'Zone', x: 400, y: 160, highlight: highlight === 0 },
    { id: 'retailer', label: 'Retailer', x: 640, y: 140 },
  ];

  return (
    <FlowShell title="Network topology">
      <svg ref={ref} viewBox="0 0 760 220" className="w-full h-auto">
        <line x1={120} y1={60} x2={320} y2={40} stroke="white" strokeOpacity={0.2} />
        <line x1={320} y1={40} x2={520} y2={60} stroke="white" strokeOpacity={0.2} />
        <line x1={400} y1={160} x2={640} y2={140} stroke="white" strokeOpacity={0.2} />
        {nodes.map((n) => (
          <g key={n.id} className="topo-node">
            <rect
              x={n.x - 50}
              y={n.y - 16}
              width={100}
              height={32}
              fill={n.highlight ? 'white' : 'none'}
              stroke="white"
              strokeOpacity={n.highlight ? 1 : 0.5}
            />
            <text
              x={n.x}
              y={n.y + 4}
              textAnchor="middle"
              className={`text-[10px] font-mono uppercase ${n.highlight ? 'fill-black' : 'fill-white'}`}
            >
              {n.label}
            </text>
          </g>
        ))}
      </svg>
    </FlowShell>
  );
}
