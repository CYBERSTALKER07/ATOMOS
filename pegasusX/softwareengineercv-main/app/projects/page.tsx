'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import { projects, getAllCategories } from '../data/projects';
import PillNav from '../components/PillNav';
import Footer from '../components/Footer';
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
        className="fixed bottom-8 right-8 z-50 px-6 py-3 md:px-8 md:py-4 bg-white text-black border-2 border-white hover:bg-[#FFA500] hover:border-[#FFA500] transition-all duration-300 font-bold text-base md:text-lg shadow-lg rounded-3xl focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-black outline-none"
      >
        Request Demo →
      </Link>

      <div className="container mx-auto px-4 py-20 md:py-32">
        {/* Header */}
        <div ref={headerRef} className="text-center mb-12 md:mb-16">
          <h1 className="text-4xl md:text-6xl lg:text-7xl font-bold mb-4 md:mb-6 text-white">
            All Projects
          </h1>
          <div className="w-20 h-1 bg-white rounded-full mx-auto mb-6 md:mb-8" />
          <p className="text-base md:text-xl text-gray-300 max-w-3xl mx-auto px-4">
            Explore {filteredProjects.length} {selectedCategory !== 'All' ? selectedCategory : ''} projects showcasing innovative solutions and creative development
          </p>
        </div>

        {/* Category Filter */}
        <div className="flex flex-wrap justify-center gap-3 md:gap-4 mb-12 md:mb-16">
          {categories.map((category) => (
            <button
              key={category}
              onClick={() => setSelectedCategory(category)}
              className={`px-4 md:px-6 py-2 md:py-3 border-2 border-white rounded-2xl font-bold text-sm md:text-base transition-all duration-300 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-white outline-none ${
                selectedCategory === category
                  ? 'bg-white text-black'
                  : 'bg-black text-white hover:bg-[#8DDC96] hover:text-black hover:border-[#8DDC96]'
              }`}
            >
              {category}
            </button>
          ))}
        </div>

        {/* Projects Grid */}
        <div 
          ref={gridRef}
          className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6 md:gap-8"
        >
          {filteredProjects.map((project) => (
            <Link
              key={project.id}
              href={`/projects/${project.slug}`}
              className="group block"
            >
              <div 
                className="bg-black border-2 border-white rounded-3xl p-6 md:p-8 h-full min-h-[320px] md:min-h-[380px] flex flex-col justify-between transition-all duration-300 hover:scale-105 hover:shadow-2xl"
                style={{
                  backgroundColor: '#0D0D0D',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = project.color;
                  e.currentTarget.style.backgroundColor = project.color;
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = '#FFFFFF';
                  e.currentTarget.style.backgroundColor = '#0D0D0D';
                }}
              >
                <div>
                  {/* Status Badge */}
                  <div className="mb-4">
                    <span className={`inline-block px-3 py-1 text-xs font-bold rounded-xl border-2 ${
                      project.status === 'completed' 
                        ? 'bg-white text-black border-white' 
                        : project.status === 'in-progress'
                        ? 'bg-[#FBFF63] text-black border-[#FBFF63]'
                        : 'bg-black text-white border-white'
                    }`}>
                      {project.status === 'completed' ? '✓ COMPLETED' :
                       project.status === 'in-progress' ? '⚡ IN PROGRESS' :
                       '📦 ARCHIVED'}
                    </span>
                  </div>

                  {/* Title */}
                  <h3 className="text-xl md:text-2xl font-bold mb-3 text-white group-hover:text-black transition-colors">
                    {project.title}
                  </h3>

                  {/* Description */}
                  <p className="text-sm md:text-base text-gray-300 mb-4 line-clamp-3 group-hover:text-black transition-colors">
                    {project.description}
                  </p>

                  {/* Category & Date */}
                  <div className="flex items-center gap-3 mb-4 text-xs md:text-sm">
                    <span className="px-3 py-1 bg-white text-black rounded-lg font-semibold group-hover:bg-black group-hover:text-white transition-colors">
                      {project.category}
                    </span>
                    <span className="text-gray-400 group-hover:text-black transition-colors">{project.date}</span>
                  </div>
                </div>

                {/* Technologies */}
                <div className="flex flex-wrap gap-2 mt-4">
                  {project.technologies.slice(0, 3).map((tech, idx) => (
                    <span
                      key={idx}
                      className="px-2 py-1 text-xs border border-white rounded-lg text-white group-hover:border-black group-hover:text-black transition-colors"
                    >
                      {tech}
                    </span>
                  ))}
                  {project.technologies.length > 3 && (
                    <span className="px-2 py-1 text-xs text-gray-400 group-hover:text-black transition-colors">
                      +{project.technologies.length - 3} more
                    </span>
                  )}
                </div>

                {/* View Details Arrow */}
                <div className="mt-6 flex items-center gap-2 text-white group-hover:text-black transition-colors">
                  <span className="font-bold text-sm">View Details</span>
                  <span className="transform group-hover:translate-x-2 transition-transform">→</span>
                </div>
              </div>
            </Link>
          ))}
        </div>

        {/* Empty State */}
        {filteredProjects.length === 0 && (
          <div className="text-center py-20">
            <p className="text-xl text-gray-400">No projects found in this category.</p>
          </div>
        )}
      </div>

      <Footer />
    </div>
  );
}
