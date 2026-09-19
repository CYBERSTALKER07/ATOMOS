'use client';

import React from 'react';
import { Cookie } from 'lucide-react';
import { useCookieConsent } from '@/app/context/CookieConsentContext';
import { useLanguage } from '@/app/context/LanguageContext';

export default function CookieSettingsTrigger({
  className = '',
  variant = 'text',
}: {
  className?: string;
  variant?: 'text' | 'button';
}) {
  const { openModal } = useCookieConsent();
  const { language } = useLanguage();
  const isRu = language === 'ru';

  if (variant === 'button') {
    return (
      <button
        type="button"
        onClick={openModal}
        className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-none border text-xs font-mono uppercase tracking-wider transition-colors border-black/10 hover:border-black/30 dark:border-white/10 dark:hover:border-white/30 text-zinc-700 dark:text-white/70 hover:text-black dark:hover:text-white ${className}`}
      >
        <Cookie className="w-3.5 h-3.5" />
        <span>{isRu ? 'Настройки cookie' : 'Cookie Preferences'}</span>
      </button>
    );
  }

  return (
    <button
      type="button"
      onClick={openModal}
      className={`text-sm text-zinc-600 hover:text-black dark:text-white/70 dark:hover:text-white transition-colors cursor-pointer text-left ${className}`}
    >
      {isRu ? 'Настройки cookie' : 'Cookie Preferences'}
    </button>
  );
}
