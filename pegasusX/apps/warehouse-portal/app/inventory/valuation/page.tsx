'use client';

import Link from 'next/link';
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { createWarehouseApi } from '@/lib/api';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import {
  createWarehouseValuationApi,
  extractValuationError,
  formatWarehouseMinor,
  parseMajorToSafeMinor,
  type InventoryCostLayer,
  type InventoryMovement,
  type InventoryValuationResponse,
  type SkuValuationSummary,
} from '@/lib/valuation-api';
import { usePortalT } from '@/lib/i18n';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PortalInput, PortalSection, PortalSelect } from '@/components/portal';
import Icon from '@/components/Icon';

type ValuationTab = 'layers' | 'skus' | 'movements';

const MOVEMENT_TYPES = [
  'ALL',
  'RECEIPT',
  'ISSUE_COGS',
  'ADJUSTMENT_GAIN',
  'ADJUSTMENT_SHRINK',
  'RETURN',
] as const;

function normalizeTab(raw: string | null): ValuationTab {
  switch ((raw || '').trim().toLowerCase()) {
    case 'skus':
      return 'skus';
    case 'movements':
      return 'movements';
    default:
      return 'layers';
  }
}

function formatIso(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

function formatSignedQty(delta: number): string {
  return delta > 0 ? `+${delta}` : String(delta);
}

function movementChipStyle(movementType: string): React.CSSProperties {
  switch (movementType.toUpperCase()) {
    case 'RECEIPT':
    case 'ADJUSTMENT_GAIN':
    case 'RETURN':
      return {
        background: 'var(--desk-success-soft)',
        color: 'var(--desk-success)',
        borderColor: 'var(--desk-border-strong)',
      };
    case 'ADJUSTMENT_SHRINK':
      return {
        background: 'var(--desk-danger-soft)',
        color: 'var(--desk-danger)',
        borderColor: 'var(--desk-border-strong)',
      };
    default:
      return {
        background: 'var(--desk-accent-soft)',
        color: 'var(--desk-accent-strong)',
        borderColor: 'var(--desk-border)',
      };
  }
}

export default function WarehouseInventoryValuationPage() {
  return (
    <Suspense fallback={null}>
      <WarehouseInventoryValuationInner />
    </Suspense>
  );
}

function WarehouseInventoryValuationInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const valuationApi = useMemo(() => createWarehouseValuationApi(createWarehouseApi()), []);

  const initialTab = normalizeTab(searchParams.get('tab'));
  const initialWarehouse = (searchParams.get('warehouse_id') || warehouseHomeNodeId() || '').trim();
  const initialProduct = (searchParams.get('product_id') || '').trim();

  const [activeTab, setActiveTab] = useState<ValuationTab>(initialTab);
  const [warehouseFilter, setWarehouseFilter] = useState<string>(initialWarehouse);
  const [productFilter, setProductFilter] = useState<string>(initialProduct);
  const [expiryRiskDays, setExpiryRiskDays] = useState<string>('14');
  const [movementTypeFilter, setMovementTypeFilter] = useState<string>('ALL');

  const [valuation, setValuation] = useState<InventoryValuationResponse | null>(null);
  const [movements, setMovements] = useState<InventoryMovement[]>([]);

  // Receipt form state
  const [showReceiptForm, setShowReceiptForm] = useState<boolean>(false);
  const [rcptWarehouseId, setRcptWarehouseId] = useState<string>(initialWarehouse);
  const [rcptProductId, setRcptProductId] = useState<string>('');
  const [rcptLotId, setRcptLotId] = useState<string>('');
  const [rcptRefId, setRcptRefId] = useState<string>('');
  const [rcptQuantity, setRcptQuantity] = useState<string>('');
  const [rcptUnitCostMajor, setRcptUnitCostMajor] = useState<string>('');
  const [rcptVatMajor, setRcptVatMajor] = useState<string>('0');
  const [rcptCreditAccount, setRcptCreditAccount] = useState<string>('AP:VENDOR');
  const [rcptExpiryDate, setRcptExpiryDate] = useState<string>('');
  const [rcptCurrency, setRcptCurrency] = useState<string>('UZS');

  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    const paramTab = searchParams.get('tab');
    if (paramTab) setActiveTab(normalizeTab(paramTab));
  }, [searchParams]);

  const loadAll = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      try {
        const riskDaysNum = Number.parseInt(expiryRiskDays, 10);
        const [valRes, movRes] = await Promise.all([
          valuationApi.getValuation({
            warehouse_id: warehouseFilter.trim() || undefined,
            product_id: productFilter.trim() || undefined,
            expiry_risk_days: Number.isInteger(riskDaysNum) && riskDaysNum > 0 ? riskDaysNum : 14,
          }),
          valuationApi.listMovements({
            warehouse_id: warehouseFilter.trim() || undefined,
            product_id: productFilter.trim() || undefined,
            movement_type: movementTypeFilter !== 'ALL' ? movementTypeFilter : undefined,
            limit: 100,
          }),
        ]);
        setValuation(valRes);
        setMovements(movRes.movements ?? []);
      } catch (err) {
        setError(extractValuationError(err, tRef.current('warehouse_portal.valuation.load_failed')));
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [expiryRiskDays, movementTypeFilter, productFilter, valuationApi, warehouseFilter],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useWarehouseSessionReconcile(() => {
    void loadAll(true);
  });

  const handleRecordReceipt = async (e: React.FormEvent) => {
    e.preventDefault();
    const wh = rcptWarehouseId.trim() || warehouseFilter.trim();
    const pid = rcptProductId.trim();
    const ref = rcptRefId.trim();
    const qty = Number.parseInt(rcptQuantity, 10);
    const unitCostMinor = parseMajorToSafeMinor(rcptUnitCostMajor, false);
    const vatMinor = rcptVatMajor.trim() ? parseMajorToSafeMinor(rcptVatMajor.trim(), false) : 0;

    if (
      !wh ||
      !pid ||
      !ref ||
      !Number.isInteger(qty) ||
      qty <= 0 ||
      unitCostMinor === null ||
      unitCostMinor <= 0 ||
      vatMinor === null
    ) {
      setError(t('warehouse_portal.valuation.receipt_fields_invalid'));
      return;
    }

    setSubmitting(true);
    setError(null);
    setNotice(null);
    try {
      const res = await valuationApi.recordCostLayerReceipt({
        warehouse_id: wh,
        product_id: pid,
        lot_id: rcptLotId.trim() || undefined,
        receipt_ref_type: 'PURCHASE_RECEIPT',
        receipt_ref_id: ref,
        quantity: qty,
        unit_cost_minor: unitCostMinor,
        vat_minor: vatMinor,
        credit_account: rcptCreditAccount,
        currency: rcptCurrency.trim() || 'UZS',
        expiry_date: rcptExpiryDate.trim() || undefined,
      });
      setNotice(
        t('warehouse_portal.valuation.receipt_recorded', {
          layerId: res.layer.layer_id,
          glEntryId: res.gl_entry_id || '—',
        }),
      );
      setRcptProductId('');
      setRcptLotId('');
      setRcptRefId('');
      setRcptQuantity('');
      setRcptUnitCostMajor('');
      setRcptVatMajor('0');
      setShowReceiptForm(false);
      await loadAll(true);
    } catch (err) {
      setError(extractValuationError(err, t('warehouse_portal.valuation.load_failed')));
    } finally {
      setSubmitting(false);
    }
  };

  const layers: InventoryCostLayer[] = valuation?.layers ?? [];
  const skus: SkuValuationSummary[] = valuation?.skus ?? [];
  const currency = valuation?.currency || 'UZS';
  const nearExpiryCount = useMemo(
    () => layers.filter((layer) => layer.near_expiry).length,
    [layers],
  );

  return (
    <PageTransition>
      <PageChrome
        icon="treasury"
        title={t('warehouse_portal.valuation.title')}
        description={t('warehouse_portal.valuation.subtitle')}
        loading={loading && !valuation && !error}
        skeletonVariant="table"
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Link href="/inventory" className="portal-btn portal-btn--outline">
              <Icon name="left" size={16} />
              <span>{t('portal.nav.inventory')}</span>
            </Link>
            <button
              type="button"
              onClick={() => setShowReceiptForm((v) => !v)}
              className="portal-btn portal-btn--outline"
            >
              <Icon name="plus" size={16} />
              <span>{t('warehouse_portal.valuation.record_receipt_button')}</span>
            </button>
            <button
              type="button"
              disabled={loading || refreshing}
              onClick={() => void loadAll(true)}
              className="portal-btn portal-btn--primary"
            >
              <Icon name="refresh" size={16} className={refreshing ? 'animate-spin' : ''} />
              <span>{refreshing ? t('common.status.loading') : t('common.action.refresh')}</span>
            </button>
          </div>
        }
      >
        {error ? (
          <div className="portal-alert portal-alert--error mb-6" role="alert">
            <p className="portal-alert-text font-semibold">{error}</p>
          </div>
        ) : null}

        {notice ? (
          <div className="portal-alert portal-alert--info mb-6">
            <p className="portal-alert-text">{notice}</p>
          </div>
        ) : null}

        <KpiStatGrid columns={4}>
          <KpiStatCard
            bay="inventory"
            icon="treasury"
            label={t('warehouse_portal.valuation.kpi_total_value')}
            value={formatWarehouseMinor(valuation?.total_value_minor ?? 0, currency)}
            sub={t('warehouse_portal.valuation.kpi_total_value_sub')}
          />
          <KpiStatCard
            bay="inventory"
            icon="inventory"
            label={t('warehouse_portal.valuation.kpi_qty_remaining')}
            value={valuation?.total_qty_remaining ?? 0}
            sub={t('warehouse_portal.valuation.kpi_skus_count', { count: skus.length })}
          />
          <KpiStatCard
            bay="inventory"
            icon="catalog"
            label={t('warehouse_portal.valuation.kpi_layers_count')}
            value={layers.length}
            sub={t('warehouse_portal.valuation.kpi_movements_count', { count: movements.length })}
          />
          <KpiStatCard
            bay="inventory"
            icon="warning"
            flag={nearExpiryCount > 0 ? 'alert' : 'ok'}
            label={t('warehouse_portal.valuation.kpi_near_expiry')}
            value={nearExpiryCount}
            sub={t('warehouse_portal.valuation.kpi_expiry_window', { days: valuation?.expiry_risk_days ?? 14 })}
          />
        </KpiStatGrid>

        {showReceiptForm ? (
          <div className="mt-6">
            <PortalSection
              icon="plus"
              title={t('warehouse_portal.valuation.receipt_form_title')}
              description={t('warehouse_portal.valuation.receipt_form_desc')}
            >
              <form onSubmit={(e) => void handleRecordReceipt(e)} className="grid grid-cols-1 md:grid-cols-5 gap-3 items-end">
                <div>
                  <label htmlFor="rcpt-wh" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.warehouse_id_label')}
                  </label>
                  <PortalInput
                    id="rcpt-wh"
                    type="text"
                    value={rcptWarehouseId}
                    onChange={(e) => setRcptWarehouseId(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="rcpt-product" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.product_id_label')}
                  </label>
                  <PortalInput
                    id="rcpt-product"
                    type="text"
                    placeholder="sku-..."
                    value={rcptProductId}
                    onChange={(e) => setRcptProductId(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="rcpt-lot" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.lot_id_label')}
                  </label>
                  <PortalInput
                    id="rcpt-lot"
                    type="text"
                    placeholder="lot-..."
                    value={rcptLotId}
                    onChange={(e) => setRcptLotId(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="rcpt-ref" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.receipt_ref_label')}
                  </label>
                  <PortalInput
                    id="rcpt-ref"
                    type="text"
                    placeholder="po-rcpt-1"
                    value={rcptRefId}
                    onChange={(e) => setRcptRefId(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="rcpt-qty" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.quantity_label')}
                  </label>
                  <PortalInput
                    id="rcpt-qty"
                    type="number"
                    min={1}
                    value={rcptQuantity}
                    onChange={(e) => setRcptQuantity(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="rcpt-cost" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.unit_cost_major_label')}
                  </label>
                  <PortalInput
                    id="rcpt-cost"
                    type="text"
                    placeholder="12500"
                    value={rcptUnitCostMajor}
                    onChange={(e) => setRcptUnitCostMajor(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="rcpt-vat" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.vat_major_label')}
                  </label>
                  <PortalInput
                    id="rcpt-vat"
                    type="text"
                    placeholder="0"
                    value={rcptVatMajor}
                    onChange={(e) => setRcptVatMajor(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="rcpt-credit-acct" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.credit_account_label')}
                  </label>
                  <PortalSelect
                    id="rcpt-credit-acct"
                    value={rcptCreditAccount}
                    onChange={(e) => setRcptCreditAccount(e.target.value)}
                  >
                    <option value="AP:VENDOR">AP:VENDOR</option>
                    <option value="GRNI">GRNI</option>
                  </PortalSelect>
                </div>
                <div>
                  <label htmlFor="rcpt-expiry" className="portal-label block mb-1">
                    {t('warehouse_portal.valuation.expiry_date_label')}
                  </label>
                  <PortalInput
                    id="rcpt-expiry"
                    type="date"
                    value={rcptExpiryDate}
                    onChange={(e) => setRcptExpiryDate(e.target.value)}
                  />
                </div>
                <div className="flex gap-2">
                  <PortalSelect
                    id="rcpt-currency"
                    aria-label={t('warehouse_portal.valuation.currency_label')}
                    value={rcptCurrency}
                    onChange={(e) => setRcptCurrency(e.target.value)}
                  >
                    <option value="UZS">UZS</option>
                    <option value="KZT">KZT</option>
                    <option value="USD">USD</option>
                  </PortalSelect>
                  <button
                    type="submit"
                    disabled={submitting}
                    className="portal-btn portal-btn--primary"
                  >
                    <span>{submitting ? t('common.status.loading') : t('warehouse_portal.valuation.submit_receipt_button')}</span>
                  </button>
                </div>
              </form>
            </PortalSection>
          </div>
        ) : null}

        <div className="desk-card p-4 mt-6 mb-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
            <div>
              <label htmlFor="val-warehouse-filter" className="portal-label block mb-1">
                {t('warehouse_portal.valuation.warehouse_id_label')}
              </label>
              <PortalInput
                id="val-warehouse-filter"
                type="text"
                placeholder="wh-..."
                value={warehouseFilter}
                onChange={(e) => setWarehouseFilter(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="val-product-filter" className="portal-label block mb-1">
                {t('warehouse_portal.valuation.product_id_label')}
              </label>
              <PortalInput
                id="val-product-filter"
                type="text"
                placeholder="sku-..."
                value={productFilter}
                onChange={(e) => setProductFilter(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="val-risk-days" className="portal-label block mb-1">
                {t('warehouse_portal.valuation.expiry_risk_days_label')}
              </label>
              <PortalInput
                id="val-risk-days"
                type="number"
                min={1}
                value={expiryRiskDays}
                onChange={(e) => setExpiryRiskDays(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="val-movement-filter" className="portal-label block mb-1">
                {t('warehouse_portal.valuation.movement_type_label')}
              </label>
              <PortalSelect
                id="val-movement-filter"
                value={movementTypeFilter}
                onChange={(e) => setMovementTypeFilter(e.target.value)}
              >
                {MOVEMENT_TYPES.map((mt) => (
                  <option key={mt} value={mt}>
                    {mt === 'ALL' ? t('warehouse_portal.valuation.all_movement_types') : mt}
                  </option>
                ))}
              </PortalSelect>
            </div>
          </div>
        </div>

        <div className="mb-4 flex flex-wrap gap-2" role="tablist" aria-label={t('warehouse_portal.valuation.tabs_label')}>
          {(
            [
              ['layers', 'warehouse_portal.valuation.tab_layers'],
              ['skus', 'warehouse_portal.valuation.tab_skus'],
              ['movements', 'warehouse_portal.valuation.tab_movements'],
            ] as const
          ).map(([tabKey, labelKey]) => {
            const active = activeTab === tabKey;
            return (
              <button
                key={tabKey}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => setActiveTab(tabKey)}
                className={`portal-btn ${active ? 'portal-btn--primary' : 'portal-btn--outline'}`}
              >
                <span>{t(labelKey)}</span>
              </button>
            );
          })}
        </div>

        {activeTab === 'layers' ? (
          <PortalSection
            icon="inventory"
            title={t('warehouse_portal.valuation.layers_title')}
            description={t('warehouse_portal.valuation.layers_desc')}
          >
            {layers.length === 0 ? (
              <p className="text-sm" style={{ color: 'var(--desk-text-tertiary)' }}>
                {t('warehouse_portal.valuation.no_layers')}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: 'var(--desk-border)',
                        color: 'var(--desk-text-tertiary)',
                        background: 'var(--desk-surface-subtle)',
                      }}
                    >
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_layer_lot')}</th>
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_sku_wh')}</th>
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_receipt_ref')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_qty')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_unit_cost')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_layer_value')}</th>
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_expiry_fefo')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: 'var(--desk-border)' }}>
                    {layers.map((layer) => {
                      const ccy = layer.currency || currency;
                      return (
                        <tr key={layer.layer_id}>
                          <td className="py-2.5 px-3 font-mono">
                            <div className="font-semibold">{layer.layer_id}</div>
                            <div style={{ color: 'var(--desk-text-tertiary)' }}>{layer.lot_id || '—'}</div>
                          </td>
                          <td className="py-2.5 px-3 font-mono">
                            <div className="font-medium">{layer.product_id}</div>
                            <div style={{ color: 'var(--desk-text-tertiary)' }}>{layer.warehouse_id}</div>
                          </td>
                          <td className="py-2.5 px-3 font-mono">
                            {layer.receipt_ref_type || 'RECEIPT'}:{layer.receipt_ref_id || '—'}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-right">
                            {layer.qty_remaining} / {layer.original_qty}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-right">
                            {formatWarehouseMinor(layer.unit_cost_minor, ccy)}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-right font-semibold">
                            {formatWarehouseMinor(layer.total_value_minor, ccy)}
                          </td>
                          <td className="py-2.5 px-3 text-xs">
                            <div className="flex flex-wrap items-center gap-1.5">
                              <span className="font-mono">{layer.expiry_date || '—'}</span>
                              {layer.near_expiry ? (
                                <span
                                  className="inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold"
                                  style={{
                                    background: 'var(--desk-warning-soft)',
                                    color: 'var(--desk-warning)',
                                    borderColor: 'var(--desk-border-strong)',
                                  }}
                                >
                                  {t('warehouse_portal.valuation.near_expiry_badge', {
                                    days: layer.days_to_expiry ?? 0,
                                  })}
                                </span>
                              ) : null}
                            </div>
                            <div className="text-[11px]" style={{ color: 'var(--desk-text-tertiary)' }}>
                              {formatIso(layer.received_at)}
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        ) : null}

        {activeTab === 'skus' ? (
          <PortalSection
            icon="catalog"
            title={t('warehouse_portal.valuation.skus_title')}
            description={t('warehouse_portal.valuation.skus_desc')}
          >
            {skus.length === 0 ? (
              <p className="text-sm" style={{ color: 'var(--desk-text-tertiary)' }}>
                {t('warehouse_portal.valuation.no_skus')}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: 'var(--desk-border)',
                        color: 'var(--desk-text-tertiary)',
                        background: 'var(--desk-surface-subtle)',
                      }}
                    >
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.product_id_label')}</th>
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.warehouse_id_label')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_layer_count')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_qty_on_hand')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_moving_avg_cost')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_total_value')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: 'var(--desk-border)' }}>
                    {skus.map((sku) => {
                      const ccy = sku.currency || currency;
                      return (
                        <tr
                          key={`${sku.warehouse_id}:${sku.product_id}`}
                          onClick={() => {
                            setProductFilter(sku.product_id);
                            setActiveTab('layers');
                          }}
                          className="cursor-pointer hover:bg-[var(--desk-surface-subtle)]"
                        >
                          <td className="py-2.5 px-3 font-mono font-semibold">{sku.product_id}</td>
                          <td className="py-2.5 px-3 font-mono">{sku.warehouse_id}</td>
                          <td className="py-2.5 px-3 font-mono text-right">{sku.layer_count}</td>
                          <td className="py-2.5 px-3 font-mono text-right">{sku.total_qty_remaining}</td>
                          <td className="py-2.5 px-3 font-mono text-right">
                            {formatWarehouseMinor(sku.moving_avg_unit_cost_minor, ccy)}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-right font-semibold">
                            {formatWarehouseMinor(sku.total_value_minor, ccy)}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        ) : null}

        {activeTab === 'movements' ? (
          <PortalSection
            icon="manifests"
            title={t('warehouse_portal.valuation.movements_title')}
            description={t('warehouse_portal.valuation.movements_desc')}
          >
            {movements.length === 0 ? (
              <p className="text-sm" style={{ color: 'var(--desk-text-tertiary)' }}>
                {t('warehouse_portal.valuation.no_movements')}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: 'var(--desk-border)',
                        color: 'var(--desk-text-tertiary)',
                        background: 'var(--desk-surface-subtle)',
                      }}
                    >
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_movement')}</th>
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_type')}</th>
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_sku_lot')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_qty_delta')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_unit_cost')}</th>
                      <th className="py-2.5 px-3 text-right">{t('warehouse_portal.valuation.col_total_cost')}</th>
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_gl_entry')}</th>
                      <th className="py-2.5 px-3">{t('warehouse_portal.valuation.col_created_at')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: 'var(--desk-border)' }}>
                    {movements.map((mov) => {
                      const ccy = mov.currency || currency;
                      return (
                        <tr key={mov.movement_id}>
                          <td className="py-2.5 px-3 font-mono">
                            <div className="font-semibold">{mov.movement_id}</div>
                            <div style={{ color: 'var(--desk-text-tertiary)' }}>
                              {mov.reference_type || 'REF'}:{mov.reference_id || '—'}
                            </div>
                          </td>
                          <td className="py-2.5 px-3">
                            <span
                              className="inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold"
                              style={movementChipStyle(mov.movement_type)}
                            >
                              {mov.movement_type}
                            </span>
                          </td>
                          <td className="py-2.5 px-3 font-mono">
                            <div>{mov.product_id}</div>
                            <div style={{ color: 'var(--desk-text-tertiary)' }}>
                              {mov.lot_id || mov.layer_id || '—'}
                            </div>
                          </td>
                          <td className="py-2.5 px-3 font-mono text-right font-semibold">
                            {formatSignedQty(mov.qty_delta)}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-right">
                            {formatWarehouseMinor(mov.unit_cost_minor, ccy)}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-right font-semibold">
                            {formatWarehouseMinor(mov.total_cost_minor, ccy)}
                          </td>
                          <td className="py-2.5 px-3 font-mono">{mov.gl_entry_id || '—'}</td>
                          <td className="py-2.5 px-3 text-[11px]" style={{ color: 'var(--desk-text-secondary)' }}>
                            {formatIso(mov.created_at)}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        ) : null}
      </PageChrome>
    </PageTransition>
  );
}
