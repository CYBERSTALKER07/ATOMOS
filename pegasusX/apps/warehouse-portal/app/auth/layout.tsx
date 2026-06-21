'use client';

import { useCallback, useEffect, useState } from 'react';
import { useTheme } from '@/components/ThemeProvider';

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
      type="button"
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

function BrandMark({ size = 96 }: { size?: number }) {
  return (
    <div className="wh-auth-mark" style={{ width: size, height: size }} aria-hidden>
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
    <div className={`auth-shell wh-auth-shell ${isDark ? 'auth-dark' : 'auth-light'}`}>
      {!splashDone && (
        <div className={`auth-splash ${mounted ? 'auth-splash-exit' : ''}`}>
          <BrandMark size={96} />
        </div>
      )}

      <aside className="auth-brand-panel wh-auth-brand" aria-label="pegasusX Warehouse">
        <div className={`auth-brand-content wh-auth-brand-content ${mounted ? 'auth-brand-enter' : ''}`}>
          <BrandMark size={88} />
          <div className="wh-auth-brand-copy">
            <p className="wh-auth-brand-eyebrow">pegasusX</p>
            <h1 className="wh-auth-brand-title">Warehouse node</h1>
            <p className="wh-auth-brand-sub">
              Dispatch, inventory, and fleet control for your depot — one account, one node.
            </p>
          </div>
        </div>
        <p className="auth-brand-footer wh-auth-brand-footer">pegasusX &copy; 2026</p>
      </aside>

      <div className="auth-form-panel wh-auth-form">
        <div className="wh-auth-form-top">
          <p className="wh-auth-form-kicker lg:hidden">pegasusX Warehouse</p>
          <ThemeToggle
            isDark={isDark}
            onToggle={toggleTheme}
            ariaLabel={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
          />
        </div>
        <div className="wh-auth-form-scroll">
          <div className={`auth-form-inner wh-auth-form-inner ${mounted ? 'auth-form-enter' : ''}`}>
            {children}
          </div>
        </div>
      </div>
    </div>
  );
}
