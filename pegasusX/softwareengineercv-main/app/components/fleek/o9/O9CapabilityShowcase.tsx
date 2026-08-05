'use client';

import ContentCard from '@/app/components/ContentCard';
import { O9SectionLabel } from '@/app/components/page-sections/o9/O9Sections';

export type O9CapabilityCard = {
  title: string;
  description: string;
  href: string;
  image: string;
  tag?: string;
};

type O9CapabilityShowcaseProps = {
  items: O9CapabilityCard[];
  title?: string;
  label?: string;
};

export default function O9CapabilityShowcase({
  items,
  title = 'What this solution enables',
  label = 'CORE CAPABILITIES',
}: O9CapabilityShowcaseProps) {
  if (!items.length) return null;

  return (
    <section className="o9-section">
      <O9SectionLabel>{label}</O9SectionLabel>
      <h2 className="o9-section__title">{title}</h2>
      <div className="o9-cap-grid">
        {items.slice(0, 6).map((item) => (
          <ContentCard
            key={item.href + item.title}
            variant="vertical"
            tone="dark"
            tag={item.tag ?? 'Capability'}
            title={item.title}
            description={item.description}
            href={item.href}
            image={item.image}
            ctaLabel="Read more"
          />
        ))}
      </div>
    </section>
  );
}
