'use client';

import { memo, useCallback, useEffect, useRef } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import type { Route } from 'next';

export const EDITORIAL_IMAGES = [
  '/original-6ba13d16f8042b23a4c50a692bb23176.webp',
  '/original-1ec76e0e43bf8113cf940c5b4cdaed7a.webp',
  '/original-7b36f739e822f2c737563ad11d6c58df.webp',
  '/original-1137dbc252460b3d33f86e46095e1b12.webp',
  '/original-2096796ee8410886b847d3c0a93c3d4e.webp',
  '/d43ad1fb95964cffaa808bf7a0364307.webp',
  '/original-66e7443f7182936b9a9732295fbfc121.webp',
  '/original-50941087b949fb6cd5a51dc535e09879.webp',
] as const;

export type ContentCardVariant = 'vertical' | 'split' | 'featured';
export type ContentCardTone = 'dark' | 'light';

function formatTag(tag: string) {
  const trimmed = tag.trim();
  if (trimmed.startsWith('[') && trimmed.endsWith(']')) return trimmed;
  return `[${trimmed.toUpperCase()}]`;
}

export function ContentCardTag({
  children,
  className = '',
}: {
  children: string;
  className?: string;
}) {
  return (
    <p className={`editorial-tag ${className}`}>{formatTag(children)}</p>
  );
}

export function ContentCardEyebrow({
  children,
  className = '',
}: {
  children: string;
  className?: string;
}) {
  return <p className={`editorial-eyebrow ${className}`}>{children}</p>;
}

export function ContentCardButton({
  children,
  href,
  className = '',
  inverted = false,
}: {
  children: React.ReactNode;
  href?: string;
  className?: string;
  inverted?: boolean;
}) {
  const classes = `editorial-btn ${inverted ? 'editorial-btn--inverted' : ''} ${className}`;
  if (href) {
    return (
      <Link href={href as Route} prefetch={false} className={classes}>
        {children}
      </Link>
    );
  }
  return <span className={classes}>{children}</span>;
}

export function ContentCardLink({
  children = 'READ MORE',
  href,
  className = '',
}: {
  children?: string;
  href: string;
  className?: string;
}) {
  return (
    <Link href={href as Route} prefetch={false} className={`editorial-link ${className}`}>
      {children} <span aria-hidden="true">&gt;</span>
    </Link>
  );
}

export function CardHoverOverlay({ label = 'WATCH' }: { label?: string }) {
  const actionLabel = label.trim().toUpperCase() || 'WATCH';
  const mediaRef = useRef<HTMLDivElement>(null);
  const controlRef = useRef<HTMLDivElement>(null);
  const targetRef = useRef({ x: 0, y: 0 });
  const currentRef = useRef({ x: 0, y: 0 });
  const frameRef = useRef(0);
  const trackingRef = useRef(false);
  const canTrackRef = useRef(false);

  useEffect(() => {
    canTrackRef.current = window.matchMedia('(hover: hover) and (pointer: fine)').matches;
    return () => cancelAnimationFrame(frameRef.current);
  }, []);

  const clampOffset = useCallback((x: number, y: number) => {
    const media = mediaRef.current;
    const control = controlRef.current;
    if (!media || !control) return { x, y };

    const mediaRect = media.getBoundingClientRect();
    const controlRect = control.getBoundingClientRect();
    const maxX = Math.max(0, mediaRect.width / 2 - controlRect.width / 2);
    const maxY = Math.max(0, mediaRect.height / 2 - controlRect.height / 2);

    return {
      x: Math.min(maxX, Math.max(-maxX, x)),
      y: Math.min(maxY, Math.max(-maxY, y)),
    };
  }, []);

  const applyTransform = useCallback(() => {
    if (!controlRef.current) return;
    controlRef.current.style.transform = `translate3d(${currentRef.current.x}px, ${currentRef.current.y}px, 0)`;
  }, []);

  const tick = useCallback(() => {
    frameRef.current = 0;
    const lerp = 0.16;
    currentRef.current.x += (targetRef.current.x - currentRef.current.x) * lerp;
    currentRef.current.y += (targetRef.current.y - currentRef.current.y) * lerp;
    applyTransform();

    const dx = Math.abs(targetRef.current.x - currentRef.current.x);
    const dy = Math.abs(targetRef.current.y - currentRef.current.y);
    if (dx > 0.4 || dy > 0.4) {
      frameRef.current = requestAnimationFrame(tick);
    } else {
      currentRef.current = { ...targetRef.current };
      applyTransform();
    }
  }, [applyTransform]);

  const ensureFrame = useCallback(() => {
    if (!frameRef.current) frameRef.current = requestAnimationFrame(tick);
  }, [tick]);

  const startTracking = useCallback(() => {
    trackingRef.current = true;
    ensureFrame();
  }, [ensureFrame]);

  const handlePointerMove = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (!canTrackRef.current) return;

      const media = mediaRef.current;
      if (!media) return;

      const rect = media.getBoundingClientRect();
      const next = clampOffset(
        event.clientX - rect.left - rect.width / 2,
        event.clientY - rect.top - rect.height / 2
      );

      targetRef.current = next;
      startTracking();
    },
    [clampOffset, startTracking]
  );

  const handlePointerEnter = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (!canTrackRef.current) return;

      const media = mediaRef.current;
      if (!media) return;

      const rect = media.getBoundingClientRect();
      const next = clampOffset(
        event.clientX - rect.left - rect.width / 2,
        event.clientY - rect.top - rect.height / 2
      );

      targetRef.current = next;
      currentRef.current = next;
      applyTransform();
    },
    [applyTransform, clampOffset]
  );

  const handlePointerLeave = useCallback(() => {
    trackingRef.current = false;
    targetRef.current = { x: 0, y: 0 };
    ensureFrame();
  }, [ensureFrame]);

  return (
    <div
      ref={mediaRef}
      className="card-hover-action"
      aria-hidden="true"
      onPointerMove={handlePointerMove}
      onPointerEnter={handlePointerEnter}
      onPointerLeave={handlePointerLeave}
    >
      <div ref={controlRef} className="card-hover-action__control">
        <span className="card-hover-action__icon" aria-hidden="true">
          <span className="card-hover-action__play" />
        </span>
        <span className="card-hover-action__label">{actionLabel}</span>
      </div>
    </div>
  );
}

function resolveHoverLabel(ctaLabel?: string, hoverLabel?: string) {
  if (hoverLabel) return hoverLabel;
  if (!ctaLabel) return 'WATCH';
  const normalized = ctaLabel.trim().toUpperCase();
  if (normalized.includes('DEMO')) return 'WATCH';
  if (normalized.includes('READ')) return 'VIEW';
  if (normalized.includes('CONTACT')) return 'VIEW';
  return normalized;
}

export interface ContentCardProps {
  variant?: ContentCardVariant;
  tone?: ContentCardTone;
  tag?: string;
  eyebrow?: string;
  title: string;
  description?: string;
  image?: string;
  imageAlt?: string;
  href?: string;
  ctaLabel?: string;
  ctaStyle?: 'button' | 'link';
  className?: string;
  imagePriority?: boolean;
  hoverLabel?: string;
  children?: React.ReactNode;
}

function ContentCard({
  variant = 'vertical',
  tone = 'dark',
  tag,
  eyebrow,
  title,
  description,
  image,
  imageAlt,
  href,
  ctaLabel,
  ctaStyle = 'link',
  className = '',
  imagePriority = false,
  hoverLabel,
  children,
}: ContentCardProps) {
  const isLight = tone === 'light' || variant === 'featured';
  const actionLabel = resolveHoverLabel(ctaLabel, hoverLabel);
  const shellClass = [
    'editorial-card',
    'editorial-card--interactive',
    `editorial-card--${variant}`,
    isLight ? 'editorial-card--light' : 'editorial-card--dark',
    className,
  ]
    .filter(Boolean)
    .join(' ');

  const cta =
    ctaLabel ? (
      ctaStyle === 'button' ? (
        href ? (
          <span className={`editorial-btn ${isLight ? 'editorial-btn--inverted' : ''}`}>{ctaLabel}</span>
        ) : (
          <ContentCardButton inverted={isLight}>{ctaLabel}</ContentCardButton>
        )
      ) : href ? (
        <span className="editorial-link">
          {ctaLabel} <span aria-hidden="true">&gt;</span>
        </span>
      ) : null
    ) : null;

  const body = (
    <div className="editorial-card__body">
      {eyebrow ? <ContentCardEyebrow>{eyebrow}</ContentCardEyebrow> : null}
      {tag ? <ContentCardTag>{tag}</ContentCardTag> : null}
      <h3 className="editorial-card__title">{title}</h3>
      {description ? <p className="editorial-card__description">{description}</p> : null}
      {children}
      {cta ? <div className="editorial-card__cta">{cta}</div> : null}
    </div>
  );

  const media = (
    <div className="editorial-card__media">
      {image ? (
        <Image
          src={image}
          alt={imageAlt || title}
          fill
          className="editorial-card__image object-cover"
          sizes="(max-width: 768px) 100vw, 50vw"
          priority={imagePriority}
          loading={imagePriority ? undefined : 'lazy'}
        />
      ) : (
        <div className="editorial-card__media--placeholder absolute inset-0" aria-hidden="true" />
      )}
      {image ? <CardHoverOverlay label={actionLabel} /> : null}
    </div>
  );

  const inner = (
    <>
      {media}
      {body}
    </>
  );

  if (href) {
    return (
      <Link href={href as Route} prefetch={false} className={shellClass}>
        {inner}
      </Link>
    );
  }

  return <article className={shellClass}>{inner}</article>;
}

export default memo(ContentCard);
