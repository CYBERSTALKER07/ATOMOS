'use client';

import React, { useState, useRef, useEffect } from 'react';
import { Globe, Check, ChevronDown } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';
import { useTheme } from '../context/ThemeContext';
import type { Language } from '../lib/i18n/translations';

const GLOBAL_LANGUAGES: { code: Language; label: string; native: string; flag: string }[] = [
  { code: 'en', label: 'English', native: 'English', flag: '🇺🇸' },
  { code: 'ru', label: 'Russian', native: 'Русский', flag: '🇷🇺' },
  { code: 'uz', label: 'Uzbek', native: 'Oʻzbek', flag: '🇺🇿' },
  { code: 'es', label: 'Spanish', native: 'Español', flag: '🇪🇸' },
  { code: 'de', label: 'German', native: 'Deutsch', flag: '🇩🇪' },
  { code: 'fr', label: 'French', native: 'Français', flag: '🇫🇷' },
  { code: 'zh', label: 'Chinese', native: '中文', flag: '🇨🇳' },
  { code: 'ja', label: 'Japanese', native: '日本語', flag: '🇯🇵' },
  { code: 'ar', label: 'Arabic', native: 'العربية', flag: '🇦🇪' },
  { code: 'pt', label: 'Portuguese', native: 'Português', flag: '🇧🇷' },
  { code: 'tr', label: 'Turkish', native: 'Türkçe', flag: '🇹🇷' },
];

export default function LanguageSwitcher({ className = '' }: { className?: string }) {
  const { language, setLanguage } = useLanguage();
  const { resolvedTheme } = useTheme();
  const isLight = resolvedTheme === 'light';
  const [hovered, setHovered] = useState<string | null>(null);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const isCoreLang = language === 'en' || language === 'ru';

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

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
    <div ref={containerRef} className={`inline-flex items-center gap-1.5 ${className}`}>
      {/* Primary EN | RU Sliding Pill Toggle (Always Front and Center) */}
      <div
        className={`lang-switcher relative inline-grid grid-cols-2 items-center rounded-none border p-0.5 backdrop-blur-md ${
          isLight
            ? 'border-black/15 bg-black/5'
            : 'border-white/20 bg-black/60'
        }`}
        role="group"
        aria-label="Language Toggle"
      >
        {isCoreLang && (
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
        )}
        {(['en', 'ru'] as const).map((code) => (
          <button
            key={code}
            type="button"
            onClick={() => {
              setLanguage(code);
              setIsDropdownOpen(false);
            }}
            onMouseEnter={() => setHovered(code)}
            onMouseLeave={() => setHovered(null)}
            onFocus={() => setHovered(code)}
            onBlur={() => setHovered(null)}
            className="lang-switcher__btn relative z-10 px-2.5 py-1 text-[11px] font-mono font-semibold tracking-wider rounded-none transition-[color,background-color,transform] duration-200 ease-out"
            style={btnStyle(code)}
            aria-pressed={language === code}
            aria-label={code === 'en' ? 'Switch language to English' : 'Переключить язык на Русский'}
          >
            {code.toUpperCase()}
          </button>
        ))}
      </div>

      {/* Global Languages Dropdown Pill (for UZ, ES, DE, ZH, etc.) */}
      <div className="relative">
        <button
          type="button"
          onClick={() => setIsDropdownOpen(!isDropdownOpen)}
          className={`flex items-center gap-1 px-2 py-1 rounded-none border transition-all text-[11px] font-mono backdrop-blur-md ${
            !isCoreLang
              ? (isLight ? 'border-black bg-black text-white font-semibold' : 'border-white bg-white text-black font-semibold')
              : (isLight ? 'border-black/15 bg-black/5 hover:bg-black/10 text-zinc-700 hover:text-black' : 'border-white/20 bg-black/60 hover:bg-white/10 text-white/60 hover:text-white')
          }`}
          aria-expanded={isDropdownOpen}
          aria-label="Other international languages"
          title="More languages"
        >
          <Globe className="w-3 h-3" />
          {!isCoreLang && <span className="uppercase">{language}</span>}
          <ChevronDown
            className={`w-2.5 h-2.5 opacity-60 transition-transform ${isDropdownOpen ? 'rotate-180' : ''}`}
          />
        </button>

        {isDropdownOpen && (
          <div
            role="listbox"
            className={`absolute right-0 mt-2 w-52 max-h-80 overflow-y-auto rounded-none border p-1.5 shadow-2xl backdrop-blur-2xl z-[9999] animate-in fade-in zoom-in-95 duration-150 ${
              isLight
                ? 'border-black/10 bg-white/98 text-zinc-900 shadow-xl'
                : 'border-white/15 bg-black/95 text-white shadow-2xl'
            }`}
          >
            <div className={`px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-widest border-b mb-1 ${
              isLight ? 'text-zinc-500 border-black/10' : 'text-white/40 border-white/10'
            }`}>
              Global Corridors & Languages
            </div>
            {GLOBAL_LANGUAGES.map((lang) => {
              const isSelected = language === lang.code;
              return (
                <button
                  key={lang.code}
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  onClick={() => {
                    setLanguage(lang.code);
                    setIsDropdownOpen(false);
                  }}
                  className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-none text-xs font-mono transition-colors text-left ${
                    isSelected
                      ? (isLight ? 'bg-black text-white font-bold' : 'bg-white text-black font-bold')
                      : (isLight ? 'text-zinc-800 hover:bg-zinc-100 hover:text-black' : 'text-white/80 hover:bg-white/10 hover:text-white')
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm">{lang.flag}</span>
                    <span>{lang.native}</span>
                    <span className={`text-[10px] uppercase ${isLight ? 'text-zinc-400' : 'text-white/40'}`}>({lang.code})</span>
                  </div>
                  {isSelected && <Check className={`w-3.5 h-3.5 ${isLight ? 'text-white' : 'text-black'}`} />}
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
