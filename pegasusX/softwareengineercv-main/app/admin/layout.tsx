import type { Metadata } from 'next';
import SiteNav from '@/app/components/explore/SiteNav';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  return pageMetadata({
    title: lang === 'ru' ? 'Админ' : 'Admin',
    description: lang === 'ru' ? 'Внутренняя админ-панель Pegasus.' : 'Internal Pegasus admin.',
    path: '/admin',
    noIndex: true,
    language: lang
  });
}

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <SiteNav activeHref="/admin" />
      <div className="pt-[4.5rem] md:pt-20">{children}</div>
    </>
  );
}
