import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Мобильные приложения' : 'Mobile Apps',
    description: isRu ? 'Нативные Android и iOS приложения Pegasus для водителей, ритейлеров, поставщиков, склада, завода и ворот.' : 'Pegasus native Android and iOS apps for drivers, retailers, suppliers, warehouse, factory, and payload gate teams.',
    path: '/mobile-apps',
    language: lang
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
