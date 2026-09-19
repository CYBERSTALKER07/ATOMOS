import type { Metadata } from 'next';
import { projects } from '@/app/data/projects';
import { getProjectBySlugLocalized } from '@/app/data/projects_ru';
import { notFound } from 'next/navigation';
import ProjectDetailClient from './ProjectDetailClient';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export function generateStaticParams() {
  return projects.map((project) => ({
    slug: project.slug,
  }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const lang = await getServerLanguage();
  const project = getProjectBySlugLocalized(slug, lang);
  if (!project) return { title: lang === 'ru' ? 'Модуль' : 'Project' };

  return pageMetadata({
    title: project.title,
    description: project.description,
    path: `/projects/${slug}`,
    image: project.image,
    imageAlt:
      lang === 'ru'
        ? `${project.title} — модуль Pegasus · ${project.category}`
        : `${project.title} — Pegasus ${project.category} module`,
  });
}

export default async function ProjectPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const lang = await getServerLanguage();
  const project = getProjectBySlugLocalized(slug, lang);

  if (!project) {
    notFound();
  }

  return <ProjectDetailClient project={project} />;
}
