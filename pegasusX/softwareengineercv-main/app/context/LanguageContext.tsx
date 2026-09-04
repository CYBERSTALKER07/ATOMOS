'use client';

import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { translations, TranslationKey } from '../lib/i18n/translations';

export type Language = 'en' | 'ru';

interface LanguageContextType {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: (key: TranslationKey | string, fallback?: string) => string;
}

const LanguageContext = createContext<LanguageContextType | undefined>(undefined);

export const LanguageProvider: React.FC<{ children: React.ReactNode; initialLanguage?: Language }> = ({
  children,
  initialLanguage = 'en',
}) => {
  const [language, setLanguageState] = useState<Language>(initialLanguage);

  useEffect(() => {
    // Prefer explicit ?lang= for hreflang / shareable language links
    const params = new URLSearchParams(window.location.search);
    const queryLang = params.get('lang');
    if (queryLang === 'en' || queryLang === 'ru') {
      setLanguageState(queryLang);
      localStorage.setItem('pegasus_lang', queryLang);
      document.cookie = `pegasus_lang=${queryLang}; path=/; max-age=31536000; SameSite=Lax`;
      document.documentElement.lang = queryLang;
      return;
    }

    // Try to get from cookie first (as it's the source of truth for SSR)
    const cookies = document.cookie.split('; ');
    const langCookie = cookies.find((row) => row.startsWith('pegasus_lang='));
    let storedLang = langCookie ? (langCookie.split('=')[1] as Language) : null;

    // Fallback to localStorage if cookie isn't there (e.g. migration)
    if (!storedLang) {
      storedLang = localStorage.getItem('pegasus_lang') as Language;
      if (storedLang === 'en' || storedLang === 'ru') {
        // Sync cookie from local storage
        document.cookie = `pegasus_lang=${storedLang}; path=/; max-age=31536000; SameSite=Lax`;
      }
    }

    if (storedLang === 'en' || storedLang === 'ru') {
      setLanguageState(storedLang);
      document.documentElement.lang = storedLang;
    } else if (initialLanguage) {
      setLanguageState(initialLanguage);
      document.documentElement.lang = initialLanguage;
    } else {
      const browserLang = navigator.language.startsWith('ru') ? 'ru' : 'en';
      setLanguageState(browserLang);
      document.cookie = `pegasus_lang=${browserLang}; path=/; max-age=31536000; SameSite=Lax`;
      document.documentElement.lang = browserLang;
    }
  }, [initialLanguage]);

  const setLanguage = useCallback((lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem('pegasus_lang', lang);
    document.cookie = `pegasus_lang=${lang}; path=/; max-age=31536000; SameSite=Lax`;
    document.documentElement.lang = lang;
  }, []);

  const t = useCallback((key: TranslationKey | string, fallback?: string): string => {
    const langDict = translations[language] || translations.en;
    const value = (langDict as Record<string, string>)[key];
    if (value) return value;
    const enValue = (translations.en as Record<string, string>)[key];
    if (enValue) return enValue;
    return fallback || key;
  }, [language]);

  const value = useMemo(() => ({ language, setLanguage, t }), [language, setLanguage, t]);

  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
};

export const useLanguage = () => {
  const context = useContext(LanguageContext);
  if (context === undefined) {
    throw new Error('useLanguage must be used within a LanguageProvider');
  }
  return context;
};
