import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Десктоп-приложения' : 'Desktop Apps',
    description: isRu ? 'Десктопные командные центры Pegasus для диспетчерских — мультимониторная диспетчеризация, казначейство и контроль сети.' : 'Pegasus desktop command centers for control-room teams — multi-monitor dispatch, treasury, and network oversight.',
    path: '/desktop-apps',
    language: lang
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
