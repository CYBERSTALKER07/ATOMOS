import React from 'react';
import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import DocsLayout from '@/app/components/docs/DocsLayout';
import DocsHeroBanner from '@/app/components/docs/DocsHeroBanner';
import DocsBodyRenderer from '@/app/components/docs/DocsBodyRenderer';
import DocsRelatedTopics from '@/app/components/docs/DocsRelatedTopics';
import { getDocArticle, getAllDocSlugs } from '@/app/data/docsData';

type PageProps = {
  params: Promise<{
    category: string;
    slug: string;
  }>;
};

export async function generateStaticParams() {
  const slugs = getAllDocSlugs();
  return slugs.map((item) => ({
    category: item.category,
    slug: item.slug,
  }));
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { category, slug } = await params;
  const article = getDocArticle(category, slug);

  if (!article) {
    return {
      title: 'Doc Not Found · Pegasus OS',
    };
  }

  return {
    title: `${article.title} · Pegasus OS Docs`,
    description: article.leadSentence,
    openGraph: {
      title: `${article.title} · Pegasus Documentation`,
      description: article.leadSentence,
      type: 'article',
    },
  };
}

export default async function DocArticlePage({ params }: PageProps) {
  const { category, slug } = await params;
  const article = getDocArticle(category, slug);

  if (!article) {
    notFound();
  }

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
