'use client';

import React, { useState, useRef, useEffect } from 'react';
import { Globe, Check, ChevronDown } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';
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
        color: 'black',
        backgroundColor: 'white',
      };
    }

    return {
      color: isHover ? '#fff' : 'rgba(255,255,255,0.6)',
      backgroundColor: isHover ? 'rgba(255,255,255,0.1)' : 'transparent',
    };
  };

  return (
    <div ref={containerRef} className={`inline-flex items-center gap-1.5 ${className}`}>
      {/* Primary EN | RU Sliding Pill Toggle (Always Front and Center) */}
      <div
        className="lang-switcher relative inline-grid grid-cols-2 items-center rounded-full border border-white/20 bg-black/60 p-0.5 backdrop-blur-md"
        role="group"
        aria-label="Language Toggle"
      >
        {isCoreLang && (
          <span
            aria-hidden
            className="lang-switcher__thumb pointer-events-none absolute inset-y-0.5 left-0.5 w-[calc(50%-2px)] rounded-full bg-white shadow-[0_0_0_1px_rgba(255,255,255,0.08),0_4px_12px_rgba(0,0,0,0.35)]"
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
            className="lang-switcher__btn relative z-10 px-2.5 py-1 text-[11px] font-mono font-semibold tracking-wider rounded-full transition-[color,background-color,transform] duration-200 ease-out"
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
          className={`flex items-center gap-1 px-2 py-1 rounded-full border transition-all text-[11px] font-mono backdrop-blur-md ${
            !isCoreLang
              ? 'border-white bg-white text-black font-semibold'
              : 'border-white/20 bg-black/60 hover:bg-white/10 text-white/60 hover:text-white'
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
            className="absolute right-0 mt-2 w-52 max-h-80 overflow-y-auto rounded-xl border border-white/15 bg-black/95 p-1.5 shadow-2xl backdrop-blur-2xl z-[9999] animate-in fade-in zoom-in-95 duration-150"
          >
            <div className="px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-widest text-white/40 border-b border-white/10 mb-1">
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
                  className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-lg text-xs font-mono transition-colors text-left ${
                    isSelected
                      ? 'bg-white text-black font-bold'
                      : 'text-white/80 hover:bg-white/10 hover:text-white'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm">{lang.flag}</span>
                    <span>{lang.native}</span>
                    <span className="text-[10px] text-white/40 uppercase">({lang.code})</span>
                  </div>
                  {isSelected && <Check className="w-3.5 h-3.5 text-black" />}
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
