import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Присоединиться' : 'Join Pegasus',
    description: isRu ? 'Запросите демо Pegasus или узнайте о партнёрстве и карьере в команде операционной системы логистики.' : 'Request a Pegasus demo or explore partnership and careers with the logistics operating system team.',
    path: '/join',
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
