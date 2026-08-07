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
  const content = topic.content[language] || topic.content.en;
  const config = getTopicLayoutConfig(content.flow);
  
  // Clone topic but override content with the resolved localized content
  // We cast to any because O9DetailLayout might expect BilingualContent later but it's simpler this way
  const localizedTopic = { ...topic, content };
  
  return (
    <O9DetailLayout topic={localizedTopic as any} showFleetShowcase={config.showFleetShowcase} />
  );
}
