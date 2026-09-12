import type { FlowVariant } from '@/app/data/topicTypes';

/**
 * Topic layout config — optional visuals only.
 * Section order is fixed by O9DetailLayout (hero → proof → why → caps → … → tour).
 */
export type TopicLayoutConfig = {
  showFleetShowcase?: boolean;
};

const FLEET_FLOWS: FlowVariant[] = ['dispatchBoard', 'fleetMap'];

export const topicLayoutConfigs: Partial<Record<FlowVariant, TopicLayoutConfig>> = {
  dispatchBoard: { showFleetShowcase: true },
  fleetMap: { showFleetShowcase: true },
};

export function getTopicLayoutConfig(flow: FlowVariant): TopicLayoutConfig {
  return (
    topicLayoutConfigs[flow] ?? {
      showFleetShowcase: FLEET_FLOWS.includes(flow),
    }
  );
}
