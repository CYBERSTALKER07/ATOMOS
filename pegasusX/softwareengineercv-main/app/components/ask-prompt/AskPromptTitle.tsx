'use client';

import { cn } from '@/lib/utils';

type AskPromptTitleProps = {
  title: string;
  compact?: boolean;
  className?: string;
};

export default function AskPromptTitle({
  title,
  compact,
  className,
}: AskPromptTitleProps) {
  return (
    <h2
      className={cn(
        'text-center font-normal leading-[1.08] tracking-[-0.04em] text-white max-w-5xl mx-auto',
        compact
          ? 'text-[1.85rem] sm:text-[2.35rem] md:text-[2.85rem] lg:text-[clamp(2.35rem,5vw,3.75rem)]'
          : 'text-[clamp(2.35rem,5vw,3.75rem)]',
        className
      )}
    >
      {title}
    </h2>
  );
}
