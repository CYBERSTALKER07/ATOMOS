'use client';

import { PackChip } from '@pegasusx/ui-kit/pack';
import { fiscalReceiptLabel, useMarketPack } from '@pegasusx/api-client';
import { readTokenFromCookie, warehouseApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  const token = readTokenFromCookie();
  const { pack, session } = useMarketPack({ baseUrl: warehouseApiBaseUrl(), token });
  return (
    <PackChip
      currency={pack?.currency_code}
      receipts={fiscalReceiptLabel(pack?.fiscal_adapter)}
      title={[pack?.timezone, pack?.maps_adapter, session?.checkout_reads_this === false ? 'checkout_reads_this=false' : '']
        .filter(Boolean)
        .join(' · ')}
    />
  );
}
