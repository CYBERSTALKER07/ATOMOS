'use client';

import PageSectionBlock from './PageSectionBlock';
import { cn } from '@/lib/utils';

type ProblemStatementProps = {
  problem: string;
  variant?: 'quote' | 'plain';
};

export default function ProblemStatement({ problem, variant = 'quote' }: ProblemStatementProps) {
  return (
    <PageSectionBlock eyebrow="Problem" title="The problem">
      {variant === 'quote' ? (
        <blockquote className="relative max-w-3xl border-l-2 border-white/30 pl-6 md:pl-8">
          <p className="text-xl font-light leading-relaxed text-white/85 md:text-2xl">{problem}</p>
          <p className="mt-4 font-mono text-[10px] uppercase tracking-widest text-white/35">
            Field observation
          </p>
        </blockquote>
      ) : (
        <p className={cn('max-w-3xl leading-relaxed text-white/70')}>{problem}</p>
      )}
    </PageSectionBlock>
  );
}
