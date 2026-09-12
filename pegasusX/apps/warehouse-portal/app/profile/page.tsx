'use client';

import { usePortalT } from "@/lib/i18n";
import { useMemo } from 'react';
import Link from 'next/link';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { clearSession, readTokenFromCookie } from '@/lib/auth';

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

export default function WarehouseProfilePage() {
  const t = usePortalT();
  const session = useMemo(() => {
    const token = readTokenFromCookie();
    if (!token) return null;
    return decodeJwtPayload(token);
  }, []);

  const warehouseId = typeof session?.home_node_id === 'string' ? session.home_node_id : '';
  const subject = typeof session?.sub === 'string' ? session.sub : '';
  const role = typeof session?.role === 'string' ? session.role : 'WAREHOUSE';

  return (
    <PageTransition>
      <PageChrome
        icon="warehouse"
        title={t("portal.nav.profile")}
        description={t("warehouse_portal.residual.text.warehouse_operator_session_and_scope")}
      >
        <div className="max-w-xl space-y-4">
          <div className="md-card p-4 space-y-2">
            <p className="md-typescale-label-large" style={{ color: 'var(--color-md-on-surface-variant)' }}>{t("warehouse_portal.profile.text.signed_in_as")}</p>
            <p className="md-typescale-title-large">{subject || 'Warehouse operator'}</p>
            <p className="md-typescale-body-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              Role: {role}
            </p>
            {warehouseId ? (
              <p className="md-typescale-body-medium font-mono text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                Warehouse: {warehouseId}
              </p>
            ) : null}
          </div>

          <div className="flex flex-wrap gap-3">
            <Link href="/settings" className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2">
              Warehouse settings
            </Link>
            <button
              type="button"
              className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2"
              onClick={() => {
                clearSession();
                window.location.href = '/auth/login';
              }}
            >
              Sign out
            </button>
          </div>
        </div>
      </PageChrome>
    </PageTransition>
  );
}
