import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import {
  getAllTopicParamsForCategory,
  getCategoryHub,
  getTopicByPath,
} from '@/app/data/topicPages';
import CategoryHubClient from '@/app/components/explore/CategoryHubClient';
import TopicPageClient from '@/app/components/explore/TopicPageClient';
import type { ExploreCategoryId } from '@/app/data/topicTypes';
import { pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';

export function createCategoryHubPage(categoryId: ExploreCategoryId) {
  return function CategoryHubPage() {
    const hub = getCategoryHub(categoryId);
    if (!hub) notFound();
    return <CategoryHubClient hub={hub} />;
  };
}

export function createCategoryHubMetadata(categoryId: ExploreCategoryId) {
  return function generateMetadata(): Metadata {
    const hub = getCategoryHub(categoryId);
    if (!hub) return { title: 'Explore' };
    const description =
      hub.promo?.body ??
      `Explore ${hub.label} on Pegasus — dispatch, fleet tracking, payments, and role-specific logistics software.`;
    return pageMetadata({
      title: hub.label,
      description,
      path: `/${categoryId}`,
    });
  };
}

export function createTopicPage(categoryId: ExploreCategoryId) {
  return function TopicPage({ params }: { params: Promise<{ slug: string }> }) {
    return <TopicPageAsync categoryId={categoryId} params={params} />;
  };
}

async function TopicPageAsync({
  categoryId,
  params,
}: {
  categoryId: ExploreCategoryId;
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const topic = getTopicByPath(categoryId, slug);
  if (!topic) notFound();
  return <TopicPageClient topic={topic} />;
}

export function createTopicStaticParams(categoryId: ExploreCategoryId) {
  return function generateStaticParams() {
    return getAllTopicParamsForCategory(categoryId);
  };
}

export function createTopicMetadata(categoryId: ExploreCategoryId) {
  return async function generateMetadata({
    params,
  }: {
    params: Promise<{ slug: string }>;
  }): Promise<Metadata> {
    const { slug } = await params;
    const topic = getTopicByPath(categoryId, slug);
    if (!topic) return { title: 'Topic' };
    const lang = await getServerLanguage();
    const content = topic.content[lang] || topic.content.en;
    return pageMetadata({
      title: content.title,
      description: content.summary,
      path: topic.href,
    });
  };
}
