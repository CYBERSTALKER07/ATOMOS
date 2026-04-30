'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { isTauri, storeToken } from '@/lib/bridge';
import { exchangeCustomToken } from '@/lib/firebase';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export default function FactoryLoginPage() {
  const router = useRouter();
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await fetch(`${API}/v1/auth/factory/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone, password }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Login failed (${res.status})`);
      }

      const data = await res.json();
      const token = data.token || data.access_token;

      if (!token) throw new Error('No token received');

      // Store in cookie
      document.cookie = `factory_jwt=${encodeURIComponent(token)}; path=/; max-age=86400; SameSite=Lax`;

      // Desktop: store in OS keyring
      if (isTauri()) {
        await storeToken(token, data.refresh_token || '').catch(() => {});
      }

      // Firebase custom token exchange (if provided)
      if (data.custom_token) {
        await exchangeCustomToken(data.custom_token).catch(() => {});
      }

      router.replace('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--background)] p-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold tracking-tight">Factory Portal</h1>
          <p className="text-sm text-[var(--muted)] mt-1">Sign in to access loading bay operations</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div
              className="px-4 py-3 rounded-lg text-sm"
              style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}
            >
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-1.5">Phone</label>
            <input
              type="tel"
              value={phone}
              onChange={e => setPhone(e.target.value)}
              placeholder="+998 XX XXX XX XX"
              required
              className="w-full px-3 py-2.5 rounded-lg text-sm border input-root outline-none focus:border-[var(--focus)] transition-colors"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5">Password</label>
            <input
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder="Enter password"
              required
              className="w-full px-3 py-2.5 rounded-lg text-sm border input-root outline-none focus:border-[var(--focus)] transition-colors"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2.5 rounded-lg text-sm font-semibold button--primary disabled:opacity-50 transition-colors"
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>

        <p className="text-center text-xs text-[var(--muted)] mt-6">
          Pegasus — Factory Operations
        </p>
      </div>
    </div>
  );
}
