'use client';

import type { LotColdAssessment } from '@/lib/cold-lot-warnings';
import { COLD_LOT_MIGRATION_WARNING } from '@/lib/cold-lot-warnings';

type Translator = (key: string, params?: Record<string, string | number>) => string;

export interface ColdLotWarningBannerProps {
  t: Translator;
  coldWarningLots: LotColdAssessment[];
  stagingLots?: LotColdAssessment[];
  expiryAlertLots?: LotColdAssessment[];
  missingManifestBaseline?: boolean;
  onSelectPutawayLot?: (assessment: LotColdAssessment) => void;
}

/**
 * Surfaces migrated cold-chain lot placement warnings (`cold_lot_migrated_without_live_baseline`),
 * staging putaway requirements, zone mismatches, and short-shelf-life alerts.
 */
export function ColdLotWarningBanner({
  t,
  coldWarningLots,
  stagingLots = [],
  expiryAlertLots = [],
  missingManifestBaseline = false,
  onSelectPutawayLot,
}: ColdLotWarningBannerProps) {
  if (
    coldWarningLots.length === 0 &&
    stagingLots.length === 0 &&
    expiryAlertLots.length === 0 &&
    !missingManifestBaseline
  ) {
    return null;
  }

  return (
    <div
      role="alert"
      data-testid="cold-lot-warning-banner"
      className="mb-4 rounded-xl border border-amber-500/50 bg-amber-50/90 p-4 text-sm text-amber-950 shadow-sm dark:bg-amber-950/30 dark:text-amber-100"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="font-semibold">
          {coldWarningLots.length > 0 || missingManifestBaseline
            ? t('warehouse_portal.cold_lot.warning_banner')
            : `Migration staging putaway required (${stagingLots.length})`}
        </div>
        {(coldWarningLots.length > 0 || missingManifestBaseline) && (
          <span className="rounded border border-amber-600/40 bg-amber-100 px-2 py-0.5 font-mono text-[11px] uppercase tracking-wide text-amber-900 dark:bg-amber-900/50 dark:text-amber-100">
            {COLD_LOT_MIGRATION_WARNING}
          </span>
        )}
      </div>

      {missingManifestBaseline && (
        <p className="mt-1.5 text-xs font-medium text-red-700 dark:text-red-300">
          Pre-seal gate: cold-chain manifest requires a live bay temperature reading before payload seal.
        </p>
      )}

      {stagingLots.length > 0 && (
        <p className="mt-1 text-xs opacity-90">
          {stagingLots.length} migrated lot(s) are sitting in receiving staging (<code>recv-default</code> /{' '}
          <code>STAGE</code>) and must be put away into final pick/chill bins.
        </p>
      )}

      {(coldWarningLots.length > 0 || stagingLots.length > 0) && (
        <ul className="mt-2.5 space-y-1.5">
          {(coldWarningLots.length > 0 ? coldWarningLots : stagingLots).slice(0, 6).map((item) => {
            const skuLabel = item.lot.product_name || item.lot.product_id;
            const lotCode = item.lot.lot_code || item.lot.lot_id.slice(0, 8);
            return (
              <li
                key={item.lot.lot_id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-amber-500/30 bg-white/75 px-3 py-1.5 text-xs dark:bg-black/20"
              >
                <div className="space-x-2">
                  <span className="font-semibold">{skuLabel}</span>
                  <span className="font-mono opacity-80">
                    {t('warehouse_portal.cold_lot.lot_code_label', { lot_code: lotCode })}
                  </span>
                  <span className="font-mono opacity-75">
                    @ {item.lot.location_id || 'recv-default'}
                    {item.effectiveZone ? ` (${item.effectiveZone})` : ''}
                  </span>
                  {item.zoneMismatch && item.requiredZone && (
                    <span className="font-medium text-red-700 dark:text-red-300">
                      {t('warehouse_portal.cold_lot.zone_mismatch', {
                        sku: skuLabel,
                        required_zone: item.requiredZone,
                      })}
                    </span>
                  )}
                  {item.shortExpiry && item.lot.expiry_date && (
                    <span className="font-medium text-amber-800 dark:text-amber-200">
                      {t('warehouse_portal.cold_lot.expiry_alert', { expires_at: item.lot.expiry_date })}
                    </span>
                  )}
                </div>
                {onSelectPutawayLot && (
                  <button
                    type="button"
                    onClick={() => onSelectPutawayLot(item)}
                    className="rounded border border-amber-600/50 bg-amber-600 px-2.5 py-1 text-[11px] font-medium text-white hover:bg-amber-700"
                  >
                    {item.requiredZone
                      ? t('warehouse_portal.cold_lot.confirm_cold_putaway')
                      : 'Stage putaway'}
                  </button>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
