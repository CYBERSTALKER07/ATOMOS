'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import Image from 'next/image';
import { CardHoverOverlay } from '@/app/components/ContentCard';
import { useReducedMotion } from '@/app/hooks/useDevice';
import {
  ORDER_LIFECYCLE_GEMINI_SHARE,
  ORDER_LIFECYCLE_POSTER,
  ORDER_LIFECYCLE_STEPS,
  ORDER_LIFECYCLE_VIDEO_MP4,
} from '@/app/lib/lifecycleAssets';
import OrderLifecycleFlow from '@/app/components/flows/OrderLifecycleFlow';

type OrderLifecycleVideoProps = {
  variant?: 'hero' | 'inline';
  showCaption?: boolean;
  hoverLabel?: string;
};

export default function OrderLifecycleVideo({
  variant = 'hero',
  showCaption = true,
  hoverLabel = 'WATCH',
}: OrderLifecycleVideoProps) {
  const reduced = useReducedMotion();
  const videoRef = useRef<HTMLVideoElement>(null);
  const captionRef = useRef<HTMLParagraphElement>(null);
  const [videoMissing, setVideoMissing] = useState(false);
  const [videoReady, setVideoReady] = useState(false);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || videoMissing) return;

    const updateCaption = () => {
      if (!captionRef.current || !video.duration || !Number.isFinite(video.duration)) return;
      const progress = video.currentTime / video.duration;
      const stepIndex = Math.min(
        ORDER_LIFECYCLE_STEPS.length - 1,
        Math.floor(progress * ORDER_LIFECYCLE_STEPS.length)
      );
      captionRef.current.textContent = ORDER_LIFECYCLE_STEPS[stepIndex];
    };

    video.addEventListener('timeupdate', updateCaption);
    return () => video.removeEventListener('timeupdate', updateCaption);
  }, [videoMissing]);

  const playWithSound = useCallback(async () => {
    const video = videoRef.current;
    if (!video || videoMissing || reduced) return;

    video.muted = false;
    video.volume = 1;

    try {
      await video.play();
    } catch {
      video.muted = true;
      try {
        await video.play();
      } catch {
        // Browser blocked playback until explicit tap.
      }
    }
  }, [reduced, videoMissing]);

  const pauseVideo = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    video.pause();
  }, []);

  const handlePointerEnter = useCallback(() => {
    void playWithSound();
  }, [playWithSound]);

  const handlePointerLeave = useCallback(() => {
    pauseVideo();
  }, [pauseVideo]);

  const handlePointerDown = useCallback(() => {
    void playWithSound();
  }, [playWithSound]);

  const frameClass =
    variant === 'hero'
      ? 'order-lifecycle-video order-lifecycle-video--hero'
      : 'order-lifecycle-video order-lifecycle-video--inline';

  return (
    <section className={frameClass} aria-label="Order lifecycle animation">
      <div className="order-lifecycle-video__pin">
        <div
          className="order-lifecycle-video__frame bw-visual bw-visual--chamfer editorial-card--interactive"
          tabIndex={0}
          onPointerEnter={handlePointerEnter}
          onPointerLeave={handlePointerLeave}
          onPointerDown={handlePointerDown}
          onFocus={handlePointerEnter}
          onBlur={handlePointerLeave}
        >
          <div className="editorial-card__media order-lifecycle-video__media">
            {!videoMissing ? (
              <>
                <video
                  ref={videoRef}
                  className="order-lifecycle-video__el editorial-card__image object-cover"
                  src={ORDER_LIFECYCLE_VIDEO_MP4}
                  playsInline
                  preload="none"
                  poster={ORDER_LIFECYCLE_POSTER}
                  onLoadedData={() => setVideoReady(true)}
                  onError={() => setVideoMissing(true)}
                />
                {!videoReady ? (
                  <Image
                    src={ORDER_LIFECYCLE_POSTER}
                    alt=""
                    fill
                    className="order-lifecycle-video__poster bw-visual__img object-cover"
                    sizes="(max-width: 1280px) 100vw, 1280px"
                    priority
                  />
                ) : null}
                <CardHoverOverlay label={hoverLabel} />
              </>
            ) : (
              <div className="order-lifecycle-video__fallback">
                <Image
                  src={ORDER_LIFECYCLE_POSTER}
                  alt=""
                  fill
                  className="bw-visual__img object-cover opacity-40"
                  sizes="100vw"
                />
                <div className="order-lifecycle-video__fallback-inner">
                  <p className="font-mono text-xs uppercase tracking-[0.2em] text-white/50">
                    Video pending
                  </p>
                  <p className="mt-2 max-w-sm text-sm text-white/60">
                    Export from Gemini and save as{' '}
                    <code className="text-white/80">public/Minimal_black_and_white_line_a.mp4</code>
                  </p>
                  <a
                    href={ORDER_LIFECYCLE_GEMINI_SHARE}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-4 inline-block font-mono text-xs uppercase tracking-wider text-white/70 underline"
                  >
                    Open Gemini source ↗
                  </a>
                  <div className="mt-8 w-full max-w-2xl">
                    <OrderLifecycleFlow />
                  </div>
                </div>
              </div>
            )}
          </div>

          {showCaption && !videoMissing ? (
            <p ref={captionRef} className="order-lifecycle-video__caption">
              {ORDER_LIFECYCLE_STEPS[0]}
            </p>
          ) : null}
        </div>

        {variant === 'hero' && !videoMissing ? (
          <p className="order-lifecycle-video__hint font-mono text-[10px] uppercase tracking-[0.18em] text-white/35">
            Hover to play with sound · Black & white line-art lifecycle
          </p>
        ) : null}
      </div>
    </section>
  );
}
