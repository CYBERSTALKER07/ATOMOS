'use client';

import type { TopicCard, WhyItMatters } from '@/app/data/topicTypes';
import ProcessRGrid from '@/app/components/visuals/ProcessRGrid';
import { cn } from '@/lib/utils';
import { useLanguage } from '@/app/context/LanguageContext';

export function O9SectionLabel({ children }: { children: string }) {
  return (
    <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">{children}</p>
  );
}

export function O9WhyItMatters({
  why,
  problemFallback,
}: {
  why?: WhyItMatters;
  problemFallback: string;
}) {
  const { t } = useLanguage();
  const headline = why?.headline ?? t('sec_why_it_matters_title');
  const body = why?.body ?? problemFallback;
  const insights = why?.insights ?? [];

  return (
    <section className="docs-section">
      <O9SectionLabel>{t('sec_why_it_matters_label')}</O9SectionLabel>
      <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight md:text-4xl">{headline}</h2>
      <p className="docs-body mt-6 max-w-3xl text-base leading-relaxed text-white/70 md:text-lg">{body}</p>
      {insights.length > 0 ? (
        <div className="mt-10">
          {insights.length >= 3 && insights.length <= 8 ? (
            <ProcessRGrid
              steps={insights.map((insight) => ({
                title: insight.title,
                description: insight.body,
              }))}
            />
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {insights.map((insight) => (
                <div key={insight.title} className="docs-card">
                  <h3 className="text-lg font-semibold">{insight.title}</h3>
                  <p className="mt-3 text-sm leading-relaxed text-white/65">{insight.body}</p>
                </div>
              ))}
            </div>
          )}
        </div>
      ) : null}
    </section>
  );
}

export function O9CapabilityGrid({
  capabilities,
  label,
  title,
  layout = 'cards',
}: {
  capabilities: TopicCard[];
  label?: string;
  title?: string;
  layout?: 'cards' | 'r-grid';
}) {
  const { t } = useLanguage();
  if (!capabilities.length) return null;

  const resolvedLabel = label ?? 'CORE CAPABILITIES';
  const resolvedTitle = title ?? 'What this solution enables';

  const useRGrid = layout === 'r-grid' && capabilities.length >= 3 && capabilities.length <= 8;
  const rGridSteps = capabilities.map((cap) => ({
    title: cap.title,
    description: cap.description,
  }));

  return (
    <section className="docs-section">
      <O9SectionLabel>{resolvedLabel}</O9SectionLabel>
      <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight md:text-4xl">{resolvedTitle}</h2>
      {useRGrid ? (
        <div className="mt-10">
          <ProcessRGrid steps={rGridSteps} />
        </div>
      ) : (
        <div className="docs-cap-grid mt-10">
          {capabilities.map((cap, i) => (
            <article
              key={cap.title}
              className={cn(
                'docs-card group',
                i === 0 && capabilities.length >= 3 && 'lg:min-h-full'
              )}
            >
              <span className="font-mono text-[10px] uppercase tracking-widest text-white/40">
                {String(i + 1).padStart(2, '0')}
              </span>
              <h3 className={cn('mt-3 font-semibold', i === 0 ? 'text-2xl' : 'text-xl')}>{cap.title}</h3>
              <p className="mt-3 text-sm leading-relaxed text-white/65 transition-colors duration-200 group-hover:text-white/85">
                {cap.description}
              </p>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

export function O9DifferentiatorList({ items }: { items: TopicCard[] }) {
  const { t } = useLanguage();
  if (!items.length) return null;
  return (
    <section className="docs-section">
      <O9SectionLabel>{t('sec_key_differentiators_label')}</O9SectionLabel>
      <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight md:text-4xl">
        {t('sec_key_differentiators_title')}
      </h2>
      <div className="mt-10 space-y-4">
        {items.map((item, i) => (
          <div
            key={`${item.title}-${i}`}
            className="docs-card grid gap-3 md:grid-cols-[minmax(0,14rem)_1fr] md:gap-8"
          >
            <h3 className="font-semibold text-white">{item.title}</h3>
            <p className="text-sm leading-relaxed text-white/65">{item.description}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

export function O9HowItWorks({
  steps,
  variant = 'list',
}: {
  steps: { title: string; description: string }[];
  variant?: 'list' | 'r-grid';
}) {
  const { t } = useLanguage();
  if (!steps.length) return null;
  return (
    <section className="docs-section">
      <O9SectionLabel>{t('sec_how_it_works_label')}</O9SectionLabel>
      <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight md:text-4xl">
        {t('sec_how_it_works_title')}
      </h2>
      {variant === 'r-grid' ? (
        <div className="mt-10">
          <ProcessRGrid steps={steps} />
        </div>
      ) : (
        <ol className="mt-10 space-y-4">
          {steps.map((step, i) => (
            <li key={step.title} className="docs-card flex gap-5 md:gap-8">
              <span
                className={cn(
                  'flex h-11 w-11 shrink-0 items-center justify-center border font-mono text-xs',
                  i === 0 ? 'border-white bg-white text-black' : 'border-white/30 text-white/70'
                )}
              >
                {String(i + 1).padStart(2, '0')}
              </span>
              <div>
                <h3 className="font-semibold">{step.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-white/65">{step.description}</p>
              </div>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

export function O9EdgeCaseGrid({ items }: { items?: TopicCard[] }) {
  const { t } = useLanguage();
  if (!items?.length) return null;
  return (
    <section className="docs-section">
      <O9SectionLabel>{t('sec_edge_cases_label')}</O9SectionLabel>
      <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight md:text-4xl">
        {t('sec_edge_cases_title')}
      </h2>
      <div className="mt-10 grid gap-4 md:grid-cols-2">
        {items.map((item) => (
          <article
            key={item.title}
            className="docs-card border-l-2 border-l-white/40"
          >
            <h3 className="font-semibold">{item.title}</h3>
            <p className="mt-3 text-sm leading-relaxed text-white/65">{item.description}</p>
          </article>
        ))}
      </div>
    </section>
  );
}

export function O9AiDataPanel({ items }: { items?: TopicCard[] }) {
  const { t } = useLanguage();
  if (!items?.length) return null;
  return (
    <section className="docs-section">
      <O9SectionLabel>{t('sec_ai_data_label')}</O9SectionLabel>
      <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight md:text-4xl">
        {t('sec_ai_data_title')}
      </h2>
      <div className="mt-10 grid gap-4 lg:grid-cols-3">
        {items.map((item) => (
          <article key={item.title} className="docs-card">
            <p className="font-mono text-[10px] uppercase tracking-widest text-white/45">Layer</p>
            <h3 className="mt-2 font-semibold">{item.title}</h3>
            <p className="mt-3 text-sm leading-relaxed text-white/65">{item.description}</p>
          </article>
        ))}
      </div>
    </section>
  );
}
