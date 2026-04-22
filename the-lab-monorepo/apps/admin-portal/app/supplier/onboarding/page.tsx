/**
 * Onboarding is now fully integrated into the supplier registration wizard at /auth/register.
 * This page simply redirects to the dashboard (or back to register if unauthenticated).
 * Profile settings (category changes, payment gateway, business details) are managed at /supplier/profile.
 */
'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { readTokenFromCookie as getToken } from '@/lib/auth';

export default function SupplierOnboardingRedirect() {
  const router = useRouter();
  useEffect(() => {
    const token = getToken();
    if (!token) { router.replace('/auth/register'); return; }
    router.replace('/supplier/dashboard');
  }, [router]);
  return (
    <div className="min-h-screen flex items-center justify-center" style={{ background: 'var(--background)' }}>
      <div className="flex flex-col items-center gap-4">
        <svg className="animate-spin h-8 w-8" viewBox="0 0 24 24" fill="none" style={{ color: 'var(--accent)' }}>
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        <p className="text-sm" style={{ color: 'var(--muted)' }}>Redirecting to dashboard…</p>
      </div>
    </div>
  );
}

