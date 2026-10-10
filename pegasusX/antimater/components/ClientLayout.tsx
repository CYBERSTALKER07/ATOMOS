'use client';

import React, { useEffect } from 'react';
import { usePathname } from 'next/navigation';
import SiteAssistant from '@/app/components/SiteAssistant';
import { LanguageProvider } from '@/app/context/LanguageContext';
import { ThemeProvider } from '@/app/context/ThemeContext';
import type { Language } from '@/app/lib/i18n/translations';
import { ReactLenis, useLenis } from 'lenis/react';
import { usePerfProfile } from '@/app/hooks/useDevice';

import SplashCursor from '@/app/components/SplashCursor';
import { CookieConsentProvider } from '@/app/context/CookieConsentContext';
import CookieBanner from '@/app/components/cookies/CookieBanner';
import CookiePreferenceModal from '@/app/components/cookies/CookiePreferenceModal';
import { initGSAP } from '@/app/lib/gsap';

function NavigationScrollManager() {
  const pathname = usePathname();
  const lenis = useLenis();

  useEffect(() => {
    if (typeof window === 'undefined') return;

    // Respect in-page hash anchors if present
    if (window.location.hash) {
      const hash = window.location.hash;
      try {
        const el = document.querySelector(hash);
        if (el) {
          if (lenis) {
            lenis.scrollTo(el as HTMLElement, { immediate: true });
          } else {
            el.scrollIntoView({ behavior: 'instant' });
          }
          return;
        }
      } catch {}
    }

    // Immediately reset scroll to the top
    if (lenis) {
      lenis.scrollTo(0, { immediate: true });
    }
    window.scrollTo({ top: 0, left: 0, behavior: 'instant' });
    document.documentElement.scrollTop = 0;
    document.body.scrollTop = 0;

    // Re-assert top position on next animation frames to counter any layout shifts or Lenis recalculations
    const rafId = requestAnimationFrame(() => {
      if (window.location.hash) return;
      if (lenis) {
        lenis.scrollTo(0, { immediate: true });
      }
      window.scrollTo({ top: 0, left: 0, behavior: 'instant' });
      document.documentElement.scrollTop = 0;
      document.body.scrollTop = 0;
    });

    const timer = setTimeout(() => {
      if (window.location.hash) return;
      if (window.scrollY > 0) {
        if (lenis) {
          lenis.scrollTo(0, { immediate: true });
        }
        window.scrollTo({ top: 0, left: 0, behavior: 'instant' });
      }
    }, 60);

    return () => {
      cancelAnimationFrame(rafId);
      clearTimeout(timer);
    };
  }, [pathname, lenis]);

  return null;
}

// Prevent OpenUI devtools from auto-mounting
if (typeof window !== 'undefined') {
  try {
    const flag = Symbol.for('openui.devtools.autoMount');
    (window as unknown as Record<symbol, boolean>)[flag] = true;
  } catch {}
}

interface ClientLayoutProps {
  children: React.ReactNode;
  initialLanguage?: Language;
}

const ClientLayout: React.FC<ClientLayoutProps> = ({ children, initialLanguage }) => {
  const pathname = usePathname();
  const isAssistantPage = pathname?.startsWith('/assistant');
  const { allowHeavyFx, isLowEnd, isMobile, prefersReducedMotion } = usePerfProfile();

  // Disable browser automatic scroll restoration so page transitions always start at top
  useEffect(() => {
    if (typeof window !== 'undefined' && 'scrollRestoration' in window.history) {
      window.history.scrollRestoration = 'manual';
    }
  }, []);

  useEffect(() => {
    initGSAP(isLowEnd, prefersReducedMotion);
  }, [isLowEnd, prefersReducedMotion]);

  useEffect(() => {
    // Clean up any OpenUI devtools widgets if previously mounted
    try {
      document.querySelectorAll('[data-openui-devtools-root], [data-openui-devtools-auto-mount]').forEach((el) => el.remove());
    } catch {}

    try {
      if (sessionStorage.getItem('hasSeenSplash')) {
        document.documentElement.classList.add('splash-done');
        const splash = document.getElementById('app-splash-screen');
        if (splash) splash.style.display = 'none';
        return;
      }

      const holdDuration = isLowEnd || prefersReducedMotion ? 400 : 700;
      const timer = setTimeout(() => {
        const splash = document.getElementById('app-splash-screen');
        if (splash) {
          splash.classList.add('splash-screen-fadeout');
          setTimeout(() => {
            splash.style.display = 'none';
            document.documentElement.classList.add('splash-done');
            try {
              sessionStorage.setItem('hasSeenSplash', 'true');
            } catch {}
          }, 300);
        } else {
          document.documentElement.classList.add('splash-done');
          try {
            sessionStorage.setItem('hasSeenSplash', 'true');
          } catch {}
        }
      }, holdDuration);

      return () => clearTimeout(timer);
    } catch {}
  }, [isLowEnd, prefersReducedMotion]);

  return (
    <ThemeProvider>
      <LanguageProvider initialLanguage={initialLanguage}>
        <CookieConsentProvider>
          {isAssistantPage ? (
            <>
              <NavigationScrollManager />
              {children}
            </>
          ) : (
            <ReactLenis
              root
              options={{
                lerp: 0.08,
                duration: 1.2,
                smoothWheel: !isLowEnd && !isMobile,
                syncTouch: false,
              }}
            >
              <NavigationScrollManager />
              {allowHeavyFx ? <SplashCursor COLOR="#10B981" RAINBOW_MODE={false} /> : null}
              {children}
              <SiteAssistant />
            </ReactLenis>
          )}
          <CookieBanner />
          <CookiePreferenceModal />
        </CookieConsentProvider>
      </LanguageProvider>
    </ThemeProvider>
  );
};

export default ClientLayout;
