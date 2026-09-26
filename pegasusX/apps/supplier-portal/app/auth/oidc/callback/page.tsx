"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { exchangeOIDC } from "@/lib/oidc";

export default function OIDCCallbackPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const hash = typeof window !== "undefined" ? window.location.hash.replace(/^#/, "") : "";
    const params = new URLSearchParams(hash);
    const idToken = params.get("id_token") || "";
    const supplierId = sessionStorage.getItem("oidc_supplier_id") || "";
    const nonce = sessionStorage.getItem("oidc_nonce") || "";
    if (!idToken || !supplierId) {
      setError("Missing id_token or company id");
      return;
    }
    void exchangeOIDC(supplierId, idToken, nonce)
      .then(() => {
        sessionStorage.removeItem("oidc_supplier_id");
        sessionStorage.removeItem("oidc_nonce");
        router.replace("/dashboard");
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : "OIDC sign-in failed");
      });
  }, [router]);

  return (
    <div className="mx-auto max-w-md p-8">
      <h1 className="text-lg font-semibold">Signing in with IdP…</h1>
      {error ? <p className="mt-3 text-sm text-red-600">{error}</p> : <p className="mt-3 text-sm">Please wait.</p>}
    </div>
  );
}
