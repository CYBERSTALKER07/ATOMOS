'use client';

import React, { useEffect } from 'react';
import { usePathname } from 'next/navigation';
import SiteAssistant from '@/app/components/SiteAssistant';
import { LanguageProvider } from '@/app/context/LanguageContext';
import { ThemeProvider } from '@/app/context/ThemeContext';
import type { Language } from '@/app/lib/i18n/translations';
import { ReactLenis } from 'lenis/react';
import { usePerfProfile } from '@/app/hooks/useDevice';

import TargetCursor from '@/app/components/TargetCursor';
import SplashCursor from '@/app/components/SplashCursor';
import { CookieConsentProvider } from '@/app/context/CookieConsentContext';
import CookieBanner from '@/app/components/cookies/CookieBanner';
import CookiePreferenceModal from '@/app/components/cookies/CookiePreferenceModal';
import { initGSAP } from '@/app/lib/gsap';

// Prevent OpenUI devtools from auto-mounting
if (typeof window !== 'undefined') {
  try {
    const flag = Symbol.for('openui.devtools.autoMount');
    (window as unknown as Record<symbol, boolean>)[flag] = true;
  } catch (e) {}
}

interface ClientLayoutProps {
  children: React.ReactNode;
  initialLanguage?: Language;
}

const ClientLayout: React.FC<ClientLayoutProps> = ({ children, initialLanguage }) => {
  const pathname = usePathname();
  const isAssistantPage = pathname?.startsWith('/assistant');
  const { allowHeavyFx, isLowEnd, isMobile, prefersReducedMotion } = usePerfProfile();

  // Instant scroll-to-top on route navigation
  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: 'instant' });
  }, [pathname]);

  useEffect(() => {
    initGSAP(isLowEnd, prefersReducedMotion);
  }, [isLowEnd, prefersReducedMotion]);

  useEffect(() => {
    // Clean up any OpenUI devtools widgets if previously mounted
    try {
      document.querySelectorAll('[data-openui-devtools-root], [data-openui-devtools-auto-mount]').forEach((el) => el.remove());
    } catch (e) {}

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
            } catch (e) {}
          }, 300);
        } else {
          document.documentElement.classList.add('splash-done');
          try {
            sessionStorage.setItem('hasSeenSplash', 'true');
          } catch (e) {}
        }
      }, holdDuration);

      return () => clearTimeout(timer);
    } catch (e) {}
  }, [isLowEnd, prefersReducedMotion]);

  return (
    <ThemeProvider>
      <LanguageProvider initialLanguage={initialLanguage}>
        <CookieConsentProvider>
          {isAssistantPage ? (
            children
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
              {allowHeavyFx ? <SplashCursor COLOR="#10B981" RAINBOW_MODE={false} /> : null}
              <TargetCursor
                spinDuration={2}
                hideDefaultCursor
                parallaxOn
                hoverDuration={0.2}
                cursorColor="#10B981"
                cursorColorOnTarget="#B497CF"
              />
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
