"use client";

import React, { useState, useEffect, useCallback, type ReactNode } from "react";
import { useAdminToken } from "@/lib/session";
import { api } from "@/lib/api";

type MfaGate =
  | { kind: "ok" }
  | { kind: "loading" }
  | { kind: "need_enroll" }
  | { kind: "enroll"; secret: string; otpauth: string }
  | { kind: "verify" }
  | { kind: "error"; message: string };

export function AdminAuthGate({ children }: { children: ReactNode }) {
  const { token, setToken, isLoading, login } = useAdminToken();
  const [mfa, setMfa] = useState<MfaGate>({ kind: "loading" });
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);

  const refreshMfa = useCallback(async (tok: string) => {
    setMfa({ kind: "loading" });
    try {
      const st = await api.mfaStatus(tok);
      if (st.verified || (!st.enrolled && !st.required)) {
        setMfa({ kind: "ok" });
        return;
      }
      if (!st.enrolled && st.required) {
        setMfa({ kind: "need_enroll" });
        return;
      }
      setMfa({ kind: "verify" });
    } catch (err) {
      setMfa({
        kind: "error",
        message: err instanceof Error ? err.message : "MFA status check failed",
      });
    }
  }, []);

  useEffect(() => {
    if (!token) {
      setMfa({ kind: "loading" });
      return;
    }
    void refreshMfa(token);
  }, [token, refreshMfa]);

  if (isLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center p-6 bg-gray-50">
        <div className="text-center space-y-2">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-600 border-t-transparent mx-auto" />
          <p className="text-xs text-gray-500 font-medium">Initializing PegasusX session…</p>
        </div>
      </main>
    );
  }

  if (!token) {
    return (
      <main className="flex min-h-screen items-center justify-center p-6 bg-gray-50">
        <div className="w-full max-w-md rounded-2xl border bg-white p-8 shadow-sm space-y-4">
          <div className="text-center space-y-1">
            <span className="inline-flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-600 font-bold text-white text-lg">
              P
            </span>
            <h1 className="text-xl font-bold text-gray-900">PegasusX Control Tower</h1>
            <p className="text-xs text-gray-500">
              Platform Owner Governance Console. Sign in with PLATFORM_ADMIN credentials.
            </p>
          </div>
          <LoginForm onLogin={login} onPasteToken={setToken} />
        </div>
      </main>
    );
  }

  if (mfa.kind === "loading") {
    return (
      <main className="flex min-h-screen items-center justify-center p-6 bg-gray-50">
        <div className="text-center space-y-2">
          <div className="h-6 w-6 animate-spin rounded-full border-3 border-indigo-600 border-t-transparent mx-auto" />
          <p className="text-xs text-gray-500 font-medium">Verifying MFA claims…</p>
        </div>
      </main>
    );
  }

  if (mfa.kind === "error") {
    return (
      <main className="flex min-h-screen items-center justify-center p-6 bg-gray-50">
        <div className="w-full max-w-md rounded-2xl border bg-white p-6 shadow-sm space-y-4 text-center">
          <div className="text-red-600 text-sm font-semibold">{mfa.message}</div>
          <div className="flex justify-center gap-3">
            <button
              type="button"
              className="rounded border px-4 py-1.5 text-xs font-semibold hover:bg-gray-50"
              onClick={() => void refreshMfa(token)}
            >
              Retry
            </button>
            <button
              type="button"
              className="rounded border px-4 py-1.5 text-xs font-semibold text-red-600 hover:bg-red-50"
              onClick={() => setToken("")}
            >
              Sign out
            </button>
          </div>
        </div>
      </main>
    );
  }

  if (mfa.kind === "need_enroll") {
    return (
      <main className="flex min-h-screen items-center justify-center p-6 bg-gray-50">
        <div className="w-full max-w-md rounded-2xl border bg-white p-8 shadow-sm space-y-4">
          <h1 className="text-lg font-bold text-gray-900">MFA Enrollment Required</h1>
          <p className="text-xs text-gray-600">
            Production platform governance mandates Time-based One-Time Password (TOTP) enforcement.
          </p>
          <button
            type="button"
            disabled={busy}
            className="w-full rounded-lg bg-indigo-600 px-4 py-2.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            onClick={async () => {
              setBusy(true);
              try {
                const en = await api.mfaEnroll(token);
                setMfa({ kind: "enroll", secret: en.secret, otpauth: en.otpauth_url });
              } catch (err) {
                setMfa({
                  kind: "error",
                  message: err instanceof Error ? err.message : "Enroll failed",
                });
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? "Generating secret…" : "Start Authenticator Setup"}
          </button>
        </div>
      </main>
    );
  }

  if (mfa.kind === "enroll" || mfa.kind === "verify") {
    return (
      <main className="flex min-h-screen items-center justify-center p-6 bg-gray-50">
        <div className="w-full max-w-md rounded-2xl border bg-white p-8 shadow-sm space-y-4">
          <h1 className="text-lg font-bold text-gray-900">
            {mfa.kind === "enroll" ? "Bind Authenticator" : "Two-Factor Verification"}
          </h1>
          {mfa.kind === "enroll" && (
            <div className="space-y-2 rounded-lg bg-gray-50 p-3 text-xs">
              <p className="text-gray-600">Enter this key into your authenticator app (Google Auth, 1Password):</p>
              <p className="font-mono font-bold select-all break-all text-indigo-700">{mfa.secret}</p>
            </div>
          )}
          <form
            className="space-y-3"
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              try {
                if (mfa.kind === "enroll") {
                  const res = await api.mfaConfirm(token, code.trim());
                  if (res.token) setToken(res.token);
                  setMfa({ kind: "ok" });
                } else {
                  const res = await api.mfaVerify(token, code.trim());
                  if (res.token) setToken(res.token);
                  setMfa({ kind: "ok" });
                }
              } catch (err) {
                setMfa({
                  kind: "error",
                  message: err instanceof Error ? err.message : "Code verification failed",
                });
              } finally {
                setBusy(false);
              }
            }}
          >
            <label className="block text-xs font-medium text-gray-700">6-Digit TOTP Code</label>
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              maxLength={6}
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
              className="w-full rounded-lg border px-3 py-2 text-center text-lg font-mono tracking-widest"
              placeholder="000000"
              autoFocus
            />
            <button
              type="submit"
              disabled={busy || code.trim().length !== 6}
              className="w-full rounded-lg bg-indigo-600 px-4 py-2.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {busy ? "Verifying…" : "Confirm & Enter Tower"}
            </button>
          </form>
        </div>
      </main>
    );
  }

  return <>{children}</>;
}

function LoginForm({
  onLogin,
  onPasteToken,
}: {
  onLogin: (s?: string, p?: string, d?: string) => Promise<{ ok: boolean; error?: string }>;
  onPasteToken: (t: string) => void;
}) {
  const [subject, setSubject] = useState("");
  const [password, setPassword] = useState("");
  const [paste, setPaste] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const allowPaste =
    process.env.NEXT_PUBLIC_ALLOW_TOKEN_PASTE === "true" || process.env.NODE_ENV === "development";

  return (
    <div className="space-y-4 pt-2">
      <form
        className="space-y-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setErr("");
          const res = await onLogin(subject.trim(), password);
          if (!res.ok) {
            setErr(res.error || "Login failed");
          }
          setBusy(false);
        }}
      >
        <div>
          <label className="block text-xs font-medium text-gray-700">Username / Email</label>
          <input
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            className="mt-1 w-full rounded-lg border px-3 py-2 text-xs"
            placeholder="admin@pegasusx.uz"
            autoFocus
            autoComplete="username"
            required
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-700">Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="mt-1 w-full rounded-lg border px-3 py-2 text-xs"
            autoComplete="current-password"
            required
          />
        </div>
        {err && <div className="rounded-lg bg-red-50 p-2 text-xs text-red-700 font-medium">{err}</div>}
        <button
          type="submit"
          disabled={busy || !subject.trim() || !password}
          className="w-full rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors"
        >
          {busy ? "Authenticating…" : "Sign In to Control Tower"}
        </button>
      </form>

      {allowPaste && (
        <div className="border-t pt-3">
          <p className="text-[11px] text-gray-500 mb-1.5 font-medium">Break-Glass Development Token</p>
          <div className="flex gap-2">
            <input
              type="password"
              value={paste}
              onChange={(e) => setPaste(e.target.value)}
              placeholder="Paste mint-dev-jwt token"
              className="w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
            />
            <button
              type="button"
              disabled={!paste.trim()}
              onClick={() => onPasteToken(paste.trim())}
              className="rounded-lg border bg-gray-50 px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-100 disabled:opacity-50"
            >
              Apply
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
