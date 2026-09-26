'use client';

import { useEffect, useRef } from 'react';
import { gsap, ScrollTrigger } from '@/app/lib/gsap';
import type { FlowConfig } from '@/app/data/topicTypes';
import { usePerfProfile } from '@/app/hooks/useDevice';
import { FlowShell, StepNode } from './FlowShell';

const STEPS = ['Confirm', 'Change event', 'Notify', 'Refresh', 'Live update'];

type Props = { config?: FlowConfig };

export default function RealtimePipelineFlow({ config }: Props) {
 const ref = useRef<HTMLDivElement>(null);
 const { isLowEnd, prefersReducedMotion } = usePerfProfile();
 const reduced = prefersReducedMotion || isLowEnd;
 const highlight = config?.highlightStep ?? 4;

 useEffect(() => {
 if (!ref.current) return;
 const pulses = ref.current.querySelectorAll('.flow-pulse');

 if (reduced) {
 gsap.set(pulses, { opacity: 1, x: 0 });
 return;
 }

 const ctx = gsap.context(() => {
 gsap.to(pulses, {
 opacity: 1,
 x: 0,
 stagger: 0.2,
 ease: 'pegasus',
 scrollTrigger: { trigger: ref.current, start: 'top 75%', fastScrollEnd: true },
 });
 }, ref);

 return () => ctx.revert();
 }, [reduced]);

 return (
 <FlowShell title="Reliable update pipeline">
 <div ref={ref} className="flex flex-wrap items-center justify-between gap-3">
 {STEPS.map((label, i) => (
 <div key={label} className="flow-pulse flex items-center gap-2 opacity-40" style={{ transform: 'translateX(-8px)' }}>
 <StepNode label={label} index={i} active={i <= highlight} />
 {i < STEPS.length - 1 ? <span className="hidden text-white/30 md:inline">→</span> : null}
 </div>
 ))}
 </div>
 </FlowShell>
 );
}
