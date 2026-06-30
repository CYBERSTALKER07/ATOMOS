'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import type { FlowConfig } from '@/app/data/topicTypes';
import { useReducedMotion } from '@/app/hooks/useDevice';
import { FlowShell } from './FlowShell';

gsap.registerPlugin(ScrollTrigger);

type Props = { config?: FlowConfig };

export default function FleetMapFlow({ config }: Props) {
  const ref = useRef<SVGSVGElement>(null);
  const reduced = useReducedMotion();
  const progress = (config?.highlightStep ?? 3) / 6;

  useEffect(() => {
    if (reduced || !ref.current) return;
    const planned = ref.current.querySelector('.route-planned');
    const actual = ref.current.querySelector('.route-actual');
    if (!planned || !actual) return;
    gsap.fromTo(
      [planned, actual],
      { strokeDashoffset: 400 },
      {
        strokeDashoffset: 0,
        scrollTrigger: { trigger: ref.current, start: 'top 75%', end: 'bottom 45%', scrub: 1 },
      }
    );
  }, [reduced]);

  const path = 'M 40 180 Q 200 40 400 120 T 760 80';

  return (
    <FlowShell title="Planned vs actual route">
      <svg ref={ref} viewBox="0 0 800 220" className="w-full h-auto">
        <path d={path} fill="none" stroke="white" strokeOpacity={0.2} strokeWidth={2} strokeDasharray="8 6" />
        <path
          className="route-planned"
          d={path}
          fill="none"
          stroke="white"
          strokeOpacity={0.5}
          strokeWidth={2}
          strokeDasharray="400"
          strokeDashoffset={400 * (1 - progress)}
        />
        <path
          className="route-actual"
          d="M 40 180 Q 220 60 420 100 T 760 100"
          fill="none"
          stroke="#f97316"
          strokeWidth={2}
          strokeDasharray="400"
          strokeDashoffset={400}
        />
        <circle cx={40 + 720 * progress} cy={80 + 40 * progress} r={8} fill="white" />
        <text x={20} y={30} fill="white" className="text-[10px] font-mono opacity-60">
          PLANNED
        </text>
        <text x={20} y={50} fill="#f97316" className="text-[10px] font-mono">
          ACTUAL
        </text>
      </svg>
    </FlowShell>
  );
}
