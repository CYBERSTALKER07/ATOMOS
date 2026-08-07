import { notFound } from 'next/navigation';
import { ROLES_DATA, getRolesData } from '@/app/data/rolesData';
import { getTopicByPath } from '@/app/data/topicPages';
import { getTopicContent } from '@/app/data/topicContent';
import { rolesTopics } from '@/app/data/topicContent/roles';
import type { TopicPage } from '@/app/data/topicTypes';
import { topicHref } from '@/app/data/topicTypes';
import TopicPageClient from '@/app/components/explore/TopicPageClient';
import SiteNav from '@/app/components/explore/SiteNav';
import RoleDetailClient from './RoleDetailClient';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

function getRoleTopic(roleId: string): TopicPage | undefined {
  const fromNav = getTopicByPath('roles', roleId);
  if (fromNav) return fromNav;

  const content = getTopicContent('roles', roleId);
  if (!content) return undefined;

  return {
    categoryId: 'roles',
    categoryLabel: 'Roles',
    slug: roleId,
    label: content.en.title,
    href: topicHref('roles', roleId),
    content,
  };
}

export function generateStaticParams() {
  const roleParams = ROLES_DATA.map((role) => ({ role: role.id }));
  const topicParams = Object.keys(rolesTopics).map((slug) => ({ role: slug }));
  const seen = new Set<string>();
  return [...roleParams, ...topicParams].filter((p) => {
    if (seen.has(p.role)) return false;
    seen.add(p.role);
    return true;
  });
}

export async function generateMetadata({ params }: { params: Promise<{ role: string }> }) {
  const { role: roleId } = await params;
  const lang = await getServerLanguage();
  const topic = getRoleTopic(roleId);
  if (topic) {
    const topicContent = topic.content[lang] || topic.content.en;
    return pageMetadata({
      title: topicContent.title,
      description: topicContent.summary,
      path: `/roles/${roleId}`,
    });
  }

  const role = getRolesData(lang).find((r) => r.id === roleId);
  if (!role) {
    return { title: lang === 'ru' ? 'Роль не найдена' : 'Role Not Found' };
  }
  return pageMetadata({
    title: role.name,
    description: role.description,
    path: `/roles/${roleId}`,
  });
}

export default async function RolePage({ params }: { params: Promise<{ role: string }> }) {
  const { role: roleId } = await params;
  const topic = getRoleTopic(roleId);
  if (topic) {
    return <TopicPageClient topic={topic} />;
  }

  const lang = await getServerLanguage();
  const role = getRolesData(lang).find((r) => r.id === roleId);
  if (!role) {
    notFound();
  }

  return (
    <div className="bg-[var(--bg)] min-h-screen pb-24 text-[var(--text)]">
      <SiteNav activeHref="/roles" />
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-[calc(4.5rem+2rem)] md:pt-[calc(5rem+2rem)]">
        <div className="mb-16">
          <h1 className="text-5xl font-bold mb-6 tracking-tight text-[var(--text)]">{role.name}</h1>
          <p className="text-xl text-[var(--text-secondary)] max-w-3xl">{role.description}</p>
        </div>
        <RoleDetailClient role={role} />
      </div>
    </div>
  );
}
