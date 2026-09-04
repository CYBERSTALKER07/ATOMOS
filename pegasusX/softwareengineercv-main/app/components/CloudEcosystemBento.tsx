'use client';

import type { ReactNode } from 'react';
import Link from 'next/link';
import {
  SiGooglecloud,
  SiGooglebigquery,
  SiApachekafka,
  SiRedis,
  SiKubernetes,
  SiDocker,
  SiFirebase,
  SiGo,
  SiNextdotjs,
  SiReact,
  SiTerraform,
  SiGithubactions,
  SiNetlify,
  SiPrometheus,
  SiGrafana,
  SiNginx,
  SiPostgresql,
  SiTypescript,
  SiAndroid,
  SiApple,
  SiOpentelemetry,
  SiCloudflare,
} from 'react-icons/si';
import { useLanguage } from '../context/LanguageContext';
import {
  type CloudTechIconId,
  type CloudTechItem,
  type CloudTechSpan,
} from '@/app/lib/cloudEcosystem';
import { cn } from '@/lib/utils';

const ICONS: Record<CloudTechIconId, ReactNode> = {
  googlecloud: <SiGooglecloud />,
  spanner: <SiGooglecloud />,
  kafka: <SiApachekafka />,
  redis: <SiRedis />,
  kubernetes: <SiKubernetes />,
  docker: <SiDocker />,
  firebase: <SiFirebase />,
  go: <SiGo />,
  nextjs: <SiNextdotjs />,
  react: <SiReact />,
  terraform: <SiTerraform />,
  githubactions: <SiGithubactions />,
  bigquery: <SiGooglebigquery />,
  netlify: <SiNetlify />,
  prometheus: <SiPrometheus />,
  grafana: <SiGrafana />,
  nginx: <SiNginx />,
  postgresql: <SiPostgresql />,
  typescript: <SiTypescript />,
  android: <SiAndroid />,
  apple: <SiApple />,
  opentelemetry: <SiOpentelemetry />,
  cloudflare: <SiCloudflare />,
};

const SPAN_CLASS: Record<CloudTechSpan, string> = {
  '1x1': 'md:col-span-1 md:row-span-1',
  '2x1': 'md:col-span-2 md:row-span-1',
  '2x2': 'md:col-span-2 md:row-span-2',
  '3x1': 'md:col-span-3 md:row-span-1',
  '3x2': 'md:col-span-3 md:row-span-2',
  '4x1': 'md:col-span-4 md:row-span-1',
};

type CloudEcosystemBentoProps = {
  items: CloudTechItem[];
  compact?: boolean;
};

function TechTile({ item, compact }: { item: CloudTechItem; compact?: boolean }) {
  const { language } = useLanguage();
  const isRu = language === 'ru';
  const name = isRu ? item.nameRu : item.name;
  const blurb = isRu ? item.blurbRu : item.blurb;
  const icon = ICONS[item.icon];
  const isExternal = Boolean(item.href?.startsWith('http'));

  const body = (
    <>
      <div className="flex items-start justify-between gap-3">
        <span
          className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/[0.04] text-[1.55rem] transition-transform duration-300 group-hover:scale-110"
          style={{ color: item.brand }}
          aria-hidden
        >
          {icon}
        </span>
        {item.featured ? (
          <span className="font-mono text-[9px] uppercase tracking-[0.18em] text-emerald-400/80">
            Core
          </span>
        ) : null}
      </div>
      <div className="mt-auto space-y-2 pt-5">
        <h3 className="text-base md:text-lg font-medium tracking-tight text-white group-hover:text-emerald-100 transition-colors">
          {name}
        </h3>
        {!compact || item.featured ? (
          <p className="text-xs md:text-sm font-light leading-relaxed text-white/55 group-hover:text-white/75 transition-colors line-clamp-3">
            {blurb}
          </p>
        ) : null}
      </div>
      <div
        className="pointer-events-none absolute inset-0 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
        style={{
          background: `radial-gradient(420px circle at 20% 0%, ${item.brand}22, transparent 55%)`,
        }}
      />
    </>
  );

  const className = cn(
    'group relative flex h-full min-h-[8.5rem] flex-col overflow-hidden border border-white/10 bg-[#070707] p-5 md:p-6',
    'transition-all duration-300 hover:-translate-y-0.5 hover:border-emerald-400/45',
    'hover:shadow-[0_0_36px_rgba(141,220,150,0.18)]',
    SPAN_CLASS[item.span],
    item.featured && 'min-h-[12rem]',
  );

  if (!item.href) {
    return <div className={className}>{body}</div>;
  }

  if (isExternal) {
    return (
      <a
        href={item.href}
        target="_blank"
        rel="noopener noreferrer"
        className={className}
        aria-label={name}
      >
        {body}
      </a>
    );
  }

  return (
    <Link href={item.href} className={className} aria-label={name}>
      {body}
    </Link>
  );
}

export default function CloudEcosystemBento({ items, compact = false }: CloudEcosystemBentoProps) {
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-6 md:auto-rows-[minmax(8.75rem,auto)] md:grid-flow-dense">
      {items.map((item) => (
        <TechTile key={item.id} item={item} compact={compact} />
      ))}
    </div>
  );
}
