import { supplierApiBaseUrl, supplierFetch, persistSession } from "@/lib/auth";

export type OIDCConfig = {
  supplier_id: string;
  issuer: string;
  client_id: string;
  audience?: string;
  authorization_endpoint?: string;
  redirect_uri?: string;
  enabled: boolean;
};

export type OIDCDiscovery = {
  enabled: boolean;
  supplier_id: string;
  issuer: string;
  client_id: string;
  redirect_uri?: string;
  authorization_url: string;
};

export function oidcCallbackUrl(): string {
  if (typeof window === "undefined") return "";
  return `${window.location.origin}/auth/oidc/callback`;
}

export async function discoverOIDC(supplierId: string, nonce: string): Promise<OIDCDiscovery> {
  const base = supplierApiBaseUrl();
  const q = new URLSearchParams({
    supplier_id: supplierId,
    nonce,
    redirect_uri: oidcCallbackUrl(),
  });
  const res = await fetch(`${base}/v1/auth/oidc/discovery?${q.toString()}`);
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || "IdP is not attached for this company");
  }
  return (await res.json()) as OIDCDiscovery;
}

export async function exchangeOIDC(supplierId: string, idToken: string, nonce: string): Promise<void> {
  const base = supplierApiBaseUrl();
  const res = await fetch(`${base}/v1/auth/oidc/exchange`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ supplier_id: supplierId, id_token: idToken, nonce }),
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || "OIDC exchange failed");
  }
  const data = (await res.json()) as { token?: string; refresh_token?: string };
  if (!data.token) throw new Error("OIDC exchange returned no token");
  persistSession(data.token, data.refresh_token);
}

export async function getSupplierOIDC(): Promise<OIDCConfig | null> {
  const res = await supplierFetch("/v1/supplier/oidc");
  if (res.status === 404) return null;
  if (!res.ok) throw new Error("Failed to load OIDC settings");
  return (await res.json()) as OIDCConfig;
}

export async function putSupplierOIDC(body: {
  issuer: string;
  client_id: string;
  audience?: string;
  authorization_endpoint?: string;
  redirect_uri?: string;
}): Promise<OIDCConfig> {
  const res = await supplierFetch("/v1/supplier/oidc", {
    method: "PUT",
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error || "Failed to attach IdP");
  }
  return (await res.json()) as OIDCConfig;
}

export async function deleteSupplierOIDC(): Promise<void> {
  const res = await supplierFetch("/v1/supplier/oidc", { method: "DELETE" });
  if (!res.ok && res.status !== 404) throw new Error("Failed to detach IdP");
}
