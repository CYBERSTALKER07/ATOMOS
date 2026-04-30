'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export default function WarehouseLogin() {
  const router = useRouter();
  const [phone, setPhone] = useState('');
  const [pin, setPin] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await fetch(`${API}/v1/auth/warehouse/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone, pin }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || 'Login failed');
        return;
      }

      const data = await res.json();
      document.cookie = `warehouse_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      router.replace('/');
    } catch {
      setError('Network error. Check connection.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-background">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-sm mx-auto p-8 rounded-xl border border-[var(--border)]"
        style={{ background: 'var(--surface)' }}
      >
        <h1 className="text-xl font-bold tracking-tight mb-1">Warehouse Portal</h1>
        <p className="text-sm text-[var(--muted)] mb-6">Sign in with your phone and PIN</p>

        {error && (
          <div className="mb-4 p-3 rounded-lg text-sm" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}>
            {error}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">Phone Number</label>
            <input
              type="tel"
              value={phone}
              onChange={e => setPhone(e.target.value)}
              placeholder="+998..."
              required
              className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none transition-colors"
              style={{
                background: 'var(--field-background)',
                color: 'var(--field-foreground)',
                borderColor: 'var(--field-border)',
              }}
            />
          </div>

          <div>
            <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">PIN</label>
            <input
              type="password"
              value={pin}
              onChange={e => setPin(e.target.value)}
              placeholder="6+ digit PIN"
              required
              minLength={6}
              className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none transition-colors"
              style={{
                background: 'var(--field-background)',
                color: 'var(--field-foreground)',
                borderColor: 'var(--field-border)',
              }}
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2.5 rounded-lg text-sm font-semibold transition-opacity button--primary disabled:opacity-50"
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </div>
      </form>
    </div>
  );
}
