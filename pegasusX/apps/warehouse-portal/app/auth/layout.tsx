"use client";

import { useCallback } from "react";
import { useTheme } from "@/components/ThemeProvider";

function ThemeToggle({
  isDark,
  onToggle,
}: {
  isDark: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="auth-theme-toggle"
      aria-label={isDark ? "Switch to light mode" : "Switch to dark mode"}
    >
      {isDark ? (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
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
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
      )}
    </button>
  );
}

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  const { resolved, setMode } = useTheme();
  const isDark = resolved === "dark";

  const toggleTheme = useCallback(() => {
    setMode(isDark ? "light" : "dark");
  }, [isDark, setMode]);

  return (
    <div className="auth-shell">
      <div className="auth-brand-panel">
        <div className="auth-brand-content">
          <div className="text-center">
            <div className="auth-brand-mark">W</div>
            <h1 className="auth-brand-heading">pegasusX Warehouse</h1>
            <p className="auth-brand-sub">
              Node-scoped dispatch, inventory, and supply operations for warehouse administrators.
            </p>
          </div>
        </div>
        <p className="auth-brand-footer">pegasusX © 2026</p>
      </div>

      <div className="auth-form-panel">
        <div className="flex items-center justify-end pt-4 pr-6 px-6 shrink-0">
          <ThemeToggle isDark={isDark} onToggle={toggleTheme} />
        </div>
        <div className="flex-1 overflow-y-auto min-h-0">
          <div className="flex flex-col items-center py-8 px-6">
            <div className="auth-mobile-brand">
              <div className="auth-brand-mark" style={{ width: 72, height: 72, fontSize: 32 }}>
                W
              </div>
              <h1 className="md-typescale-title-large">pegasusX Warehouse</h1>
            </div>
            <div className="auth-form-inner">{children}</div>
          </div>
        </div>
      </div>
    </div>
  );
}
