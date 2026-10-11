'use client';

import Link from 'next/link';
import { useLanguage } from '../context/LanguageContext';
import ParticleText from './ParticleText';
import DitheredFaceCanvas from './visuals/DitheredFaceCanvas';

export default function EarlyAccessSection() {
  const { t } = useLanguage();

  return (
    <section
      className="relative w-full min-h-[580px] md:min-h-[660px] lg:min-h-[740px] bg-[#000000] text-white overflow-hidden flex flex-col justify-between py-10 sm:py-14 px-6 sm:px-12 lg:px-20 border-t border-white/10 select-none keep-dark"
      data-keep-dark
      data-keep-white
      aria-label="Request Early Access"
    >
      {/* Background Interactive Dithered Face & Hand Mesh (Bayer 8x8 matrix shader matching About) */}
      <div className="absolute inset-0 pointer-events-none select-none overflow-hidden">
        <DitheredFaceCanvas />
      </div>

      {/* Top Spacer */}
      <div className="relative z-10 w-full" />

      {/* Center-Right Main Content Block */}
      <div className="relative z-10 w-full flex justify-end items-center my-auto py-10 md:py-14">
        <div className="w-full max-w-xl md:max-w-2xl lg:max-w-3xl lg:pl-16 text-left">
          {/* Eyebrow with leading dash line */}
          <div className="flex items-center gap-3 mb-4 sm:mb-6">
            <span className="w-8 h-[1px] bg-white/40 inline-block shrink-0" />
            <span className="text-[11px] sm:text-xs font-mono uppercase tracking-[0.25em] text-white/70">
              {t('early_access_eyebrow', 'JOIN THE NEXT GENERATION OF LOGISTICS')}
            </span>
          </div>

          {/* Headline in Interactive Particle Text Style (same ParticleText as Hero) */}
          <div className="w-full max-w-[620px] h-[150px] sm:h-[175px] md:h-[200px]">
            <ParticleText
              text={`${t('early_access_title_1', 'Request')}\n${t('early_access_title_2', 'Early')} *${t('early_access_title_3', 'Access.')}*`}
              fontSize="clamp(2.5rem, 5.5vw, 4.2rem)"
              fontWeight={800}
              particleSize={2.0}
              density={2.5}
              color="#ffffff"
              highlightColor="#c084fc"
              scatter={140}
              gatherDuration={1350}
              stagger={260}
              pointerRepel={45}
              repelRadius={105}
              idleDrift={0.25}
              trigger="view"
              textAlign="left"
              glow={false}
            />
          </div>

          {/* CTA Action Button */}
          <div className="mt-6 sm:mt-8">
            <Link
              href="/join"
              className="inline-flex items-center justify-center gap-2 px-8 py-3 transition-all text-sm sm:text-base font-medium bg-white/10 text-white hover:bg-white/20 backdrop-blur-sm group cursor-pointer"
            >
              <span>{t('early_access_btn', 'Request Access')}</span>
              <span className="text-lg leading-none mt-[-2px] transition-transform group-hover:translate-x-0.5">›</span>
            </Link>
          </div>
        </div>
      </div>

      {/* Bottom Bar: Pegasus Brand on Left, Legal & Socials on Right */}
      <div className="relative z-10 w-full flex flex-wrap items-center justify-between gap-4 pt-6 border-t border-white/5">
        {/* Brand mark */}
        <div className="flex items-center gap-2.5 text-white/80 select-none">
          <span className="text-white/60 text-sm">❖</span>
          <span className="font-bold tracking-[0.2em] text-xs uppercase text-white font-mono">
            PEGASUS
          </span>
        </div>

        {/* Minimal Footer Links */}
        <div className="flex items-center gap-4 text-xs font-mono text-white/50">
          <a
            href="https://linkedin.com"
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-white transition-colors"
          >
            Linkedin
          </a>
          <span className="text-white/20">|</span>
          <Link href="/cookie-policy" className="hover:text-white transition-colors">
            Privacy
          </Link>
          <Link href="/cookie-policy" className="hover:text-white transition-colors">
            Terms
          </Link>
        </div>
      </div>
    </section>
  );
}
