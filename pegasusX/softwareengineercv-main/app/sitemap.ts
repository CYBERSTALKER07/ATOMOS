import type { MetadataRoute } from 'next';
import { EXPLORE_CATEGORY_IDS } from '@/app/data/topicTypes';
import { ALL_TOPICS } from '@/app/data/topicPages';
import { SOLUTIONS_ACCORDION_DATA } from '@/app/data/solutionsAccordionData';
import { ROLES_DATA } from '@/app/data/rolesData';
import { projects } from '@/app/data/projects';
import { SITE_URL } from '@/app/lib/seo';

export default function sitemap(): MetadataRoute.Sitemap {
  const staticPages: MetadataRoute.Sitemap = [
    { url: SITE_URL, changeFrequency: 'weekly', priority: 1 },
    { url: `${SITE_URL}/solutions`, changeFrequency: 'weekly', priority: 0.95 },
    { url: `${SITE_URL}/roles`, changeFrequency: 'weekly', priority: 0.9 },
    { url: `${SITE_URL}/projects`, changeFrequency: 'weekly', priority: 0.85 },
    { url: `${SITE_URL}/desktop-apps`, changeFrequency: 'monthly', priority: 0.8 },
    { url: `${SITE_URL}/web-apps`, changeFrequency: 'monthly', priority: 0.8 },
    { url: `${SITE_URL}/mobile-apps`, changeFrequency: 'monthly', priority: 0.8 },
    { url: `${SITE_URL}/join`, changeFrequency: 'monthly', priority: 0.75 },
    { url: `${SITE_URL}/contact`, changeFrequency: 'monthly', priority: 0.7 },
    { url: `${SITE_URL}/resume`, changeFrequency: 'monthly', priority: 0.5 },
  ];

  const solutionPages: MetadataRoute.Sitemap = SOLUTIONS_ACCORDION_DATA.flatMap((sol) =>
    sol.useCases
      .filter((uc) => uc.slug)
      .map((uc) => ({
        url: `${SITE_URL}/solutions/${uc.slug}`,
        changeFrequency: 'monthly' as const,
        priority: 0.75,
      }))
  );

  const rolePages: MetadataRoute.Sitemap = ROLES_DATA.map((role) => ({
    url: `${SITE_URL}/roles/${role.id}`,
    changeFrequency: 'monthly' as const,
    priority: 0.7,
  }));

  const projectPages: MetadataRoute.Sitemap = projects.map((project) => ({
    url: `${SITE_URL}/projects/${project.slug}`,
    changeFrequency: 'monthly' as const,
    priority: 0.65,
  }));

  const hubs: MetadataRoute.Sitemap = EXPLORE_CATEGORY_IDS.map((id) => ({
    url: `${SITE_URL}/${id}`,
    changeFrequency: 'weekly' as const,
    priority: 0.85,
  }));

  const topics: MetadataRoute.Sitemap = ALL_TOPICS.map((t) => ({
    url: `${SITE_URL}${t.href}`,
    changeFrequency: 'monthly' as const,
    priority: 0.7,
  }));

  return [
    ...staticPages,
    ...solutionPages,
    ...rolePages,
    ...projectPages,
    ...hubs,
    ...topics,
  ];
}
