import type {
  FlowVariant,
  TopicCard,
  TopicContent,
  WhyItMatters,
  ProofItem,
} from '../topicTypes';

export const DEFAULT_PROOF: ProofItem[] = [
  { label: 'Roles', value: '6 connected' },
  { label: 'System of record', value: 'One shared truth' },
  { label: 'Live updates', value: 'Boards stay in sync' },
  { label: 'Surfaces', value: 'Portal · Mobile · Desktop' },
];

export const DEFAULT_PROOF_RU: ProofItem[] = [
  { label: 'Роли', value: '6 подключено' },
  { label: 'Система учета', value: 'Единая истина' },
  { label: 'Обновления', value: 'Синхронизация досок' },
  { label: 'Интерфейсы', value: 'Портал · Мобильный · Десктоп' },
];

export const DEFAULT_AI_DATA: TopicCard[] = [
  {
    title: 'Reliable change events',
    description:
      'Every confirmed update notifies the right apps in the same step — screens never diverge from the order truth.',
  },
  {
    title: 'Live coordination',
    description:
      'Dispatch boards, fleet maps, and retailer tracking stay aligned automatically — no manual refresh.',
  },
  {
    title: 'AI with proven fallback',
    description:
      'Smart assist for dispatch and recommendations always falls back to proven planning rules — never blocks the floor if models are slow.',
  },
];

export const DEFAULT_AI_DATA_RU: TopicCard[] = [
  {
    title: 'Надежные события изменений',
    description:
      'Каждое подтвержденное обновление уведомляет нужные приложения в том же шаге — экраны никогда не расходятся с истиной заказа.',
  },
  {
    title: 'Живая координация',
    description:
      'Диспетчерские доски, карты автопарка и отслеживание ритейлера автоматически синхронизируются без ручного обновления.',
  },
  {
    title: 'ИИ с надежным откатом',
    description:
      'Умный помощник диспетчера всегда откатывается к проверенным правилам планирования — работа не останавливается, если модели задерживаются.',
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
          'Operators resolve via audited override while the shared order record and live updates stay consistent.',
      ],
      [
        'Cross-role notify',
        'Downstream roles receive live alerts and screen refreshes so views do not diverge after recovery.',
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
