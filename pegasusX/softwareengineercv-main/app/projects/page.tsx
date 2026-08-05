'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import Link from 'next/link';
import ContentCard, { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { projects, getAllCategories } from '@/app/data/projects';
import { bentoPlacement, bentoVariant } from '@/app/lib/bento';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import ImpactMetricCard from '@/app/components/fleek/cards/ImpactMetricCard';
import { O9TourCTA } from '@/app/components/page-sections/o9/O9PageChrome';

export default function AllProjectsPage() {
  const gridRef = useRef<HTMLDivElement>(null);
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  const categories = ['All', ...getAllCategories()];
  const filteredProjects =
    selectedCategory === 'All'
      ? projects
      : projects.filter((p) => p.category === selectedCategory);

  const featured = filteredProjects[0];

  useEffect(() => {
    if (!gridRef.current) return;
    gsap.fromTo(
      gridRef.current.children,
      { opacity: 0, y: 24 },
      { opacity: 1, y: 0, duration: 0.5, stagger: 0.06, ease: 'power3.out' }
    );
  }, [selectedCategory]);

  return (
    <FleekSecondaryLayout
      activeHref="/projects"
      sectionTitle="MODULES"
      title="All Modules"
      summary={`Explore ${filteredProjects.length} modules powering supplier-led logistics — dispatch, payments, fleet, and role apps on one Spanner spine.`}
      secondaryHref="/platform"
      secondaryLabel="EXPLORE PLATFORM"
      hubId="capabilities"
      dataExtra={
        <ImpactMetricCard
          metric={{
            client: 'NOVA',
            title: 'Module coverage',
            description: 'Performance score across dispatch, fleet, and treasury modules.',
            value: 72,
            unit: '%',
          }}
        />
      }
      section06={
        <>
          <Link
            href="/join"
            className="fleek-btn fleek-btn--accent fixed bottom-8 right-8 z-50"
          >
            Request Demo →
          </Link>

          <div className="flex flex-wrap gap-2">
            {categories.map((category) => (
              <button
                key={category}
                type="button"
                onClick={() => setSelectedCategory(category)}
                className={`fleek-btn ${selectedCategory === category ? 'fleek-btn--accent' : ''}`}
              >
                {category}
              </button>
            ))}
          </div>

          {featured && selectedCategory === 'All' ? (
            <div className="mt-12">
              <ContentCard
                variant="featured"
                tone="light"
                tag={featured.category}
                title={featured.title}
                description={featured.description}
                image={featured.image || EDITORIAL_IMAGES[0]}
                href={`/projects/${featured.slug}`}
                ctaLabel="VIEW MODULE"
                ctaStyle="button"
              />
            </div>
          ) : null}

          <section className="docs-section mt-16">
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              Modules covered on the Pegasus platform
            </h2>
            <div ref={gridRef} className="editorial-bento mt-10 max-w-7xl">
              {filteredProjects
                .filter((p) => !(selectedCategory === 'All' && p.id === featured?.id))
                .map((project, index) => (
                  <ContentCard
                    key={project.id}
                    variant={bentoVariant(index)}
                    tone={index % 7 === 0 ? 'light' : 'dark'}
                    tag={project.category}
                    title={project.title}
                    description={project.description}
                    image={project.image || EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length]}
                    href={`/projects/${project.slug}`}
                    ctaLabel="READ MORE"
                    ctaStyle="link"
                    className={bentoPlacement(index)}
                  />
                ))}
            </div>
          </section>
          <O9TourCTA />
        </>
      }
    />
  );
}
