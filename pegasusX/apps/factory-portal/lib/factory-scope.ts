import { readTokenFromCookie } from '@/lib/auth';

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(payload) as Record<string, unknown>;
  } catch {
    return null;
  }
}

/** Factory operator id from JWT (used for deterministic idempotency keys). */
export function factoryOperatorId(): string {
  const token = readTokenFromCookie();
  if (!token) return '';
  const session = decodeJwtPayload(token);
  if (typeof session?.factory_id === 'string' && session.factory_id) {
    return session.factory_id;
  }
  if (typeof session?.sub === 'string') {
    return session.sub;
  }
  return '';
}
