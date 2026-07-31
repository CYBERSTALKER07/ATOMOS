"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import StatusBadge from "@/components/StatusBadge";

const api = createSupplierApi();

type CreditNoteRow = {
  credit_note_id: string;
  order_id: string;
  type: string;
  status: string;
  reason_code: string;
  total_gross_minor: number;
  created_at: string;
};

export default function CreditNotesPage() {
  const [rows, setRows] = useState<CreditNoteRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [orderId, setOrderId] = useState("");
  const [reasonCode, setReasonCode] = useState("MANUAL_ADJUSTMENT");
  const [reasonText, setReasonText] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await api.listCreditNotes({ status: "DRAFT", limit: 100 });
      setRows((resp.credit_notes ?? []) as CreditNoteRow[]);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const issue = async (id: string) => {
    try {
      await api.issueCreditNote(id);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "issue_failed");
    }
  };

  const createManual = async () => {
    if (!orderId.trim()) return;
    try {
      await api.createCreditNote({
        order_id: orderId.trim(),
        reason_code: reasonCode.trim() || "MANUAL_ADJUSTMENT",
        reason_text: reasonText.trim(),
        lines: [],
      });
      setOrderId("");
      setReasonText("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "create_failed");
    }
  };

  return (
    <PageChrome
      title="Credit notes"
      description="Draft credit notes and issue reverse-logistics tasks."
      icon="treasury"
      loading={loading}
      error={error}
    >
      <section className="mb-6 p-4 md-card space-y-3">
        <h2 className="md-typescale-title-medium">Create manual draft</h2>
        <input className="md-input w-full" placeholder="Order ID" value={orderId} onChange={(e) => setOrderId(e.target.value)} />
        <input className="md-input w-full" placeholder="Reason code" value={reasonCode} onChange={(e) => setReasonCode(e.target.value)} />
        <input className="md-input w-full" placeholder="Reason text" value={reasonText} onChange={(e) => setReasonText(e.target.value)} />
        <button type="button" className="md-btn md-btn-filled" onClick={() => void createManual()}>Create draft</button>
      </section>
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {rows.map((row) => (
          <li key={row.credit_note_id} className="p-4">
            <div className="flex flex-wrap gap-2 items-center">
              <span className="font-mono text-sm">{row.credit_note_id}</span>
              <StatusBadge state={row.status} />
              <span>{row.type}</span>
            </div>
            <p className="mt-1">Order {row.order_id} · {row.total_gross_minor.toLocaleString()} minor</p>
            <button type="button" className="md-btn md-btn-outlined mt-2" onClick={() => void issue(row.credit_note_id)}>
              Issue
            </button>
          </li>
        ))}
      </ul>
    </PageChrome>
  );
}
