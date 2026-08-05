'use client';

import type { TopicPage } from '@/app/data/topicTypes';
import O9DetailLayout from '@/app/components/explore/O9DetailLayout';
import { getTopicLayoutConfig } from '@/app/lib/explore/topicLayouts';

type TopicPageClientProps = {
  topic: TopicPage;
};

export default function TopicPageClient({ topic }: TopicPageClientProps) {
  const config = getTopicLayoutConfig(topic.content.flow);
  return (
    <O9DetailLayout topic={topic} showFleetShowcase={config.showFleetShowcase} />
  );
}
