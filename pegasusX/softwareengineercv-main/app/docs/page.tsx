import React from 'react';
import type { Metadata } from 'next';
import DocsLayout from '@/app/components/docs/DocsLayout';
import DocsHeroBanner from '@/app/components/docs/DocsHeroBanner';
import DocsBodyRenderer from '@/app/components/docs/DocsBodyRenderer';
import DocsRelatedTopics from '@/app/components/docs/DocsRelatedTopics';
import { DEFAULT_DOC_ARTICLE } from '@/app/data/docsData';

export const metadata: Metadata = {
  title: 'Documentation · Pegasus OS Sovereign Logistics Engine',
  description:
    'Official operational blueprints, mathematical dispatch algorithms, and interface specifications for the Pegasus multi-tenant global logistics operating system.',
};

export default function DocsRootPage() {
  const article = DEFAULT_DOC_ARTICLE;

  return (
    <DocsLayout activeArticle={article}>
      <DocsHeroBanner article={article} />
      <DocsBodyRenderer article={article} />
      <DocsRelatedTopics
        currentCategory={article.categoryId}
        currentSlug={article.slug}
        relatedTopics={article.relatedTopics}
      />
    </DocsLayout>
  );
}
