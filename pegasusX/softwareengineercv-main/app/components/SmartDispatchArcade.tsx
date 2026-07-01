'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ChamferButton from './ChamferButton';
import { useReducedMotion } from '@/app/hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

const TRUCKS = [
  { id: 'A', slots: [1, 1, 0], score: 94 },
  { id: 'B', slots: [0, 0, 0], score: 88 },
  { id: 'C', slots: [1, 0, 0], score: 71 },
] as const;

type Phase = 'scan' | 'suggest' | 'confirm' | 'locked';

export default function SmartDispatchArcade() {
  const sectionRef = useRef<HTMLElement>(null);
  const cabinetRef = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();
  const [activeTruck, setActiveTruck] = useState(0);
  const [phase, setPhase] = useState<Phase>('scan');
  const [confirmed, setConfirmed] = useState(false);

  useEffect(() => {
    if (!sectionRef.current || !cabinetRef.current || reduced) return;
    gsap.fromTo(
      cabinetRef.current,
      { opacity: 0, y: 36, scale: 0.96 },
      {
        opacity: 1,
        y: 0,
        scale: 1,
        duration: 0.9,
        ease: 'power3.out',
        scrollTrigger: { trigger: sectionRef.current, start: 'top 75%' },
      }
    );
  }, [reduced]);

  useEffect(() => {
    if (reduced) return;

    const sequence: Phase[] = ['scan', 'suggest', 'confirm', 'locked'];
    let step = 0;
    let truck = 0;

    const tick = window.setInterval(() => {
      step = (step + 1) % sequence.length;
      const nextPhase = sequence[step];

      if (nextPhase === 'scan') {
        truck = (truck + 1) % TRUCKS.length;
        setActiveTruck(truck);
        setConfirmed(false);
      }

      if (nextPhase === 'locked') {
        setConfirmed(true);
      }

      setPhase(nextPhase);
    }, 1400);

    return () => window.clearInterval(tick);
  }, [reduced]);

  return (
    <section ref={sectionRef} className="border-t border-white/10 bg-black py-20 md:py-28 text-white">
      <div className="container mx-auto max-w-7xl px-4">
        <div className="mb-10 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="editorial-eyebrow">Smart dispatch</p>
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              Arcade assist — warehouse always in control
            </h2>
            <p className="mt-3 max-w-xl text-sm text-white/55">
              Ranked truck suggestions, never auto-commit. The floor lead confirms every load.
            </p>
          </div>
          <ChamferButton href="/capabilities/smarter-dispatch" variant="ghost">
            Smarter dispatch
          </ChamferButton>
        </div>

        <div
          ref={cabinetRef}
          className="smart-dispatch-arcade mx-auto max-w-4xl border border-white/15 bg-black p-4 md:p-6"
        >
          <div className="flex items-center justify-between border-b border-white/10 pb-3 font-mono text-[10px] uppercase tracking-[0.2em] text-white/45">
            <span>Smart dispatch v1</span>
            <span className="smart-dispatch-arcade__blink">Warehouse online</span>
            <span>Hi-score: 0 misloads</span>
          </div>

          <div className="smart-dispatch-arcade__screen relative mt-4 overflow-hidden border border-white/20 bg-black p-4 md:p-6">
            <div className="smart-dispatch-arcade__scanlines pointer-events-none absolute inset-0" aria-hidden />

            <div className="relative z-[1] space-y-3">
              {TRUCKS.map((truck, index) => {
                const isActive = index === activeTruck;
                const isSuggested = isActive && (phase === 'suggest' || phase === 'confirm' || phase === 'locked');

                return (
                  <div
                    key={truck.id}
                    className={`smart-dispatch-arcade__row flex items-center gap-3 border px-3 py-2 font-mono text-xs transition-colors md:gap-4 md:px-4 md:py-3 ${
                      isSuggested
                        ? 'border-white bg-white/5'
                        : 'border-white/15 text-white/50'
                    } ${confirmed && isActive ? 'smart-dispatch-arcade__row--locked' : ''}`}
                  >
                    <span className="w-16 shrink-0 uppercase tracking-wider text-white/70">
                      Truck {truck.id}
                    </span>
                    <div className="flex flex-1 gap-1.5 md:gap-2">
                      {truck.slots.map((filled, slotIndex) => (
                        <span
                          key={slotIndex}
                          className={`smart-dispatch-arcade__slot h-6 w-6 border md:h-7 md:w-7 ${
                            filled || (isActive && phase === 'locked' && slotIndex < 2)
                              ? 'smart-dispatch-arcade__slot--filled border-white bg-white'
                              : 'border-white/30'
                          } ${isActive && phase === 'scan' ? 'smart-dispatch-arcade__slot--scan' : ''}`}
                        />
                      ))}
                    </div>
                    <span
                      className={`w-14 shrink-0 text-right tabular-nums ${
                        isSuggested ? 'text-white' : 'text-white/35'
                      }`}
                    >
                      {truck.score}%
                    </span>
                    {isSuggested ? (
                      <span className="smart-dispatch-arcade__cursor hidden text-white sm:inline">
                        ◄
                      </span>
                    ) : (
                      <span className="hidden w-4 sm:inline" />
                    )}
                  </div>
                );
              })}
            </div>

            <div className="relative z-[1] mt-6 flex flex-col gap-3 border-t border-white/10 pt-4 sm:flex-row sm:items-center sm:justify-between">
              <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-white/40">
                {phase === 'scan' && 'Scanning yard capacity…'}
                {phase === 'suggest' && 'Suggestion ranked — dashed ghost slot'}
                {phase === 'confirm' && 'Awaiting warehouse confirm'}
                {phase === 'locked' && 'Load committed · manifest issued'}
              </p>
              <div
                className={`smart-dispatch-arcade__confirm inline-flex items-center gap-2 border px-4 py-2 font-mono text-[11px] uppercase tracking-[0.16em] ${
                  phase === 'confirm'
                    ? 'smart-dispatch-arcade__confirm--pulse border-white text-white'
                    : confirmed
                      ? 'border-white bg-white text-black'
                      : 'border-white/25 text-white/40'
                }`}
              >
                <span className="text-white/50">&gt;</span>
                {confirmed ? 'Confirmed' : 'Confirm load'}
              </div>
            </div>
          </div>

          <p className="mt-4 text-center font-mono text-[10px] uppercase tracking-[0.14em] text-white/30">
            Human in the loop · No auto-commit · Audit on override
          </p>
        </div>
      </div>
    </section>
  );
}
