'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { projects } from '@/app/data/projects';
import PillNav from '@/app/components/PillNav';
import Footer from '@/app/components/Footer';
import type { Project } from '@/app/data/projects';

gsap.registerPlugin(ScrollTrigger);

export default function ProjectDetailClient({ project }: { project: Project }) {
  const heroRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const featuresRef = useRef<HTMLDivElement>(null);
  const techRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!heroRef.current) return;

    const timeline = gsap.timeline({ defaults: { ease: 'power3.out' } });

    timeline
      .fromTo(heroRef.current,
        { opacity: 0, y: 50 },
        { opacity: 1, y: 0, duration: 1 }
      )
      .fromTo(contentRef.current,
        { opacity: 0, y: 30 },
        { opacity: 1, y: 0, duration: 0.8 },
        '-=0.5'
      );

    if (featuresRef.current) {
      gsap.fromTo(featuresRef.current.children,
        { opacity: 0, y: 30 },
        {
          opacity: 1,
          y: 0,
          duration: 0.6,
          stagger: 0.1,
          scrollTrigger: {
            trigger: featuresRef.current,
            start: 'top 80%',
          }
        }
      );
    }

    if (techRef.current) {
      gsap.fromTo(techRef.current.children,
        { opacity: 0, scale: 0.8 },
        {
          opacity: 1,
          scale: 1,
          duration: 0.5,
          stagger: 0.05,
          scrollTrigger: {
            trigger: techRef.current,
            start: 'top 80%',
          }
        }
      );
    }
  }, []);

  return (
    <main className="min-h-screen bg-black text-white">
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

      {/* Hero Section */}
      <section className="min-h-screen relative flex items-center justify-center overflow-hidden pt-20">
        <div className="container mx-auto px-4 py-12 md:py-20 relative z-10">
          <div ref={heroRef} className="max-w-5xl mx-auto text-center">
            {/* Back Button */}
            <Link 
              href="/projects"
              className="inline-flex items-center gap-2 mb-6 md:mb-8 px-4 md:px-6 py-2 md:py-3 border-2 border-white hover:bg-white hover:text-black transition-all duration-300 rounded-2xl text-sm md:text-base"
            >
              ← Back to Projects
            </Link>

            {/* Status Badge */}
            <div className="mb-4 md:mb-6">
              <span className={`inline-block px-4 md:px-6 py-2 border-2 border-white rounded-2xl font-bold text-xs md:text-sm ${
                project.status === 'completed' ? 'bg-white text-black' :
                project.status === 'in-progress' ? 'bg-[#FBFF63] text-black border-[#FBFF63]' :
                'bg-black text-white opacity-70'
              }`}>
                {project.status === 'completed' ? '✓ COMPLETED' :
                 project.status === 'in-progress' ? '⚡ IN PROGRESS' :
                 '📦 ARCHIVED'}
              </span>
            </div>

            {/* Title */}
            <h1 className="text-3xl md:text-5xl lg:text-7xl font-bold mb-4 md:mb-6">
              {project.title}
            </h1>

            <div className="w-20 md:w-24 h-1 bg-white rounded-full mx-auto mb-6 md:mb-8" />

            {/* Description */}
            <p className="text-base md:text-xl lg:text-2xl text-gray-300 mb-6 md:mb-8 max-w-3xl mx-auto">
              {project.description}
            </p>

            {/* Category & Date */}
            <div className="flex flex-col sm:flex-row items-center justify-center gap-3 sm:gap-6 mb-8 md:mb-12">
              <span 
                className="px-4 py-2 text-black font-bold rounded-2xl"
                style={{ backgroundColor: project.color }}
              >
                {project.category}
              </span>
              <span className="hidden sm:block text-gray-400">•</span>
              <span className="text-gray-400 text-sm md:text-base">{project.date}</span>
            </div>

            {/* Action Buttons */}
            <div className="flex flex-col sm:flex-row gap-3 md:gap-4 justify-center">
              <a
                href={project.github}
                target="_blank"
                rel="noopener noreferrer"
                className="px-6 md:px-8 py-3 md:py-4 bg-white text-black border-2 border-white hover:bg-[#A9EBF9] hover:border-[#A9EBF9] transition-all duration-300 font-bold rounded-2xl text-sm md:text-base"
              >
                View on GitHub →
              </a>
              {project.liveUrl && (
                <a
                  href={project.liveUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="px-6 md:px-8 py-3 md:py-4 border-2 border-white hover:bg-[#8DDC96] hover:text-black hover:border-[#8DDC96] transition-all duration-300 font-bold rounded-2xl text-sm md:text-base"
                >
                  Live Demo →
                </a>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* Content Section */}
      <section className="py-12 md:py-20 bg-white text-black">
        <div className="container mx-auto px-4">
          <div ref={contentRef} className="max-w-5xl mx-auto">
            <div className="mb-12 md:mb-16">
              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">About This Project</h2>
              <div className="w-20 h-1 bg-black rounded-full mb-4 md:mb-6" />
              <p className="text-base md:text-lg lg:text-xl leading-relaxed text-gray-800">
                {project.longDescription}
              </p>
            </div>

            <div className="mb-12 md:mb-16">
              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-6 md:mb-8">Technologies Used</h2>
              <div className="w-20 h-1 bg-black rounded-full mb-6 md:mb-8" />
              <div ref={techRef} className="flex flex-wrap gap-2 md:gap-3">
                {project.technologies.map((tech, index) => (
                  <span
                    key={index}
                    className="px-4 md:px-6 py-2 md:py-3 bg-black text-white border-2 border-black hover:bg-white hover:text-black transition-all duration-300 font-bold rounded-2xl text-sm md:text-base"
                  >
                    {tech}
                  </span>
                ))}
              </div>
            </div>

            <div className="mb-12 md:mb-16">
              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-6 md:mb-8">Tags</h2>
              <div className="w-20 h-1 bg-black rounded-full mb-6 md:mb-8" />
              <div className="flex flex-wrap gap-2 md:gap-3">
                {project.tags.map((tag, index) => (
                  <span
                    key={index}
                    className="px-3 md:px-4 py-2 border-2 border-black rounded-2xl text-xs md:text-sm font-medium hover:bg-black hover:text-white transition-all duration-300"
                  >
                    #{tag}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-12 md:py-20 bg-black text-white">
        <div className="container mx-auto px-4">
          <div className="max-w-5xl mx-auto">
            <div className="text-center mb-12 md:mb-16">
              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4">Key Features</h2>
              <div className="w-20 h-1 bg-white rounded-full mx-auto" />
            </div>

            <div ref={featuresRef} className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-6">
              {project.features.map((feature, index) => (
                <div
                  key={index}
                  className="p-4 md:p-6 border-2 border-white rounded-2xl hover:bg-white hover:text-black transition-all duration-300"
                >
                  <div className="flex items-start gap-3 md:gap-4">
                    <span className="text-xl md:text-2xl">✓</span>
                    <p className="text-sm md:text-lg">{feature}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* Challenges & Learnings */}
      <section className="py-12 md:py-20 bg-white text-black">
        <div className="container mx-auto px-4">
          <div className="max-w-5xl mx-auto">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 md:gap-12">
              <div>
                <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">Challenges</h2>
                <div className="w-20 h-1 bg-black rounded-full mb-6 md:mb-8" />
                <ul className="space-y-3 md:space-y-4">
                  {project.challenges.map((challenge, index) => (
                    <li key={index} className="flex items-start gap-3">
                      <span className="text-lg md:text-xl mt-1">⚠️</span>
                      <p className="text-sm md:text-base lg:text-lg">{challenge}</p>
                    </li>
                  ))}
                </ul>
              </div>

              <div>
                <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">Key Learnings</h2>
                <div className="w-20 h-1 bg-black rounded-full mb-6 md:mb-8" />
                <ul className="space-y-3 md:space-y-4">
                  {project.learnings.map((learning, index) => (
                    <li key={index} className="flex items-start gap-3">
                      <span className="text-lg md:text-xl mt-1">💡</span>
                      <p className="text-sm md:text-base lg:text-lg">{learning}</p>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Related Projects */}
      <section className="py-12 md:py-20 bg-black text-white">
        <div className="container mx-auto px-4">
          <div className="max-w-6xl mx-auto">
            <div className="text-center mb-12 md:mb-16">
              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4">More Projects</h2>
              <div className="w-20 h-1 bg-white rounded-full mx-auto mb-4 md:mb-6" />
              <p className="text-sm md:text-lg text-gray-300">
                Check out other projects in the {project.category} category
              </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 md:gap-8 mb-8 md:mb-12">
              {projects
                .filter(p => p.category === project.category && p.id !== project.id)
                .slice(0, 3)
                .map((relatedProject) => (
                  <Link
                    key={relatedProject.id}
                    href={`/projects/${relatedProject.slug}`}
                    className="block group"
                  >
                    <div 
                      className="bg-black border-2 border-white rounded-3xl p-6 h-full min-h-[280px] md:min-h-[300px] flex flex-col justify-between transition-all duration-300 hover:scale-105"
                      style={{ backgroundColor: '#0D0D0D' }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.borderColor = relatedProject.color;
                        e.currentTarget.style.backgroundColor = relatedProject.color;
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.borderColor = '#FFFFFF';
                        e.currentTarget.style.backgroundColor = '#0D0D0D';
                      }}
                    >
                      <div>
                        <h3 className="text-xl md:text-2xl font-bold mb-2 md:mb-3 text-white group-hover:text-black transition-colors">
                          {relatedProject.title}
                        </h3>
                        <p className="text-xs md:text-sm text-white/90 group-hover:text-black/90 mb-3 md:mb-4 line-clamp-2 transition-colors">
                          {relatedProject.description}
                        </p>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        {relatedProject.technologies.slice(0, 3).map((tech, idx) => (
                          <span
                            key={idx}
                            className="px-2 py-1 text-xs bg-white text-black border border-white rounded-lg group-hover:bg-black group-hover:text-white transition-colors"
                          >
                            {tech}
                          </span>
                        ))}
                      </div>
                    </div>
                  </Link>
                ))}
            </div>

            <div className="text-center">
              <Link
                href="/projects"
                className="inline-block px-6 md:px-8 py-3 md:py-4 bg-white text-black border-2 border-white hover:bg-[#FFA500] hover:border-[#FFA500] transition-all duration-300 font-bold rounded-2xl text-sm md:text-base"
              >
                View All Projects
              </Link>
            </div>
          </div>
        </div>
      </section>

      <Footer />
    </main>
  );
}
