import { platformTopics } from './platform';
import { solutionsTopics } from './solutions';
import { rolesTopics } from './roles';
import { capabilitiesTopics } from './capabilities';
import { technologyTopics } from './technology';
import { aiVisionTopics } from './ai-vision';
import { operationsTopics } from './operations';
import { appsDeployTopics } from './apps-deploy';
import type { TopicContent, BilingualContent } from '../topicTypes';

// English content (default)
export const TOPIC_CONTENT_EN: Record<string, TopicContent> = {
  ...Object.fromEntries(Object.entries(platformTopics).map(([slug, c]) => [`platform/${slug}`, c])),
  ...Object.fromEntries(Object.entries(solutionsTopics).map(([slug, c]) => [`solutions/${slug}`, c])),
  ...Object.fromEntries(Object.entries(rolesTopics).map(([slug, c]) => [`roles/${slug}`, c])),
  ...Object.fromEntries(Object.entries(capabilitiesTopics).map(([slug, c]) => [`capabilities/${slug}`, c])),
  ...Object.fromEntries(Object.entries(technologyTopics).map(([slug, c]) => [`technology/${slug}`, c])),
  ...Object.fromEntries(Object.entries(aiVisionTopics).map(([slug, c]) => [`ai-vision/${slug}`, c])),
  ...Object.fromEntries(Object.entries(operationsTopics).map(([slug, c]) => [`operations/${slug}`, c])),
  ...Object.fromEntries(Object.entries(appsDeployTopics).map(([slug, c]) => [`apps-deploy/${slug}`, c])),
};

// Russian content (partial, imported gracefully later as they are created)
import { platformTopicsRu } from './platform_ru';
import { solutionsTopicsRu } from './solutions_ru';
import { rolesTopicsRu } from './roles_ru';
import { technologyTopicsRu } from './technology_ru';
import { capabilitiesTopicsRu } from './capabilities_ru';
import { aiVisionTopicsRu } from './ai-vision_ru';
import { operationsTopicsRu } from './operations_ru';
import { appsDeployTopicsRu } from './apps-deploy_ru';

export const TOPIC_CONTENT_RU: Record<string, TopicContent> = {
  ...Object.fromEntries(Object.entries(platformTopicsRu).map(([slug, c]) => [`platform/${slug}`, c])),
  ...Object.fromEntries(Object.entries(solutionsTopicsRu).map(([slug, c]) => [`solutions/${slug}`, c])),
  ...Object.fromEntries(Object.entries(rolesTopicsRu).map(([slug, c]) => [`roles/${slug}`, c])),
  ...Object.fromEntries(Object.entries(technologyTopicsRu).map(([slug, c]) => [`technology/${slug}`, c])),
  ...Object.fromEntries(Object.entries(capabilitiesTopicsRu).map(([slug, c]) => [`capabilities/${slug}`, c])),
  ...Object.fromEntries(Object.entries(aiVisionTopicsRu).map(([slug, c]) => [`ai-vision/${slug}`, c])),
  ...Object.fromEntries(Object.entries(operationsTopicsRu).map(([slug, c]) => [`operations/${slug}`, c])),
  ...Object.fromEntries(Object.entries(appsDeployTopicsRu).map(([slug, c]) => [`apps-deploy/${slug}`, c])),
};

export function getTopicContent(categoryId: string, slug: string): BilingualContent | undefined {
  const path = `${categoryId}/${slug}`;
  const en = TOPIC_CONTENT_EN[path];
  if (!en) return undefined;
  
  return {
    en,
    ru: TOPIC_CONTENT_RU[path]
  };
}
