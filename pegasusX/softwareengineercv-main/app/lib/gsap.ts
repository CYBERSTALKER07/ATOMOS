'use client';

import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { ScrollToPlugin } from 'gsap/ScrollToPlugin';
import { Flip } from 'gsap/Flip';
import { CustomEase } from 'gsap/CustomEase';
import { Observer } from 'gsap/Observer';

let isRegistered = false;

/**
 * Initializes and registers the core GSAP plugins and custom tactical eases.
 * Configures the GSAP engine and ScrollTrigger specifically for battery efficiency
 * and low-end hardware performance.
 */
export function initGSAP(isLowEnd: boolean = false, prefersReducedMotion: boolean = false): void {
  if (typeof window === 'undefined') return;

  if (!isRegistered) {
    gsap.registerPlugin(ScrollTrigger, ScrollToPlugin, Flip, CustomEase, Observer);

    try {
      CustomEase.create('pegasus', 'M0,0 C0.16,1 0.3,1 1,1');
      CustomEase.create('tacticalSnap', 'M0,0 C0.25,1 0.5,1 1,1');
      CustomEase.create('tacticalSpring', 'M0,0 C0.175,0.885 0.32,1.1 1,1');
    } catch {
      // Eases already registered
    }

    isRegistered = true;
  }

  // Optimize GSAP core engine for low-end / battery profile
  gsap.config({
    // autoSleep puts ticker to sleep when animations stop (huge CPU/battery win)
    autoSleep: isLowEnd ? 30 : 60,
    force3D: 'auto',
    nullTargetWarn: false,
  });

  // Optimize ScrollTrigger performance
  ScrollTrigger.config({
    limitCallbacks: true,
    ignoreMobileResize: true,
    autoRefreshEvents: 'visibilitychange,DOMContentLoaded,load',
  });

  // On low-end or reduced motion, enable fastScrollEnd to avoid catch-up animation lag
  if (isLowEnd || prefersReducedMotion) {
    ScrollTrigger.defaults({
      fastScrollEnd: true,
    });
  }
}

// Auto-register upon client module evaluation
if (typeof window !== 'undefined') {
  initGSAP();
}

export interface SmoothScrollOptions {
  offsetY?: number;
  duration?: number;
  ease?: string;
  onComplete?: () => void;
  instant?: boolean;
}

/**
 * Universal hardware-accelerated smooth scroll-to helper.
 * Automatically respects prefers-reduced-motion and low-end constraints.
 */
export function smoothScrollTo(
  target: string | Element | number,
  options: SmoothScrollOptions = {}
): void {
  if (typeof window === 'undefined') return;

  const {
    offsetY = 70,
    duration = 1.0,
    ease = 'pegasus',
    onComplete,
    instant = false,
  } = options;

  const isReducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches;

  if (instant || isReducedMotion) {
    if (typeof target === 'number') {
      window.scrollTo({ top: target, behavior: 'instant' });
    } else {
      const el = typeof target === 'string' ? document.querySelector(target) : target;
      if (el) {
        const top = el.getBoundingClientRect().top + window.scrollY - offsetY;
        window.scrollTo({ top, behavior: 'instant' });
      }
    }
    onComplete?.();
    return;
  }

  const selectedEase = CustomEase.get(ease) ? ease : 'power3.inOut';

  gsap.to(window, {
    duration,
    scrollTo: {
      y: target,
      offsetY,
      autoKill: true,
    },
    ease: selectedEase,
    onComplete,
    overwrite: 'auto',
  });
}

export { gsap, ScrollTrigger, ScrollToPlugin, Flip, CustomEase, Observer };
export default gsap;
