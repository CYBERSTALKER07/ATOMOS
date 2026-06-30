import { platformTopics } from './platform';
import { solutionsTopics } from './solutions';
import { rolesTopics } from './roles';
import { capabilitiesTopics } from './capabilities';
import { technologyTopics } from './technology';
import { aiVisionTopics } from './ai-vision';
import { operationsTopics } from './operations';
import { appsDeployTopics } from './apps-deploy';
import type { TopicContent } from '../topicTypes';

export const TOPIC_CONTENT_BY_PATH: Record<string, TopicContent> = {
  ...Object.fromEntries(Object.entries(platformTopics).map(([slug, c]) => [`platform/${slug}`, c])),
  ...Object.fromEntries(Object.entries(solutionsTopics).map(([slug, c]) => [`solutions/${slug}`, c])),
  ...Object.fromEntries(Object.entries(rolesTopics).map(([slug, c]) => [`roles/${slug}`, c])),
  ...Object.fromEntries(Object.entries(capabilitiesTopics).map(([slug, c]) => [`capabilities/${slug}`, c])),
  ...Object.fromEntries(Object.entries(technologyTopics).map(([slug, c]) => [`technology/${slug}`, c])),
  ...Object.fromEntries(Object.entries(aiVisionTopics).map(([slug, c]) => [`ai-vision/${slug}`, c])),
  ...Object.fromEntries(Object.entries(operationsTopics).map(([slug, c]) => [`operations/${slug}`, c])),
  ...Object.fromEntries(Object.entries(appsDeployTopics).map(([slug, c]) => [`apps-deploy/${slug}`, c])),
};

export function getTopicContent(categoryId: string, slug: string): TopicContent | undefined {
  return TOPIC_CONTENT_BY_PATH[`${categoryId}/${slug}`];
}
