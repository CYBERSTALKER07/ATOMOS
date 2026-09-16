'use client';

import React, { useState, useEffect } from 'react';
import { Sun, Moon } from 'lucide-react';
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
        className={`w-8 h-8 rounded-full flex items-center justify-center border border-white/10 bg-white/5 text-white/40 ${className}`}
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
      className={`relative w-8 h-8 rounded-full flex items-center justify-center transition-all duration-200 outline-none focus-visible:ring-2 ${
        isLight
          ? 'bg-black/5 hover:bg-black/10 text-zinc-800 border border-black/10 focus-visible:ring-black'
          : 'bg-white/5 hover:bg-white/10 text-zinc-200 border border-white/15 focus-visible:ring-white'
      } ${className}`}
    >
      {isLight ? (
        <Sun className="w-3.5 h-3.5 text-amber-600 transition-transform hover:rotate-45 duration-300" />
      ) : (
        <Moon className="w-3.5 h-3.5 text-cyan-400 transition-transform hover:-rotate-12 duration-300" />
      )}
    </button>
  );
}
