'use client';

import React, { useState, useEffect } from 'react';
import SplashScreen from './SplashScreen';
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
  const [showSplash, setShowSplash] = useState(false);
  const { allowHeavyFx, allowHoverFx, isLowEnd, isMobile, prefersReducedMotion } = usePerfProfile();

  useEffect(() => {
    try {
      const hasSeenSplash = sessionStorage.getItem('hasSeenSplash');
      if (!hasSeenSplash) {
        setShowSplash(true);
        sessionStorage.setItem('hasSeenSplash', 'true');
      }
    } catch (e) {
      // ignore
    } finally {
      // Clean up static pre-splash overlay once React hydrates
      const pre = document.getElementById('pre-splash-overlay');
      if (pre) {
        pre.style.display = 'none';
      }
      document.documentElement.classList.remove('needs-splash');
    }
  }, []);

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
        {showSplash && (
          <SplashScreen
            onComplete={() => setShowSplash(false)}
            duration={isLowEnd || prefersReducedMotion ? 1200 : 2500}
          />
        )}
        {children}
        {!showSplash ? <SiteAssistant /> : null}
      </ReactLenis>
    </LanguageProvider>
  );
};

export default ClientLayout;
