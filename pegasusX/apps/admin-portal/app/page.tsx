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

type Tab = "tenants" | "flags" | "audit" | "match" | "partner";

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
          Break-glass platform governance. Paste a PLATFORM_ADMIN bearer token to continue.
        </p>
        <TokenForm onSubmit={setToken} />
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
  const [tab, setTab] = useState<Tab>("tenants");
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
        {(["tenants", "flags", "audit", "match", "partner"] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 text-sm font-medium capitalize ${
              tab === t ? "border-b-2 border-indigo-600 text-indigo-700" : "text-gray-500 hover:text-gray-800"
            }`}
          >
            {t === "match" ? "Match queue" : t === "partner" ? "Partner / dunning" : t}
          </button>
        ))}
      </nav>
      {tab === "tenants" && <TenantsPanel token={token} refreshKey={refreshKey} />}
      {tab === "flags" && <FlagsPanel token={token} />}
      {tab === "audit" && <AuditPanel token={token} refreshKey={refreshKey} />}
      {tab === "match" && <MatchQueuePanel token={token} />}
      {tab === "partner" && <PartnerPanel token={token} />}
    </main>
  );
}

function TokenForm({ onSubmit }: { onSubmit: (t: string) => void }) {
  const [v, setV] = useState("");
  return (
    <form
      className="mt-6 space-y-3"
      onSubmit={(e) => {
        e.preventDefault();
        if (v.trim()) onSubmit(v);
      }}
    >
      <input
        type="password"
        value={v}
        onChange={(e) => setV(e.target.value)}
        placeholder="PLATFORM_ADMIN token"
        className="w-full rounded border px-3 py-2 text-sm"
        autoFocus
      />
      <button type="submit" className="w-full rounded bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700">
        Continue
      </button>
    </form>
  );
}
