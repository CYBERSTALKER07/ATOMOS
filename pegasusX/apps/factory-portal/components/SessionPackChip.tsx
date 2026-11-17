'use client';

import { useEffect, useState } from 'react';
import { useMarketPack } from '@pegasusx/api-react';
import { fiscalReceiptLabel } from '@pegasusx/api-core';
import { PackChip } from './ui/PackChip';
import { readTokenFromCookie, factoryApiBaseUrl } from '@/lib/auth';

export function SessionPackChip({
  baseUrl,
  token,
}: {
  baseUrl?: string;
  token?: string;
} = {}) {
  const [mounted, setMounted] = useState(false);
  const resolvedBaseUrl = baseUrl ?? factoryApiBaseUrl();
  const resolvedToken = token ?? readTokenFromCookie();
  const { pack, session } = useMarketPack({ baseUrl: resolvedBaseUrl, token: resolvedToken });

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) return null;

  return (
    <PackChip
      currency={pack?.currency_code}
      receipts={fiscalReceiptLabel(pack?.fiscal_adapter)}
      title={[
        pack?.timezone,
        pack?.maps_adapter,
        session?.checkout_reads_this === false ? 'checkout_reads_this=false' : '',
      ]
        .filter(Boolean)
        .join(' · ')}
    />
  );
}
