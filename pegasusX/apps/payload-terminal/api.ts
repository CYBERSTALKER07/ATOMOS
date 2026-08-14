import { authFetch } from './authSession';
import { readApiError } from './explainBanner';

export const API_BASE = (process.env.EXPO_PUBLIC_API_URL?.trim() || '') ||
  (__DEV__ ? 'http://localhost:8180' : (process.env.EXPO_PUBLIC_RELEASE_API_URL?.trim() || 'https://api.pegasusx.app'));

/**
 * Payload Terminal API
 * Gap-hunter pass endpoints. Authenticated calls use authFetch (401 → refresh).
 */
export type LoadingBayManifestSource = 'payloader' | 'factory';

export type LoadingBayManifest = {
    manifest_id: string;
    state?: string;
    truck_id?: string;
    vehicle_id?: string;
    total_volume_vu?: number;
    max_volume_vu?: number;
    stop_count?: number;
    region_code?: string;
    source: LoadingBayManifestSource;
};

function normalizeManifests(
    raw: unknown,
    source: LoadingBayManifestSource,
): LoadingBayManifest[] {
    const list = Array.isArray((raw as { manifests?: unknown })?.manifests)
        ? ((raw as { manifests: Record<string, unknown>[] }).manifests)
        : [];
    return list.map((m) => ({
        manifest_id: String(m.manifest_id ?? ''),
        state: typeof m.state === 'string' ? m.state : undefined,
        truck_id: typeof m.truck_id === 'string' ? m.truck_id : undefined,
        vehicle_id: typeof m.vehicle_id === 'string' ? m.vehicle_id : undefined,
        total_volume_vu: typeof m.total_volume_vu === 'number' ? m.total_volume_vu : undefined,
        max_volume_vu: typeof m.max_volume_vu === 'number' ? m.max_volume_vu : undefined,
        stop_count: typeof m.stop_count === 'number' ? m.stop_count : undefined,
        region_code: typeof m.region_code === 'string' ? m.region_code : undefined,
        source,
    })).filter((m) => m.manifest_id);
}

export const PayloadTerminalApi = {
    getSupplierManifests: async (_token: string, state: string = 'DRAFT') => {
        const res = await authFetch(`/v1/supplier/manifests?state=${state}`);
        if (!res.ok) throw new Error('Failed to fetch supplier manifests');
        return res.json();
    },

    /** Payloader + factory loading-bay manifests (P1-18 Class A bridge). */
    listLoadingBayManifests: async (
        _token: string,
        state: string = 'DRAFT',
    ): Promise<{ manifests: LoadingBayManifest[] }> => {
        const q = `state=${encodeURIComponent(state)}`;
        const [payloaderRes, factoryRes] = await Promise.all([
            authFetch(`/v1/payloader/manifests?${q}`),
            authFetch(`/v1/factory/manifests?${q}`),
        ]);
        const out: LoadingBayManifest[] = [];
        const seen = new Set<string>();
        if (payloaderRes.ok) {
            for (const m of normalizeManifests(await payloaderRes.json(), 'payloader')) {
                if (seen.has(m.manifest_id)) continue;
                seen.add(m.manifest_id);
                out.push(m);
            }
        } else {
            // Legacy alias still mounted for PAYLOAD on payloaderroutes.
            const supplierRes = await authFetch(`/v1/supplier/manifests?${q}`);
            if (supplierRes.ok) {
                for (const m of normalizeManifests(await supplierRes.json(), 'payloader')) {
                    if (seen.has(m.manifest_id)) continue;
                    seen.add(m.manifest_id);
                    out.push(m);
                }
            }
        }
        if (factoryRes.ok) {
            for (const m of normalizeManifests(await factoryRes.json(), 'factory')) {
                if (seen.has(m.manifest_id)) continue;
                seen.add(m.manifest_id);
                out.push(m);
            }
        }
        return { manifests: out };
    },

    factoryStartLoading: async (_token: string, manifestId: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = {};
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await authFetch(`/v1/factory/manifests/${encodeURIComponent(manifestId)}/start-loading`, {
            method: 'POST',
            headers,
        });
        if (!res.ok) throw new Error('Failed to start loading factory manifest');
        return res.json();
    },

    factorySealManifest: async (_token: string, manifestId: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = {};
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await authFetch(`/v1/factory/manifests/${encodeURIComponent(manifestId)}/seal`, {
            method: 'POST',
            headers,
        });
        if (!res.ok) throw await readApiError(res);
        return res.json();
    },

    lookupBarcode: async (ean: string) => {
        const encoded = encodeURIComponent(ean.trim());
        const res = await authFetch(`/v1/catalog/barcode/${encoded}`);
        if (!res.ok) throw new Error('Barcode lookup failed');
        return res.json() as Promise<{
            product_id?: string;
            sku_id?: string;
            name?: string;
            barcode?: string;
        }>;
    },

    getSupplierManifestDetail: async (_token: string, manifestId: string) => {
        const res = await authFetch(`/v1/supplier/manifests/${manifestId}`);
        if (!res.ok) throw new Error('Failed to fetch supplier manifest detail');
        return res.json();
    },

    supplierStartLoading: async (_token: string, manifestId: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = {};
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await authFetch(`/v1/supplier/manifests/${manifestId}/start-loading`, {
            method: 'POST',
            headers,
        });
        if (!res.ok) throw new Error('Failed to start loading supplier manifest');
        return res.json();
    },

    supplierInjectOrder: async (_token: string, manifestId: string, orderId: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await authFetch(`/v1/supplier/manifests/${manifestId}/inject-order`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ order_id: orderId }),
        });
        if (!res.ok) throw new Error('Failed to inject order to supplier manifest');
        return res.json();
    },

    supplierSealManifest: async (_token: string, manifestId: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = {};
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await authFetch(`/v1/supplier/manifests/${manifestId}/seal`, {
            method: 'POST',
            headers,
        });
        if (!res.ok) throw new Error('Failed to seal supplier manifest');
        return res.json();
    },

    sealCompletedManifests: async (_token: string, manifestIds: string[], idempotencyKey?: string) => {
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await authFetch('/v1/payloader/manifests/seal-completed', {
            method: 'POST',
            headers,
            body: JSON.stringify({ manifest_ids: manifestIds }),
        });
        if (!res.ok) throw await readApiError(res);
        return res.json();
    },

    sealAllManifests: async (_token: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await authFetch('/v1/payloader/manifests/seal-all', {
            method: 'POST',
            headers,
            body: '{}',
        });
        if (!res.ok) throw await readApiError(res);
        return res.json();
    },

    getManifestExceptions: async (_token: string, limit: number = 50, offset: number = 0) => {
        const res = await authFetch(`/v1/payloader/manifest-exceptions?limit=${limit}&offset=${offset}`);
        if (!res.ok) throw new Error('Failed to fetch manifest exceptions');
        return res.json();
    },

    reassignOrder: async (
        _token: string,
        orderId: string,
        toDriverId: string,
        toManifestId?: string,
        reason: string = 'payload-redispatch',
        idempotencyKey?: string,
    ) => {
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const body: Record<string, string> = {
            order_id: orderId,
            to_driver_id: toDriverId,
            reason,
        };
        if (toManifestId) body.to_manifest_id = toManifestId;
        const res = await authFetch('/v1/payloader/reassign-order', {
            method: 'POST',
            headers,
            body: JSON.stringify(body),
        });
        if (!res.ok) throw new Error('Failed to reassign order');
        return res.json();
    },

    getClientPolicy: async (platform: string, version: string, channel: string = 'production') => {
        const params = new URLSearchParams({
            role: 'PAYLOAD',
            platform,
            version,
            channel,
        });
        const res = await fetch(`${API_BASE}/v1/platform/client-policy?${params.toString()}`, {
            headers: { 'X-Trace-Id': crypto.randomUUID() },
        });
        if (!res.ok) throw new Error('Failed to fetch client policy');
        return res.json() as Promise<{
            outdated: boolean;
            force_update: boolean;
            minimum_version: string;
            defer_reason?: string;
        }>;
    },

    registerDeviceToken: async (token: string, platform: string) => {
        const res = await authFetch('/v1/user/device-token', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token, platform }),
        });
        if (!res.ok) throw new Error('Failed to register device token');
        return res.json();
    },
};
