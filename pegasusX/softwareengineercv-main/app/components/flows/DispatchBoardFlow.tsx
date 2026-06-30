'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import type { FlowConfig } from '@/app/data/topicTypes';
import { useReducedMotion } from '@/app/hooks/useDevice';
import { FlowShell } from './FlowShell';

gsap.registerPlugin(ScrollTrigger);

type Props = { config?: FlowConfig };

export default function DispatchBoardFlow({ config }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();
  const trucks = 3;
  const orders = config?.highlightStep !== undefined ? config.highlightStep + 2 : 4;

  useEffect(() => {
    if (reduced || !ref.current) return;
    gsap.fromTo(
      ref.current.querySelectorAll('.dispatch-chip'),
      { opacity: 0, y: 12 },
      { opacity: 1, y: 0, stagger: 0.08, scrollTrigger: { trigger: ref.current, start: 'top 78%' } }
    );
  }, [reduced]);

  return (
    <FlowShell title="Visual dispatch board">
      <div ref={ref} className="grid gap-4 md:grid-cols-3">
        {Array.from({ length: trucks }).map((_, t) => (
          <div key={t} className="border border-white/20 p-4">
            <p className="mb-3 font-mono text-xs uppercase text-white/60">Truck {t + 1}</p>
            <div className="flex flex-wrap gap-2">
              {Array.from({ length: Math.max(1, orders - t) }).map((_, o) => (
                <span
                  key={o}
                  className="dispatch-chip border border-white/40 px-2 py-1 font-mono text-[10px] uppercase"
                >
                  Order
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </FlowShell>
  );
}
