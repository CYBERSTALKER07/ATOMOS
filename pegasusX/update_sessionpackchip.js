const fs = require('fs');
const glob = require('glob');

const files = glob.sync('apps/*/components/SessionPackChip.tsx');
files.forEach(f => {
  let content = fs.readFileSync(f, 'utf8');
  let newContent = '';
  if (f.includes('factory-portal')) {
    newContent = \`'use client';
import { SessionPackChip as GenericSessionPackChip } from '@pegasusx/ui-kit/portal';
import { readTokenFromCookie, factoryApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={factoryApiBaseUrl()} token={readTokenFromCookie()} />;
}
\`;
  } else if (f.includes('warehouse-portal')) {
    newContent = \`'use client';
import { SessionPackChip as GenericSessionPackChip } from '@pegasusx/ui-kit/portal';
import { readTokenFromCookie, warehouseApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={warehouseApiBaseUrl()} token={readTokenFromCookie()} />;
}
\`;
  } else if (f.includes('supplier-portal')) {
    newContent = \`'use client';
import { SessionPackChip as GenericSessionPackChip } from '@pegasusx/ui-kit/portal';
import { readTokenFromCookie, supplierApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={supplierApiBaseUrl()} token={readTokenFromCookie()} />;
}
\`;
  } else if (f.includes('retailer-app-desktop')) {
    newContent = \`'use client';
import { SessionPackChip as GenericSessionPackChip } from '@pegasusx/ui-kit/portal';
import { readTokenFromCookie, retailStoreApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={retailStoreApiBaseUrl()} token={readTokenFromCookie()} />;
}
\`;
  }

  if (newContent) {
    fs.writeFileSync(f, newContent);
  }
});
