'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { gsap } from 'gsap';
import Link from 'next/link';
import ContentCard, { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { getAllCategories } from '@/app/data/projects';
import {
  getAllCategoriesLocalized,
  getEnCategoryForSlug,
  getProjects,
} from '@/app/data/projects_ru';
import { bentoPlacement, bentoVariant } from '@/app/lib/bento';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import ImpactMetricCard from '@/app/components/fleek/cards/ImpactMetricCard';
import { useLanguage } from '@/app/context/LanguageContext';

export default function AllProjectsPage() {
  const { t, language } = useLanguage();
  const gridRef = useRef<HTMLDivElement>(null);
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  const projects = useMemo(() => getProjects(language), [language]);
  const enCategories = useMemo(() => getAllCategories(), []);
  const localizedCategories = useMemo(
    () => getAllCategoriesLocalized(language),
    [language]
  );
  // Keep filter keyed to EN category so switching language doesn't break selection.
  const categories = useMemo(
    () => ['All', ...enCategories],
    [enCategories]
  );
  const categoryLabel = (enCat: string) => {
    if (enCat === 'All') return t('projects_filter_all', 'All');
    if (language !== 'ru') return enCat;
    const idx = enCategories.indexOf(enCat);
    return idx >= 0 ? localizedCategories[idx] ?? enCat : enCat;
  };
  const filteredProjects =
    selectedCategory === 'All'
      ? projects
      : projects.filter((p) => getEnCategoryForSlug(p.slug) === selectedCategory);

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
      sectionTitle={t('projects_section_title', 'MODULES')}
      title={t('projects_title', 'All Modules')}
      summary={`${t('projects_summary_prefix', 'Explore ')}${filteredProjects.length}${t('projects_summary_suffix', ' modules powering supplier-led logistics — dispatch, payments, fleet, and role apps on one shared order record.')}`}
      secondaryHref="/platform"
      secondaryLabel={t('btn_explore_platform', 'EXPLORE PLATFORM')}
      hubId="capabilities"
      dataExtra={
        <ImpactMetricCard
          metric={{
            client: 'NOVA',
            title: t('projects_coverage_title', 'Module coverage'),
            description: t('projects_coverage_desc', 'Performance score across dispatch, fleet, and treasury modules.'),
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
            {t('nav_demo', 'Request Demo →')}
          </Link>

          <div className="flex flex-wrap gap-2">
            {categories.map((category) => (
              <button
                key={category}
                type="button"
                onClick={() => setSelectedCategory(category)}
                className={`fleek-btn ${selectedCategory === category ? 'fleek-btn--accent' : ''}`}
              >
                {categoryLabel(category)}
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
                ctaLabel={t('nav_modules', 'VIEW MODULE')}
                ctaStyle="button"
              />
            </div>
          ) : null}

          <section className="docs-section mt-16">
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              {t('projects_heading', 'Modules covered on the Pegasus platform')}
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
                    ctaLabel={t('btn_read_more', 'READ MORE')}
                    ctaStyle="link"
                    className={bentoPlacement(index)}
                  />
                ))}
            </div>
          </section>
        </>
      }
    />
  );
}
