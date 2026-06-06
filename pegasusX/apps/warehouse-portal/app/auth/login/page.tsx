'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';

import { persistSession, warehouseApiBaseUrl } from '@/lib/auth';

const API = warehouseApiBaseUrl;

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
      persistSession(data.token, data.refresh_token);
      router.replace('/');
    } catch {
      setError('Network error. Check connection.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="auth-card space-y-5">
      <div>
        <h1 className="md-typescale-headline-medium" style={{ margin: 0 }}>
          Warehouse sign in
        </h1>
        <p className="desk-page-subtitle">Use your phone number and PIN.</p>
      </div>

      {error ? (
        <p className="md-typescale-body-small" style={{ color: 'var(--desk-danger)' }}>
          {error}
        </p>
      ) : null}

      <label className="block space-y-1">
        <span className="md-typescale-label-medium">Phone</span>
        <input
          type="tel"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          placeholder="+998..."
          required
          autoComplete="tel"
          className="md-input-outlined w-full"
        />
      </label>

      <label className="block space-y-1">
        <span className="md-typescale-label-medium">PIN</span>
        <input
          type="password"
          value={pin}
          onChange={(e) => setPin(e.target.value)}
          placeholder="6+ digit PIN"
          required
          minLength={6}
          autoComplete="current-password"
          className="md-input-outlined w-full"
        />
      </label>

      <button type="submit" disabled={loading} className="md-btn md-btn-filled w-full">
        {loading ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  );
}
