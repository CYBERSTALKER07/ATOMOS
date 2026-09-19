'use client';

import { useMemo } from 'react';
import type { AskPromptCard as AskPromptCardType } from './types';
import AskPromptCard from './AskPromptCard';
import { cn } from '@/lib/utils';

type AskPromptCollageProps = {
  cards: AskPromptCardType[];
  interactive?: boolean;
  className?: string;
};

export default function AskPromptCollage({
  cards,
  interactive = false,
  className,
}: AskPromptCollageProps) {
  const mobileCards = useMemo(
    () =>
      [...cards].sort(
        (a, b) => (a.layout.mobileOrder ?? 99) - (b.layout.mobileOrder ?? 99)
      ),
    [cards]
  );

  return (
    <div className={cn('relative w-full', className)}>
      {/* Mobile / tablet — stacked */}
      <div className="flex flex-col gap-3 sm:gap-4 lg:hidden max-w-xl mx-auto">
        {mobileCards.map((card) => (
          <AskPromptCard
            key={card.id}
            category={card.category}
            question={card.question}
            featured={card.featured}
            interactive={interactive}
            className={cn(card.featured && 'ring-1 ring-violet-400/35')}
          />
        ))}
      </div>

      {/* Desktop — overlapping collage with edge fade (Basedash-style) */}
      <div className="relative hidden lg:block">
        <div className="relative mx-auto h-[34rem] xl:h-[36rem] max-w-[72rem] overflow-hidden">
          {cards.map((card) => (
            <AskPromptCard
              key={card.id}
              category={card.category}
              question={card.question}
              featured={card.featured}
              interactive={interactive}
              className={cn('absolute', card.layout.desktop, card.featured && 'min-h-[10.5rem]')}
            />
          ))}

          {/* Side fades — cards appear to extend beyond viewport */}
          <div
            className="pointer-events-none absolute inset-y-0 left-0 z-20 w-28 bg-gradient-to-r from-black via-black/80 to-transparent"
            aria-hidden
          />
          <div
            className="pointer-events-none absolute inset-y-0 right-0 z-20 w-28 bg-gradient-to-l from-black via-black/80 to-transparent"
            aria-hidden
          />
          <div
            className="pointer-events-none absolute inset-x-0 bottom-0 z-20 h-16 bg-gradient-to-t from-black to-transparent"
            aria-hidden
          />
        </div>
      </div>
    </div>
  );
}
