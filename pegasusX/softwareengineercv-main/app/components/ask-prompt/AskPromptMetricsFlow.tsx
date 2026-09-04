'use client';

import { cn } from '@/lib/utils';
import type { AskPromptMetric } from './types';

function VerifiedBadge() {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/40 bg-emerald-500/10 px-2 py-0.5 text-[10px] sm:text-[11px] font-medium text-emerald-400">
      <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden>
        <circle cx="5" cy="5" r="4" stroke="currentColor" strokeWidth="1" />
        <path d="M3 5l1.5 1.5L7 4" stroke="currentColor" strokeWidth="1" strokeLinecap="round" />
      </svg>
      Verified
    </span>
  );
}

function QueryBlock({ tokens }: { tokens: AskPromptMetric['queryLines'] }) {
  const colorMap = {
    keyword: 'text-violet-400',
    function: 'text-sky-400',
    string: 'text-emerald-400/90',
    identifier: 'text-white/85',
    plain: 'text-white/70',
  } as const;

  return (
    <pre className="ask-metrics-code mt-4 overflow-x-auto rounded-lg border border-white/[0.08] bg-[#050505] p-4 text-[11px] sm:text-xs leading-relaxed font-mono">
      <code>
        {tokens.map((token, i) => (
          <span key={i} className={colorMap[token.type]}>
            {token.text}
          </span>
        ))}
      </code>
    </pre>
  );
}

function MetricDefinitionCard({ metric }: { metric: AskPromptMetric }) {
  return (
    <div className="ask-metrics-card rounded-xl border border-white/[0.12] bg-[#000000] p-5 sm:p-6 h-full flex flex-col">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-lg sm:text-xl font-medium text-white tracking-tight">{metric.label}</p>
        </div>
        {metric.verified && <VerifiedBadge />}
      </div>
      <p className="mt-2 text-sm text-white/45 leading-relaxed">{metric.description}</p>
      <QueryBlock tokens={metric.queryLines} />
    </div>
  );
}

function MetricChartCard({ metric, animate }: { metric: AskPromptMetric; animate?: boolean }) {
  const max = Math.max(...metric.chartBars, 1);

  return (
    <div className="ask-metrics-card rounded-xl border border-white/[0.12] bg-[#000000] p-5 sm:p-6 h-full flex flex-col">
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm text-white/45">{metric.chartTitle}</p>
        {metric.verified && <VerifiedBadge />}
      </div>
      <p className="mt-2 text-3xl sm:text-4xl font-light tracking-tight text-white">{metric.chartValue}</p>
      <div className="mt-5 flex flex-1 items-end gap-2 sm:gap-3 min-h-[8rem]" aria-hidden>
        {metric.chartBars.map((value, i) => (
          <div key={metric.chartLabels[i]} className="flex flex-1 flex-col items-center gap-2 h-full justify-end">
            <div
              className={cn(
                'w-full max-w-8 rounded-t-[2px] bg-white/90 origin-bottom',
                animate && 'prompt-dash-bar'
              )}
              style={{
                height: `${(value / max) * 100}%`,
                animationDelay: animate ? `${i * 60}ms` : undefined,
              }}
            />
            <span className="text-[10px] font-mono text-white/30">{metric.chartLabels[i]}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function MetricsPromptBridge({ prompt }: { prompt: string }) {
  return (
    <div className="relative flex min-h-[10rem] sm:min-h-[12rem] lg:min-h-0 lg:h-full items-center justify-center py-6 lg:py-0">
      {/* Double chevron + dot field */}
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center overflow-hidden" aria-hidden>
        <div className="relative h-[70%] w-[85%] max-w-[16rem]">
          <div className="absolute inset-0 opacity-70 [background-image:radial-gradient(rgba(167,139,250,0.35)_1px,transparent_1px)] [background-size:6px_6px]" />
          <svg viewBox="0 0 120 80" className="h-full w-full" fill="none">
            <path
              d="M8 8 L52 40 L8 72 Z"
              fill="url(#chev1)"
              opacity="0.85"
            />
            <path
              d="M38 8 L82 40 L38 72 Z"
              fill="url(#chev2)"
              opacity="0.55"
            />
            <defs>
              <linearGradient id="chev1" x1="0" y1="0" x2="1" y2="1">
                <stop stopColor="rgba(124,58,237,0.9)" />
                <stop offset="1" stopColor="rgba(76,29,149,0.4)" />
              </linearGradient>
              <linearGradient id="chev2" x1="0" y1="0" x2="1" y2="1">
                <stop stopColor="rgba(167,139,250,0.7)" />
                <stop offset="1" stopColor="rgba(49,16,98,0.3)" />
              </linearGradient>
            </defs>
          </svg>
        </div>
      </div>

      {/* Prompt pill */}
      <div className="relative z-10 w-full max-w-[15rem] sm:max-w-[17rem] mx-auto px-2">
        <div className="rounded-full border border-violet-400/35 bg-[linear-gradient(180deg,rgba(88,28,180,0.92),rgba(49,16,98,0.95))] px-4 py-2.5 sm:px-5 sm:py-3 shadow-[0_0_36px_rgba(124,58,237,0.38)]">
          <p className="text-center text-[0.72rem] sm:text-xs text-violet-100/95 font-light leading-snug">
            {prompt}
          </p>
        </div>
      </div>
    </div>
  );
}

type AskPromptMetricsFlowProps = {
  metric: AskPromptMetric;
  animate?: boolean;
  className?: string;
};

export default function AskPromptMetricsFlow({ metric, animate, className }: AskPromptMetricsFlowProps) {
  return (
    <div
      className={cn(
        'grid grid-cols-1 lg:grid-cols-[1fr_minmax(10rem,14rem)_1fr] gap-4 lg:gap-5 items-stretch max-w-5xl mx-auto',
        className
      )}
    >
      <MetricDefinitionCard metric={metric} />
      <MetricsPromptBridge prompt={metric.prompt} />
      <MetricChartCard metric={metric} animate={animate} />
    </div>
  );
}
