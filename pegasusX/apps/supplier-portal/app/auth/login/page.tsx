"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { supplierLoginSchema } from "@pegasusx/validation";
import { createSupplierApi } from "@/lib/api";
import { persistSession } from "@/lib/auth";

export default function SupplierLoginPage() {
  const router = useRouter();
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const parsed = supplierLoginSchema.safeParse({ phone: phone.trim(), password });
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message ?? "Invalid credentials");
      return;
    }
    setLoading(true);
    try {
      const api = createSupplierApi();
      const resp = await api.loginSupplier(parsed.data);
      if (resp.token) {
        persistSession(resp.token, resp.refresh_token);
      }
      router.replace(resp.is_configured ? "/dashboard" : "/setup/billing");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="auth-card space-y-5">
      <div>
        <h1 className="md-typescale-headline-medium" style={{ margin: 0 }}>
          Supplier sign in
        </h1>
        <p className="desk-page-subtitle">
          Use the phone and password from your supplier registration.
        </p>
      </div>

      {error ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      ) : null}

      <label className="block space-y-1">
        <span className="md-typescale-label-medium">Phone</span>
        <input
          className="md-input-outlined"
          type="tel"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          required
          autoComplete="tel"
        />
      </label>

      <label className="block space-y-1">
        <span className="md-typescale-label-medium">Password</span>
        <input
          className="md-input-outlined"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          autoComplete="current-password"
        />
      </label>

      <button type="submit" className="md-btn md-btn-filled w-full" disabled={loading}>
        {loading ? "Signing in…" : "Sign in"}
      </button>

      <p className="md-typescale-body-small text-center" style={{ color: "var(--desk-text-secondary)" }}>
        New supplier?{" "}
        <Link href="/auth/register" className="underline">
          Register
        </Link>
      </p>
    </form>
  );
}
