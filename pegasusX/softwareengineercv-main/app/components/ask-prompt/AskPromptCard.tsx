'use client';

import { forwardRef } from 'react';
import { cn } from '@/lib/utils';

export type AskPromptCardProps = {
  category: string;
  question: string;
  featured?: boolean;
  interactive?: boolean;
  className?: string;
  style?: React.CSSProperties;
};

const AskPromptCard = forwardRef<HTMLElement, AskPromptCardProps>(function AskPromptCard(
  { category, question, featured = false, interactive = false, className, style },
  ref
) {
  return (
    <article
      ref={ref}
      style={style}
      className={cn(
        'ask-prompt-card relative flex flex-col justify-between rounded-2xl border p-5 sm:p-6',
        'backdrop-blur-md transition-[transform,border-color,box-shadow] duration-300',
        featured
          ? cn(
              'min-h-[9.5rem] border-violet-400/55',
              'bg-[linear-gradient(145deg,rgba(88,28,180,0.95),rgba(49,16,98,0.98))]',
              'shadow-[0_0_60px_rgba(124,58,237,0.42),inset_0_1px_0_rgba(255,255,255,0.08)]',
              interactive &&
                'md:hover:border-violet-300/80 md:hover:shadow-[0_0_72px_rgba(167,139,250,0.5)] md:hover:-translate-y-1'
            )
          : cn(
              'min-h-[8.25rem] border-white/[0.12]',
              'bg-[#121212]/88',
              'bg-[radial-gradient(circle_at_1px_1px,rgba(255,255,255,0.07)_1px,transparent_0)] bg-[size:16px_16px]',
              'shadow-[0_8px_32px_rgba(0,0,0,0.35)]',
              interactive &&
                'md:hover:border-white/25 md:hover:bg-[#171717]/92 md:hover:-translate-y-0.5 md:hover:shadow-[0_14px_40px_rgba(0,0,0,0.45)]'
            ),
        className
      )}
    >
      {featured && (
        <>
          <div
            className="pointer-events-none absolute -inset-16 -z-10 rounded-full bg-violet-600/25 blur-3xl"
            aria-hidden
          />
          <div
            className="pointer-events-none absolute inset-0 rounded-2xl ring-1 ring-inset ring-violet-300/20"
            aria-hidden
          />
        </>
      )}

      <p
        className={cn(
          'font-mono text-[10px] sm:text-[11px] uppercase tracking-[0.14em] mb-3 sm:mb-4',
          featured ? 'text-violet-100/75' : 'text-white/38'
        )}
      >
        {category}
      </p>
      <p
        className={cn(
          'text-[0.95rem] sm:text-base md:text-[1.05rem] leading-snug',
          featured ? 'text-white font-normal' : 'text-white/82 font-light'
        )}
      >
        {question}
      </p>
    </article>
  );
});

export default AskPromptCard;
