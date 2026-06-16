import { authFetch } from './authSession';

export const API_BASE = (process.env.EXPO_PUBLIC_API_URL?.trim() || '') ||
  (__DEV__ ? 'http://localhost:8180' : 'https://api.pegasus.uz');

/**
 * Payload Terminal API
 * Gap-hunter pass endpoints. Authenticated calls use authFetch (401 → refresh).
 */
export const PayloadTerminalApi = {
    getSupplierManifests: async (_token: string, state: string = 'DRAFT') => {
        const res = await authFetch(`/v1/supplier/manifests?state=${state}`);
        if (!res.ok) throw new Error('Failed to fetch supplier manifests');
        return res.json();
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
        if (!res.ok) throw new Error('Failed to seal completed manifests');
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
