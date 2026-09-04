'use client';

import Link from 'next/link';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';
import CloudEcosystemBento from './CloudEcosystemBento';
import { useLanguage } from '../context/LanguageContext';
import { cloudEcosystemHomeItems } from '@/app/lib/cloudEcosystem';

export default function CloudEcosystemSection() {
  const { t } = useLanguage();
  const items = cloudEcosystemHomeItems();

  return (
    <PageSection id="cloud-ecosystem" className="border-t border-white/10">
      <SectionHeader
        align="center"
        eyebrow={t('cloud_eco_eyebrow', 'Cloud ecosystem')}
        title={t('cloud_eco_title', 'Fully backed by the best of Google Cloud')}
        description={t(
          'cloud_eco_desc',
          'Databases, servers, messaging, and delivery — Spanner, Kafka, Redis, GKE, and the GCP services that keep every role online.',
        )}
        className="mb-10"
      />

      <CloudEcosystemBento items={items} compact />

      <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-3">
        <Link
          href="/cloud-ecosystem"
          className="inline-flex items-center justify-center min-h-11 px-5 text-xs font-semibold uppercase tracking-[0.1em] bg-white text-black hover:bg-emerald-300 transition-colors"
        >
          {t('cloud_eco_cta_page', 'Explore full stack')}
        </Link>
        <Link
          href="/technology"
          className="inline-flex items-center justify-center min-h-11 px-5 text-xs font-semibold uppercase tracking-[0.1em] border border-white/30 text-white hover:border-emerald-400/60 hover:text-emerald-200 transition-colors"
        >
          {t('cloud_eco_cta_tech', 'Technology hub')}
        </Link>
      </div>
    </PageSection>
  );
}
