'use client';

import { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { RoleData, getRolesData } from '@/app/data/rolesData';
import { useLanguage } from '@/app/context/LanguageContext';

gsap.registerPlugin(ScrollTrigger);

const PLATFORM_ICONS = {
  web: (
    <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
    </svg>
  ),
  mobile: (
    <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
    </svg>
  ),
  desktop: (
    <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9delivery zonesm9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
    </svg>
  )
};

export default function RoleDetailClient({ role: roleProp }: { role: RoleData }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const { t, language } = useLanguage();
  const role =
    getRolesData(language).find((r) => r.id === roleProp.id) ?? roleProp;

  const platformLabel = (platform: string) => {
    if (language !== 'ru') return `${platform} App`;
    const map: Record<string, string> = {
      web: 'Веб-приложение',
      mobile: 'Мобильное приложение',
      desktop: 'Десктопное приложение',
    };
    return map[platform] ?? platform;
  };

  useEffect(() => {
    if (!containerRef.current) return;

    const ctx = gsap.context(() => {
      // Platform cards animation
      gsap.fromTo('.platform-card', 
        { y: 30, opacity: 0 },
        { 
          y: 0, 
          opacity: 1, 
          stagger: 0.1, 
          duration: 0.8, 
          ease: 'power3.out',
          scrollTrigger: {
            trigger: '.platform-section',
            start: 'top 80%',
          }
        }
      );

      // Flow sections animation
      gsap.utils.toArray('.flow-section').forEach((section: any) => {
        gsap.fromTo(section,
          { y: 50, opacity: 0 },
          {
            y: 0,
            opacity: 1,
            duration: 0.8,
            ease: 'power3.out',
            scrollTrigger: {
              trigger: section,
              start: 'top 85%',
            }
          }
        );
      });
    }, containerRef);

    return () => ctx.revert();
  }, [role.id]);

  return (
    <div ref={containerRef} className="space-y-24">
      
      {/* App Presentation / Platforms */}
      <div className="platform-section border-t border-[var(--border)] pt-16">
        <h2 className="text-3xl font-semibold mb-8 text-[var(--text)]">{t('role_available_platforms')}</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {role.platforms.map((platform) => (
            <div key={platform} className="platform-card bg-[var(--surface)] border border-[var(--border)] rounded-[24px] overflow-hidden">
              <div className="p-6 border-b border-[var(--border)] flex items-center text-[var(--text)] font-medium capitalize">
                {PLATFORM_ICONS[platform]}
                {platformLabel(platform)}
              </div>
              <div className="aspect-[4/3] bg-[var(--bg)] flex items-center justify-center p-8">
                {/* PLACEHOLDER FOR IMAGES */}
                <div className="w-full h-full border-2 border-dashed border-[var(--border)] rounded-xl flex items-center justify-center text-[var(--text-secondary)] text-sm font-mono text-center px-4">
                  {language === 'ru' ? (
                    <>
                      [ ИЗОБРАЖЕНИЕ {platform.toUpperCase()} ]<br />
                      Цель: {role.name} / {platform}
                    </>
                  ) : (
                    <>
                      [ {platform.toUpperCase()} PRESENTATION IMAGE ]<br />
                      Target: {role.name} / {platform}
                    </>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Subtopics / Flow Breakdown */}
      <div className="space-y-16">
        <h2 className="text-3xl font-semibold mb-8 text-[var(--text)] border-b border-[var(--border)] pb-4">{t('role_capabilities_title')}</h2>
        
        {role.subtopics.map((topic, index) => (
          <div 
            key={topic.id} 
            id={topic.id}
            className="flow-section grid grid-cols-1 lg:grid-cols-2 gap-12 items-center"
          >
            {/* Text Content */}
            <div className={`space-y-6 ${index % 2 !== 0 ? 'lg:order-2' : ''}`}>
              <div className="inline-block px-3 py-1 bg-[var(--surface)] border border-[var(--border)] rounded-full text-xs font-mono text-[var(--text-secondary)]">
                {String(index + 1).padStart(2, '0')} // {language === 'ru' ? 'ВОЗМОЖНОСТЬ' : 'CAPABILITY'}
              </div>
              <h3 className="text-3xl font-bold text-[var(--text)] leading-tight">
                {topic.title}
              </h3>
              <p className="text-lg text-[var(--text-secondary)]">
                {topic.description}
              </p>
              
              <div className="space-y-4 pt-4">
                <div className="bg-[var(--surface)] p-6 rounded-[20px] border border-[var(--border)]">
                  <h4 className="text-sm font-semibold uppercase tracking-wider text-[var(--text)] mb-2">{t('role_business_logic')}</h4>
                  <p className="text-[var(--text-secondary)] leading-relaxed">{topic.businessLogic}</p>
                </div>
                
                <div className="bg-[var(--surface)] p-6 rounded-[20px] border border-[var(--border)]">
                  <h4 className="text-sm font-semibold uppercase tracking-wider text-[var(--text)] mb-2">{t('role_edge_cases')}</h4>
                  <p className="text-[var(--text-secondary)] leading-relaxed">{topic.edgeCases}</p>
                </div>
              </div>
            </div>

            {/* Visualization */}
            <div className={`aspect-square sm:aspect-[4/3] lg:aspect-square bg-[var(--surface)] rounded-[32px] border border-[var(--border)] flex items-center justify-center overflow-hidden relative ${index % 2 !== 0 ? 'lg:order-1' : ''}`}>
              {/* PLACEHOLDER FOR FEATURE IMAGE */}
              <div className="absolute inset-8 border-2 border-dashed border-[var(--border)] rounded-[20px] flex flex-col items-center justify-center text-[var(--text-secondary)] text-sm font-mono text-center p-6 bg-[var(--bg)]/50 backdrop-blur-sm">
                <svg className="w-12 h-12 mb-4 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                [ {language === 'ru' ? 'НУЖНА ВИЗУАЛИЗАЦИЯ' : 'VISUALIZATION IMAGE NEEDED'} ]<br/>
                <span className="mt-2 text-xs opacity-75">{topic.title}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

    </div>
  );
}
