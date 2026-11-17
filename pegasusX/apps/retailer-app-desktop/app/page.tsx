'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { readToken } from '../lib/auth';
import { getStoredToken, isTauri } from '../lib/bridge';

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

function isTokenExpired(token: string): boolean {
  const payload = decodeJwtPayload(token);
  if (!payload || typeof payload.exp !== 'number') return true;
  return payload.exp * 1000 < Date.now();
}

export default function RootPage() {
  const router = useRouter();

  useEffect(() => {
    let active = true;

    async function checkAuthAndRedirect() {
      let token = readToken();
      if (!token && isTauri()) {
        try {
          const stored = await getStoredToken();
          if (stored) token = stored;
        } catch {
          // ignore keyring error
        }
      }

      if (!active) return;

      if (!token || isTokenExpired(token)) {
        router.replace('/auth/login');
        return;
      }

      const payload = decodeJwtPayload(token);
      if (payload && payload.is_configured === false) {
        router.replace('/setup/tax');
        return;
      }

      router.replace('/dashboard');
    }

    void checkAuthAndRedirect();

    return () => {
      active = false;
    };
  }, [router]);

  return null;
}
