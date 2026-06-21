'use client';

import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useState, type ReactNode } from 'react';

export type ThemeMode = 'system' | 'light' | 'dark';

type ThemeContextValue = {
  mode: ThemeMode;
  resolved: 'light' | 'dark';
  setMode: (mode: ThemeMode) => void;
  cycle: () => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

const STORAGE_KEY = 'pegasusx-supplier-theme';
const CYCLE_ORDER: ThemeMode[] = ['system', 'light', 'dark'];

function getSystemPreference(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function getStoredMode(): ThemeMode {
  if (typeof window === 'undefined') return 'system';
  const stored = localStorage.getItem(STORAGE_KEY) as ThemeMode | null;
  return stored && CYCLE_ORDER.includes(stored) ? stored : 'system';
}

function resolveMode(mode: ThemeMode): 'light' | 'dark' {
  return mode === 'system' ? getSystemPreference() : mode;
}

function applyTheme(resolved: 'light' | 'dark') {
  const root = document.documentElement;
  root.classList.toggle('dark', resolved === 'dark');
  root.style.colorScheme = resolved;
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>('system');
  const [resolved, setResolved] = useState<'light' | 'dark'>('light');
  const [mounted, setMounted] = useState(false);

  useLayoutEffect(() => {
    const stored = getStoredMode();
    const effective = resolveMode(stored);
    setModeState(stored);
    setResolved(effective);
    applyTheme(effective);
    setMounted(true);
    document.documentElement.setAttribute('data-hydrated', '');

    if (typeof window !== 'undefined' && (window as unknown as Record<string, unknown>).__TAURI_INTERNALS__) {
      document.documentElement.setAttribute('data-tauri', '');
    }
  }, []);

  useEffect(() => {
    if (!mounted) return;

    const effective = resolveMode(mode);
    setResolved(effective);
    applyTheme(effective);

    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => {
      if (mode === 'system') {
        const next = resolveMode('system');
        setResolved(next);
        applyTheme(next);
      }
    };
    media.addEventListener('change', handler);
    return () => media.removeEventListener('change', handler);
  }, [mode, mounted]);

  const setMode = useCallback((next: ThemeMode) => {
    const effective = resolveMode(next);
    setModeState(next);
    setResolved(effective);
    applyTheme(effective);
    localStorage.setItem(STORAGE_KEY, next);
  }, []);

  const cycle = useCallback(() => {
    setModeState((current) => {
      const index = CYCLE_ORDER.indexOf(current);
      const next = CYCLE_ORDER[(index + 1) % CYCLE_ORDER.length];
      localStorage.setItem(STORAGE_KEY, next);
      return next;
    });
  }, []);

  return (
    <ThemeContext.Provider value={{ mode, resolved, setMode, cycle }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return ctx;
}
