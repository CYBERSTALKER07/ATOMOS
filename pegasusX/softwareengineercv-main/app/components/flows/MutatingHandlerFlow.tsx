'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import type { FlowConfig } from '@/app/data/topicTypes';
import { useReducedMotion } from '@/app/hooks/useDevice';
import { FlowShell, StepNode } from './FlowShell';

gsap.registerPlugin(ScrollTrigger);

const STEPS = ['Verify', 'Validate', 'Save', 'Refresh', 'Notify'];

type Props = { config?: FlowConfig };

export default function MutatingHandlerFlow({ config }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();
  const highlight = config?.highlightStep ?? 2;

  useEffect(() => {
    if (reduced || !ref.current) return;
    const bar = ref.current.querySelector('.flow-progress');
    if (!bar) return;
    gsap.fromTo(
      bar,
      { scaleX: 0 },
      { scaleX: 1, ease: 'none', scrollTrigger: { trigger: ref.current, start: 'top 70%', end: 'bottom 50%', scrub: 1 } }
    );
  }, [reduced]);

  return (
    <FlowShell title="Mutating handler contract">
      <div ref={ref} className="relative">
        <div className="absolute left-0 top-5 h-px w-full bg-white/20" />
        <div
          className="flow-progress absolute left-0 top-5 h-px w-full origin-left bg-white"
          style={{ transform: 'scaleX(0)' }}
        />
        <div className="relative flex flex-wrap justify-between gap-4">
          {STEPS.map((label, i) => (
            <StepNode key={label} label={label} index={i} active={i <= highlight} />
          ))}
        </div>
      </div>
    </FlowShell>
  );
}
