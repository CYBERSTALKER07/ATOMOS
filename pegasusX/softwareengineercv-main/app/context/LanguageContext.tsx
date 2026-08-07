'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { translations, TranslationKey } from '../lib/i18n/translations';

export type Language = 'en' | 'ru';

interface LanguageContextType {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: (key: TranslationKey | string, fallback?: string) => string;
}

const LanguageContext = createContext<LanguageContextType | undefined>(undefined);

export const LanguageProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [language, setLanguageState] = useState<Language>('en');

  useEffect(() => {
    const storedLang = localStorage.getItem('pegasus_lang') as Language;
    if (storedLang === 'en' || storedLang === 'ru') {
      setLanguageState(storedLang);
      document.documentElement.lang = storedLang;
    } else {
      const browserLang = navigator.language.startsWith('ru') ? 'ru' : 'en';
      setLanguageState(browserLang);
      document.documentElement.lang = browserLang;
    }
  }, []);

  const setLanguage = (lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem('pegasus_lang', lang);
    document.documentElement.lang = lang;
  };

  const t = (key: TranslationKey | string, fallback?: string): string => {
    const langDict = translations[language] || translations['en'];
    const value = (langDict as Record<string, string>)[key];
    if (value) return value;
    const enValue = (translations['en'] as Record<string, string>)[key];
    if (enValue) return enValue;
    return fallback || key;
  };

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  );
};

export const useLanguage = (): LanguageContextType => {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error('useLanguage must be used within a LanguageProvider');
  }
  return context;
};
