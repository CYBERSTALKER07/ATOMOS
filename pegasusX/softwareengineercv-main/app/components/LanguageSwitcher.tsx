'use client';

import React, { useState } from 'react';
import { useLanguage } from '../context/LanguageContext';
import { useTheme } from '../context/ThemeContext';

export default function LanguageSwitcher({ className = '' }: { className?: string }) {
  const { language, setLanguage } = useLanguage();
  const { resolvedTheme } = useTheme();
  const isLight = resolvedTheme === 'light';
  const [hovered, setHovered] = useState<string | null>(null);

  const btnStyle = (code: 'en' | 'ru'): React.CSSProperties => {
    const active = language === code;
    const isHover = hovered === code;

    if (active) {
      return {
        color: isLight ? 'white' : 'black',
        backgroundColor: isLight ? 'black' : 'white',
      };
    }

    return {
      color: isLight
        ? (isHover ? '#000' : 'rgba(0,0,0,0.65)')
        : (isHover ? '#fff' : 'rgba(255,255,255,0.6)'),
      backgroundColor: isHover
        ? (isLight ? 'rgba(0,0,0,0.06)' : 'rgba(255,255,255,0.1)')
        : 'transparent',
    };
  };

  return (
    <div className={`inline-flex items-center ${className}`}>
      {/* Primary EN | RU Sliding Pill Toggle */}
      <div
        className={`lang-switcher relative inline-grid grid-cols-2 items-center rounded-none border p-0.5 backdrop-blur-md ${
          isLight
            ? 'border-black bg-white text-black'
            : 'border-white bg-black text-white'
        }`}
        role="group"
        aria-label="Language Toggle"
      >
        <span
          aria-hidden
          className={`lang-switcher__thumb pointer-events-none absolute inset-y-0.5 left-0.5 w-[calc(50%-2px)] rounded-none shadow-sm ${
            isLight
              ? 'bg-black shadow-[0_2px_8px_rgba(0,0,0,0.25)]'
              : 'bg-white shadow-[0_0_0_1px_rgba(255,255,255,0.08),0_4px_12px_rgba(0,0,0,0.35)]'
          }`}
          style={{
            transform: language === 'ru' ? 'translateX(100%)' : 'translateX(0)',
            transition: 'transform 300ms cubic-bezier(0.22, 1, 0.36, 1)',
          }}
        />
        {(['en', 'ru'] as const).map((code) => (
          <button
            key={code}
            type="button"
            onClick={() => setLanguage(code)}
            onMouseEnter={() => setHovered(code)}
            onMouseLeave={() => setHovered(null)}
            onFocus={() => setHovered(code)}
            onBlur={() => setHovered(null)}
            className="lang-switcher__btn relative z-10 px-2.5 py-1 text-[11px] font-mono font-semibold tracking-wider rounded-none transition-[color,background-color,transform] duration-200 ease-out cursor-pointer"
            style={btnStyle(code)}
            aria-pressed={language === code}
            aria-label={code === 'en' ? 'Switch language to English' : 'Переключить язык на Русский'}
          >
            {code.toUpperCase()}
          </button>
        ))}
      </div>
    </div>
  );
}
