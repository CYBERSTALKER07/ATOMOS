'use client';

import dynamic from 'next/dynamic';
import type { FlowConfig, FlowVariant } from '@/app/data/topicTypes';

const FlowSlot = dynamic(() => import('@/app/components/explore/FlowSlot'), { ssr: false });
const OrderLifecycleVideo = dynamic(
  () => import('@/app/components/lifecycle/OrderLifecycleVideo'),
  { ssr: false }
);

type FlowHeroProps = {
  flow: FlowVariant;
  flowConfig?: FlowConfig;
  slug?: string;
  useLifecycleVideo?: boolean;
};

export default function FlowHero({ flow, flowConfig, slug, useLifecycleVideo }: FlowHeroProps) {
  const lifecycleSlugs = new Set(['order-lifecycle', 'how-pegasus-works']);
  const showVideo = useLifecycleVideo ?? (slug ? lifecycleSlugs.has(slug) : false);

  return (
    <div className="-mx-4 md:-mx-[calc((100vw-100%)/2+1rem)]">
      {showVideo ? (
        <OrderLifecycleVideo variant="hero" />
      ) : (
        <FlowSlot variant={flow} config={flowConfig} />
      )}
    </div>
  );
}
