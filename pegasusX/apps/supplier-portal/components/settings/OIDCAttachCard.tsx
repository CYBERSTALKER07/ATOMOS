"use client";

import { useEffect, useState } from "react";
import { deleteSupplierOIDC, getSupplierOIDC, oidcCallbackUrl, putSupplierOIDC, type OIDCConfig } from "@/lib/oidc";

export function OIDCAttachCard() {
  const [cfg, setCfg] = useState<OIDCConfig | null>(null);
  const [issuer, setIssuer] = useState("");
  const [clientId, setClientId] = useState("");
  const [audience, setAudience] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void getSupplierOIDC()
      .then((row) => {
        setCfg(row);
        if (row) {
          setIssuer(row.issuer);
          setClientId(row.client_id);
          setAudience(row.audience || "");
        }
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "load failed"));
  }, []);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const row = await putSupplierOIDC({
        issuer,
        client_id: clientId,
        audience: audience || undefined,
        redirect_uri: oidcCallbackUrl(),
      });
      setCfg(row);
      setNotice("IdP attached. Client secret is not stored — use PKCE / id_token.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "save failed");
    } finally {
      setBusy(false);
    }
  }

  async function detach() {
    setBusy(true);
    setError(null);
    try {
      await deleteSupplierOIDC();
      setCfg(null);
      setNotice("IdP detached. Password login is unchanged.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "detach failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="space-y-3 rounded-lg border border-[var(--desk-border)] p-4">
      <h2 className="text-lg font-semibold">Company IdP (OIDC)</h2>
      <p className="text-sm text-[var(--desk-text-secondary)]">
        Optional SSO for this supplier org. Staff and driver apps keep HS256 passwords. SAML/SCIM are not in this slice.
        {cfg?.enabled ? " IdP is attached." : " No IdP attached."}
      </p>
      {error && <p className="text-sm text-red-600">{error}</p>}
      {notice && <p className="text-sm text-emerald-700">{notice}</p>}
      <form className="space-y-2" onSubmit={save}>
        <input
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Issuer (https://…)"
          value={issuer}
          onChange={(e) => setIssuer(e.target.value)}
          required
        />
        <input
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Client ID"
          value={clientId}
          onChange={(e) => setClientId(e.target.value)}
          required
        />
        <input
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Audience (optional; defaults to client ID)"
          value={audience}
          onChange={(e) => setAudience(e.target.value)}
        />
        <div className="flex gap-2">
          <button type="submit" disabled={busy} className="rounded bg-[var(--desk-text)] px-3 py-2 text-sm text-white">
            {busy ? "Saving…" : "Attach IdP"}
          </button>
          {cfg && (
            <button type="button" disabled={busy} onClick={() => void detach()} className="rounded border px-3 py-2 text-sm">
              Detach
            </button>
          )}
        </div>
      </form>
    </section>
  );
}
