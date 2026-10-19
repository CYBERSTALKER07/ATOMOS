import { Metadata } from 'next';
import SolutionsAccordion from './SolutionsAccordion';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { translations } from '@/app/lib/i18n/translations';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const dict = translations[lang] ?? translations.en;
  return pageMetadata({
    title: dict.meta_solutions_title,
    description: dict.meta_solutions_desc,
    path: '/solutions',
    language: lang
  });
}

export default function SolutionsPage() {
  return <SolutionsAccordion />;
}
