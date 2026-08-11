'use client';

import Link from 'next/link';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import PageSection from '@/app/components/layout/PageSection';
import SectionHeader from '@/app/components/layout/SectionHeader';
import CloudEcosystemBento from '@/app/components/CloudEcosystemBento';
import { useLanguage } from '@/app/context/LanguageContext';
import {
  CLOUD_ECOSYSTEM_CATEGORIES,
  CLOUD_ECOSYSTEM_TECH,
  type CloudTechCategory,
} from '@/app/lib/cloudEcosystem';
import { useMemo, useState } from 'react';

export default function CloudEcosystemPageClient() {
  const { t, language } = useLanguage();
  const isRu = language === 'ru';
  const [active, setActive] = useState<CloudTechCategory | 'all'>('all');

  const items = useMemo(() => {
    if (active === 'all') return CLOUD_ECOSYSTEM_TECH;
    return CLOUD_ECOSYSTEM_TECH.filter((item) => item.category === active);
  }, [active]);

  return (
    <div className="relative bg-black min-h-screen">
      <SiteNav activeHref="/cloud-ecosystem" />

      <PageSection className="pt-28 md:pt-32 border-b border-white/10">
        <SectionHeader
          align="left"
          eyebrow={t('cloud_eco_eyebrow', 'Cloud ecosystem')}
          title={t('cloud_eco_page_title', 'Every layer of the cloud, mapped')}
          description={t(
            'cloud_eco_page_desc',
            'Pegasus runs on Google Cloud with Spanner as the system of record, Kafka for live events, Redis for hot cache, GKE for servers, and the delivery stack that ships every role app.',
          )}
          className="mb-8 max-w-3xl"
        />

        <div className="flex flex-wrap gap-2 mb-10">
          <button
            type="button"
            onClick={() => setActive('all')}
            className={`px-3.5 py-2 text-[10px] font-mono uppercase tracking-[0.16em] border transition-colors ${
              active === 'all'
                ? 'border-emerald-400/60 bg-emerald-500/10 text-emerald-200'
                : 'border-white/15 text-white/55 hover:border-white/35 hover:text-white'
            }`}
          >
            {t('cloud_eco_filter_all', 'All')}
          </button>
          {CLOUD_ECOSYSTEM_CATEGORIES.map((cat) => (
            <button
              key={cat.id}
              type="button"
              onClick={() => setActive(cat.id)}
              className={`px-3.5 py-2 text-[10px] font-mono uppercase tracking-[0.16em] border transition-colors ${
                active === cat.id
                  ? 'border-emerald-400/60 bg-emerald-500/10 text-emerald-200'
                  : 'border-white/15 text-white/55 hover:border-white/35 hover:text-white'
              }`}
            >
              {isRu ? cat.labelRu : cat.label}
            </button>
          ))}
        </div>

        <CloudEcosystemBento items={items} />

        <div className="mt-12 flex flex-col sm:flex-row gap-3">
          <Link
            href="/technology"
            className="inline-flex items-center justify-center min-h-11 px-5 text-xs font-semibold uppercase tracking-[0.1em] bg-white text-black hover:bg-emerald-300 transition-colors"
          >
            {t('cloud_eco_cta_tech', 'Technology hub')}
          </Link>
          <Link
            href="/join"
            className="inline-flex items-center justify-center min-h-11 px-5 text-xs font-semibold uppercase tracking-[0.1em] border border-white/30 text-white hover:border-emerald-400/60 transition-colors"
          >
            {t('nav_demo', 'Request demo')}
          </Link>
        </div>
      </PageSection>

      <Footer />
    </div>
  );
}
