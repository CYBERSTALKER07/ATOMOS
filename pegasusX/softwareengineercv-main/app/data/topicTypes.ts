export type FlowVariant =
  | 'orderLifecycle'
  | 'controlPlane'
  | 'mutatingHandler'
  | 'realtimePipeline'
  | 'dispatchBoard'
  | 'fleetMap'
  | 'paymentFlow'
  | 'roleJourney'
  | 'topologyMap'
  | 'techStack'
  | 'aiAssist'
  | 'exceptionPlaybook'
  | 'appsMatrix';

export type FlowConfig = {
  highlightStep?: number;
  roles?: string[];
};

export type TopicCard = {
  title: string;
  description: string;
};

export type WhyItMatters = {
  headline: string;
  body: string;
  insights?: { title: string; body: string }[];
};

export type ProofItem = {
  label: string;
  value: string;
};

export type TopicContent = {
  title: string;
  summary: string;
  problem: string;
  outcomes: string[];
  howItWorks: { title: string; description: string }[];
  crossRole?: { role: string; touchpoint: string }[];
  specs?: { label: string; value: string }[];
  flow: FlowVariant;
  flowConfig?: FlowConfig;
  relatedProjectSlug?: string;
  /** o9 CORE CAPABILITIES */
  capabilities?: TopicCard[];
  /** o9 KEY DIFFERENTIATORS */
  differentiators?: TopicCard[];
  /** o9 why it matters block */
  whyItMatters?: WhyItMatters;
  /** Exception / edge-case paths from order flow docs */
  edgeCases?: TopicCard[];
  /** AI workers + Spanner/Redis/Kafka/WS data layer */
  aiAndData?: TopicCard[];
  /** Proof strip facts */
  proofItems?: ProofItem[];
};

export type TopicPage = {
  categoryId: string;
  categoryLabel: string;
  slug: string;
  label: string;
  description?: string;
  badge?: 'NEW';
  href: string;
  content: TopicContent;
};

export const EXPLORE_CATEGORY_IDS = [
  'platform',
  'roles',
  'capabilities',
  'technology',
  'ai-vision',
  'operations',
  'apps-deploy',
] as const;

export type ExploreCategoryId = (typeof EXPLORE_CATEGORY_IDS)[number];

export function isExploreCategoryId(id: string): id is ExploreCategoryId {
  return (EXPLORE_CATEGORY_IDS as readonly string[]).includes(id);
}

export function labelToSlug(label: string): string {
  return label
    .toLowerCase()
    .replace(/\s*\/\s*/g, '-')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}

export function topicHref(categoryId: string, slug: string): string {
  return `/${categoryId}/${slug}`;
}
