'use client';
'use client';

import { useState, useCallback, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@heroui/react';
import { exchangeCustomToken } from '../../../lib/firebase';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export default function AdminLoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // Clear any stale auth cookies so the user can log in fresh
  useEffect(() => {
    document.cookie = 'pegasus_admin_jwt=; Max-Age=0; path=/';
    document.cookie = 'pegasus_supplier_jwt=; Max-Age=0; path=/';
  }, []);

  const handleLogin = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      const res = await fetch(`${API}/v1/auth/admin/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: 'Login failed' }));
        const errorMessage = body.error === 'rate_limit_exceeded' 
          ? 'Too many requests. Please try again later.' 
          : (body.error || `HTTP ${res.status}`);
        setError(errorMessage);
        setSubmitting(false);
        return;
      }

      const data = await res.json();
      document.cookie = `pegasus_admin_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      document.cookie = `admin_name=${encodeURIComponent(data.display_name || '')}; path=/; max-age=86400; SameSite=Lax`;
      // Exchange Firebase custom token for ID token session (graceful — legacy cookie still works)
      if (data.firebase_token) {
        await exchangeCustomToken(data.firebase_token);
      }
      router.push('/');
    } catch {
      setError('Network error \u2014 is the backend running?');
    } finally {
      setSubmitting(false);
    }
  }, [email, password, router]);

  return (
    <div className="w-full">
      {/* Brand — visible on mobile where the left panel is hidden */}
      <div className="flex items-center gap-3 mb-8 lg:hidden">
        <div
          className="w-10 h-10 rounded-xl flex items-center justify-center"
          style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
        >
          <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z"/>
          </svg>
        </div>
        <div>
          <h1 className="md-typescale-title-large" style={{ color: 'var(--foreground)' }}>
            Lab Hub
          </h1>
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
            Supplier Operations Portal
          </p>
        </div>
      </div>

      {/* Login Form */}
      <div className="py-2">
        <h2 className="md-typescale-headline-small mb-1" style={{ color: 'var(--foreground)' }}>
          Sign in
        </h2>
        <p className="md-typescale-body-small mb-6" style={{ color: 'var(--muted)' }}>
          Enter your credentials to access the portal
        </p>

        <form onSubmit={handleLogin} className="space-y-5">
          {error && (
            <div
              className="px-4 py-3 md-shape-lg md-typescale-body-small flex items-center gap-3"
              style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}
              role="alert"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" className="shrink-0">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
              </svg>
              {error}
            </div>
          )}

          <div>
            <label htmlFor="email" className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              placeholder="admin@void.pegasus.uz"
              required
              autoFocus
              className="md-input-outlined w-full"
            />
          </div>

          <div>
            <label htmlFor="password" className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>
              Password
            </label>
            <div className="relative">
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="Enter your password"
                required
                className="md-input-outlined w-full pr-12"
              />
              <button
                type="button"
                onClick={() => setShowPassword(v => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 rounded-full"
                style={{ color: 'var(--muted)' }}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 7c2.76 0 5 2.24 5 5 0 .65-.13 1.26-.36 1.83l2.92 2.92c1.51-1.26 2.7-2.89 3.43-4.75-1.73-4.39-6-7.5-11-7.5-1.4 0-2.74.25-3.98.7l2.16 2.16C10.74 7.13 11.35 7 12 7zM2 4.27l2.28 2.28.46.46C3.08 8.3 1.78 10.02 1 12c1.73 4.39 6 7.5 11 7.5 1.55 0 3.03-.3 4.38-.84l.42.42L19.73 22 21 20.73 3.27 3 2 4.27zM7.53 9.8l1.55 1.55c-.05.21-.08.43-.08.65 0 1.66 1.34 3 3 3 .22 0 .44-.03.65-.08l1.55 1.55c-.67.33-1.41.53-2.2.53-2.76 0-5-2.24-5-5 0-.79.2-1.53.53-2.2zm4.31-.78 3.15 3.15.02-.16c0-1.66-1.34-3-3-3l-.17.01z"/></svg>
                ) : (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/></svg>
                )}
              </button>
            </div>
          </div>

          <Button
            type="submit"
            variant="primary"
            fullWidth
            className="py-3"
            isDisabled={submitting}
          >
            {submitting ? (
              <span className="flex items-center justify-center gap-2">
                <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Authenticating...
              </span>
            ) : 'Sign In'}
          </Button>

          <p className="text-center md-typescale-body-small" style={{ color: 'var(--muted)' }}>
            Don&apos;t have an account?{' '}
            <a href="/auth/register" className="font-semibold hover:underline" style={{ color: 'var(--accent)' }}>
              Create account
            </a>
          </p>
        </form>
      </div>

      <p className="text-center mt-6 md-typescale-label-small lg:hidden" style={{ color: 'var(--muted)', opacity: 0.6 }}>
        Pegasus &copy; 2026
      </p>
    </div>
  );
}
