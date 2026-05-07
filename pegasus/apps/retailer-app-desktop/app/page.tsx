'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { getStoredToken, isTauri, storeToken } from '../lib/bridge';
import { readToken } from '../lib/auth';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

type LoginErrorBody = {
  error?: string;
  detail?: string;
  title?: string;
  message?: string;
};

type LoginSuccessBody = {
  token?: string;
  refresh_token?: string;
  user?: unknown;
};

async function parseLoginError(res: Response): Promise<string> {
  const contentType = (res.headers.get('content-type') || '').toLowerCase();
  if (contentType.includes('application/json') || contentType.includes('application/problem+json')) {
    const payload = (await res.json().catch(() => null)) as LoginErrorBody | null;
    if (payload) {
      return payload.error || payload.detail || payload.title || payload.message || `Login failed (${res.status})`;
    }
  }

  const raw = (await res.text().catch(() => '')).trim();
  return raw || `Login failed (${res.status})`;
}

export default function Home() {
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const router = useRouter();

  useEffect(() => {
    let cancelled = false;

    const restoreSession = async () => {
      const cookieToken = readToken();
      if (cookieToken) {
        router.replace('/dashboard');
        return;
      }

      if (!isTauri()) return;

      const storedToken = await getStoredToken();
      if (!storedToken || cancelled) return;

      document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(storedToken)}; path=/; max-age=86400; SameSite=Lax`;
      router.replace('/dashboard');
    };

    void restoreSession();
    return () => {
      cancelled = true;
    };
  }, [router]);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const res = await fetch(`${API}/v1/auth/retailer/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone_number: phone, password }),
      });

      if (!res.ok) {
        throw new Error(await parseLoginError(res));
      }

      const data = (await res.json().catch(() => null)) as LoginSuccessBody | null;
      if (!data?.token) {
        throw new Error('Login response is missing token');
      }
      
      // Store in cookie for Next.js routing/middleware
      document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      
      // Store in OS Keyring / LocalStorage
      await storeToken(data.token, data.refresh_token || '');

      // Store user profile for use across the app
      if (data.user) {
        localStorage.setItem('retailer_profile', JSON.stringify(data.user));
      }

      router.replace('/dashboard');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="min-h-screen p-8 flex flex-col items-center justify-center" style={{ background: 'var(--surface)' }}>
      <div className="rounded-2xl shadow-lg p-8 max-w-md w-full border border-[var(--border)]" style={{ backgroundColor: 'var(--background)' }}>
        <h1 className="md-typescale-display-small mb-2 text-center font-bold" style={{ color: 'var(--accent)' }}>Retailer Portal</h1>
        <p className="md-typescale-body-large text-muted mb-8 text-center">
          Sign in to your account
        </p>
        
        {error && (
          <div className="p-3 mb-4 rounded-lg text-sm" style={{ background: 'rgba(220,38,38,0.08)', color: 'var(--danger)' }}>
            {error}
          </div>
        )}

        <form onSubmit={handleLogin} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <label className="md-typescale-label-medium text-foreground">Phone Number</label>
            <input 
              type="tel" 
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="+998901234567"
              className="p-3 w-full border rounded-lg border-[var(--border)] focus:outline-none focus:border-[var(--accent)] bg-transparent text-foreground"
              required
            />
          </div>
          
          <div className="flex flex-col gap-1 mb-4">
            <label className="md-typescale-label-medium text-foreground">Password</label>
            <input 
              type="password" 
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="p-3 w-full border rounded-lg border-[var(--border)] focus:outline-none focus:border-[var(--accent)] bg-transparent text-foreground"
              required
            />
          </div>

          <button 
            type="submit" 
            disabled={loading}
            className="md-typescale-label-large px-6 py-3 w-full flex justify-center disabled:opacity-50 rounded-full font-bold transition-opacity hover:opacity-90"
            style={{ backgroundColor: 'var(--accent)', color: 'var(--accent-foreground)' }}
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </main>
  );
}
