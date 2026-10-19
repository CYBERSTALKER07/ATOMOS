'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { createWarehouseApi } from '@/lib/api';
import { catalogOmits, gatewayLabel } from '@/lib/coverage';
import { useWarehousePaymentCatalog } from '@/lib/use-payment-catalog';
import type { PSPListing } from '@pegasusx/types';

const api = createWarehouseApi();

export default function PaymentConfigPage() {
  const t = usePortalT();
  const { currency, catalog: hookCatalog } = useWarehousePaymentCatalog();
  const [catalog, setCatalog] = useState<PSPListing[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await api.getWarehouseOpsPaymentConfig();
      setCatalog(data.catalog ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_payment_config_failed");
      setCatalog(hookCatalog);
    } finally {
      setLoading(false);
    }
  }, [hookCatalog]);

  useEffect(() => { void load(); }, [load]);

  const rows = catalog.length ? catalog : hookCatalog;
  const packCurrency = currency || "";

  return (
    <PageTransition>
      <PageChrome
        icon="payment"
        title={t("portal.nav.payment_config")}
        description={t("warehouse_portal.residual.text.read_only_view_payment_gateways_are_configured_by_the_supplier_a")}
        actions={
          <button type="button" onClick={() => { setLoading(true); void load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <div className="space-y-4">
      {packCurrency ? (
        <p className="text-sm text-[var(--muted)]">Pack currency {packCurrency} (read-only).</p>
      ) : null}
      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 3 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : error ? (
        <p className="text-sm text-[var(--danger)]">{error}</p>
      ) : rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="payment" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">{t("warehouse_portal.payment_config.text.no_payment_gateways_configured")}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {rows.map(listing => (
            <div key={listing.code} className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-semibold">{gatewayLabel(listing.code)}</h3>
                <span className={`status-chip ${listing.selectable ? 'status-chip--stable' : 'status-chip--draft'}`}>
                  {listing.selectable ? 'Selectable' : listing.status}
                </span>
              </div>
              <div className="space-y-1 text-xs text-[var(--muted)]">
                <div className="flex justify-between">
                  <span>{t("warehouse_portal.payment_config.text.provider")}</span>
                  <span className="font-medium text-[var(--foreground)]">{listing.code}</span>
                </div>
                <div className="flex justify-between">
                  <span>{t("warehouse_portal.payment_config.text.mode")}</span>
                  <span className="font-mono">{listing.status}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
      {catalogOmits(rows, ["STRIPE", "ADYEN"]) ? (
        <p className="text-xs text-[var(--muted)]">This pack does not list Stripe or Adyen.</p>
      ) : null}
        </div>
      </PageChrome>
    </PageTransition>
  );
}
