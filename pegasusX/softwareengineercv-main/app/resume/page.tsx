'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { getProjects } from '../data/projects_ru';
import SiteNav from '../components/explore/SiteNav';
import Link from 'next/link';
import { useLanguage } from '../context/LanguageContext';

export default function ResumePage() {
  const { t, language } = useLanguage();
  const resumeProjects = getProjects(language).slice(0, 4);
  const resumeRef = useRef<HTMLDivElement>(null);
  const headerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Set document title
    document.title = `${t('resume_title', 'Platform Overview')} | Pegasus`;
    
    // Add print styles
    const style = document.createElement('style');
    style.textContent = `
      @media print {
        body {
          background: white !important;
          color: black !important;
        }
        .no-print {
          display: none !important;
        }
        .print-section {
          page-break-inside: avoid;
          break-inside: avoid;
        }
        .resume-container {
          max-width: 100% !important;
          padding: 40px !important;
          box-shadow: none !important;
        }
        a {
          color: black !important;
          text-decoration: none !important;
        }
        .border-white {
          border-color: black !important;
        }
        .text-white {
          color: black !important;
        }
        .bg-black {
          background: white !important;
        }
      }
    `;
    document.head.appendChild(style);
    
    if (headerRef.current) {
      gsap.fromTo(
        headerRef.current,
        { opacity: 0, y: -30 },
        { opacity: 1, y: 0, duration: 0.8, ease: 'power3.out' }
      );
    }
    
    return () => {
      document.head.removeChild(style);
    };
  }, [t]);

  const handleDownloadPDF = () => {
    if (typeof window !== 'undefined') {
      window.print();
    }
  };

  return (
    <>
      <div className="pegasus-docs min-h-screen bg-black text-white no-print">
        <SiteNav activeHref="/resume" />

        {/* Action Buttons */}
        <div className="fixed top-24 right-8 z-50 flex flex-col gap-4 no-print">
          <button type="button" onClick={handleDownloadPDF} className="editorial-btn editorial-btn--inverted editorial-btn--shadow">
            {t('resume_download', '📄 Download PDF')}
          </button>
          <Link href="/" className="editorial-btn editorial-btn--shadow text-center">
            {t('contact_back_home', '← Back Home')}
          </Link>
        </div>

        {/* Header */}
        <div ref={headerRef} className="border-b border-white/10 px-4 pb-12 pt-28 text-center no-print md:pt-32">
          <p className="editorial-eyebrow">{t('resume_title', 'Platform overview')}</p>
          <h1 className="docs-hero-title mt-4 text-4xl font-semibold tracking-tight md:text-5xl">Pegasus at a glance</h1>
          <p className="docs-body mx-auto mt-4 max-w-xl text-white/60">
            {t('resume_sub', 'Six roles · shared system of record · live sync after every change · Portal · Mobile · Desktop')}
          </p>
          <div className="mx-auto mt-8 grid max-w-2xl grid-cols-2 gap-2 font-mono text-[10px] uppercase tracking-wider text-white/45 sm:grid-cols-4">
            {[
              [t('nav_roles', 'Roles'), '6 connected'],
              [t('sec_source_truth', 'Source of truth'), 'Shared order record'],
              ['Realtime', 'Outbox → WS'],
              [t('sec_surfaces', 'Surfaces'), 'All channels'],
            ].map(([l, v]) => (
              <div key={l} className="border border-white/15 px-2 py-3 text-left sm:text-center">
                <p className="text-white/30">{l}</p>
                <p className="mt-1 text-white/80">{v}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Resume Content */}
      <div className="min-h-screen bg-white text-black py-12 px-4">
        <div
          ref={resumeRef}
          className="resume-container max-w-4xl mx-auto bg-white p-8 md:p-16 shadow-2xl rounded-2xl"
        >
          {/* Header Section */}
          <div className="text-center mb-12 print-section border-b-2 border-black pb-8">
            <h1 className="text-5xl md:text-7xl font-light mb-3 tracking-tight">SHAKZHOD SOLIYEV</h1>
            <p className="text-xl md:text-2xl text-gray-700 mb-6 font-semibold">LOGISTICS OPERATING SYSTEM</p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm max-w-3xl mx-auto">
              <div className="flex items-center justify-center gap-2">
                <span className="font-light">📧 Email:</span>
                <a href="mailto:demo@pegasus.io" className="hover:text-[#FFA500] transition-colors">
                  demo@pegasus.io
                </a>
              </div>
              <div className="flex items-center justify-center gap-2">
                <span className="font-light">📱 Sales:</span>
                <a href="mailto:sales@pegasus.io" className="hover:text-[#FFA500] transition-colors">
                  sales@pegasus.io
                </a>
              </div>
              <div className="flex items-center justify-center gap-2">
                <span className="font-light">🌐 Platform:</span>
                <a href="https://pegasus.io" target="_blank" rel="noopener noreferrer" className="hover:text-[#FFA500] transition-colors truncate max-w-[250px]">
                  pegasus.io
                </a>
              </div>
              <div className="flex items-center justify-center gap-2">
                <span className="font-light">📍 Focus:</span>
                <span>Supplier-led logistics networks</span>
              </div>
            </div>
          </div>

          {/* About Me Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-light mb-4 pb-2 border-b-2 border-black">
              ABOUT PEGASUS
            </h2>
            <p className="text-base leading-relaxed text-gray-800">
              Pegasus is the logistics operating system for supplier-led networks. From morning dispatch
              to live fleet tracking and payment reconciliation, every team works from the same source of truth.
            </p>
          </div>

          {/* Education Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-light mb-6 pb-2 border-b-2 border-black">
              SIX NETWORK ROLES
            </h2>
            <div className="border-l-4 border-black pl-6">
              <div className="mb-2">
                <h3 className="text-xl font-light">Connected Role Row</h3>
                <p className="text-lg text-gray-700 font-semibold">Supplier · Warehouse · Factory · Driver · Retailer · Payload</p>
              </div>
              <p className="text-base text-gray-800 leading-relaxed">
                Each role has dedicated portal and mobile apps with shared contracts across the network.
              </p>
            </div>
          </div>

          {/* Skills Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-light mb-6 pb-2 border-b-2 border-black">
              PLATFORM CAPABILITIES
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <h3 className="text-lg font-light mb-3 text-gray-800">Operations</h3>
                <div className="flex flex-wrap gap-2">
                  {['Dispatch Engine', 'Fleet Telemetry', 'Gate Seal', 'Pre-Orders'].map((skill, idx) => (
                    <span key={idx} className="editorial-btn editorial-btn--sm editorial-btn--on-light cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-lg font-light mb-3 text-gray-800">Finance</h3>
                <div className="flex flex-wrap gap-2">
                  {['Payment Integrity', 'Cash on Delivery', 'Treasury', 'Reconciliation'].map((skill, idx) => (
                    <span key={idx} className="editorial-btn editorial-btn--sm editorial-btn--on-light cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-lg font-light mb-3 text-gray-800">Network</h3>
                <div className="flex flex-wrap gap-2">
                  {['Topology', 'Service Zones', 'Multi-Site', 'Role Parity'].map((skill, idx) => (
                    <span key={idx} className="editorial-btn editorial-btn--sm editorial-btn--on-light cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-lg font-light mb-3 text-gray-800">Mobile</h3>
                <div className="flex flex-wrap gap-2">
                  {['Driver App', 'Warehouse Android', 'Gate Terminal'].map((skill, idx) => (
                    <span key={idx} className="editorial-btn editorial-btn--sm editorial-btn--on-light cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-lg font-light mb-3 text-gray-800">Realtime</h3>
                <div className="flex flex-wrap gap-2">
                  {['Live Sync', 'live updates', 'Event Contracts', 'Cache Invalidation'].map((skill, idx) => (
                    <span key={idx} className="editorial-btn editorial-btn--sm editorial-btn--on-light cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>

          {/* Deployment Tiers Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-light mb-6 pb-2 border-b-2 border-black">
              DEPLOYMENT TIERS
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {[
                { lang: 'Starter', level: 'Single-site dispatch & tracking' },
                { lang: 'Professional', level: 'Multi-site + payments' },
                { lang: 'Enterprise', level: 'Full network + SLA' },
              ].map((item, idx) => (
                <div key={idx} className="text-center p-4 border-2 border-black rounded-xl hover:bg-black hover:text-white transition-all duration-300">
                  <p className="font-light text-lg">{item.lang}</p>
                  <p className="text-sm opacity-80">{item.level}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Solutions Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-light mb-6 pb-2 border-b-2 border-black">
              KEY OUTCOMES
            </h2>
            <div className="border-l-4 border-black pl-6 space-y-8">
              <div>
                <div className="flex justify-between items-start mb-3">
                  <div>
                    <h3 className="text-xl font-light">Dispatch Accuracy</h3>
                    <p className="text-gray-700 font-semibold">Visual boards with smart assist at peak hours</p>
                  </div>
                </div>
                <ul className="list-disc list-inside space-y-2 text-gray-800">
                  <li>Truck-and-order matching with capacity-aware load planning</li>
                  <li>Gate seal workflow before driver departure</li>
                  <li>Overflow handling across trucks without overselling</li>
                </ul>
              </div>
              <div>
                <div className="flex justify-between items-start mb-3">
                  <div>
                    <h3 className="text-xl font-light">Payment Confidence</h3>
                    <p className="text-gray-700 font-semibold">Checkout through collection to treasury</p>
                  </div>
                </div>
                <ul className="list-disc list-inside space-y-2 text-gray-800">
                  <li>Card and cash-on-delivery paths with duplicate protection</li>
                  <li>Driver cash collection tied to delivery proof</li>
                  <li>Supplier treasury dashboards operators actually use</li>
                </ul>
              </div>
            </div>
          </div>

          {/* Modules Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-light mb-6 pb-2 border-b-2 border-black">
              PLATFORM MODULES
            </h2>
            <div className="space-y-6">
              {resumeProjects.map((project) => (
                <div key={project.id} className="border-l-4 border-black pl-6 hover:border-[#FFA500] transition-colors">
                  <div className="flex justify-between items-start mb-2">
                    <h3 className="text-lg font-light">{project.title}</h3>
                    <span className="text-sm text-gray-600 whitespace-nowrap ml-4">{project.date}</span>
                  </div>
                  <p className="text-sm text-gray-700 mb-3 leading-relaxed">{project.description}</p>
                  <div className="flex flex-wrap gap-2 mb-3">
                    {project.technologies.slice(0, 5).map((tech, idx) => (
                      <span key={idx} className="editorial-btn editorial-btn--sm editorial-btn--on-light editorial-btn--inverted">
                        {tech}
                      </span>
                    ))}
                  </div>
                  <div className="space-y-1 text-xs text-gray-700">
                    {project.features.slice(0, 3).map((feature, idx) => (
                      <p key={idx} className="flex items-start">
                        <span className="mr-2">▸</span>
                        <span>{feature}</span>
                      </p>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Contact Information */}
          <div className="mt-12 pt-8 border-t-2 border-black text-center print-section">
            <h3 className="text-2xl font-light mb-4">REQUEST A DEMO</h3>
            <div className="flex flex-wrap justify-center gap-6 text-sm">
              <a href="mailto:demo@pegasus.io" className="hover:text-[#FFA500] transition-colors font-semibold">
                📧 demo@pegasus.io
              </a>
              <a href="mailto:sales@pegasus.io" className="hover:text-[#FFA500] transition-colors font-semibold">
                📧 sales@pegasus.io
              </a>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
