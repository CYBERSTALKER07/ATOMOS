'use client';

import { useEffect, useRef } from 'react';
import { gsap, ScrollTrigger } from '@/app/lib/gsap';
import type { FlowConfig } from '@/app/data/topicTypes';
import { usePerfProfile } from '@/app/hooks/useDevice';
import { FlowShell, StepNode } from './FlowShell';

const STEPS = ['Placed', 'Vetted', 'Loaded', 'In Transit', 'Arrived', 'Paid', 'Completed'];

type Props = { config?: FlowConfig };

export default function OrderLifecycleFlow({ config }: Props) {
 const ref = useRef<HTMLDivElement>(null);
 const { isLowEnd, prefersReducedMotion } = usePerfProfile();
 const reduced = prefersReducedMotion || isLowEnd;
 const highlight = config?.highlightStep ?? 3;

 useEffect(() => {
 if (!ref.current) return;
 const steps = ref.current.querySelectorAll('.flow-step');

 if (reduced) {
 gsap.set(steps, { opacity: 1, scale: 1 });
 return;
 }

 const ctx = gsap.context(() => {
 const tl = gsap.timeline({
 scrollTrigger: { trigger: ref.current, start: 'top 75%', end: 'bottom 40%', scrub: 1, fastScrollEnd: true },
 });
 steps.forEach((step, i) => {
 tl.to(step, { opacity: 1, scale: 1.05, duration: 0.5 }, i * 0.4);
 if (i < steps.length - 1) tl.to(step, { opacity: 0.4, scale: 1, duration: 0.2 }, i * 0.4 + 0.35);
 });
 }, ref);

 return () => ctx.revert();
 }, [reduced]);

 return (
 <FlowShell title="Order lifecycle">
 <div ref={ref} className="flex flex-wrap items-start justify-between gap-4 md:gap-2">
 {STEPS.map((label, i) => (
 <StepNode key={label} label={label} index={i} active={i <= highlight} />
 ))}
 </div>
 </FlowShell>
 );
}
