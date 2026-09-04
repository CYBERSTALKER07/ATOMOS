'use client';

import { SessionPackChip as GenericSessionPackChip } from '@pegasusx/ui-kit/portal';
import { readTokenFromCookie, warehouseApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={warehouseApiBaseUrl()} token={readTokenFromCookie()} />;
}
