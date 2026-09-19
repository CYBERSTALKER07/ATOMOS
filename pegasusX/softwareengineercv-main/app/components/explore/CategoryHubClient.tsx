'use client';

import type { CategoryHub } from '@/app/data/topicPages';
import HubLayoutRenderer from '@/app/components/explore/hubs/HubLayoutRenderer';
import { getHubLayoutConfig } from '@/app/lib/explore/hubLayouts';

type CategoryHubClientProps = {
  hub: CategoryHub;
};

export default function CategoryHubClient({ hub }: CategoryHubClientProps) {
  const config = getHubLayoutConfig(hub.id);
  return <HubLayoutRenderer hub={hub} config={config} />;
}
