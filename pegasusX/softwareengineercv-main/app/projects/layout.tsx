import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Модули' : 'Projects',
    description: isRu ? 'Модули Pegasus для логистики под управлением поставщика — диспетчеризация, платежи, автопарк и ролевые приложения.' : 'Explore Pegasus modules powering supplier-led logistics — dispatch, payments, fleet, and role apps.',
    path: '/projects',
    language: lang
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
