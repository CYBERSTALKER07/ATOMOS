'use client';

import React, { useState, useEffect } from 'react';
import SplashScreen from './SplashScreen';
import SiteAssistant from '@/app/components/SiteAssistant';
import { LanguageProvider } from '@/app/context/LanguageContext';
import type { Language } from '@/app/lib/i18n/translations';

import TargetCursor from '@/app/components/TargetCursor';
import SplashCursor from '@/app/components/SplashCursor';

interface ClientLayoutProps {
  children: React.ReactNode;
  initialLanguage?: Language;
}

const ClientLayout: React.FC<ClientLayoutProps> = ({ children, initialLanguage }) => {
  const [showSplash, setShowSplash] = useState(false);

  useEffect(() => {
    const hasSeenSplash = sessionStorage.getItem('hasSeenSplash');
    if (!hasSeenSplash) {
      setShowSplash(true);
      sessionStorage.setItem('hasSeenSplash', 'true');
    }
  }, []);

  return (
    <LanguageProvider initialLanguage={initialLanguage}>
      <SplashCursor COLOR="#10B981" RAINBOW_MODE={false} />
      <TargetCursor
        targetSelector=".cursor-target, button, a[href], [role='button'], input[type='submit']"
        spinDuration={2}
        cursorColor="#ffffff"
        cursorColorOnTarget="#10B981"
      />
      {showSplash && <SplashScreen onComplete={() => setShowSplash(false)} duration={3000} />}
      {children}
      {!showSplash ? <SiteAssistant /> : null}
    </LanguageProvider>
  );
};

export default ClientLayout;
