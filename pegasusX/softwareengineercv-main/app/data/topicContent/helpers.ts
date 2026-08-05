import type {
  FlowVariant,
  TopicCard,
  TopicContent,
  WhyItMatters,
  ProofItem,
} from '../topicTypes';

export const DEFAULT_PROOF: ProofItem[] = [
  { label: 'Roles', value: '6 connected' },
  { label: 'System of record', value: 'Cloud Spanner' },
  { label: 'Realtime', value: 'Outbox → Kafka → WS' },
  { label: 'Surfaces', value: 'Portal · Mobile · Desktop' },
];

export const DEFAULT_AI_DATA: TopicCard[] = [
  {
    title: 'Transactional outbox',
    description:
      'Every Spanner write emits outbox events in the same RW transaction — cache invalidation and Kafka fanout stay consistent with domain state.',
  },
  {
    title: 'Realtime coordination',
    description:
      'Redis + WebSocket hubs per role keep dispatch boards, fleet maps, and retailer tracking aligned without manual refresh.',
  },
  {
    title: 'AI with deterministic fallback',
    description:
      'ai-worker paths (dispatch optimizer, pre-orders, freeze locks) always degrade to pure deterministic engines — never block operations on model latency.',
  },
];

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
  capabilities?: TopicCard[];
  differentiators?: TopicCard[];
  whyItMatters?: WhyItMatters;
  edgeCases?: TopicCard[];
  aiAndData?: TopicCard[];
  proofItems?: ProofItem[];
};

/** Derive capability cards from outcomes when authors omit explicit capabilities. */
function capabilitiesFromOutcomes(outcomes: string[]): TopicCard[] {
  return outcomes.slice(0, 4).map((o, i) => ({
    title: o.split(/[—–.-]/)[0]?.trim().slice(0, 48) || `Capability ${i + 1}`,
    description: o,
  }));
}

function whyFromProblem(problem: string, outcomes: string[]): WhyItMatters {
  return {
    headline: 'Why this matters for supplier-led networks',
    body: problem,
    insights: outcomes.slice(0, 2).map((o) => ({
      title: o.split(/[—–.-]/)[0]?.trim().slice(0, 40) || 'Outcome',
      body: o,
    })),
  };
}

function differentiatorsFromSpecs(
  specs: TopicContent['specs'],
  outcomes: string[],
): TopicCard[] {
  if (specs && specs.length > 0) {
    return specs.slice(0, 3).map((s) => ({
      title: s.label,
      description: s.value,
    }));
  }
  return outcomes.slice(0, 3).map((o) => ({
    title: 'Platform advantage',
    description: o,
  }));
}

export function seedContent(seed: ContentSeed): TopicContent {
  const capabilities =
    seed.capabilities ?? capabilitiesFromOutcomes(seed.outcomes);
  const whyItMatters = seed.whyItMatters ?? whyFromProblem(seed.problem, seed.outcomes);
  const differentiators =
    seed.differentiators ?? differentiatorsFromSpecs(seed.specs, seed.outcomes);
  const edgeCases =
    seed.edgeCases ??
    cards([
      [
        'Operational exception',
        seed.problem.length > 180 ? `${seed.problem.slice(0, 180)}…` : seed.problem,
      ],
      [
        'Recovery path',
        seed.howItWorks[seed.howItWorks.length - 1]?.description ??
          'Operators resolve via audited override while Spanner and outbox stay consistent.',
      ],
      [
        'Cross-role notify',
        'Downstream roles receive WS / notification envelopes so screens do not diverge after recovery.',
      ],
    ]);

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
    capabilities,
    differentiators,
    whyItMatters,
    edgeCases,
    aiAndData: seed.aiAndData ?? DEFAULT_AI_DATA,
    proofItems: seed.proofItems ?? DEFAULT_PROOF,
  };
}

export function defaultHowItWorks(
  steps: [string, string][],
): { title: string; description: string }[] {
  return steps.map(([title, description]) => ({ title, description }));
}

export function cards(pairs: [string, string][]): TopicCard[] {
  return pairs.map(([title, description]) => ({ title, description }));
}
