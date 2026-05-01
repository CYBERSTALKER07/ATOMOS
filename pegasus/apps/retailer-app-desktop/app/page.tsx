'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { storeToken } from '../lib/bridge';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export default function Home() {
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const router = useRouter();

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
        const d = await res.json();
        throw new Error(d.error || 'Login failed');
      }

      const data = await res.json();
      
      // Store in cookie for Next.js routing/middleware
      document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      
      // Store in OS Keyring / LocalStorage
      await storeToken(data.token, data.refresh_token || '');

      // Store user profile for use across the app
      if (data.user) {
        localStorage.setItem('retailer_profile', JSON.stringify(data.user));
      }

      router.push('/dashboard');
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
