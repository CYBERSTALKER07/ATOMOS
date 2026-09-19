'use client';

import React, { useState, useEffect } from 'react';
import { Sun, Moon } from '@/components/icons';
import { useTheme } from '@/app/context/ThemeContext';

export default function ThemeSwitcher({ className = '' }: { className?: string }) {
  const { resolvedTheme, toggleTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return (
      <button
        type="button"
        aria-label="Toggle theme"
        className={`w-8 h-8 rounded-none flex items-center justify-center border border-black/20 dark:border-white/20 bg-white dark:bg-black text-black/40 dark:text-white/40 ${className}`}
        disabled
      >
        <Moon className="w-3.5 h-3.5" />
      </button>
    );
  }

  const isLight = resolvedTheme === 'light';

  return (
    <button
      type="button"
      onClick={toggleTheme}
      aria-label={isLight ? 'Switch to dark theme' : 'Switch to light theme'}
      title={isLight ? 'Switch to dark theme' : 'Switch to light theme'}
      className={`theme-switcher__btn group relative w-8 h-8 rounded-none flex items-center justify-center transition-colors duration-200 outline-none focus-visible:ring-2 cursor-pointer ${
        isLight
          ? 'bg-white hover:bg-black hover:text-white text-black border border-black focus-visible:ring-black'
          : 'bg-black hover:bg-white hover:text-black text-white border border-white focus-visible:ring-white'
      } ${className}`}
    >
      {isLight ? (
        <Sun size={14} className="text-current" />
      ) : (
        <Moon size={14} className="text-current" />
      )}
    </button>
  );
}
