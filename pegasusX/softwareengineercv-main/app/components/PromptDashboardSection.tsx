'use client';

import { useLanguage } from '../context/LanguageContext';
import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import PageSection from './layout/PageSection';
import { usePerfProfile } from '../hooks/useDevice';
import { useInView } from '../hooks/useInView';
import { cn } from '@/lib/utils';

gsap.registerPlugin(ScrollTrigger);

/* ── Basedash reference data (Screenshot 5.14.49 AM) ── */
const REVENUE_LINE = [210, 248, 225, 290, 318, 302, 355, 372, 348, 395, 382, 410];
const REVENUE_LINE_ALT = [235, 242, 258, 268, 285, 298, 312, 328, 338, 352, 365, 378];
const SUBSCRIPTION_BARS = [
  120, 165, 140, 200, 245, 280, 260, 310, 340, 320, 370, 395, 380, 420, 450, 435, 475, 490,
  465, 500, 508, 480, 455, 490, 475, 508, 495, 508,
];

const COHORT_ROW = { cohort: '1', w1: 248, w2: 200, w3: 140, w4: 108, w5: 64, w6: 25 };

const PROMPT_TEXT_EN = 'Generate a dashboard showing our core growth KPIs';
const PROMPT_TEXT_MOBILE_EN = 'Generate a core growth KPIs dashboard';
const PROMPT_TEXT_RU = 'Сгенерируй дашборд с ключевыми KPI роста';
const PROMPT_TEXT_MOBILE_RU = 'Сгенерируй дашборд KPI роста';

const TITLE_EN = 'Create dashboards with a prompt';
const TITLE_RU = 'Создавайте дашборды промптом';

function SectionTitle({ compact, title }: { compact: boolean; title: string }) {
  return (
    <h2
      className={cn(
        'text-center font-normal leading-[1.08] tracking-[-0.04em] text-white max-w-4xl mx-auto',
        compact
          ? 'text-[1.75rem] sm:text-[2.1rem] md:text-[2.6rem] lg:text-[clamp(2.25rem,4.8vw,3.5rem)]'
          : 'text-[clamp(2.25rem,4.8vw,3.5rem)]'
      )}
    >
      {title}
    </h2>
  );
}

function MetricChange({
  pct,
  delta,
  positive,
}: {
  pct: string;
  delta: string;
  positive: boolean;
}) {
  return (
    <span
      className={cn(
        'text-xs sm:text-sm font-normal tabular-nums',
        positive ? 'text-emerald-400' : 'text-rose-400'
      )}
    >
      {pct}{' '}
      <span className={positive ? 'text-emerald-400/80' : 'text-rose-400/80'}>({delta})</span>
    </span>
  );
}

function BentoCard({ className, children }: { className?: string; children: React.ReactNode }) {
  return (
    <div
      className={cn(
        'rounded-xl border border-white/[0.12] bg-[#000000] p-4 sm:p-5 md:p-6 overflow-hidden',
        className
      )}
    >
      {children}
    </div>
  );
}

function RevenueChart({ animate }: { animate: boolean }) {
  const padL = 14;
  const padR = 4;
  const padT = 4;
  const padB = 4;
  const w = 100;
  const h = 64;
  const innerW = w - padL - padR;
  const innerH = h - padT - padB;
  const yMin = 100;
  const yMax = 400;

  const toPoints = (values: number[]) =>
    values
      .map((v, i) => {
        const x = padL + (i / (values.length - 1)) * innerW;
        const y = padT + innerH - ((v - yMin) / (yMax - yMin)) * innerH;
        return `${x},${y}`;
      })
      .join(' ');

  const yLabels = [400, 300, 200, 100];

  return (
    <div className="mt-3 w-full overflow-hidden">
      <div className="relative h-[8.5rem] sm:h-[9rem] overflow-hidden">
        <svg
          viewBox={`0 0 ${w} ${h}`}
          className="absolute inset-0 h-full w-full"
          preserveAspectRatio="xMidYMid meet"
          aria-hidden
        >
          {yLabels.map((label) => {
            const y = padT + innerH - ((label - yMin) / (yMax - yMin)) * innerH;
            return (
              <g key={label}>
                <line
                  x1={padL}
                  y1={y}
                  x2={w - padR}
                  y2={y}
                  stroke="rgba(255,255,255,0.06)"
                  strokeWidth="0.4"
                />
                <text
                  x={padL - 2}
                  y={y + 1}
                  textAnchor="end"
                  fill="rgba(255,255,255,0.28)"
                  fontSize="3.2"
                  fontFamily="ui-monospace, monospace"
                >
                  {label}
                </text>
              </g>
            );
          })}
          <polyline
            points={toPoints(REVENUE_LINE_ALT)}
            fill="none"
            stroke="rgba(251,113,133,0.55)"
            strokeWidth="1.1"
            strokeLinecap="round"
            strokeLinejoin="round"
            className={cn(animate && 'prompt-dash-chart-in prompt-dash-chart-in--delay')}
          />
          <polyline
            points={toPoints(REVENUE_LINE)}
            fill="none"
            stroke="rgba(255,255,255,0.92)"
            strokeWidth="1.35"
            strokeLinecap="round"
            strokeLinejoin="round"
            className={cn(animate && 'prompt-dash-chart-in')}
          />
        </svg>
      </div>
      <div className="mt-1.5 flex justify-between pl-8 text-[10px] font-mono text-white/30">
        <span>Aug 1, 2026</span>
        <span>Aug 31, 2026</span>
      </div>
    </div>
  );
}

function SubscriptionsChart({ animate }: { animate: boolean }) {
  const yMax = 500;
  const yLabels = [500, 250];

  return (
    <div className="mt-3 w-full overflow-hidden">
      <div className="relative h-[8.5rem] sm:h-[9rem] overflow-hidden">
        {/* Y-axis */}
        <div className="pointer-events-none absolute left-0 top-0 bottom-6 z-10 flex w-7 flex-col justify-between text-[10px] font-mono text-white/28">
          {yLabels.map((label) => (
            <span key={label}>{label}</span>
          ))}
        </div>

        {/* Bars */}
        <div className="ml-7 flex h-full items-end gap-[1.5px] overflow-hidden pb-6">
          {SUBSCRIPTION_BARS.map((v, i) => (
            <div key={i} className="flex h-full min-w-0 flex-1 flex-col justify-end overflow-hidden">
              <div
                className={cn(
                  'w-full max-h-full rounded-t-[1px] bg-white/88 origin-bottom',
                  animate && 'prompt-dash-bar'
                )}
                style={{
                  height: `${(v / yMax) * 100}%`,
                  animationDelay: animate ? `${i * 20}ms` : undefined,
                }}
              />
            </div>
          ))}
        </div>
      </div>
      <div className="mt-1.5 flex justify-between pl-7 text-[10px] font-mono text-white/30">
        <span>Aug 1, 2026</span>
        <span>Aug 31, 2026</span>
      </div>
    </div>
  );
}

function PromptOverlay({
  reducedMotion,
  isLowEnd,
  isMobile,
  inView,
  promptText,
  promptTextMobile,
}: {
  reducedMotion: boolean;
  isLowEnd: boolean;
  isMobile: boolean;
  inView: boolean;
  promptText: string;
  promptTextMobile: string;
}) {
  const [cursorOn, setCursorOn] = useState(true);
  const showFx = !reducedMotion && !isLowEnd && inView;

  useEffect(() => {
    if (reducedMotion) {
      setCursorOn(false);
      return;
    }
    const id = window.setInterval(() => setCursorOn((v) => !v), 530);
    return () => window.clearInterval(id);
  }, [reducedMotion]);

  return (
    <div className="relative w-full max-w-[36rem] mx-auto px-4">
      {showFx && (
        <div
          className="pointer-events-none absolute left-1/2 top-1/2 h-[4.5rem] w-[min(140vw,64rem)] -translate-x-1/2 -translate-y-1/2"
          aria-hidden
        >
          <div className="absolute inset-0 bg-[linear-gradient(90deg,transparent_0%,rgba(124,58,237,0.1)_18%,rgba(167,139,250,0.42)_50%,rgba(124,58,237,0.1)_82%,transparent_100%)] blur-[6px]" />
          <div className="absolute inset-0 opacity-50 [background-image:repeating-linear-gradient(90deg,rgba(255,255,255,0.1)_0,rgba(255,255,255,0.1)_1px,transparent_1px,transparent_4px)]" />
          <div className="absolute top-1/2 inset-x-[10%] h-px -translate-y-1/2 bg-violet-300/40" />
        </div>
      )}
      <div
        className={cn(
          'relative rounded-full border border-violet-400/30 px-5 py-3 sm:px-6 sm:py-3.5',
          'bg-[linear-gradient(180deg,rgba(88,28,180,0.88),rgba(49,16,98,0.94))]',
          'shadow-[0_0_40px_rgba(124,58,237,0.35),inset_0_1px_0_rgba(255,255,255,0.1)]',
          showFx && 'prompt-dash-prompt-glow'
        )}
      >
        <p className="text-center text-[0.8rem] sm:text-sm text-white/95 font-light leading-snug">
          {isMobile ? promptTextMobile : promptText}
          {!reducedMotion && (
            <span
              className={cn(
                'inline-block w-[2px] h-[1em] align-[-0.1em] ml-0.5 bg-white/80',
                cursorOn ? 'opacity-100' : 'opacity-0'
              )}
              aria-hidden
            />
          )}
        </p>
      </div>
    </div>
  );
}

export default function PromptDashboardSection() {
  const { t, language } = useLanguage();

  const PROMPT_TEXT = language === 'ru' ? PROMPT_TEXT_RU : PROMPT_TEXT_EN;
  const PROMPT_TEXT_MOBILE = language === 'ru' ? PROMPT_TEXT_MOBILE_RU : PROMPT_TEXT_MOBILE_EN;
  const TITLE = language === 'ru' ? TITLE_RU : TITLE_EN;

  const { isMobile, isTablet, prefersReducedMotion, isLowEnd } = usePerfProfile();
  const { ref: sectionRef, isInView } = useInView<HTMLElement>({ rootMargin: '80px' });
  const headerRef = useRef<HTMLDivElement>(null);
  const promptRef = useRef<HTMLDivElement>(null);
  const [chartsAnimated, setChartsAnimated] = useState(false);
  const compact = isMobile || isTablet;

  useEffect(() => {
    if (isInView && !prefersReducedMotion && !isLowEnd) setChartsAnimated(true);
  }, [isInView, prefersReducedMotion, isLowEnd]);

  useEffect(() => {
    const section = sectionRef.current;
    if (!section || !headerRef.current || prefersReducedMotion || isMobile || isLowEnd) return;

    const ctx = gsap.context(() => {
      gsap.from(headerRef.current, {
        opacity: 0,
        y: 18,
        duration: 0.55,
        ease: 'power3.out',
        scrollTrigger: { trigger: section, start: 'top 85%', once: true },
      });
      if (promptRef.current) {
        gsap.from(promptRef.current, {
          opacity: 0,
          scale: 0.98,
          duration: 0.4,
          ease: 'back.out(1.4)',
          scrollTrigger: { trigger: section, start: 'top 80%', once: true },
          delay: 0.2,
        });
      }
    }, section);

    return () => ctx.revert();
  }, [prefersReducedMotion, isMobile, isLowEnd, sectionRef]);

  return (
    <PageSection
      ref={sectionRef}
      id="prompt-dashboard"
      bleed
      className="overflow-hidden border-t border-white/5 !py-12 sm:!py-16 md:!py-20"
    >
      <div className="max-w-[68rem] mx-auto px-4 sm:px-6 md:px-10 lg:px-12">
        {/* Header */}
        <div ref={headerRef} className="text-center mb-10 sm:mb-12 md:mb-14">
          <SectionTitle compact={compact} title={TITLE} />
          <p className="mt-4 sm:mt-5 text-sm sm:text-base text-white/50 max-w-xl mx-auto leading-relaxed font-light">
            {t(
              'prompt_subtitle',
              'Describe what you want to track and let Pegasus generate a custom dashboard for you in minutes.'
            )}
          </p>
        </div>

        {/* Bento dashboard */}
        <div className="relative">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-12 gap-3 md:gap-3.5">
            {/* Revenue */}
            <BentoCard className="lg:col-span-6">
              <p className="text-xs sm:text-sm text-white/45 font-normal">Revenue</p>
              <p className="mt-1 text-3xl sm:text-[2.35rem] font-light tracking-tight text-white">
                $28.3K
              </p>
              <MetricChange pct="+68%" delta="16,800" positive />
              <RevenueChart animate={chartsAnimated} />
            </BentoCard>

            {/* New subscriptions */}
            <BentoCard className="lg:col-span-6">
              <div className="flex items-start justify-between gap-2">
                <p className="text-xs sm:text-sm text-white/45 font-normal">{t('prompt_new_subs', 'New subscriptions')}</p>
                <span className="inline-flex items-center gap-1.5 text-[10px] sm:text-xs text-emerald-400">
                  <span
                    className={cn(
                      'h-1.5 w-1.5 rounded-full bg-emerald-400',
                      !prefersReducedMotion && 'prompt-dash-live'
                    )}
                  />
                  Live
                </span>
              </div>
              <p className="mt-1 text-3xl sm:text-[2.35rem] font-light tracking-tight text-white">
                508
              </p>
              <MetricChange pct="+189%" delta="176" positive />
              <SubscriptionsChart animate={chartsAnimated} />
            </BentoCard>

            {/* Mobile / tablet prompt */}
            <div className="sm:col-span-2 lg:hidden py-2">
              <PromptOverlay
                reducedMotion={prefersReducedMotion}
                isLowEnd={isLowEnd}
                isMobile={isMobile}
                inView={isInView}
                promptText={PROMPT_TEXT}
                promptTextMobile={PROMPT_TEXT_MOBILE}
              />
            </div>

            {/* CAC */}
            <BentoCard className="sm:col-span-1 lg:col-span-3 flex min-h-[7.5rem] flex-col items-center justify-center text-center">
              <p className="text-xs sm:text-sm text-white/45">CAC</p>
              <p className="mt-2 text-2xl sm:text-3xl font-light tracking-tight text-white">
                14 <span className="text-lg sm:text-xl text-white/75">wks</span>
              </p>
            </BentoCard>

            {/* Burnrate */}
            <BentoCard className="sm:col-span-1 lg:col-span-3 flex min-h-[7.5rem] flex-col items-center justify-center text-center">
              <p className="text-xs sm:text-sm text-white/45">Burnrate</p>
              <p className="mt-2 text-2xl sm:text-3xl font-light tracking-tight text-white">
                $32.5K
              </p>
            </BentoCard>

            {/* Cohort retention */}
            <BentoCard className="sm:col-span-2 lg:col-span-6">
              <p className="text-xs sm:text-sm text-white/45">{t('prompt_cohort', 'Cohort retention')}</p>
              <div className="mt-1 flex flex-wrap items-baseline gap-x-3 gap-y-1">
                <p className="text-2xl sm:text-3xl font-light tracking-tight text-white">457</p>
                <MetricChange pct="-31%" delta="659" positive={false} />
              </div>
              <div className="mt-4 overflow-x-auto">
                <table className="w-full min-w-[20rem] text-left text-xs">
                  <thead>
                    <tr className="text-white/35">
                      <th className="pb-2 pr-4 font-normal">Cohort</th>
                      <th className="pb-2 pr-3 font-normal">Week 1</th>
                      <th className="pb-2 pr-3 font-normal">Week 2</th>
                      <th className="pb-2 pr-3 font-normal">Week 3</th>
                      <th className="pb-2 pr-3 font-normal">Week 4</th>
                      <th className="pb-2 pr-3 font-normal">Week 5</th>
                      <th className="pb-2 font-normal">Week 6</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr className="border-t border-white/10 text-white/70">
                      <td className="py-2.5 pr-4">{COHORT_ROW.cohort}</td>
                      <td className="py-2.5 pr-3 tabular-nums">{COHORT_ROW.w1}</td>
                      <td className="py-2.5 pr-3 tabular-nums">{COHORT_ROW.w2}</td>
                      <td className="py-2.5 pr-3 tabular-nums">{COHORT_ROW.w3}</td>
                      <td className="py-2.5 pr-3 tabular-nums">{COHORT_ROW.w4}</td>
                      <td className="py-2.5 pr-3 tabular-nums">{COHORT_ROW.w5}</td>
                      <td className="py-2.5 tabular-nums">{COHORT_ROW.w6}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </BentoCard>
          </div>

          {/* Desktop prompt — centered overlay between rows (Basedash reference) */}
          <div
            ref={promptRef}
            className="pointer-events-none absolute inset-x-0 top-[42%] z-30 hidden -translate-y-1/2 lg:block"
          >
            <PromptOverlay
              reducedMotion={prefersReducedMotion}
              isLowEnd={isLowEnd}
              isMobile={false}
              inView={isInView}
              promptText={PROMPT_TEXT}
              promptTextMobile={PROMPT_TEXT_MOBILE}
            />
          </div>
        </div>
      </div>
    </PageSection>
  );
}
