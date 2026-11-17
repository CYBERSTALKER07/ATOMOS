import type { PickTask, PickWave, StockLot, WarehouseBinLocation } from '@pegasusx/types';

export const COLD_LOT_MIGRATION_WARNING = 'cold_lot_migrated_without_live_baseline';
const SEVEN_DAYS_MS = 7 * 24 * 60 * 60 * 1000;

export interface LotColdAssessment {
  lot: StockLot;
  requiredZone: 'CHILL' | 'FREEZE' | null;
  effectiveZone: string;
  isStaging: boolean;
  zoneMismatch: boolean;
  hasColdWarning: boolean;
  shortExpiry: boolean;
}

export interface WarehouseColdSummary {
  byLotId: Map<string, LotColdAssessment>;
  coldWarningLots: LotColdAssessment[];
  stagingLots: LotColdAssessment[];
  expiryAlertLots: LotColdAssessment[];
}

export interface PickWavePreSealColdSummary {
  hasPreSealWarning: boolean;
  missingBaseline: boolean;
  flaggedTasks: PickTask[];
  zoneMismatchLots: LotColdAssessment[];
  warnings: string[];
}

export function isColdCompatibleZone(zone: string | undefined | null, requiredZone: 'CHILL' | 'FREEZE'): boolean {
  const normalized = (zone ?? '').trim().toUpperCase();
  if (!normalized) return false;
  if (requiredZone === 'FREEZE') {
    return normalized === 'FREEZE' || normalized === 'FROZEN' || normalized === 'FREEZER';
  }
  return (
    normalized === 'CHILL' ||
    normalized === 'CHILLED' ||
    normalized === 'COLD' ||
    normalized === 'REEFER' ||
    normalized === 'FREEZE' ||
    normalized === 'FROZEN'
  );
}

export function resolveLotRequiredZone(lot: Pick<StockLot, 'required_zone' | 'requires_cold_chain' | 'is_perishable' | 'handling_class' | 'cold_lot_warning'>): 'CHILL' | 'FREEZE' | null {
  const explicit = (lot.required_zone ?? '').trim().toUpperCase();
  if (explicit === 'FREEZE' || explicit === 'CHILL') {
    return explicit;
  }
  const hc = (lot.handling_class ?? '').trim().toUpperCase();
  if (hc === 'FROZEN') {
    return 'FREEZE';
  }
  if (
    lot.requires_cold_chain ||
    lot.is_perishable ||
    hc === 'CHILLED' ||
    hc === 'COLD' ||
    hc === 'PERISHABLE' ||
    lot.cold_lot_warning === COLD_LOT_MIGRATION_WARNING
  ) {
    return 'CHILL';
  }
  return null;
}

export function isStagingLocation(locationId: string | undefined | null, bin?: WarehouseBinLocation): boolean {
  const normalized = (locationId ?? '').trim().toLowerCase();
  if (normalized === 'recv-default' || normalized.startsWith('recv-')) {
    return true;
  }
  return (bin?.location_type ?? '').trim().toUpperCase() === 'STAGE';
}

export function assessStockLot(lot: StockLot, bin?: WarehouseBinLocation, nowMs = Date.now()): LotColdAssessment {
  const requiredZone = resolveLotRequiredZone(lot);
  const effectiveZone = (bin?.zone || lot.location_zone || '').trim().toUpperCase();
  const isStaging = Boolean(lot.migrated_staging) || isStagingLocation(lot.location_id, bin);
  const zoneMismatch = requiredZone !== null && (isStaging || !isColdCompatibleZone(effectiveZone, requiredZone));
  const hasColdWarning = lot.cold_lot_warning === COLD_LOT_MIGRATION_WARNING || zoneMismatch;

  let shortExpiry = false;
  if (lot.expiry_date) {
    const parsed = Date.parse(lot.expiry_date);
    if (Number.isFinite(parsed) && parsed - nowMs <= SEVEN_DAYS_MS) {
      shortExpiry = true;
    }
  }

  return {
    lot,
    requiredZone,
    effectiveZone,
    isStaging,
    zoneMismatch,
    hasColdWarning,
    shortExpiry,
  };
}

export function analyzeWarehouseLots(lots: StockLot[], bins: WarehouseBinLocation[], nowMs = Date.now()): WarehouseColdSummary {
  const binById = new Map(bins.map((b) => [b.location_id, b]));
  const byLotId = new Map<string, LotColdAssessment>();
  const coldWarningLots: LotColdAssessment[] = [];
  const stagingLots: LotColdAssessment[] = [];
  const expiryAlertLots: LotColdAssessment[] = [];

  for (const lot of lots) {
    const assessment = assessStockLot(lot, binById.get(lot.location_id), nowMs);
    byLotId.set(lot.lot_id, assessment);
    if (lot.quantity_on_hand <= 0) {
      continue;
    }
    if (assessment.hasColdWarning) {
      coldWarningLots.push(assessment);
    }
    if (assessment.isStaging) {
      stagingLots.push(assessment);
    }
    if (assessment.shortExpiry) {
      expiryAlertLots.push(assessment);
    }
  }

  return { byLotId, coldWarningLots, stagingLots, expiryAlertLots };
}

export function recommendedColdBins(bins: WarehouseBinLocation[], requiredZone: 'CHILL' | 'FREEZE' | null): WarehouseBinLocation[] {
  if (!requiredZone) {
    return bins.filter((b) => b.location_type !== 'STAGE' && b.location_type !== 'QUARANTINE');
  }
  const matching = bins.filter((b) => b.is_active !== false && isColdCompatibleZone(b.zone, requiredZone));
  return matching.length > 0 ? matching : bins.filter((b) => b.location_type !== 'STAGE' && b.location_type !== 'QUARANTINE');
}

export function analyzePickWavePreSealColdState(
  wave: PickWave,
  lots: StockLot[],
  bins: WarehouseBinLocation[],
  tempReadingsCount?: number | null,
): PickWavePreSealColdSummary {
  const lotSummary = analyzeWarehouseLots(lots, bins);
  const byLotId = lotSummary.byLotId;
  const tasks = wave.tasks ?? [];
  const flaggedTasks: PickTask[] = [];
  const zoneMismatchLots: LotColdAssessment[] = [];
  const warnings = new Set<string>(wave.cold_lot_warnings ?? []);
  let hasColdInvolvement = Boolean(wave.cold_lot_migrated_without_live_baseline || wave.cold_chain_baseline_missing);

  for (const task of tasks) {
    const direct = byLotId.get(task.lot_id);
    const fallbackMatches = direct ? [direct] : lotSummary.coldWarningLots.filter((a) => a.lot.product_id === task.product_id);
    let taskFlagged = false;
    for (const match of fallbackMatches) {
      if (match.requiredZone !== null) {
        hasColdInvolvement = true;
      }
      if (match.hasColdWarning || match.isStaging) {
        taskFlagged = true;
        if (!zoneMismatchLots.some((item) => item.lot.lot_id === match.lot.lot_id)) {
          zoneMismatchLots.push(match);
        }
        if (match.requiredZone) {
          warnings.add(`${match.lot.product_id}:${match.requiredZone}@${match.lot.location_id || 'recv-default'}`);
        }
      }
    }
    if (!direct) {
      const bin = bins.find((b) => b.location_id === task.location_id);
      if (isStagingLocation(task.location_id, bin)) {
        taskFlagged = true;
        warnings.add(`${task.product_id}@${task.location_id || 'recv-default'}`);
      }
    }
    if (taskFlagged) {
      flaggedTasks.push(task);
    }
  }

  const missingBaseline =
    Boolean(wave.cold_chain_baseline_missing) ||
    (hasColdInvolvement && tempReadingsCount === 0);

  if (missingBaseline) {
    warnings.add('cold_chain_baseline_required');
  }

  return {
    hasPreSealWarning:
      Boolean(wave.cold_lot_migrated_without_live_baseline) ||
      missingBaseline ||
      flaggedTasks.length > 0 ||
      zoneMismatchLots.length > 0,
    missingBaseline,
    flaggedTasks,
    zoneMismatchLots,
    warnings: Array.from(warnings),
  };
}
