'use client';

import { SessionPackChip as GenericSessionPackChip } from '@pegasusx/ui-kit/portal';
import { readTokenFromCookie, factoryApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={factoryApiBaseUrl()} token={readTokenFromCookie()} />;
}
