"use client";

import { useState } from "react";
import { useAdminToken } from "@/lib/session";
import TenantsPanel from "@/components/TenantsPanel";
import FlagsPanel from "@/components/FlagsPanel";
import AuditPanel from "@/components/AuditPanel";

type Tab = "tenants" | "flags" | "audit";

export default function Home() {
  const { token, setToken } = useAdminToken();
  const [tab, setTab] = useState<Tab>("tenants");

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

  return (
    <main className="mx-auto max-w-6xl p-6">
      <header className="mb-6 flex items-center justify-between">
        <h1 className="text-xl font-semibold">PegasusX Admin Console</h1>
        <button
          onClick={() => setToken("")}
          className="rounded border px-3 py-1 text-sm text-gray-600 hover:bg-gray-100"
        >
          Sign out
        </button>
      </header>
      <nav className="mb-4 flex gap-2 border-b">
        {(["tenants", "flags", "audit"] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 text-sm font-medium capitalize ${
              tab === t ? "border-b-2 border-indigo-600 text-indigo-700" : "text-gray-500 hover:text-gray-800"
            }`}
          >
            {t}
          </button>
        ))}
      </nav>
      {tab === "tenants" && <TenantsPanel token={token} />}
      {tab === "flags" && <FlagsPanel token={token} />}
      {tab === "audit" && <AuditPanel token={token} />}
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
