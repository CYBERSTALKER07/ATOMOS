'use client';

import { useState, useEffect } from 'react';

export function useIsMobile() {
  const [isMobile, setIsMobile] = useState(false);
  const [isTablet, setIsTablet] = useState(false);

  useEffect(() => {
    const checkDevice = () => {
      const width = window.innerWidth;
      setIsMobile(width < 768);
      setIsTablet(width >= 768 && width < 1024);
    };

    checkDevice();
    window.addEventListener('resize', checkDevice);
    return () => window.removeEventListener('resize', checkDevice);
  }, []);

  return { isMobile, isTablet, isDesktop: !isMobile && !isTablet };
}

export function useReducedMotion() {
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    setPrefersReducedMotion(mediaQuery.matches);

    const handleChange = (e: MediaQueryListEvent) => {
      setPrefersReducedMotion(e.matches);
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  return prefersReducedMotion;
}

function detectLowEnd(): boolean {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') return false;

  const nav = navigator as Navigator & {
    deviceMemory?: number;
    connection?: { saveData?: boolean; effectiveType?: string };
  };

  const cores = navigator.hardwareConcurrency ?? 8;
  const memory = nav.deviceMemory; // Chrome only
  const saveData = !!nav.connection?.saveData;
  const slowNet =
    nav.connection?.effectiveType === 'slow-2g' ||
    nav.connection?.effectiveType === '2g' ||
    nav.connection?.effectiveType === '3g';

  // Heuristic: few cores, low RAM, or data-saver / slow network
  if (cores <= 4) return true;
  if (typeof memory === 'number' && memory <= 4) return true;
  if (saveData || slowNet) return true;
  return false;
}

export type PerfProfile = {
  isMobile: boolean;
  isTablet: boolean;
  isDesktop: boolean;
  prefersReducedMotion: boolean;
  isLowEnd: boolean;
  /** Continuous canvas FX (desktop capable only) */
  allowHeavyFx: boolean;
  /** Hover canvas FX (pointer devices, not phones) */
  allowHoverFx: boolean;
  /** Suggested glyph cell size */
  cellSize: number;
};

/**
 * Central gate for heavy digital / 3D / glitch effects.
 * Mobile, tablet, reduced-motion, and low-end desktops get the light path.
 */
export function usePerfProfile(): PerfProfile {
  const { isMobile, isTablet, isDesktop } = useIsMobile();
  const prefersReducedMotion = useReducedMotion();
  const [isLowEnd, setIsLowEnd] = useState(false);

  useEffect(() => {
    setIsLowEnd(detectLowEnd());
  }, []);

  const allowHeavyFx = isDesktop && !prefersReducedMotion && !isLowEnd;
  const allowHoverFx = !isMobile && !prefersReducedMotion && !isLowEnd;
  const cellSize = isMobile ? 14 : isTablet ? 12 : isLowEnd ? 13 : 9;

  return {
    isMobile,
    isTablet,
    isDesktop,
    prefersReducedMotion,
    isLowEnd,
    allowHeavyFx,
    allowHoverFx,
    cellSize,
  };
}
