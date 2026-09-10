'use client';

import React, { useEffect } from 'react';
import SiteAssistant from '@/app/components/SiteAssistant';
import { LanguageProvider } from '@/app/context/LanguageContext';
import type { Language } from '@/app/lib/i18n/translations';
import { ReactLenis } from 'lenis/react';
import { usePerfProfile } from '@/app/hooks/useDevice';

import TargetCursor from '@/app/components/TargetCursor';
import SplashCursor from '@/app/components/SplashCursor';

interface ClientLayoutProps {
  children: React.ReactNode;
  initialLanguage?: Language;
}

const ClientLayout: React.FC<ClientLayoutProps> = ({ children, initialLanguage }) => {
  const { allowHeavyFx, allowHoverFx, isLowEnd, isMobile, prefersReducedMotion } = usePerfProfile();

  useEffect(() => {
    try {
      if (sessionStorage.getItem('hasSeenSplash')) {
        document.documentElement.classList.add('splash-done');
        const splash = document.getElementById('app-splash-screen');
        if (splash) splash.style.display = 'none';
        return;
      }

      const holdDuration = isLowEnd || prefersReducedMotion ? 900 : 1800;
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
          }, 700);
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
    <LanguageProvider initialLanguage={initialLanguage}>
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
        {allowHoverFx ? (
          <TargetCursor
            targetSelector=".cursor-target, button, a[href], [role='button'], input[type='submit']"
            spinDuration={2}
            cursorColor="#ffffff"
            cursorColorOnTarget="#10B981"
          />
        ) : null}
        {children}
        <SiteAssistant />
      </ReactLenis>
    </LanguageProvider>
  );
};

export default ClientLayout;
