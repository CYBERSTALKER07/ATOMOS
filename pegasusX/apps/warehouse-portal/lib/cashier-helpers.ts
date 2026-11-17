import {
  homeCellFromJwt,
  readCachedAuthSession,
} from '@pegasusx/api-core';

type CachedAuthSession = NonNullable<ReturnType<typeof readCachedAuthSession>>;
import {
  UZS_BANKNOTE_DENOMINATIONS,
  type WarehouseFleetLiveRoute,
} from '@pegasusx/types';
import { decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';
import {
  formatGroupedInteger,
  majorIntToMinor,
  minorToWholeTrunc,
  minorToWholeTruncBigInt,
} from '@/lib/money';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';

export interface CashierScope {
  cellId: string;
  depotId: string;
  supplierId: string;
  actorId: string;
}

export interface CashierDriverOption {
  driverId: string;
  label: string;
  routeId?: string;
}

export interface CashierRouteOption {
  routeId: string;
  label: string;
  driverId?: string;
  manifestId?: string;
}

export interface CashierDriverRecord {
  driver_id: string;
  name?: string;
}

export interface CashierManifestRecord {
  manifest_id: string;
  driver_name?: string;
  vehicle_label?: string;
}

export function formatWholeSom(
  minor: number | bigint | null | undefined,
  currency = "so'm",
): string {
  if (typeof minor === 'bigint') {
    return `${formatGroupedInteger(minorToWholeTruncBigInt(minor))} ${currency}`;
  }
  return `${formatGroupedInteger(minorToWholeTrunc(minor ?? 0))} ${currency}`;
}

export function formatSignedWholeSom(minor: number | null | undefined): string {
  const whole = minorToWholeTrunc(minor ?? 0);
  const prefix = whole > 0 ? '+' : '';
  return `${prefix}${formatGroupedInteger(whole)} so'm`;
}

export function formatWholeSomInteger(
  wholeSom: number | bigint | null | undefined,
  currency = "so'm",
): string {
  return `${formatGroupedInteger(wholeSom ?? 0)} ${currency}`;
}

export function calculateDenominationTotals(counts: Record<number, number>): {
  totalSom: number;
  totalMinor: number;
  totalNotes: number;
} {
  let totalSom = 0;
  let totalNotes = 0;
  for (const denom of UZS_BANKNOTE_DENOMINATIONS) {
    const safeCount = Math.max(0, Math.floor(counts[denom] || 0));
    totalSom += denom * safeCount;
    totalNotes += safeCount;
  }
  return {
    totalSom,
    totalMinor: majorIntToMinor(totalSom),
    totalNotes,
  };
}

function readClaimString(
  claims: Record<string, unknown> | null,
  keys: string[],
): string {
  if (!claims) return '';
  for (const key of keys) {
    const val = claims[key];
    if (typeof val === 'string' && val.trim()) {
      return val.trim();
    }
  }
  return '';
}

export function resolveCashierScope(options?: {
  token?: string;
  cachedSession?: CachedAuthSession | null;
  homeNodeId?: string;
}): CashierScope {
  const token = options?.token ?? readTokenFromCookie();
  const claims = token ? decodeJwtPayload(token) : null;
  const cached =
    options?.cachedSession !== undefined
      ? options.cachedSession
      : readCachedAuthSession();

  const cellId =
    homeCellFromJwt(token) ||
    readClaimString(claims, ['home_cell']) ||
    (cached?.home_cell?.trim() ?? '');
  const depotId =
    (options?.homeNodeId ?? warehouseHomeNodeId()).trim() ||
    readClaimString(claims, ['home_node_id', 'warehouse_id']);
  const supplierId = readClaimString(claims, ['supplier_id', 'org_id']);
  const actorId = readClaimString(claims, ['sub', 'email']);

  return { cellId, depotId, supplierId, actorId };
}

export function buildCashierSelectorOptions(input: {
  drivers?: CashierDriverRecord[];
  manifests?: CashierManifestRecord[];
  fleetRoutes?: WarehouseFleetLiveRoute[];
}): {
  driverOptions: CashierDriverOption[];
  routeOptions: CashierRouteOption[];
} {
  const driverMap = new Map<string, CashierDriverOption>();
  const routeMap = new Map<string, CashierRouteOption>();
  const seenManifestIds = new Set<string>();

  for (const driver of input.drivers ?? []) {
    const id = driver.driver_id?.trim();
    if (!id) continue;
    const name = driver.name?.trim();
    driverMap.set(id, {
      driverId: id,
      label: name ? `${name} (${id})` : id,
    });
  }

  for (const route of input.fleetRoutes ?? []) {
    const routeId = route.route_id?.trim();
    const driverId = route.driver_id?.trim() || undefined;
    const manifestId = route.manifest_id?.trim() || undefined;
    if (manifestId) seenManifestIds.add(manifestId);
    if (routeId) {
      const suffix = route.driver_name?.trim() || driverId || '';
      routeMap.set(routeId, {
        routeId,
        label: suffix ? `${routeId} · ${suffix}` : routeId,
        driverId,
        manifestId,
      });
    }
    if (driverId) {
      const existing = driverMap.get(driverId);
      if (!existing) {
        const name = route.driver_name?.trim();
        driverMap.set(driverId, {
          driverId,
          label: name ? `${name} (${driverId})` : driverId,
          routeId: routeId || undefined,
        });
      } else if (!existing.routeId && routeId) {
        existing.routeId = routeId;
      }
    }
  }

  for (const manifest of input.manifests ?? []) {
    const manifestId = manifest.manifest_id?.trim();
    if (!manifestId || seenManifestIds.has(manifestId) || routeMap.has(manifestId)) {
      continue;
    }
    const parts = [manifestId];
    if (manifest.driver_name?.trim()) parts.push(manifest.driver_name.trim());
    if (manifest.vehicle_label?.trim()) parts.push(manifest.vehicle_label.trim());
    routeMap.set(manifestId, {
      routeId: manifestId,
      label: parts.join(' · '),
      manifestId,
    });
  }

  return {
    driverOptions: Array.from(driverMap.values()),
    routeOptions: Array.from(routeMap.values()),
  };
}
