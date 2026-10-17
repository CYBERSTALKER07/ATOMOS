'use client';

import Link from 'next/link';
import { useLanguage } from '../context/LanguageContext';
import { useIsMobile } from '../hooks/useDevice';
import ParticleText from './ParticleText';
import Dither from './visuals/Dither';

export default function EarlyAccessSection() {
  const { t } = useLanguage();
  const { isMobile } = useIsMobile();

  return (
    <section
      className="relative w-full min-h-[580px] md:min-h-[660px] lg:min-h-[740px] bg-[#000000] text-white overflow-hidden flex flex-col justify-between py-10 sm:py-14 px-6 sm:px-12 lg:px-20 border-t border-white/10 select-none keep-dark"
      data-keep-dark
      data-keep-white
      aria-label="Request Early Access"
    >
      {/* Full-bleed face & hand rendered through the Dither shader */}
      <div className="absolute inset-0 pointer-events-none select-none" aria-hidden="true">
        <Dither
          image="/images/early-access-mesh.png"
          imageFit="cover"
          imagePosition={isMobile ? [0.28, 0.5] : [0.1, 0.5]}
          imageNoise={0.35}
          palette={['#000000', '#2b1f52', '#7a5cc8', '#c8b6ff', '#f5efff']}
          ditherMode="bayer"
          colorNum={4}
          pixelSize={2}
          waveSpeed={0.05}
          waveFrequency={3}
          waveAmplitude={0.3}
          mouseEffect="erase"
          mouseRadius={0.3}
          scale={1}
          contrast={1.15}
          brightness={0.05}
          mouseStrength={1}
          trail={0.5}
          enableMouseInteraction
          clickRipple
          intro
          disableAnimation={false}
        />
        {/* Subtle mobile backdrop gradient for text readability */}
        <div className="absolute inset-0 bg-gradient-to-t from-black/85 via-black/30 to-black/10 sm:hidden" />
      </div>

      {/* Top Spacer */}
      <div className="relative z-10 w-full" />

      {/* Center-Right Main Content Block */}
      <div className="relative z-10 w-full flex justify-end items-center my-auto py-10 md:py-14">
        {/* Right Content Block */}
        <div className="w-full max-w-xl md:max-w-2xl lg:max-w-3xl lg:pl-16 text-left">
          {/* Eyebrow with leading dash line */}
          <div className="flex items-center gap-3 mb-4 sm:mb-6">
            <span className="w-8 h-[1px] bg-white/40 inline-block shrink-0" />
            <span className="text-[10px] sm:text-xs font-mono uppercase tracking-[0.18em] sm:tracking-[0.25em] text-white/70">
              {t('early_access_eyebrow', 'JOIN THE NEXT GENERATION OF LOGISTICS')}
            </span>
          </div>

          {/* Headline in Interactive Particle Text Style */}
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
