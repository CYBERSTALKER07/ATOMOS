import type { MetadataRoute } from 'next';
import { EXPLORE_CATEGORY_IDS } from '@/app/data/topicTypes';
import { ALL_TOPICS } from '@/app/data/topicPages';
import { SOLUTIONS_ACCORDION_DATA } from '@/app/data/solutionsAccordionData';
import { ROLES_DATA } from '@/app/data/rolesData';
import { projects } from '@/app/data/projects';
import { SITE_URL, languageAlternates } from '@/app/lib/seo';

function entry(
  path: string,
  opts: { changeFrequency?: MetadataRoute.Sitemap[number]['changeFrequency']; priority?: number } = {}
): MetadataRoute.Sitemap[number] {
  const url = path === '/' ? SITE_URL : `${SITE_URL}${path}`;
  const languages = languageAlternates(path === '/' ? '' : path);
  return {
    url,
    lastModified: new Date(),
    changeFrequency: opts.changeFrequency ?? 'monthly',
    priority: opts.priority ?? 0.7,
    alternates: { languages },
  };
}

export default function sitemap(): MetadataRoute.Sitemap {
  const staticPages: MetadataRoute.Sitemap = [
    entry('/', { changeFrequency: 'weekly', priority: 1 }),
    entry('/solutions', { changeFrequency: 'weekly', priority: 0.95 }),
    entry('/roles', { changeFrequency: 'weekly', priority: 0.9 }),
    entry('/projects', { changeFrequency: 'weekly', priority: 0.85 }),
    entry('/desktop-apps', { changeFrequency: 'monthly', priority: 0.8 }),
    entry('/web-apps', { changeFrequency: 'monthly', priority: 0.8 }),
    entry('/mobile-apps', { changeFrequency: 'monthly', priority: 0.8 }),
    entry('/join', { changeFrequency: 'monthly', priority: 0.85 }),
    entry('/contact', { changeFrequency: 'monthly', priority: 0.8 }),
    entry('/resume', { changeFrequency: 'monthly', priority: 0.8 }),
  ];

  const solutionPages: MetadataRoute.Sitemap = SOLUTIONS_ACCORDION_DATA.flatMap((sol) =>
    sol.useCases
      .filter((uc) => uc.slug)
      .map((uc) => entry(`/solutions/${uc.slug}`, { priority: 0.75 }))
  );

  const rolePages: MetadataRoute.Sitemap = ROLES_DATA.map((role) =>
    entry(`/roles/${role.id}`, { priority: 0.7 })
  );

  const projectPages: MetadataRoute.Sitemap = projects.map((project) =>
    entry(`/projects/${project.slug}`, { priority: 0.65 })
  );

  const hubs: MetadataRoute.Sitemap = EXPLORE_CATEGORY_IDS.map((id) =>
    entry(`/${id}`, { changeFrequency: 'weekly', priority: 0.85 })
  );

  const topics: MetadataRoute.Sitemap = ALL_TOPICS.map((t) =>
    entry(t.href, { priority: 0.7 })
  );

  return [
    ...staticPages,
    ...solutionPages,
    ...rolePages,
    ...projectPages,
    ...hubs,
    ...topics,
  ];
}
