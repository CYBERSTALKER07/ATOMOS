import {
  MEGA_NAV_CATEGORIES,
  type MegaNavCategory,
  type MegaNavLink,
} from './megaNavigation';
import { getTopicContent } from './topicContent';
import type { TopicContent, TopicPage, BilingualContent } from './topicTypes';
import { topicHref } from './topicTypes';

export type CategoryHub = {
  id: string;
  label: string;
  viewAllHref: string;
  viewAllLabel?: string;
  promo?: MegaNavCategory['promo'];
  topics: TopicPage[];
};

function buildTopicPage(category: MegaNavCategory, link: MegaNavLink): TopicPage | null {
  const content = getTopicContent(category.id, link.slug);
  if (!content) return null;

  return {
    categoryId: category.id,
    categoryLabel: category.label,
    slug: link.slug,
    label: link.label,
    description: link.description,
    badge: link.badge,
    href: topicHref(category.id, link.slug),
    content,
  };
}

export const ALL_TOPICS: TopicPage[] = MEGA_NAV_CATEGORIES.flatMap((category) =>
  // Solutions accordion owns /solutions — do not register mega-nav solution
  // topics as explore TopicPages (their hrefs now point at live O9 hubs).
  category.id === 'solutions'
    ? []
    : category.links
        .map((link) => buildTopicPage(category, link))
        .filter((t): t is TopicPage => t !== null)
);

export function getTopicByPath(categoryId: string, slug: string): TopicPage | undefined {
  return ALL_TOPICS.find((t) => t.categoryId === categoryId && t.slug === slug);
}

export function getAllTopicParams(): { category: string; slug: string }[] {
  return ALL_TOPICS.map((t) => ({ category: t.categoryId, slug: t.slug }));
}

export function getAllTopicParamsForCategory(categoryId: string): { slug: string }[] {
  return ALL_TOPICS.filter((t) => t.categoryId === categoryId).map((t) => ({ slug: t.slug }));
}

export function getCategoryHub(categoryId: string): CategoryHub | undefined {
  const category = MEGA_NAV_CATEGORIES.find((c) => c.id === categoryId);
  if (!category) return undefined;

  const topics = category.links
    .map((link) => buildTopicPage(category, link))
    .filter((t): t is TopicPage => t !== null);

  return {
    id: category.id,
    label: category.label,
    viewAllHref: category.viewAllHref,
    viewAllLabel: category.viewAllLabel,
    promo: category.promo,
    topics,
  };
}

export function getSiblingTopics(categoryId: string, slug: string, limit = 4): TopicPage[] {
  return ALL_TOPICS.filter((t) => t.categoryId === categoryId && t.slug !== slug).slice(0, limit);
}

export function getTopicContentOrThrow(categoryId: string, slug: string): BilingualContent {
  const topic = getTopicByPath(categoryId, slug);
  if (!topic) throw new Error(`Missing topic: ${categoryId}/${slug}`);
  return topic.content;
}
