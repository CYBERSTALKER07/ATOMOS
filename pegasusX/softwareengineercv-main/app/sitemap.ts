import type { MetadataRoute } from 'next';
import { EXPLORE_CATEGORY_IDS } from '@/app/data/topicTypes';
import { ALL_TOPICS } from '@/app/data/topicPages';

const BASE = process.env.NEXT_PUBLIC_SITE_URL || 'https://pegasus.io';

export default function sitemap(): MetadataRoute.Sitemap {
  const staticPages: MetadataRoute.Sitemap = [
    { url: BASE, changeFrequency: 'weekly', priority: 1 },
    { url: `${BASE}/projects`, changeFrequency: 'weekly', priority: 0.9 },
    { url: `${BASE}/join`, changeFrequency: 'monthly', priority: 0.8 },
    { url: `${BASE}/contact`, changeFrequency: 'monthly', priority: 0.6 },
  ];

  const hubs: MetadataRoute.Sitemap = EXPLORE_CATEGORY_IDS.map((id) => ({
    url: `${BASE}/${id}`,
    changeFrequency: 'weekly' as const,
    priority: 0.85,
  }));

  const topics: MetadataRoute.Sitemap = ALL_TOPICS.map((t) => ({
    url: `${BASE}${t.href}`,
    changeFrequency: 'monthly' as const,
    priority: 0.7,
  }));

  return [...staticPages, ...hubs, ...topics];
}
