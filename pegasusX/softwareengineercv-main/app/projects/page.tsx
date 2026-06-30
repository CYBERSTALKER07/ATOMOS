'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import { projects, getAllCategories } from '../data/projects';
import PillNav from '../components/PillNav';
import Footer from '../components/Footer';
import ContentCard, { EDITORIAL_IMAGES } from '../components/ContentCard';
import { bentoPlacement, bentoVariant } from '../lib/bento';
import Link from 'next/link';

export default function AllProjectsPage() {
  const headerRef = useRef<HTMLDivElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  useEffect(() => {
    if (headerRef.current) {
      gsap.fromTo(
        headerRef.current,
        { opacity: 0, y: -50 },
        { opacity: 1, y: 0, duration: 1, ease: 'power3.out' }
      );
    }
  }, []);

  useEffect(() => {
    if (gridRef.current) {
      gsap.fromTo(
        gridRef.current.children,
        { opacity: 0, y: 30, scale: 0.9 },
        {
          opacity: 1,
          y: 0,
          scale: 1,
          duration: 0.6,
          stagger: 0.1,
          ease: 'power3.out'
        }
      );
    }
  }, [selectedCategory]);

  const categories = ['All', ...getAllCategories()];
  const filteredProjects = selectedCategory === 'All' 
    ? projects 
    : projects.filter(p => p.category === selectedCategory);

  return (
    <div className="min-h-screen bg-black text-white">
      <PillNav
        logo=""
        logoAlt="Pegasus Logo"
        items={[
          { label: 'Home', href: '/' },
          { label: 'About', href: '/#about' },
          { label: 'Skills', href: '/#skills' },
          { label: 'Projects', href: '/projects' },
          { label: 'Resume', href: '/resume' },
          { label: 'Contact', href: '/#contact' }
        ]}
        activeHref="/projects"
        baseColor="#000000"
        pillColor="#ffffff"
        hoveredPillTextColor="#ffffff"
        pillTextColor="#000000"
      />

      {/* Floating "Join Us" Button */}
      <Link
        href="/join"
        className="editorial-btn editorial-btn--shadow fixed bottom-8 right-8 z-50"
      >
        Request Demo →
      </Link>

      <div className="container mx-auto px-4 py-20 md:py-32">
        {/* Header */}
        <div ref={headerRef} className="text-center mb-12 md:mb-16">
          <h1 className="text-4xl md:text-6xl lg:text-7xl font-bold mb-4 md:mb-6 text-white">
            All Modules
          </h1>
          <div className="w-20 h-1 bg-white rounded-full mx-auto mb-6 md:mb-8" />
          <p className="text-base md:text-xl text-gray-300 max-w-3xl mx-auto px-4">
            Explore {filteredProjects.length} {selectedCategory !== 'All' ? selectedCategory : ''} modules powering supplier-led logistics on Pegasus
          </p>
        </div>

        {/* Category Filter */}
        <div className="flex flex-wrap justify-center gap-3 md:gap-4 mb-12 md:mb-16">
          {categories.map((category) => (
            <button
              key={category}
              onClick={() => setSelectedCategory(category)}
              className={`editorial-btn editorial-btn--sm ${
                selectedCategory === category ? 'editorial-btn--active' : ''
              }`}
            >
              {category}
            </button>
          ))}
        </div>

        {/* Projects Grid */}
        <div ref={gridRef} className="editorial-bento max-w-7xl mx-auto">
          {filteredProjects.map((project, index) => (
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

        {/* Empty State */}
        {filteredProjects.length === 0 && (
          <div className="text-center py-20">
            <p className="text-xl text-gray-400">No modules found in this category.</p>
          </div>
        )}
      </div>

      <Footer />
    </div>
  );
}
