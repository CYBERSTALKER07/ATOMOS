import type { MetadataRoute } from 'next';
import { SITE_URL } from '@/app/lib/seo';

/**
 * Crawl policy:
 * - Index public marketing pages
 * - Block admin, APIs, and interactive demo sandboxes
 * - AI search/training crawlers inherit `*` (not blanket-blocked)
 */
export default function robots(): MetadataRoute.Robots {
  const disallow = ['/api/', '/admin/', '/private/', '/demo/', '/demo'];

  return {
    rules: [
      {
        userAgent: '*',
        allow: '/',
        disallow,
      },
      // Explicit allow for major AI answer/search bots (AEO) — same public surface
      {
        userAgent: 'GPTBot',
        allow: '/',
        disallow,
      },
      {
        userAgent: 'OAI-SearchBot',
        allow: '/',
        disallow,
      },
      {
        userAgent: 'PerplexityBot',
        allow: '/',
        disallow,
      },
      {
        userAgent: 'Google-Extended',
        allow: '/',
        disallow,
      },
      {
        userAgent: 'ClaudeBot',
        allow: '/',
        disallow,
      },
    ],
    sitemap: `${SITE_URL}/sitemap.xml`,
    host: SITE_URL,
  };
}
