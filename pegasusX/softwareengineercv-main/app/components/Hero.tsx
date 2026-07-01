'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { gsap } from 'gsap';
import Image from 'next/image';
import CurvedLoop from './CurvedLoop';
import TextType from './TextType';
import ChamferButton from './ChamferButton';
import { useIsMobile, useReducedMotion } from '../hooks/useDevice';
import {
  HERO_VIDEO_LOCAL_MP4,
  HERO_VIDEO_REMOTE_MP4,
  HERO_VIDEO_POSTER,
  HERO_VIDEO_LOOP_END_SEC,
} from '../lib/heroAssets';

export default function Hero() {
  const { isMobile } = useIsMobile();
  const prefersReducedMotion = useReducedMotion();

  const textRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const descRef = useRef<HTMLParagraphElement>(null);
  const ctaRef = useRef<HTMLDivElement>(null);
  const visualRef = useRef<HTMLDivElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);

  const [videoSrc, setVideoSrc] = useState(HERO_VIDEO_LOCAL_MP4);
  const [videoReady, setVideoReady] = useState(false);
  const [videoMissing, setVideoMissing] = useState(false);

  const startVideo = useCallback(() => {
    const video = videoRef.current;
    if (!video || videoMissing || prefersReducedMotion) return;
    video.currentTime = 0;
    void video.play().catch(() => {});
  }, [videoMissing, prefersReducedMotion]);

  const handleVideoError = useCallback(() => {
    if (videoSrc === HERO_VIDEO_LOCAL_MP4) {
      setVideoSrc(HERO_VIDEO_REMOTE_MP4);
      setVideoReady(false);
      return;
    }
    setVideoMissing(true);
    setVideoReady(false);
  }, [videoSrc]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || prefersReducedMotion || videoMissing) return;

    let loopEnd = HERO_VIDEO_LOOP_END_SEC;

    const syncLoopEnd = () => {
      if (video.duration && Number.isFinite(video.duration)) {
        loopEnd = Math.min(HERO_VIDEO_LOOP_END_SEC, video.duration);
      }
    };

    const onTimeUpdate = () => {
      if (video.currentTime >= loopEnd) {
        video.pause();
        video.currentTime = loopEnd;
      }
    };

    syncLoopEnd();
    video.addEventListener('loadedmetadata', syncLoopEnd);
    video.addEventListener('timeupdate', onTimeUpdate);
    video.muted = true;
    startVideo();

    return () => {
      video.removeEventListener('loadedmetadata', syncLoopEnd);
      video.removeEventListener('timeupdate', onTimeUpdate);
    };
  }, [prefersReducedMotion, videoMissing, videoSrc, startVideo]);

  useEffect(() => {
    const ctx = gsap.context(() => {
      if (isMobile || prefersReducedMotion) {
        gsap.set([titleRef.current, subtitleRef.current, descRef.current, ctaRef.current, visualRef.current], {
          opacity: 1,
          x: 0,
          y: 0,
        });
        return;
      }

      gsap
        .timeline({ defaults: { ease: 'power3.out' } })
        .fromTo(visualRef.current, { x: 48 }, { x: 0, duration: 1.2 })
        .fromTo(titleRef.current, { opacity: 0, y: 50 }, { opacity: 1, y: 0, duration: 1 }, '-=0.8')
        .fromTo(subtitleRef.current, { opacity: 0, y: 30 }, { opacity: 1, y: 0, duration: 0.8 }, '-=0.6')
        .fromTo(descRef.current, { opacity: 0, y: 20 }, { opacity: 1, y: 0, duration: 0.8 }, '-=0.5')
        .fromTo(ctaRef.current, { opacity: 0, y: 20 }, { opacity: 1, y: 0, duration: 0.8 }, '-=0.3');
    });

    return () => ctx.revert();
  }, [isMobile, prefersReducedMotion]);

  const scrollToNext = () => {
    document.querySelector('#about')?.scrollIntoView({ behavior: 'smooth' });
  };

  const showPoster = !videoReady || videoMissing || prefersReducedMotion;

  return (
    <section id="hero" className="min-h-screen relative flex items-center bg-black overflow-hidden pt-[4.5rem] md:pt-20">
      {!isMobile && (
        <>
          <div className="absolute top-0 left-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-30 z-10">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={900}
              direction="right"
              interactive={false}
              className="fill-white"
            />
          </div>

          <div className="absolute top-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-30 z-10 scale-x-[-1]">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={500}
              direction="left"
              interactive={false}
              className="fill-white"
            />
          </div>

          <div className="absolute bottom-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-30 z-10 rotate-180 scale-x-[-1]">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={200}
              direction="right"
              interactive={false}
              className="fill-white"
            />
          </div>
        </>
      )}

      <div className="container mx-auto px-4 py-20 relative z-20">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center max-w-7xl mx-auto">
          <div ref={textRef} className="space-y-8 order-2 lg:order-1">
            <div>
              <h1
                ref={titleRef}
                className="font-title text-5xl md:text-6xl lg:text-7xl xl:text-8xl font-light mb-4 text-white"
              >
                Pegasus
              </h1>

              <div ref={subtitleRef} className="mb-6">
                <TextType
                  text={['Logistics Platform', 'Dispatch System', 'Fleet Tracking', 'Payment Confidence']}
                  typingSpeed={isMobile ? 100 : 75}
                  pauseDuration={1500}
                  deletingSpeed={isMobile ? 70 : 50}
                  showCursor={true}
                  cursorCharacter="|"
                  loop={true}
                  textColors={['#FFFFFF', '#C0C0C0']}
                  className="text-2xl md:text-3xl lg:text-4xl font-light text-white"
                  cursorClassName="text-white font-light"
                />
              </div>

              <div className="w-90 h-[0.5px] bg-white mb-6" />

              <p
                ref={descRef}
                className="text-base md:text-lg font-extralight lg:text-xl text-white leading-relaxed max-w-xl"
              >
                Run supplier-led logistics from one platform — dispatch, tracking, payments,
                and coordination across every team in your network.
              </p>
            </div>

            <div ref={ctaRef} className="flex flex-col sm:flex-row gap-3">
              <ChamferButton onClick={scrollToNext} variant="fill">
                Explore Platform
              </ChamferButton>
              <ChamferButton href="/join" variant="ghost">
                Request Demo
              </ChamferButton>
            </div>
          </div>

          <div ref={visualRef} className="relative order-1 lg:order-2">
            <div className="relative h-[400px] md:h-[500px] lg:h-[600px] overflow-hidden shadow-2xl bg-black rounded-tl-[200px] rounded-br-[100px]">
              <div className="absolute inset-0">
                {showPoster ? (
                  <Image
                    src={HERO_VIDEO_POSTER}
                    alt=""
                    fill
                    priority
                    className="object-cover"
                    sizes="(max-width: 1024px) 100vw, 50vw"
                  />
                ) : null}

                {!videoMissing && !prefersReducedMotion ? (
                  <video
                    key={videoSrc}
                    ref={videoRef}
                    src={videoSrc}
                    muted
                    playsInline
                    preload="auto"
                    poster={HERO_VIDEO_POSTER}
                    onLoadedData={() => setVideoReady(true)}
                    onError={handleVideoError}
                    className={`absolute inset-0 h-full w-full object-cover ${videoReady ? 'opacity-100' : 'opacity-0'}`}
                  />
                ) : null}
              </div>

              {isMobile && (
                <div className="absolute inset-0 pointer-events-none">
                  <div className="absolute top-0 left-0 w-20 h-20 border-t-2 border-l-2 border-white" />
                  <div className="absolute top-0 right-0 w-20 h-20 border-t-2 border-l-2 border-white" />
                  <div className="absolute bottom-0 left-0 w-20 h-20 border-b-2 border-l-2 border-white" />
                  <div className="absolute bottom-0 right-0 w-20 h-20 border-b-2 border-r-2 border-white" />
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      <button
        type="button"
        className="absolute bottom-8 left-1/2 z-10 hidden -translate-x-1/2 cursor-pointer rounded-lg p-2 outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-black md:block"
        onClick={scrollToNext}
        aria-label="Scroll to next section"
      >
        <div className="flex flex-col items-center gap-2 text-white transition-colors duration-300 hover:text-[#FBFF63]">
          <span className="text-sm font-light tracking-widest">SCROLL</span>
          <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
          </svg>
        </div>
      </button>
    </section>
  );
}
