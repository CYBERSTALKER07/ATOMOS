'use client';

import Image from 'next/image';
import { usePathname } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

// ─── Theme toggle icon (sun/moon) ──────────────────────────────────────────
function ThemeToggle({ isDark, onToggle }: { isDark: boolean; onToggle: () => void }) {
  return (
    <button
      onClick={onToggle}
      className="auth-theme-toggle"
      aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
    >
      {isDark ? (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="5" />
          <line x1="12" y1="1" x2="12" y2="3" />
          <line x1="12" y1="21" x2="12" y2="23" />
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
          <line x1="1" y1="12" x2="3" y2="12" />
          <line x1="21" y1="12" x2="23" y2="12" />
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
        </svg>
      ) : (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
      )}
    </button>
  );
}

const taglines: Record<string, { heading: string; sub: string }> = {
  '/auth/login': {
    heading: '',
    sub: 'Your logistics ecosystem is waiting.',
  },
  '/auth/register': {
    heading: 'Join the\nnetwork.',
    sub: 'Set up your supplier operations in minutes.',
  },
};

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const t = taglines[pathname] ?? taglines['/auth/login'];
  const [mounted, setMounted] = useState(false);
  const [isDark, setIsDark] = useState(true);
  const [splashDone, setSplashDone] = useState(() => {
    if (typeof window !== 'undefined') {
      return sessionStorage.getItem('auth-splash-shown') === '1';
    }
    return false;
  });

  useEffect(() => {
    setMounted(true);
    const saved = localStorage.getItem('auth-theme');
    if (saved) setIsDark(saved === 'dark');
    if (!splashDone) {
      const timer = setTimeout(() => {
        setSplashDone(true);
        sessionStorage.setItem('auth-splash-shown', '1');
      }, 1600);
      return () => clearTimeout(timer);
    }
  }, []);

  const toggleTheme = useCallback(() => {
    setIsDark(prev => {
      const next = !prev;
      localStorage.setItem('auth-theme', next ? 'dark' : 'light');
      return next;
    });
  }, []);

  return (
    <div className={`auth-shell ${isDark ? 'auth-dark' : 'auth-light'}`}>
      {/* ── Splash Screen ── */}
      {!splashDone && (
        <div className={`auth-splash ${mounted ? 'auth-splash-exit' : ''}`}>
          <Image
            className="auth-splash-logo"
            src={isDark ? '/logo-dark-square.png' : '/logo-light-square.png'}
            alt=""
            width={96}
            height={96}
            priority
          />
        </div>
      )}

      {/* ── Left: Branding Panel ── */}
      <div className="auth-brand-panel">
        <div className={`auth-brand-content relative z-10 ${mounted ? 'auth-brand-enter' : ''}`}>
          <div className="auth-brand-logo">
            <Image
              src={isDark ? '/logo-light-square.png' : '/logo-dark-square.png'}
              alt="Pegasus"
              width={500}
              height={500}
              priority
              className="auth-brand-logo-img"
            />
          </div>
        </div>

        <p className="auth-brand-footer relative z-10">
          Pegasus &copy; 2026
        </p>
      </div>

      {/* ── Right: Form Panel ── */}
      <div className="auth-form-panel">
        <div className="flex items-center justify-end pt-4 pr-6 px-6 shrink-0 relative z-10">
          <ThemeToggle isDark={isDark} onToggle={toggleTheme} />
        </div>
        <div className="flex-1 overflow-y-auto min-h-0">
          <div className="flex flex-col items-center py-8 px-6 relative z-10">
            <div className={`auth-form-inner w-full ${mounted ? 'auth-form-enter' : ''}`}>
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
