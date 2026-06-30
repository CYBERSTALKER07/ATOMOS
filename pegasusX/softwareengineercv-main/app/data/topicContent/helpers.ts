import type { FlowVariant, TopicContent } from '../topicTypes';

type ContentSeed = {
  title: string;
  summary: string;
  problem: string;
  outcomes: string[];
  howItWorks: { title: string; description: string }[];
  flow: FlowVariant;
  flowConfig?: TopicContent['flowConfig'];
  relatedProjectSlug?: string;
  crossRole?: TopicContent['crossRole'];
  specs?: TopicContent['specs'];
};

export function seedContent(seed: ContentSeed): TopicContent {
  return {
    title: seed.title,
    summary: seed.summary,
    problem: seed.problem,
    outcomes: seed.outcomes,
    howItWorks: seed.howItWorks,
    flow: seed.flow,
    flowConfig: seed.flowConfig,
    relatedProjectSlug: seed.relatedProjectSlug,
    crossRole: seed.crossRole,
    specs: seed.specs,
  };
}

export function defaultHowItWorks(
  steps: [string, string][],
): { title: string; description: string }[] {
  return steps.map(([title, description]) => ({ title, description }));
}
