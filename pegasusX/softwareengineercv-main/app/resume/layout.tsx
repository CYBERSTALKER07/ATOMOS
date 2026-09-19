import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Обзор платформы' : 'Platform Overview',
    description: isRu ? 'Обзор платформы Pegasus — модули, возможности и сводка операционной системы логистики.' : 'Pegasus platform overview — modules, capabilities, and logistics operating system summary.',
    path: '/resume',
    language: lang
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
