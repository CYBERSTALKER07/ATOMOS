'use client';

import type { TopicPage } from '@/app/data/topicTypes';
import O9DetailLayout from '@/app/components/explore/O9DetailLayout';
import { getTopicLayoutConfig } from '@/app/lib/explore/topicLayouts';
import { useLanguage } from '@/app/context/LanguageContext';

type TopicPageClientProps = {
  topic: TopicPage;
};

export default function TopicPageClient({ topic }: TopicPageClientProps) {
  const { language } = useLanguage();
  const content = (topic.content as any)?.[language] || topic.content?.en || (topic.content as any);
  const config = getTopicLayoutConfig(content?.flow);

  return (
    <O9DetailLayout topic={topic} showFleetShowcase={config.showFleetShowcase} />
  );
}
