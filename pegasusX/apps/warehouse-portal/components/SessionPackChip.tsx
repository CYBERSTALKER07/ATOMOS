'use client';

import { SessionPackChip as GenericSessionPackChip } from '@/components/ui';
import { readTokenFromCookie, warehouseApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={warehouseApiBaseUrl()} token={readTokenFromCookie()} />;
}
