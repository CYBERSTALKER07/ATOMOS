import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Веб-приложения' : 'Web Apps',
    description: isRu ? 'Веб-порталы Pegasus для поставщиков, складов, заводов и ритейлеров — диспетчеризация, казначейство и живые операции.' : 'Pegasus web portals for suppliers, warehouses, factories, and retailers — dispatch, treasury, and live ops.',
    path: '/web-apps',
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
