"use client";

import { useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";

/** Pegaus support/admin console surface: permanent disable with ticket linkage. */
export default function CreditAdminDisablePage() {
  const [supplierId, setSupplierId] = useState("");
  const [retailerId, setRetailerId] = useState("");
  const [ticketId, setTicketId] = useState("");
  const [reason, setReason] = useState("");
  const [mode, setMode] = useState<"relationship" | "program">("relationship");
  const [msg, setMsg] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    setMsg(null);
    try {
      const path =
        mode === "program"
          ? `/v1/admin/credit-program/${encodeURIComponent(supplierId)}/disable`
          : `/v1/admin/credit-relationships/${encodeURIComponent(supplierId)}/${encodeURIComponent(retailerId)}/disable`;
      const res = await supplierFetch(path, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ticket_id: ticketId, reason }),
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(body.error || `status_${res.status}`);
      setMsg("disabled_ok");
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageChrome
      title="Admin: disable credit"
      description="Support-only permanent disable. Requires ticket id + reason. Open AR remains collectible."
    >
      <div className="space-y-3 max-w-lg text-sm">
        <label className="block">
          Mode
          <select className="ml-2 border rounded px-2 py-1" value={mode} onChange={(e) => setMode(e.target.value as typeof mode)}>
            <option value="relationship">Relationship</option>
            <option value="program">Program</option>
          </select>
        </label>
        <label className="block">
          Supplier ID
          <input className="mt-1 w-full border rounded px-2 py-1" value={supplierId} onChange={(e) => setSupplierId(e.target.value)} />
        </label>
        {mode === "relationship" ? (
          <label className="block">
            Retailer ID
            <input className="mt-1 w-full border rounded px-2 py-1" value={retailerId} onChange={(e) => setRetailerId(e.target.value)} />
          </label>
        ) : null}
        <label className="block">
          Ticket ID
          <input className="mt-1 w-full border rounded px-2 py-1" value={ticketId} onChange={(e) => setTicketId(e.target.value)} />
        </label>
        <label className="block">
          Reason
          <textarea className="mt-1 w-full border rounded px-2 py-1" value={reason} onChange={(e) => setReason(e.target.value)} rows={3} />
        </label>
        <button
          type="button"
          className="rounded-lg bg-red-700 text-white px-3 py-1.5 disabled:opacity-40"
          disabled={busy || !supplierId || !ticketId || !reason || (mode === "relationship" && !retailerId)}
          onClick={() => void submit()}
        >
          Disable permanently
        </button>
        {msg ? <p className="text-sm">{msg}</p> : null}
      </div>
    </PageChrome>
  );
}
