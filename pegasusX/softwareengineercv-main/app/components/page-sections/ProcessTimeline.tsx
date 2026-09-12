'use client';

import PageSectionBlock from './PageSectionBlock';
import { cn } from '@/lib/utils';

type Step = { title: string; description: string };

type ProcessTimelineProps = {
  steps: Step[];
  variant?: 'timeline' | 'grid';
};

export default function ProcessTimeline({ steps, variant = 'timeline' }: ProcessTimelineProps) {
  if (variant === 'grid') {
    return (
      <PageSectionBlock eyebrow="Process" title="How it works">
        <div className="grid gap-4 md:grid-cols-3">
          {steps.map((step, i) => (
            <div key={step.title} className="border border-white/15 p-6">
              <p className="font-mono text-xs uppercase tracking-wider text-white/40">
                Step {String(i + 1).padStart(2, '0')}
              </p>
              <h3 className="mt-3 font-semibold">{step.title}</h3>
              <p className="mt-2 text-sm text-white/60">{step.description}</p>
            </div>
          ))}
        </div>
      </PageSectionBlock>
    );
  }

  return (
    <PageSectionBlock eyebrow="Process" title="How it works">
      <div className="relative">
        <div className="absolute left-4 top-0 hidden h-full w-px bg-white/15 md:block" aria-hidden />
        <ol className="space-y-6 md:space-y-8">
          {steps.map((step, i) => (
            <li key={step.title} className="relative flex gap-6 md:gap-10">
              <div
                className={cn(
                  'relative z-10 flex h-8 w-8 shrink-0 items-center justify-center border border-white/30 bg-black font-mono text-xs',
                  i === 0 && 'border-white bg-white text-black'
                )}
              >
                {String(i + 1).padStart(2, '0')}
              </div>
              <div className="flex-1 border border-white/10 bg-[#0a0a0a] p-5 md:p-6">
                <h3 className="font-semibold">{step.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-white/60">{step.description}</p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    </PageSectionBlock>
  );
}
