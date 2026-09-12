"use client";

import { useCallback, useEffect, useState } from "react";
import { useAdminToken } from "@/lib/session";
import { api } from "@/lib/api";
import { useAdminWsRefresh } from "@/lib/use-admin-ws-refresh";
import TenantsPanel from "@/components/TenantsPanel";
import FlagsPanel from "@/components/FlagsPanel";
import AuditPanel from "@/components/AuditPanel";
import MatchQueuePanel from "@/components/MatchQueuePanel";
import PartnerPanel from "@/components/PartnerPanel";
import OpsPanel from "@/components/OpsPanel";
import BillingPanel from "@/components/BillingPanel";
import CommandBoard from "@/components/CommandBoard";
import AccuracyPanel from "@/components/AccuracyPanel";

type Tab = "command" | "tenants" | "flags" | "audit" | "match" | "partner" | "ops" | "billing" | "accuracy";

type MfaGate =
  | { kind: "ok" }
  | { kind: "loading" }
  | { kind: "need_enroll" }
  | { kind: "enroll"; secret: string; otpauth: string }
  | { kind: "verify" }
  | { kind: "error"; message: string };

export default function Home() {
  const { token, setToken } = useAdminToken();
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
      setMfa({ kind: "error", message: err instanceof Error ? err.message : "MFA status failed" });
    }
  }, []);

  useEffect(() => {
    if (!token) {
      setMfa({ kind: "loading" });
      return;
    }
    void refreshMfa(token);
  }, [token, refreshMfa]);

  if (!token) {
    return (
      <main className="mx-auto max-w-md p-8">
        <h1 className="text-2xl font-semibold">PegasusX Admin Console</h1>
        <p className="mt-2 text-sm text-gray-600">
          Platform governance. Sign in with PLATFORM_ADMIN credentials (G4). MFA may follow.
        </p>
        <LoginForm onSubmit={setToken} />
      </main>
    );
  }

  if (mfa.kind === "loading") {
    return (
      <main className="mx-auto max-w-md p-8">
        <p className="text-sm text-gray-600">Checking MFA…</p>
      </main>
    );
  }

  if (mfa.kind === "error") {
    return (
      <main className="mx-auto max-w-md p-8 space-y-3">
        <p className="text-sm text-red-700">{mfa.message}</p>
        <button className="rounded border px-3 py-1 text-sm" onClick={() => void refreshMfa(token)}>
          Retry
        </button>
        <button className="rounded border px-3 py-1 text-sm" onClick={() => setToken("")}>
          Sign out
        </button>
      </main>
    );
  }

  if (mfa.kind === "need_enroll") {
    return (
      <main className="mx-auto max-w-md p-8 space-y-4">
        <h1 className="text-xl font-semibold">MFA required</h1>
        <p className="text-sm text-gray-600">
          Production platform-admin sessions require TOTP. Start enrollment to bind an authenticator.
        </p>
        <button
          disabled={busy}
          className="w-full rounded bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          onClick={async () => {
            setBusy(true);
            try {
              const en = await api.mfaEnroll(token);
              setMfa({ kind: "enroll", secret: en.secret, otpauth: en.otpauth_url });
            } catch (err) {
              setMfa({ kind: "error", message: err instanceof Error ? err.message : "Enroll failed" });
            } finally {
              setBusy(false);
            }
          }}
        >
          Start TOTP enrollment
        </button>
        <button className="text-sm text-gray-500 underline" onClick={() => setToken("")}>
          Sign out
        </button>
      </main>
    );
  }

  if (mfa.kind === "enroll" || mfa.kind === "verify") {
    return (
      <main className="mx-auto max-w-md p-8 space-y-4">
        <h1 className="text-xl font-semibold">
          {mfa.kind === "enroll" ? "Enroll authenticator (TOTP)" : "Verify MFA"}
        </h1>
        {mfa.kind === "enroll" ? (
          <div className="space-y-2 text-sm">
            <p className="text-gray-600">Add this secret to your authenticator app, then enter a 6-digit code.</p>
            <code className="block break-all rounded bg-gray-100 p-2 text-xs">{mfa.secret}</code>
            <a className="text-indigo-700 underline break-all text-xs" href={mfa.otpauth}>
              {mfa.otpauth}
            </a>
          </div>
        ) : (
          <p className="text-sm text-gray-600">Enter the 6-digit code from your authenticator.</p>
        )}
        <form
          className="space-y-3"
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            try {
              const res =
                mfa.kind === "enroll"
                  ? await api.mfaConfirm(token, code.trim())
                  : await api.mfaVerify(token, code.trim());
              setCode("");
              setToken(res.token);
              setMfa({ kind: "ok" });
            } catch (err) {
              setMfa({
                kind: "error",
                message: err instanceof Error ? err.message : "MFA failed",
              });
            } finally {
              setBusy(false);
            }
          }}
        >
          <input
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="123456"
            className="w-full rounded border px-3 py-2 text-sm tracking-widest"
            autoFocus
          />
          <button
            type="submit"
            disabled={busy || code.trim().length !== 6}
            className="w-full rounded bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {mfa.kind === "enroll" ? "Confirm enrollment" : "Verify"}
          </button>
        </form>
        <button className="text-sm text-gray-500 underline" onClick={() => setToken("")}>
          Sign out
        </button>
      </main>
    );
  }

  return <AdminConsole token={token} onSignOut={() => setToken("")} />;
}

function AdminConsole({ token, onSignOut }: { token: string; onSignOut: () => void }) {
  const [tab, setTab] = useState<Tab>("command");
  const [refreshKey, setRefreshKey] = useState(0);
  const [live, setLive] = useState(false);

  useAdminWsRefresh(
    token,
    () => {
      setLive(true);
      setRefreshKey((k) => k + 1);
    },
    true,
  );

  return (
    <main className="mx-auto max-w-6xl p-6">
      <header className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-semibold">PegasusX Admin Console</h1>
          <span className="rounded-full border px-2 py-0.5 text-xs text-gray-600">
            {live || refreshKey > 0 ? "WS live" : "WS connecting…"}
          </span>
        </div>
        <button onClick={onSignOut} className="rounded border px-3 py-1 text-sm text-gray-600 hover:bg-gray-100">
          Sign out
        </button>
      </header>
      <nav className="mb-4 flex flex-wrap gap-2 border-b">
        {(["command", "tenants", "flags", "ops", "billing", "accuracy", "match", "partner", "audit"] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-3 py-2 text-sm font-medium capitalize ${
              tab === t ? "border-b-2 border-indigo-600 text-indigo-700" : "text-gray-500 hover:text-gray-800"
            }`}
          >
            {t === "command"
              ? "Command"
              : t === "match"
                ? "Match queue"
                : t === "partner"
                  ? "Partner"
                  : t === "ops"
                    ? "Outbox"
                    : t === "accuracy"
                      ? "Accuracy"
                      : t}
          </button>
        ))}
      </nav>
      {tab === "command" && (
        <CommandBoard token={token} refreshKey={refreshKey} onOpenTab={(next) => setTab(next as Tab)} />
      )}
      {tab === "tenants" && <TenantsPanel token={token} refreshKey={refreshKey} />}
      {tab === "flags" && <FlagsPanel token={token} />}
      {tab === "audit" && <AuditPanel token={token} refreshKey={refreshKey} />}
      {tab === "match" && <MatchQueuePanel token={token} />}
      {tab === "partner" && <PartnerPanel token={token} />}
      {tab === "ops" && <OpsPanel token={token} />}
      {tab === "billing" && <BillingPanel token={token} />}
      {tab === "accuracy" && <AccuracyPanel token={token} />}
    </main>
  );
}

function LoginForm({ onSubmit }: { onSubmit: (t: string) => void }) {
  const [subject, setSubject] = useState("");
  const [password, setPassword] = useState("");
  const [paste, setPaste] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const allowPaste =
    process.env.NEXT_PUBLIC_ALLOW_TOKEN_PASTE === "true" ||
    process.env.NODE_ENV === "development";

  return (
    <div className="mt-6 space-y-6">
      <form
        className="space-y-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setErr("");
          try {
            const res = await api.platformAdminLogin(subject.trim(), password);
            if (!res.token) throw new Error("no_token");
            onSubmit(res.token);
          } catch (ex) {
            setErr(ex instanceof Error ? ex.message : "login_failed");
          } finally {
            setBusy(false);
          }
        }}
      >
        <input
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          placeholder="Subject or email"
          className="w-full rounded border px-3 py-2 text-sm"
          autoFocus
          autoComplete="username"
        />
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Password"
          className="w-full rounded border px-3 py-2 text-sm"
          autoComplete="current-password"
        />
        {err && <p className="text-sm text-red-700">{err}</p>}
        <button
          type="submit"
          disabled={busy || !subject.trim() || !password}
          className="w-full rounded bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {busy ? "Signing in…" : "Sign in"}
        </button>
      </form>
      {allowPaste && (
        <form
          className="space-y-2 border-t pt-4"
          onSubmit={(e) => {
            e.preventDefault();
            if (paste.trim()) onSubmit(paste);
          }}
        >
          <p className="text-xs text-gray-500">Dev break-glass: paste mint-dev-jwt token</p>
          <input
            type="password"
            value={paste}
            onChange={(e) => setPaste(e.target.value)}
            placeholder="PLATFORM_ADMIN token"
            className="w-full rounded border px-3 py-2 text-sm"
          />
          <button type="submit" className="w-full rounded border px-3 py-2 text-sm">
            Continue with token
          </button>
        </form>
      )}
    </div>
  );
}
