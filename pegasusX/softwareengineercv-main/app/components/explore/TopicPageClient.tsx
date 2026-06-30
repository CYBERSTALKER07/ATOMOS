'use client';

import dynamic from 'next/dynamic';
import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import type { TopicPage, FlowVariant } from '@/app/data/topicTypes';
import { getSiblingTopics } from '@/app/data/topicPages';
import SiteNav from './SiteNav';
import FlowSlot from './FlowSlot';
import TopicSection from './TopicSection';
import ChamferButton from '@/app/components/ChamferButton';
import Footer from '@/app/components/Footer';
import ContentCard, { ContentCardEyebrow } from '@/app/components/ContentCard';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';

const FleetScrollShowcase = dynamic(() => import('@/app/components/fleet/FleetScrollShowcase'), {
  ssr: false,
});

const OrderLifecycleVideo = dynamic(() => import('@/app/components/lifecycle/OrderLifecycleVideo'), {
  ssr: false,
});

const LIFECYCLE_SLUGS = new Set(['order-lifecycle', 'how-pegasus-works']);

const FLEET_FLOWS: FlowVariant[] = ['fleetMap', 'dispatchBoard'];

function topicImageForFlow(flow: FlowVariant, index: number): string {
  if (FLEET_FLOWS.includes(flow)) {
    return FLEET_TRUCK_IMAGES[index % FLEET_TRUCK_IMAGES.length].src;
  }
  return EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length];
}

function fleetShowcaseCopy(topic: TopicPage) {
  if (topic.content.flow === 'fleetMap') {
    return {
      eyebrow: 'Fleet telemetry',
      title: topic.content.title,
      subtitle: topic.content.summary,
      learnMoreHref: '/solutions/live-fleet-tracking',
    };
  }
  return {
    eyebrow: 'Dispatch operations',
    title: topic.content.title,
    subtitle: topic.content.summary,
    learnMoreHref: '/solutions/visual-dispatch-engine',
  };
}

type TopicPageClientProps = {
  topic: TopicPage;
};

export default function TopicPageClient({ topic }: TopicPageClientProps) {
  const heroRef = useRef<HTMLDivElement>(null);
  const { content, categoryLabel, categoryId, slug, badge } = topic;

  useEffect(() => {
    if (!heroRef.current) return;
    gsap.fromTo(heroRef.current, { opacity: 0, y: 40 }, { opacity: 1, y: 0, duration: 0.9, ease: 'power3.out' });
  }, []);

  const siblings = getSiblingTopics(categoryId, slug, 4);

  return (
    <main className="min-h-screen bg-black text-white">
      <SiteNav activeHref={`/${categoryId}`} />

      <div className="container mx-auto px-4 pb-20 pt-24 md:pt-28">
        <div ref={heroRef}>
          <Link href={`/${categoryId}`} className="editorial-btn editorial-btn--sm mb-8">
            ← All {categoryLabel}
          </Link>
          <ContentCardEyebrow>{categoryLabel}</ContentCardEyebrow>
          <h1 className="mt-4 max-w-4xl text-4xl font-semibold tracking-tight md:text-6xl">
            {content.title}
            {badge ? (
              <span className="ml-3 align-middle font-mono text-xs tracking-widest text-white/60">[NEW]</span>
            ) : null}
          </h1>
          <p className="mt-6 max-w-3xl text-lg text-white/70">{content.summary}</p>
        </div>

        <div className="-mx-4 mt-12 md:-mx-[calc((100vw-100%)/2+1rem)]">
          {LIFECYCLE_SLUGS.has(slug) ? (
            <OrderLifecycleVideo variant="hero" />
          ) : (
            <FlowSlot variant={content.flow} config={content.flowConfig} />
          )}
        </div>

        {FLEET_FLOWS.includes(content.flow) ? (
          <div className="-mx-4 mt-0 md:-mx-[calc((100vw-100%)/2+1rem)]">
            <FleetScrollShowcase {...fleetShowcaseCopy(topic)} />
          </div>
        ) : null}

        <TopicSection eyebrow="Problem" title="The problem">
          <p className="max-w-3xl text-white/70 leading-relaxed">{content.problem}</p>
        </TopicSection>

        <TopicSection eyebrow="Outcomes" title="What changes">
          <ul className="grid gap-3 md:grid-cols-2">
            {content.outcomes.map((item) => (
              <li key={item} className="border border-white/15 p-4 text-sm text-white/80">
                {item}
              </li>
            ))}
          </ul>
        </TopicSection>

        <TopicSection eyebrow="Process" title="How it works">
          <div className="grid gap-4 md:grid-cols-3">
            {content.howItWorks.map((step, i) => (
              <div key={step.title} className="border border-white/15 p-6">
                <p className="font-mono text-xs uppercase tracking-wider text-white/40">
                  Step {String(i + 1).padStart(2, '0')}
                </p>
                <h3 className="mt-3 font-semibold">{step.title}</h3>
                <p className="mt-2 text-sm text-white/60">{step.description}</p>
              </div>
            ))}
          </div>
        </TopicSection>

        {content.crossRole && content.crossRole.length > 0 ? (
          <TopicSection eyebrow="Roles" title="Who's involved">
            <div className="overflow-x-auto">
              <table className="w-full min-w-[400px] text-left text-sm">
                <thead>
                  <tr className="border-b border-white/20 font-mono text-xs uppercase text-white/50">
                    <th className="py-3 pr-4">Role</th>
                    <th className="py-3">Touchpoint</th>
                  </tr>
                </thead>
                <tbody>
                  {content.crossRole.map((row) => (
                    <tr key={row.role} className="border-b border-white/10">
                      <td className="py-3 pr-4 font-medium">{row.role}</td>
                      <td className="py-3 text-white/60">{row.touchpoint}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </TopicSection>
        ) : null}

        {content.specs && content.specs.length > 0 ? (
          <TopicSection eyebrow="Specs" title="Technical details">
            <dl className="grid gap-px bg-white/10 md:grid-cols-2">
              {content.specs.map((spec) => (
                <div key={spec.label} className="flex justify-between bg-black p-4 font-mono text-xs">
                  <dt className="text-white/50 uppercase">{spec.label}</dt>
                  <dd className="text-white/90">{spec.value}</dd>
                </div>
              ))}
            </dl>
          </TopicSection>
        ) : null}

        {siblings.length > 0 ? (
          <TopicSection eyebrow="Explore" title={`More in ${categoryLabel}`}>
            <div className="grid gap-4 md:grid-cols-2">
              {siblings.map((s, i) => (
                <ContentCard
                  key={s.slug}
                  variant="split"
                  tone="dark"
                  tag={categoryLabel}
                  title={s.label}
                  description={s.description ?? s.content.summary}
                  href={s.href}
                  image={topicImageForFlow(content.flow, i)}
                />
              ))}
            </div>
          </TopicSection>
        ) : null}

        <div className="mt-16 flex flex-col gap-3 border-t border-white/10 pt-12 sm:flex-row">
          {content.relatedProjectSlug ? (
            <ChamferButton href={`/projects/${content.relatedProjectSlug}`} variant="fill">
              View module
            </ChamferButton>
          ) : null}
          <ChamferButton href="/join" variant="ghost">
            Request demo
          </ChamferButton>
        </div>
      </div>

      <Footer />
    </main>
  );
}
