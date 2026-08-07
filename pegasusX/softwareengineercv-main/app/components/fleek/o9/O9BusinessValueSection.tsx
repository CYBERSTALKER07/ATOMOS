'use client';

import { useState } from 'react';
import { cn } from '@/lib/utils';
import type { O9ValueTab } from '@/app/data/o9FleekDefaults';
import { O9SectionLabel } from '@/app/components/page-sections/o9/O9Sections';
import { useLanguage } from '@/app/context/LanguageContext';

type O9BusinessValueSectionProps = {
  tabs: O9ValueTab[];
  title?: string;
};

export default function O9BusinessValueSection({
  tabs,
  title,
}: O9BusinessValueSectionProps) {
  const { language, t } = useLanguage();
  const [activeId, setActiveId] = useState(tabs[0]?.id ?? '');
  const active = tabs.find((tab) => tab.id === activeId) ?? tabs[0];
  if (!active) return null;

  const resolvedTitle =
    title ?? (language === 'ru' ? 'Примеры ценности бизнеса на платформе Pegasus' : 'Examples of business value on the Pegasus platform');

  return (
    <section className="o9-section">
      <O9SectionLabel>{language === 'ru' ? 'ЦЕННОСТЬ ДЛЯ БИЗНЕСА' : 'BUSINESS VALUE'}</O9SectionLabel>
      <h2 className="o9-section__title">{resolvedTitle}</h2>
      {tabs.length > 1 ? (
        <div className="o9-tab-row" role="tablist" aria-label="Business value examples">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              role="tab"
              aria-selected={tab.id === active.id}
              className={cn('o9-tab', tab.id === active.id && 'is-active')}
              onClick={() => setActiveId(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </div>
      ) : null}
      <div className="o9-stat-grid" role="tabpanel">
        {active.stats.map((stat) => (
          <div key={`${stat.label}-${stat.value}`} className="o9-stat-col">
            <p className="o9-stat-col__context">{stat.context}</p>
            <p className="o9-stat-col__value">{stat.value}</p>
            <p className="o9-stat-col__label">{stat.label}</p>
          </div>
        ))}
      </div>
    </section>
  );
}
