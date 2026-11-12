'use client';

import { SessionPackChip as GenericSessionPackChip } from '@/components/ui/SessionPackChip';
import { readTokenFromCookie, factoryApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={factoryApiBaseUrl()} token={readTokenFromCookie()} />;
}
