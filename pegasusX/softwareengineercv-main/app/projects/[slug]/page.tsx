import type { Metadata } from 'next';
import { projects, getProjectBySlug } from '@/app/data/projects';
import { notFound } from 'next/navigation';
import ProjectDetailClient from './ProjectDetailClient';
import { pageMetadata } from '@/app/lib/seo';

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
  const project = getProjectBySlug(slug);
  if (!project) return { title: 'Project' };

  return pageMetadata({
    title: project.title,
    description: project.description,
    path: `/projects/${slug}`,
    image: project.image,
    imageAlt: `${project.title} — Pegasus ${project.category} module`,
  });
}

export default async function ProjectPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const project = getProjectBySlug(slug);

  if (!project) {
    notFound();
  }

  return <ProjectDetailClient project={project} />;
}
