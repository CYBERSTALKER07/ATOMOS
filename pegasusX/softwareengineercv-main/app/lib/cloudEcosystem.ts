export type CloudTechIconId =
  | 'googlecloud'
  | 'spanner'
  | 'kafka'
  | 'redis'
  | 'kubernetes'
  | 'docker'
  | 'firebase'
  | 'go'
  | 'nextjs'
  | 'react'
  | 'terraform'
  | 'githubactions'
  | 'bigquery'
  | 'netlify'
  | 'prometheus'
  | 'grafana'
  | 'nginx'
  | 'postgresql'
  | 'typescript'
  | 'android'
  | 'apple'
  | 'opentelemetry'
  | 'cloudflare';

export type CloudTechSpan = '1x1' | '2x1' | '2x2' | '3x1' | '3x2' | '4x1';

export type CloudTechCategory =
  | 'cloud'
  | 'data'
  | 'messaging'
  | 'runtime'
  | 'delivery'
  | 'observability'
  | 'clients';

export type CloudTechItem = {
  id: string;
  name: string;
  nameRu: string;
  blurb: string;
  blurbRu: string;
  category: CloudTechCategory;
  icon: CloudTechIconId;
  href?: string;
  brand: string;
  span: CloudTechSpan;
  featured?: boolean;
};

/** Production stack shown on the Cloud Ecosystem section + page. */
export const CLOUD_ECOSYSTEM_TECH: CloudTechItem[] = [
  {
    id: 'gcp',
    name: 'Google Cloud',
    nameRu: 'Google Cloud',
    blurb: 'Primary cloud fabric — IAM, networking, and managed services under one project.',
    blurbRu: 'Основное облако — IAM, сеть и управляемые сервисы в одном проекте.',
    category: 'cloud',
    icon: 'googlecloud',
    href: 'https://cloud.google.com',
    brand: '#4285F4',
    span: '3x2',
    featured: true,
  },
  {
    id: 'spanner',
    name: 'Cloud Spanner',
    nameRu: 'Cloud Spanner',
    blurb: 'Globally consistent order truth with ACID transactions across regions.',
    blurbRu: 'Глобально согласованная правда заказа с ACID-транзакциями по регионам.',
    category: 'data',
    icon: 'spanner',
    href: '/technology/cloud-spanner',
    brand: '#4285F4',
    span: '3x2',
    featured: true,
  },
  {
    id: 'kafka',
    name: 'Apache Kafka',
    nameRu: 'Apache Kafka',
    blurb: 'Event backbone for outbox → live fanout across every role inbox.',
    blurbRu: 'Шина событий: outbox → живой fanout во все ролевые inbox.',
    category: 'messaging',
    icon: 'kafka',
    href: '/technology/redis-kafka',
    brand: '#231F20',
    span: '2x1',
  },
  {
    id: 'redis',
    name: 'Redis',
    nameRu: 'Redis',
    blurb: 'Sub-millisecond cache and hot state for catalogs, sessions, and boards.',
    blurbRu: 'Кэш и горячее состояние для каталогов, сессий и досок.',
    category: 'data',
    icon: 'redis',
    href: '/technology/redis-kafka',
    brand: '#DC382D',
    span: '2x1',
  },
  {
    id: 'gke',
    name: 'GKE / Kubernetes',
    nameRu: 'GKE / Kubernetes',
    blurb: 'Container orchestration for backend workers, relays, and APIs.',
    blurbRu: 'Оркестрация контейнеров для API, воркеров и релеев.',
    category: 'runtime',
    icon: 'kubernetes',
    href: 'https://cloud.google.com/kubernetes-engine',
    brand: '#326CE5',
    span: '2x1',
  },
  {
    id: 'memorystore',
    name: 'Memorystore',
    nameRu: 'Memorystore',
    blurb: 'Managed Redis on Google Cloud for durable cache clusters.',
    blurbRu: 'Управляемый Redis в Google Cloud для кэш-кластеров.',
    category: 'data',
    icon: 'redis',
    href: 'https://cloud.google.com/memorystore',
    brand: '#DC382D',
    span: '2x1',
  },
  {
    id: 'firebase',
    name: 'Firebase Auth',
    nameRu: 'Firebase Auth',
    blurb: 'Phone OTP for driver, factory, and gate teams in the field.',
    blurbRu: 'OTP по SMS для водителей, завода и ворот в поле.',
    category: 'cloud',
    icon: 'firebase',
    href: '/technology/firebase-otp',
    brand: '#FFCA28',
    span: '2x1',
  },
  {
    id: 'bigquery',
    name: 'BigQuery',
    nameRu: 'BigQuery',
    blurb: 'Analytics warehouse for network throughput and SLA reporting.',
    blurbRu: 'Аналитическое хранилище для пропускной способности и SLA.',
    category: 'data',
    icon: 'bigquery',
    href: 'https://cloud.google.com/bigquery',
    brand: '#669DF6',
    span: '2x1',
  },
  {
    id: 'gcs',
    name: 'Cloud Storage',
    nameRu: 'Cloud Storage',
    blurb: 'Object storage for manifests, seals, receipts, and media.',
    blurbRu: 'Объектное хранилище для манифестов, пломб, чеков и медиа.',
    category: 'cloud',
    icon: 'googlecloud',
    href: 'https://cloud.google.com/storage',
    brand: '#4285F4',
    span: '2x1',
  },
  {
    id: 'secret-manager',
    name: 'Secret Manager',
    nameRu: 'Secret Manager',
    blurb: 'Encrypted credentials for payments, partners, and webhooks.',
    blurbRu: 'Шифрованные секреты для платежей, партнёров и webhooks.',
    category: 'cloud',
    icon: 'googlecloud',
    href: 'https://cloud.google.com/secret-manager',
    brand: '#34A853',
    span: '2x1',
  },
  {
    id: 'cloud-armor',
    name: 'Cloud Armor',
    nameRu: 'Cloud Armor',
    blurb: 'Edge WAF and DDoS protection in front of public APIs.',
    blurbRu: 'WAF и защита от DDoS перед публичными API.',
    category: 'cloud',
    icon: 'googlecloud',
    href: 'https://cloud.google.com/armor',
    brand: '#EA4335',
    span: '2x1',
  },
  {
    id: 'load-balancing',
    name: 'Cloud Load Balancing',
    nameRu: 'Cloud Load Balancing',
    blurb: 'Global HTTPS entry for API and portal traffic.',
    blurbRu: 'Глобальный HTTPS-вход для API и порталов.',
    category: 'cloud',
    icon: 'googlecloud',
    href: 'https://cloud.google.com/load-balancing',
    brand: '#4285F4',
    span: '2x1',
  },
  {
    id: 'go',
    name: 'Go Backend',
    nameRu: 'Go Backend',
    blurb: 'One modular platform core serving every role surface.',
    blurbRu: 'Единое модульное ядро платформы для всех ролей.',
    category: 'runtime',
    icon: 'go',
    href: '/technology/go-backend-platform',
    brand: '#00ADD8',
    span: '2x1',
  },
  {
    id: 'docker',
    name: 'Docker',
    nameRu: 'Docker',
    blurb: 'Immutable images for services, workers, and local SSMR stacks.',
    blurbRu: 'Неизменяемые образы для сервисов, воркеров и локального SSMR.',
    category: 'runtime',
    icon: 'docker',
    href: 'https://www.docker.com',
    brand: '#2496ED',
    span: '1x1',
  },
  {
    id: 'terraform',
    name: 'Terraform',
    nameRu: 'Terraform',
    blurb: 'Infrastructure as code for GCP projects and Kafka topics.',
    blurbRu: 'Инфраструктура как код для GCP и топиков Kafka.',
    category: 'delivery',
    icon: 'terraform',
    href: 'https://www.terraform.io',
    brand: '#7B42BC',
    span: '1x1',
  },
  {
    id: 'gha',
    name: 'GitHub Actions',
    nameRu: 'GitHub Actions',
    blurb: 'CI gates for contracts, money-path, and deploy pipelines.',
    blurbRu: 'CI-гейты для контрактов, money-path и деплоев.',
    category: 'delivery',
    icon: 'githubactions',
    href: 'https://github.com/features/actions',
    brand: '#2088FF',
    span: '1x1',
  },
  {
    id: 'nextjs',
    name: 'Next.js',
    nameRu: 'Next.js',
    blurb: 'Supplier portals, ops boards, and the marketing site.',
    blurbRu: 'Порталы поставщика, ops-доски и маркетинговый сайт.',
    category: 'clients',
    icon: 'nextjs',
    href: '/technology/next-js-surfaces',
    brand: '#FFFFFF',
    span: '2x1',
  },
  {
    id: 'react',
    name: 'React',
    nameRu: 'React',
    blurb: 'Interactive surfaces shared across web and desktop shells.',
    blurbRu: 'Интерактивные поверхности для web и desktop.',
    category: 'clients',
    icon: 'react',
    href: 'https://react.dev',
    brand: '#61DAFB',
    span: '1x1',
  },
  {
    id: 'android',
    name: 'Android',
    nameRu: 'Android',
    blurb: 'Native Kotlin apps for warehouse, driver, factory, and gate.',
    blurbRu: 'Нативные Kotlin-приложения для склада, водителя, завода и ворот.',
    category: 'clients',
    icon: 'android',
    href: '/technology/native-mobile-desktop',
    brand: '#3DDC84',
    span: '1x1',
  },
  {
    id: 'ios',
    name: 'iOS',
    nameRu: 'iOS',
    blurb: 'Swift apps for retailer, supplier, and field roles.',
    blurbRu: 'Swift-приложения для ритейлера, поставщика и полевых ролей.',
    category: 'clients',
    icon: 'apple',
    href: '/technology/native-mobile-desktop',
    brand: '#FFFFFF',
    span: '1x1',
  },
  {
    id: 'netlify',
    name: 'Netlify',
    nameRu: 'Netlify',
    blurb: 'Edge delivery for the public marketing experience.',
    blurbRu: 'Edge-доставка публичного маркетингового сайта.',
    category: 'delivery',
    icon: 'netlify',
    href: 'https://www.netlify.com',
    brand: '#00C7B7',
    span: '1x1',
  },
  {
    id: 'prometheus',
    name: 'Prometheus',
    nameRu: 'Prometheus',
    blurb: 'Metrics scrape path for SLOs and worker health.',
    blurbRu: 'Метрики для SLO и здоровья воркеров.',
    category: 'observability',
    icon: 'prometheus',
    href: 'https://prometheus.io',
    brand: '#E6522C',
    span: '1x1',
  },
  {
    id: 'grafana',
    name: 'Grafana',
    nameRu: 'Grafana',
    blurb: 'Ops dashboards over latency, lag, and error budgets.',
    blurbRu: 'Ops-дашборды по latency, lag и error budget.',
    category: 'observability',
    icon: 'grafana',
    href: 'https://grafana.com',
    brand: '#F46800',
    span: '1x1',
  },
  {
    id: 'otel',
    name: 'OpenTelemetry',
    nameRu: 'OpenTelemetry',
    blurb: 'Traces and spans across API → Spanner → Kafka paths.',
    blurbRu: 'Трейсы по путям API → Spanner → Kafka.',
    category: 'observability',
    icon: 'opentelemetry',
    href: 'https://opentelemetry.io',
    brand: '#FFFFFF',
    span: '2x1',
  },
  {
    id: 'nginx',
    name: 'NGINX',
    nameRu: 'NGINX',
    blurb: 'Ingress and reverse-proxy edge for clustered services.',
    blurbRu: 'Ingress и reverse-proxy на краю кластера.',
    category: 'runtime',
    icon: 'nginx',
    href: 'https://nginx.org',
    brand: '#009639',
    span: '1x1',
  },
  {
    id: 'typescript',
    name: 'TypeScript',
    nameRu: 'TypeScript',
    blurb: 'Shared contracts across web clients and API packages.',
    blurbRu: 'Общие контракты для web-клиентов и API-пакетов.',
    category: 'clients',
    icon: 'typescript',
    href: 'https://www.typescriptlang.org',
    brand: '#3178C6',
    span: '1x1',
  },
];

export const CLOUD_ECOSYSTEM_CATEGORIES: {
  id: CloudTechCategory;
  label: string;
  labelRu: string;
}[] = [
  { id: 'cloud', label: 'Google Cloud', labelRu: 'Google Cloud' },
  { id: 'data', label: 'Data & cache', labelRu: 'Данные и кэш' },
  { id: 'messaging', label: 'Events', labelRu: 'События' },
  { id: 'runtime', label: 'Runtime', labelRu: 'Рантайм' },
  { id: 'delivery', label: 'Delivery', labelRu: 'Доставка кода' },
  { id: 'observability', label: 'Observability', labelRu: 'Наблюдаемость' },
  { id: 'clients', label: 'Client surfaces', labelRu: 'Клиентские поверхности' },
];

/** Compact subset for the home-page bento. */
export const CLOUD_ECOSYSTEM_HOME_IDS = [
  'gcp',
  'spanner',
  'kafka',
  'redis',
  'gke',
  'firebase',
  'go',
  'bigquery',
  'docker',
  'terraform',
  'gha',
  'nextjs',
  'android',
  'ios',
  'netlify',
  'otel',
] as const;

export function cloudEcosystemHomeItems(): CloudTechItem[] {
  const byId = new Map(CLOUD_ECOSYSTEM_TECH.map((item) => [item.id, item]));
  return CLOUD_ECOSYSTEM_HOME_IDS.map((id) => byId.get(id)).filter(Boolean) as CloudTechItem[];
}
