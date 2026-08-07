'use client';

import React from 'react';
import { useLanguage } from '../context/LanguageContext';

export default function LanguageSwitcher({ className = '' }: { className?: string }) {
  const { language, setLanguage } = useLanguage();

  return (
    <div className={`inline-flex items-center rounded-full border border-white/20 bg-black/60 p-0.5 backdrop-blur-md ${className}`}>
      <button
        type="button"
        onClick={() => setLanguage('en')}
        className={`px-2.5 py-1 text-[11px] font-mono font-semibold tracking-wider transition-all rounded-full ${
          language === 'en'
            ? 'bg-white text-black shadow-sm'
            : 'text-white/60 hover:text-white'
        }`}
        aria-label="Switch language to English"
      >
        EN
      </button>
      <span className="text-white/20 text-[10px] font-mono">|</span>
      <button
        type="button"
        onClick={() => setLanguage('ru')}
        className={`px-2.5 py-1 text-[11px] font-mono font-semibold tracking-wider transition-all rounded-full ${
          language === 'ru'
            ? 'bg-white text-black shadow-sm'
            : 'text-white/60 hover:text-white'
        }`}
        aria-label="Переключить язык на Русский"
      >
        RU
      </button>
    </div>
  );
}
