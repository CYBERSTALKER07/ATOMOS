'use client';

import { useEffect, useRef } from 'react';
import { gsap, ScrollTrigger } from '@/app/lib/gsap';
import type { FlowConfig } from '@/app/data/topicTypes';
import { usePerfProfile } from '@/app/hooks/useDevice';
import { FlowShell } from './FlowShell';

type Props = { config?: FlowConfig };

export default function DispatchBoardFlow({ config }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const { isLowEnd, prefersReducedMotion } = usePerfProfile();
  const reduced = prefersReducedMotion || isLowEnd;
  const trucks = 3;
  const orders = config?.highlightStep !== undefined ? config.highlightStep + 2 : 4;

  useEffect(() => {
    if (!ref.current) return;
    const chips = ref.current.querySelectorAll('.dispatch-chip');

    if (reduced) {
      gsap.set(chips, { opacity: 1, y: 0 });
      return;
    }

    const ctx = gsap.context(() => {
      gsap.fromTo(
        chips,
        { opacity: 0, y: 12 },
        { opacity: 1, y: 0, stagger: 0.08, ease: 'pegasus', scrollTrigger: { trigger: ref.current, start: 'top 78%', fastScrollEnd: true } }
      );
    }, ref);

    return () => ctx.revert();
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
