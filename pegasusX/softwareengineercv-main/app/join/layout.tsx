import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Запросить демо Pegasus' : 'Request a Live Pegasus Demo',
    description: isRu
      ? 'Запросите живое демо Pegasus — диспетчеризация, автопарк, платежи и координация шести ролей на одной платформе.'
      : 'Request a live Pegasus demo — dispatch, fleet tracking, payments, and six-role coordination on one logistics operating system.',
    path: '/join',
    language: lang,
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
