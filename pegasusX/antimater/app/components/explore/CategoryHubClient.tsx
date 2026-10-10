'use client';

import type { CategoryHub } from '@/app/data/topicPages';
import HubLayoutRenderer from '@/app/components/explore/hubs/HubLayoutRenderer';
import { getHubLayoutConfig } from '@/app/lib/explore/hubLayouts';
import { useLanguage } from '@/app/context/LanguageContext';

type CategoryHubClientProps = {
  hub: CategoryHub;
};

export default function CategoryHubClient({ hub }: CategoryHubClientProps) {
  const { language } = useLanguage();
  const config = getHubLayoutConfig(hub.id, language);
  return <HubLayoutRenderer hub={hub} config={config} />;
}
