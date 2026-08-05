'use client';

import { Box, Layers, Shield, Zap } from 'lucide-react';
import type { TopicCard } from '@/app/data/topicTypes';
import { O9SectionLabel } from '@/app/components/page-sections/o9/O9Sections';

const ICONS = [Layers, Zap, Shield, Box];

type O9DifferentiatorGridProps = {
  items: TopicCard[];
  title?: string;
};

export default function O9DifferentiatorGrid({
  items,
  title = 'What makes Pegasus different',
}: O9DifferentiatorGridProps) {
  if (!items.length) return null;

  return (
    <section className="o9-section">
      <O9SectionLabel>KEY DIFFERENTIATORS</O9SectionLabel>
      <h2 className="o9-section__title">{title}</h2>
      <div className="o9-diff-grid">
        {items.slice(0, 4).map((item, i) => {
          const Icon = ICONS[i % ICONS.length];
          return (
            <article key={item.title} className="o9-card o9-diff-card">
              <Icon className="o9-diff-card__icon" aria-hidden />
              <h3 className="o9-diff-card__title">{item.title}</h3>
              <p className="o9-diff-card__body">{item.description}</p>
            </article>
          );
        })}
      </div>
    </section>
  );
}
