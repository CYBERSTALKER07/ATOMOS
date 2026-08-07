import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Роли' : 'Roles',
    description: isRu ? 'Шесть ролей Pegasus — поставщик, склад, завод, водитель, ритейлер и ворота — на одном общем учёте заказов.' : 'Six Pegasus roles — supplier, warehouse, factory, driver, retailer, and gate — on one shared order record.',
    path: '/roles',
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
