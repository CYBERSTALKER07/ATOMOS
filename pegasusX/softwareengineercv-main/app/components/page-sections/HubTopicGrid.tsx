'use client';

import ContentCard from '@/app/components/ContentCard';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import type { TopicPage } from '@/app/data/topicTypes';
import { cn } from '@/lib/utils';
import { useLanguage } from '@/app/context/LanguageContext';

const FLEET_HUB_IDS = new Set(['solutions', 'apps-deploy']);

function hubTopicImage(hubId: string, index: number): string {
  if (FLEET_HUB_IDS.has(hubId)) {
    return FLEET_TRUCK_IMAGES[index % FLEET_TRUCK_IMAGES.length].src;
  }
  return EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length];
}

type HubTopicGridProps = {
  hubId: string;
  hubLabel: string;
  topics: TopicPage[];
  layout?: 'uniform' | 'masonry' | 'featured';
};

export default function HubTopicGrid({
  hubId,
  hubLabel,
  topics,
  layout = 'uniform',
}: HubTopicGridProps) {
  const { language, t } = useLanguage();
  const localizedTag = t(`nav_${hubId}`, hubLabel);

  if (layout === 'featured' && topics.length > 0) {
    const [featured, ...rest] = topics;
    const featuredContent = featured.content[language] || featured.content.en;
    return (
      <div className="mt-16 space-y-4">
        <ContentCard
          variant="featured"
          tone="light"
          tag={localizedTag}
          title={featuredContent.title}
          description={featuredContent.summary}
          href={featured.href}
          image={hubTopicImage(hubId, 0)}
          className="min-h-[280px]"
        />
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {rest.map((topic, i) => {
            const content = topic.content[language] || topic.content.en;
            return (
              <ContentCard
                key={topic.slug}
                variant="vertical"
                tone={i % 3 === 0 ? 'light' : 'dark'}
                tag={localizedTag}
                title={content.title}
                description={content.summary}
                href={topic.href}
                image={hubTopicImage(hubId, i + 1)}
              />
            );
          })}
        </div>
      </div>
    );
  }

  if (layout === 'masonry') {
    return (
      <div className="mt-16 editorial-bento max-w-none">
        {topics.map((topic, i) => {
          const content = topic.content[language] || topic.content.en;
          return (
            <ContentCard
              key={topic.slug}
              variant={i % 5 === 0 ? 'featured' : i % 2 === 0 ? 'split' : 'vertical'}
              tone={i % 4 === 0 ? 'light' : 'dark'}
              tag={localizedTag}
              title={content.title}
              description={content.summary}
              href={topic.href}
              image={hubTopicImage(hubId, i)}
              className={cn(i % 5 === 0 && 'md:col-span-2')}
            />
          );
        })}
      </div>
    );
  }

  return (
    <div className="mt-16 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {topics.map((topic, i) => {
        const content = topic.content[language] || topic.content.en;
        return (
          <ContentCard
            key={topic.slug}
            variant="vertical"
            tone={i % 3 === 0 ? 'light' : 'dark'}
            tag={localizedTag}
            title={content.title}
            description={content.summary}
            href={topic.href}
            image={hubTopicImage(hubId, i)}
          />
        );
      })}
    </div>
  );
}
