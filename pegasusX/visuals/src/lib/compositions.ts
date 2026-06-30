import { DURATION, PEGASUS_VIDEO, secondsToFrames } from '../style/tokens';

export type CompositionKind = 'short' | 'chapter' | 'ecosystem';

export type PegasusCompositionMeta = {
  id: string;
  category: string;
  slug: string;
  title: string;
  kind: CompositionKind;
  durationSeconds: number;
};

export const PEGASUS_COMPOSITIONS: PegasusCompositionMeta[] = [
  {
    id: 'OrderLifecycle',
    category: 'platform',
    slug: 'order-lifecycle',
    title: 'Order Lifecycle',
    kind: 'short',
    durationSeconds: DURATION.short,
  },
  {
    id: 'PegasusEcosystemFlow',
    category: 'platform',
    slug: 'pegasus-ecosystem-flow',
    title: 'Pegasus Ecosystem Flow',
    kind: 'ecosystem',
    durationSeconds: DURATION.ecosystem,
  },
];

export function compositionDurationInFrames(meta: PegasusCompositionMeta): number {
  return secondsToFrames(meta.durationSeconds, PEGASUS_VIDEO.fps);
}
