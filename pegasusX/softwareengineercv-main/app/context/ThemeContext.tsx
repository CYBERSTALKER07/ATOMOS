'use client';

import React, { createContext, useContext, useEffect, useState, useMemo, useCallback } from 'react';

export type Theme = 'system' | 'light' | 'dark';
export type ResolvedTheme = 'light' | 'dark';

interface ThemeContextType {
 theme: Theme;
 resolvedTheme: ResolvedTheme;
 setTheme: (theme: Theme) => void;
 toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

const THEME_STORAGE_KEY = 'pegasus-theme';

export function ThemeProvider({ children }: { children: React.ReactNode }) {
 const [theme, setThemeState] = useState<Theme>('system');
 const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>('dark');
 const [mounted, setMounted] = useState(false);

 // Initialize theme from localStorage or system on mount
 useEffect(() => {
 setMounted(true);
 try {
 const saved = localStorage.getItem(THEME_STORAGE_KEY) as Theme | null;
 if (saved && (saved === 'light' || saved === 'dark' || saved === 'system')) {
 setThemeState(saved);
 }
 } catch (e) {}
 }, []);

 // Compute resolved theme and update DOM classes
 useEffect(() => {
 if (!mounted) return;

 const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

 const computeTheme = (currentTheme: Theme): ResolvedTheme => {
 if (currentTheme === 'light') return 'light';
 if (currentTheme === 'dark') return 'dark';
 return mediaQuery.matches ? 'dark' : 'light';
 };

 const newResolved = computeTheme(theme);
 setResolvedTheme(newResolved);

 const root = document.documentElement;
 if (newResolved === 'light') {
 root.classList.add('light');
 root.classList.remove('dark');
 root.setAttribute('data-theme', 'light');
 } else {
 root.classList.add('dark');
 root.classList.remove('light');
 root.setAttribute('data-theme', 'dark');
 }

 const handleChange = () => {
 if (theme === 'system') {
 const updated = mediaQuery.matches ? 'dark' : 'light';
 setResolvedTheme(updated);
 if (updated === 'light') {
 root.classList.add('light');
 root.classList.remove('dark');
 root.setAttribute('data-theme', 'light');
 } else {
 root.classList.add('dark');
 root.classList.remove('light');
 root.setAttribute('data-theme', 'dark');
 }
 }
 };

 mediaQuery.addEventListener('change', handleChange);
 return () => mediaQuery.removeEventListener('change', handleChange);
 }, [theme, mounted]);

 const setTheme = useCallback((newTheme: Theme) => {
 setThemeState(newTheme);
 try {
 localStorage.setItem(THEME_STORAGE_KEY, newTheme);
 } catch (e) {}
 }, []);

 const toggleTheme = useCallback(() => {
 const next = resolvedTheme === 'dark' ? 'light' : 'dark';
 setTheme(next);
 }, [resolvedTheme, setTheme]);

 const value = useMemo(
 () => ({
 theme,
 resolvedTheme,
 setTheme,
 toggleTheme,
 }),
 [theme, resolvedTheme, setTheme, toggleTheme]
 );

 return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
 const context = useContext(ThemeContext);
 if (!context) {
 // Graceful fallback for non-wrapped components or SSR
 return {
 theme: 'dark' as Theme,
 resolvedTheme: 'dark' as ResolvedTheme,
 setTheme: () => {},
 toggleTheme: () => {},
 };
 }
 return context;
}
