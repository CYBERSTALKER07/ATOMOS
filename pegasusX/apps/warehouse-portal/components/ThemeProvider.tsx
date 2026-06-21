'use client';

import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useState, type ReactNode } from 'react';

export type ThemeMode = 'system' | 'light' | 'dark';

interface ThemeCtx {
  mode: ThemeMode;
  resolved: 'light' | 'dark';
  setMode: (m: ThemeMode) => void;
  cycle: () => void;
}

const ThemeContext = createContext<ThemeCtx>({
  mode: 'system',
  resolved: 'light',
  setMode: () => {},
  cycle: () => {},
});

export const useTheme = () => useContext(ThemeContext);

const STORAGE_KEY = 'pegasus-warehouse-theme-mode';
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
  if (resolved === 'dark') {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
  root.style.colorScheme = resolved;
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>(() => getStoredMode());
  const [resolved, setResolved] = useState<'light' | 'dark'>(() => resolveMode(getStoredMode()));
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

    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => {
      if (mode === 'system') {
        const next = resolveMode('system');
        setResolved(next);
        applyTheme(next);
      }
    };
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [mode, mounted]);

  const setMode = useCallback((m: ThemeMode) => {
    const effective = resolveMode(m);
    setModeState(m);
    setResolved(effective);
    applyTheme(effective);
    localStorage.setItem(STORAGE_KEY, m);
  }, []);

  const cycle = useCallback(() => {
    setModeState(prev => {
      const idx = CYCLE_ORDER.indexOf(prev);
      const next = CYCLE_ORDER[(idx + 1) % CYCLE_ORDER.length];
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
