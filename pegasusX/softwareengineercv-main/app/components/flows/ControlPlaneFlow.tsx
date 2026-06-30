'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import type { FlowConfig } from '@/app/data/topicTypes';
import { useReducedMotion } from '@/app/hooks/useDevice';
import { FlowShell } from './FlowShell';

gsap.registerPlugin(ScrollTrigger);

const ROLES = ['Supplier', 'Warehouse', 'Factory', 'Driver', 'Retailer', 'Payload'];

type Props = { config?: FlowConfig };

export default function ControlPlaneFlow({ config }: Props) {
  const ref = useRef<SVGSVGElement>(null);
  const reduced = useReducedMotion();
  const roles = config?.roles ?? ROLES;

  useEffect(() => {
    if (reduced || !ref.current) return;
    const nodes = ref.current.querySelectorAll('.role-node');
    gsap.fromTo(
      nodes,
      { opacity: 0.3, scale: 0.9 },
      {
        opacity: 1,
        scale: 1,
        stagger: 0.15,
        scrollTrigger: { trigger: ref.current, start: 'top 80%' },
      }
    );
  }, [reduced]);

  const cx = 400;
  const cy = 140;
  const r = 100;

  return (
    <FlowShell title="ATOMOS control plane">
      <svg ref={ref} viewBox="0 0 800 280" className="w-full h-auto" aria-hidden>
        <circle cx={cx} cy={cy} r={36} fill="white" className="opacity-90" />
        <text x={cx} y={cy + 4} textAnchor="middle" className="fill-black text-[11px] font-mono">
          SPANNER
        </text>
        {roles.map((role, i) => {
          const angle = (i / roles.length) * Math.PI * 2 - Math.PI / 2;
          const x = cx + Math.cos(angle) * r;
          const y = cy + Math.sin(angle) * r;
          return (
            <g key={role} className="role-node">
              <line x1={cx} y1={cy} x2={x} y2={y} stroke="white" strokeOpacity={0.25} />
              <rect x={x - 44} y={y - 14} width={88} height={28} fill="none" stroke="white" strokeOpacity={0.6} />
              <text x={x} y={y + 4} textAnchor="middle" fill="white" className="text-[10px] font-mono uppercase">
                {role}
              </text>
            </g>
          );
        })}
      </svg>
    </FlowShell>
  );
}
