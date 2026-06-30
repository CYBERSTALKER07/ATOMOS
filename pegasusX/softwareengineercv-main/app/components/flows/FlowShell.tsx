'use client';

import type { FlowConfig } from '@/app/data/topicTypes';

type FlowShellProps = {
  title: string;
  children: React.ReactNode;
  className?: string;
};

export function FlowShell({ title, children, className = '' }: FlowShellProps) {
  return (
    <div className={`flow-panel relative overflow-hidden border-y border-white/10 bg-zinc-950 ${className}`}>
      <div className="container mx-auto px-4 py-10 md:py-14">
        <p className="editorial-eyebrow mb-6">{title}</p>
        <div className="flow-panel__canvas min-h-[220px] md:min-h-[280px]">{children}</div>
      </div>
    </div>
  );
}

type StepNodeProps = {
  label: string;
  active?: boolean;
  index: number;
};

export function StepNode({ label, active, index }: StepNodeProps) {
  return (
    <div
      className={`flow-step flex flex-col items-center gap-2 text-center transition-colors ${
        active ? 'text-white' : 'text-white/50'
      }`}
      data-step={index}
    >
      <div
        className={`flex h-10 w-10 items-center justify-center border text-xs font-mono ${
          active ? 'border-white bg-white text-black' : 'border-white/30 bg-transparent'
        }`}
      >
        {index + 1}
      </div>
      <span className="max-w-[88px] text-[10px] font-mono uppercase leading-tight tracking-wide md:max-w-none md:text-xs">
        {label}
      </span>
    </div>
  );
}

export function useHighlightStep(config?: FlowConfig, totalSteps?: number) {
  const highlight = config?.highlightStep ?? 0;
  return (index: number) => index === highlight || (highlight >= (totalSteps ?? 0) && index === (totalSteps ?? 1) - 1);
}
