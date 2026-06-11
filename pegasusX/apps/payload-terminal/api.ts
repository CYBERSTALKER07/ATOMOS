export const API_BASE = (process.env.EXPO_PUBLIC_API_URL?.trim() || '') ||
  (__DEV__ ? 'http://localhost:8180' : 'https://api.pegasus.uz');

/**
 * Payload Terminal API 
 * Gap-hunter pass endpoints. Use these for missing functionalities.
 */
export const PayloadTerminalApi = {
    // ── Supplier Manifests ───────────────────────────────────────────────────
    getSupplierManifests: async (token: string, state: string = 'DRAFT') => {
        const res = await fetch(`${API_BASE}/v1/supplier/manifests?state=${state}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (!res.ok) throw new Error('Failed to fetch supplier manifests');
        return res.json();
    },

    getSupplierManifestDetail: async (token: string, manifestId: string) => {
        const res = await fetch(`${API_BASE}/v1/supplier/manifests/${manifestId}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (!res.ok) throw new Error('Failed to fetch supplier manifest detail');
        return res.json();
    },

    supplierStartLoading: async (token: string, manifestId: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = { 'Authorization': `Bearer ${token}` };
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await fetch(`${API_BASE}/v1/supplier/manifests/${manifestId}/start-loading`, {
            method: 'POST',
            headers
        });
        if (!res.ok) throw new Error('Failed to start loading supplier manifest');
        return res.json();
    },

    supplierInjectOrder: async (token: string, manifestId: string, orderId: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = { 
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        };
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await fetch(`${API_BASE}/v1/supplier/manifests/${manifestId}/inject-order`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ order_id: orderId })
        });
        if (!res.ok) throw new Error('Failed to inject order to supplier manifest');
        return res.json();
    },

    supplierSealManifest: async (token: string, manifestId: string, idempotencyKey?: string) => {
        const headers: Record<string, string> = { 'Authorization': `Bearer ${token}` };
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const res = await fetch(`${API_BASE}/v1/supplier/manifests/${manifestId}/seal`, {
            method: 'POST',
            headers
        });
        if (!res.ok) throw new Error('Failed to seal supplier manifest');
        return res.json();
    },

    // ── Payloader Additions ──────────────────────────────────────────────────
    getManifestExceptions: async (token: string, limit: number = 50, offset: number = 0) => {
        const res = await fetch(`${API_BASE}/v1/payloader/manifest-exceptions?limit=${limit}&offset=${offset}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (!res.ok) throw new Error('Failed to fetch manifest exceptions');
        return res.json();
    },

    reassignOrder: async (
        token: string,
        orderId: string,
        toDriverId: string,
        toManifestId?: string,
        reason: string = 'payload-redispatch',
        idempotencyKey?: string,
    ) => {
        const headers: Record<string, string> = {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
        };
        if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
        const body: Record<string, string> = {
            order_id: orderId,
            to_driver_id: toDriverId,
            reason,
        };
        if (toManifestId) body.to_manifest_id = toManifestId;
        const res = await fetch(`${API_BASE}/v1/payloader/reassign-order`, {
            method: 'POST',
            headers,
            body: JSON.stringify(body),
        });
        if (!res.ok) throw new Error('Failed to reassign order');
        return res.json();
    },
};
