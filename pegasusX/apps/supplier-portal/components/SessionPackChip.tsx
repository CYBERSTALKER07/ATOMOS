'use client';

import { SessionPackChip as GenericSessionPackChip } from '@/components/ui';
import { readTokenFromCookie, supplierApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={supplierApiBaseUrl()} token={readTokenFromCookie()} />;
}
