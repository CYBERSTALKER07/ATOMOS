"use client";

import { useCallback, useEffect, useState } from "react";
import { api, type PartnerKey } from "@/lib/api";

export default function PartnerPanel({ token }: { token: string }) {
  const [tenantType, setTenantType] = useState("SUPPLIER");
  const [tenantId, setTenantId] = useState("");
  const [keys, setKeys] = useState<PartnerKey[]>([]);
  const [as2, setAs2] = useState<Record<string, unknown> | null>(null);
  const [sftp, setSftp] = useState<Record<string, unknown> | null>(null);
  const [coa, setCoa] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [dunningMsg, setDunningMsg] = useState("");

  const load = useCallback(async () => {
    if (!tenantId.trim()) {
      setError("tenant_id required for partner lookups");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const [k, a, s, c] = await Promise.all([
        api.listPartnerKeys(token, tenantType, tenantId.trim()),
        api.getPartnerAs2(token, tenantType, tenantId.trim()),
        api.getPartnerSftp(token, tenantType, tenantId.trim()),
        api.getPartnerCoa(token, tenantType, tenantId.trim()),
      ]);
      setKeys(k.keys || []);
      setAs2(a);
      setSftp(s);
      setCoa(c);
    } catch (e) {
      setError(e instanceof Error ? e.message : "load failed");
    } finally {
      setLoading(false);
    }
  }, [token, tenantType, tenantId]);

  useEffect(() => {
    /* wait for explicit load */
  }, []);

  const runDunning = async () => {
    setDunningMsg("");
    try {
      const r = await api.runDunningOnce(token);
      setDunningMsg(JSON.stringify(r));
    } catch (e) {
      setDunningMsg(e instanceof Error ? e.message : "dunning failed");
    }
  };

  return (
    <section className="space-y-6">
      <div className="flex flex-wrap items-end gap-3">
        <label className="text-sm">
          <span className="mb-1 block text-gray-600">Tenant type</span>
          <select value={tenantType} onChange={(e) => setTenantType(e.target.value)} className="rounded border px-2 py-1 text-sm">
            <option value="SUPPLIER">SUPPLIER</option>
            <option value="RETAILER">RETAILER</option>
          </select>
        </label>
        <label className="text-sm">
          <span className="mb-1 block text-gray-600">Tenant ID</span>
          <input
            value={tenantId}
            onChange={(e) => setTenantId(e.target.value)}
            placeholder="supplier_… / retailer_…"
            className="w-64 rounded border px-2 py-1 text-sm font-mono"
          />
        </label>
        <button onClick={() => void load()} className="rounded border px-3 py-1 text-sm hover:bg-gray-100">
          Load partner config
        </button>
      </div>
      {error && <p className="rounded bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}
      {loading && <p className="text-sm text-gray-500">Loading…</p>}

      <div>
        <h2 className="mb-2 text-sm font-semibold">API keys</h2>
        {keys.length === 0 ? (
          <p className="text-sm text-gray-500">No keys (load a tenant).</p>
        ) : (
          <ul className="space-y-2 text-sm">
            {keys.map((k) => (
              <li key={k.key_id} className="flex items-center justify-between border-b py-2">
                <span className="font-mono text-xs">
                  {k.key_prefix}… · {k.status} · {(k.scopes || []).join(",")}
                </span>
                {k.status !== "REVOKED" && (
                  <button
                    className="rounded border px-2 py-0.5 text-xs hover:bg-red-50"
                    onClick={() =>
                      void api.revokePartnerKey(token, k.key_id).then(load).catch((e) => setError(String(e)))
                    }
                  >
                    Revoke
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <ConfigCard title="AS2" data={as2} />
        <ConfigCard title="SFTP" data={sftp} />
        <ConfigCard title="COA" data={coa} />
      </div>

      <div className="rounded border p-4">
        <h2 className="mb-2 text-sm font-semibold">AR dunning</h2>
        <p className="mb-3 text-xs text-gray-500">Ops trigger: POST /v1/admin/ar/dunning/run-once</p>
        <button onClick={() => void runDunning()} className="rounded bg-gray-900 px-3 py-1.5 text-sm text-white">
          Run dunning once
        </button>
        {dunningMsg && <pre className="mt-3 overflow-auto rounded bg-gray-50 p-2 text-xs">{dunningMsg}</pre>}
      </div>
    </section>
  );
}

function ConfigCard({ title, data }: { title: string; data: Record<string, unknown> | null }) {
  return (
    <div className="rounded border p-3">
      <h3 className="mb-2 text-sm font-semibold">{title}</h3>
      {!data ? (
        <p className="text-xs text-gray-500">—</p>
      ) : (
        <pre className="overflow-auto text-xs text-gray-700">{JSON.stringify(data, null, 2)}</pre>
      )}
    </div>
  );
}
