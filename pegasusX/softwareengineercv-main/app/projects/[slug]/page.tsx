import { projects, getProjectBySlug } from '@/app/data/projects';
import { notFound } from 'next/navigation';
import ProjectDetailClient from './ProjectDetailClient';

export function generateStaticParams() {
  return projects.map((project) => ({
    slug: project.slug,
  }));
}

export default function ProjectPage({ params }: { params: { slug: string } }) {
  const project = getProjectBySlug(params.slug);
  
  if (!project) {
    notFound();
  }

  return <ProjectDetailClient project={project} />;
}
