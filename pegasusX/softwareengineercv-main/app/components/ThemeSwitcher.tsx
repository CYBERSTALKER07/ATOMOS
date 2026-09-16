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
        className={`w-8 h-8 rounded-none flex items-center justify-center border border-white/20 bg-black text-white/40 ${className}`}
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
      className={`relative w-8 h-8 rounded-none flex items-center justify-center transition-all duration-200 outline-none focus-visible:ring-2 cursor-pointer ${
        isLight
          ? 'bg-white hover:bg-black hover:text-white text-black border border-black focus-visible:ring-black'
          : 'bg-black hover:bg-white hover:text-black text-white border border-white focus-visible:ring-white'
      } ${className}`}
    >
      {isLight ? (
        <Sun className="w-3.5 h-3.5 text-current transition-transform hover:rotate-45 duration-300" />
      ) : (
        <Moon className="w-3.5 h-3.5 text-current transition-transform hover:-rotate-12 duration-300" />
      )}
    </button>
  );
}
