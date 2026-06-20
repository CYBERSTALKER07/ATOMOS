'use client';

import { useCallback, useEffect, useState } from 'react';
import { useTheme } from '@/components/ThemeProvider';

// ─── Theme toggle icon (sun/moon) ──────────────────────────────────────────
function ThemeToggle({
  isDark,
  onToggle,
  ariaLabel,
}: {
  isDark: boolean;
  onToggle: () => void;
  ariaLabel: string;
}) {
  return (
    <button
      onClick={onToggle}
      className="auth-theme-toggle"
      aria-label={ariaLabel}
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

function BrandLogo({ size = 96 }: { size?: number }) {
  return (
    <div
      className="flex items-center justify-center rounded-2xl"
      style={{
        width: size,
        height: size,
        background: 'var(--desk-accent, #c4a574)',
        color: 'var(--desk-accent-on, #1a1208)',
      }}
      aria-hidden
    >
      <svg width={size * 0.45} height={size * 0.45} viewBox="0 0 24 24" fill="currentColor">
        <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z" />
      </svg>
    </div>
  );
}

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  const { resolved, setMode } = useTheme();
  const [mounted, setMounted] = useState(false);
  const isDark = resolved === 'dark';
  const [splashDone, setSplashDone] = useState(false);

  useEffect(() => {
    setMounted(true);
    if (sessionStorage.getItem('auth-splash-shown') === '1') {
      setSplashDone(true);
      return;
    }
    const timer = setTimeout(() => {
      setSplashDone(true);
      sessionStorage.setItem('auth-splash-shown', '1');
    }, 240);
    return () => clearTimeout(timer);
  }, []);

  const toggleTheme = useCallback(() => {
    setMode(isDark ? 'light' : 'dark');
  }, [isDark, setMode]);

  return (
    <div className={`auth-shell ${isDark ? 'auth-dark' : 'auth-light'}`}>
      {/* ── Splash Screen ── */}
      {!splashDone && (
        <div className={`auth-splash ${mounted ? 'auth-splash-exit' : ''}`}>
          <BrandLogo size={96} />
        </div>
      )}

      {/* ── Left: Branding Panel ── */}
      <div className="auth-brand-panel">
        <div className={`auth-brand-content relative z-10 ${mounted ? 'auth-brand-enter' : ''}`}>
          <div className="auth-brand-logo">
            <BrandLogo size={160} />
          </div>
        </div>

        <p className="auth-brand-footer relative z-10">
          pegasusX &copy; 2026
        </p>
      </div>

      {/* ── Right: Form Panel ── */}
      <div className="auth-form-panel">
        <div className="flex items-center justify-end pt-4 pr-6 px-6 shrink-0 relative z-10">
          <ThemeToggle
            isDark={isDark}
            onToggle={toggleTheme}
            ariaLabel={
              isDark
                ? 'Switch to light mode'
                : 'Switch to dark mode'
            }
          />
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
