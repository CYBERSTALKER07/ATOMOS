import type { Metadata } from 'next';
import CloudEcosystemPageClient from './CloudEcosystemPageClient';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { translations } from '@/app/lib/i18n/translations';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const dict = translations[lang] ?? translations.en;
  return pageMetadata({
    title: dict.cloud_eco_page_title,
    description: dict.cloud_eco_page_desc,
    path: '/cloud-ecosystem',
    language: lang,
  });
}

export default function CloudEcosystemPage() {
  return <CloudEcosystemPageClient />;
}
