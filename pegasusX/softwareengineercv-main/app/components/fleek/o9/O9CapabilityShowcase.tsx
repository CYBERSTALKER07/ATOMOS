'use client';

import ContentCard from '@/app/components/ContentCard';
import { O9SectionLabel } from '@/app/components/page-sections/o9/O9Sections';
import { useLanguage } from '@/app/context/LanguageContext';

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
  title,
  label,
}: O9CapabilityShowcaseProps) {
  const { language, t } = useLanguage();
  if (!items.length) return null;

  const resolvedLabel = label ?? (language === 'ru' ? 'КЛЮЧЕВЫЕ ВОЗМОЖНОСТИ' : 'CORE CAPABILITIES');
  const resolvedTitle = title ?? (language === 'ru' ? 'Что обеспечивает это решение' : 'What this solution enables');

  return (
    <section className="o9-section">
      <O9SectionLabel>{resolvedLabel}</O9SectionLabel>
      <h2 className="o9-section__title">{resolvedTitle}</h2>
      <div className="o9-cap-grid">
        {items.slice(0, 6).map((item) => (
          <ContentCard
            key={item.href + item.title}
            variant="vertical"
            tone="dark"
            tag={item.tag ?? (language === 'ru' ? 'Возможность' : 'Capability')}
            title={item.title}
            description={item.description}
            href={item.href}
            image={item.image}
            ctaLabel={t('btn_read_more', 'READ MORE')}
          />
        ))}
      </div>
    </section>
  );
}
