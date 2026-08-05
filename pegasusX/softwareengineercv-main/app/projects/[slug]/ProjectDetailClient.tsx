'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { projects } from '@/app/data/projects';
import ContentCard, { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { BENTO_THREE } from '@/app/lib/bento';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import ImpactMetricCard from '@/app/components/fleek/cards/ImpactMetricCard';
import type { Project } from '@/app/data/projects';

export default function ProjectDetailClient({ project }: { project: Project }) {
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!bodyRef.current) return;
    gsap.fromTo(bodyRef.current, { opacity: 0, y: 20 }, { opacity: 1, y: 0, duration: 0.7, ease: 'power3.out' });
  }, []);

  const related = projects
    .filter((p) => p.category === project.category && p.id !== project.id)
    .slice(0, 3);

  return (
    <FleekSecondaryLayout
      activeHref="/projects"
      sectionTitle={project.category.toUpperCase()}
      title={project.title}
      summary={project.description}
      primaryHref="/join"
      primaryLabel="REQUEST DEMO"
      secondaryHref={project.liveUrl || project.github}
      secondaryLabel={project.liveUrl ? 'LIVE DEMO' : 'GITHUB'}
      hubId="capabilities"
      stackFeatures={project.features.slice(0, 6)}
      relatedProjectSlug={project.slug}
      dataExtra={
        <ImpactMetricCard
          metric={{
            client: project.category,
            title: project.title,
            description: project.longDescription.slice(0, 120),
            value: 72,
            unit: '%',
          }}
        />
      }
      section06={
        <>
          <p className="mb-6">
            <Link href="/projects" className="fleek-btn">
              ← All modules
            </Link>
          </p>

          <div ref={bodyRef} className="space-y-8">
            <section className="docs-section">
              <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">why it matters</p>
              <h2 className="mt-3 text-3xl font-semibold tracking-tight">About this module</h2>
              <p className="mt-6 max-w-3xl leading-relaxed text-white/70">{project.longDescription}</p>
              <dl className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <div className="docs-card">
                  <dt className="font-mono text-[10px] uppercase tracking-widest text-white/40">Status</dt>
                  <dd className="mt-2 text-sm text-white/80">{project.status.replace('-', ' ')}</dd>
                </div>
                <div className="docs-card">
                  <dt className="font-mono text-[10px] uppercase tracking-widest text-white/40">Category</dt>
                  <dd className="mt-2 text-sm text-white/80">{project.category}</dd>
                </div>
                <div className="docs-card">
                  <dt className="font-mono text-[10px] uppercase tracking-widest text-white/40">Stack</dt>
                  <dd className="mt-2 text-sm text-white/80">
                    {project.technologies.slice(0, 4).join(' · ') || 'Pegasus'}
                  </dd>
                </div>
                <div className="docs-card">
                  <dt className="font-mono text-[10px] uppercase tracking-widest text-white/40">Date</dt>
                  <dd className="mt-2 text-sm text-white/80">{project.date}</dd>
                </div>
              </dl>
            </section>

            <section className="docs-section">
              <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">core capabilities</p>
              <h2 className="mt-3 text-3xl font-semibold tracking-tight">Key features</h2>
              <ul className="docs-cap-grid mt-8">
                {project.features.map((f) => (
                  <li key={f} className="docs-card text-sm text-white/80">{f}</li>
                ))}
              </ul>
            </section>

            <section className="docs-section">
              <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">edge cases</p>
              <div className="mt-8 grid gap-4 md:grid-cols-2">
                <div className="docs-card">
                  <h3 className="text-xl font-semibold">Challenges</h3>
                  <ul className="mt-4 space-y-2 text-sm text-white/65">
                    {project.challenges.map((c) => (
                      <li key={c}>· {c}</li>
                    ))}
                  </ul>
                </div>
                <div className="docs-card">
                  <h3 className="text-xl font-semibold">Learnings</h3>
                  <ul className="mt-4 space-y-2 text-sm text-white/65">
                    {project.learnings.map((l) => (
                      <li key={l}>· {l}</li>
                    ))}
                  </ul>
                </div>
              </div>
            </section>

            <div className="flex flex-wrap gap-3">
              <Link href="/join" className="fleek-btn fleek-btn--accent">Request demo</Link>
              <a
                href={project.github}
                target="_blank"
                rel="noopener noreferrer"
                className="fleek-btn"
              >
                GitHub →
              </a>
            </div>

            {project.tags.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {project.tags.slice(0, 8).map((tag) => (
                  <span
                    key={tag}
                    className="border border-white/15 px-2 py-1 font-mono text-[10px] text-white/60"
                  >
                    #{tag}
                  </span>
                ))}
              </div>
            ) : null}
          </div>

          {related.length > 0 ? (
            <section className="docs-section mt-12">
              <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">related modules</p>
              <h2 className="mt-3 text-3xl font-semibold tracking-tight">More in {project.category}</h2>
              <div className="editorial-bento mt-10">
                {related.map((r, index) => (
                  <ContentCard
                    key={r.id}
                    variant={index === 1 ? 'split' : 'vertical'}
                    tone={index === 1 ? 'light' : 'dark'}
                    tag={r.category}
                    title={r.title}
                    description={r.description}
                    image={r.image || EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length]}
                    href={`/projects/${r.slug}`}
                    ctaLabel="READ MORE"
                    className={BENTO_THREE[index]}
                  />
                ))}
              </div>
            </section>
          ) : null}
        </>
      }
    />
  );
}
