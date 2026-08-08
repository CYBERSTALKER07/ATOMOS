import type { Metadata } from 'next';
import { contactPageJsonLd, jsonLdScript, pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';
  return pageMetadata({
    title: isRu ? 'Контакты — Pegasus' : 'Contact Pegasus Logistics',
    description: isRu
      ? 'Свяжитесь с Pegasus для живого демо, вопросов по развёртыванию или enterprise-логистике.'
      : 'Contact Pegasus for a live demo, deployment questions, or enterprise logistics software inquiries.',
    path: '/contact',
    language: lang,
  });
}

export default async function Layout({ children }: { children: React.ReactNode }) {
  const lang = await getServerLanguage();
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdScript(contactPageJsonLd(lang))}
      />
      {children}
    </>
  );
}
