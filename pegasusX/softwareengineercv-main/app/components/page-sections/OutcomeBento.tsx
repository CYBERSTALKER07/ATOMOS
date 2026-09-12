'use client';

import PageSectionBlock from './PageSectionBlock';
import { cn } from '@/lib/utils';

const ACCENTS = ['hover:border-[#FBFF63]/50', 'hover:border-[#A9EBF9]/50', 'hover:border-[#8DDC96]/50', 'hover:border-[#DABDFF]/50'];

type OutcomeBentoProps = {
  outcomes: string[];
  variant?: 'bento' | 'list';
};

export default function OutcomeBento({ outcomes, variant = 'bento' }: OutcomeBentoProps) {
  if (variant === 'list') {
    return (
      <PageSectionBlock eyebrow="Outcomes" title="What changes">
        <ul className="grid gap-3 md:grid-cols-2">
          {outcomes.map((item) => (
            <li key={item} className="border border-white/15 p-4 text-sm text-white/80">
              {item}
            </li>
          ))}
        </ul>
      </PageSectionBlock>
    );
  }

  return (
    <PageSectionBlock eyebrow="Outcomes" title="What changes">
      <div className="grid gap-3 sm:grid-cols-2">
        {outcomes.map((item, i) => (
          <div
            key={item}
            className={cn(
              'group border border-white/15 bg-[#0a0a0a] p-5 transition-colors duration-300 md:p-6',
              ACCENTS[i % ACCENTS.length],
              i === 0 && outcomes.length >= 3 && 'sm:col-span-2 sm:max-w-[calc(50%-0.375rem)]'
            )}
          >
            <span className="font-mono text-[10px] uppercase tracking-widest text-white/35">
              {String(i + 1).padStart(2, '0')}
            </span>
            <p className="mt-3 text-sm leading-relaxed text-white/80 transition-colors group-hover:text-white">
              {item}
            </p>
          </div>
        ))}
      </div>
    </PageSectionBlock>
  );
}
