"use client";

import { useCallback, useEffect, useState } from "react";
import { Loader2, HandHelping } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

type Section = { section_id: string; name: string };
type Ticket = {
  ticket_id: string;
  section_id: string;
  note: string;
  status: string;
  created_at?: string;
  claimed_by_user_id?: string;
};

export default function AssistPage() {
  const [sections, setSections] = useState<Section[]>([]);
  const [sectionId, setSectionId] = useState("");
  const [note, setNote] = useState("");
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [banner, setBanner] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const [secRes, tRes] = await Promise.all([
        apiFetch("/v1/retailer/sections"),
        apiFetch("/v1/retailer/assist/tickets"),
      ]);
      if (secRes.ok) {
        const json = (await secRes.json()) as { items?: Section[] };
        const items = json.items ?? [];
        setSections(items);
        if (!sectionId && items[0]) setSectionId(items[0].section_id);
      }
      if (tRes.ok) {
        const json = (await tRes.json()) as { items?: Ticket[] };
        setTickets(json.items ?? []);
      }
    } catch {
      /* ignore */
    }
  }, [sectionId]);

  useEffect(() => {
    void load();
  }, [load]);

  const create = async () => {
    if (!sectionId || !note.trim()) return;
    setBusy(true);
    try {
      const res = await apiFetch("/v1/retailer/assist/tickets", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `assist-${Date.now()}`,
        },
        body: JSON.stringify({ section_id: sectionId, note: note.trim() }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error((json as { error?: string }).error || "create_failed");
      setNote("");
      setBanner("Ticket opened (CUSTOMER_ASSIST pack auto-enabled if deps met)");
      await load();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Create failed");
    } finally {
      setBusy(false);
    }
  };

  const act = async (id: string, action: "claim" | "complete" | "cancel") => {
    setBusy(true);
    try {
      const res = await apiFetch(`/v1/retailer/assist/tickets/${id}/${action}`, {
        method: "POST",
      });
      if (!res.ok) {
        const json = await res.json().catch(() => ({}));
        throw new Error((json as { error?: string }).error || action + "_failed");
      }
      setBanner(`Ticket ${action}ed`);
      await load();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Action failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <PageChrome
      title="Floor assist"
      description="Customer help queue by section. Claim and complete as section lead."
    >
      <div className="mx-auto flex max-w-3xl flex-col gap-6 p-4">
        {banner && (
          <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm">
            {banner}
          </div>
        )}
        <section className="rounded-xl border border-border bg-card p-4">
          <div className="mb-3 flex items-center gap-2">
            <HandHelping className="h-5 w-5 text-muted-foreground" />
            <h2 className="font-semibold">New ticket</h2>
          </div>
          <select
            className="mb-2 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
            value={sectionId}
            onChange={(e) => setSectionId(e.target.value)}
          >
            {sections.length === 0 && <option value="">Create a section first</option>}
            {sections.map((s) => (
              <option key={s.section_id} value={s.section_id}>
                {s.name}
              </option>
            ))}
          </select>
          <textarea
            className="mb-2 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
            rows={2}
            placeholder="What help is needed?"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          <button
            type="button"
            disabled={busy || !sectionId}
            onClick={() => void create()}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Open ticket
          </button>
        </section>

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 font-semibold">Queue</h2>
          {tickets.length === 0 ? (
            <p className="text-sm text-muted-foreground">No tickets.</p>
          ) : (
            <ul className="space-y-3">
              {tickets.map((t) => (
                <li
                  key={t.ticket_id}
                  className="rounded-lg border border-border px-3 py-2 text-sm"
                >
                  <div className="flex flex-wrap justify-between gap-2">
                    <span className="font-medium">{t.status}</span>
                    <span className="text-muted-foreground text-xs">
                      {t.created_at?.slice(0, 16)}
                    </span>
                  </div>
                  <p className="mt-1">{t.note}</p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {t.status === "OPEN" && (
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => void act(t.ticket_id, "claim")}
                        className="rounded-md border border-border px-2 py-1 text-xs"
                      >
                        Claim
                      </button>
                    )}
                    {(t.status === "OPEN" || t.status === "CLAIMED") && (
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => void act(t.ticket_id, "complete")}
                        className="rounded-md border border-border px-2 py-1 text-xs"
                      >
                        Complete
                      </button>
                    )}
                    {t.status !== "DONE" && t.status !== "CANCELLED" && (
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => void act(t.ticket_id, "cancel")}
                        className="rounded-md border border-border px-2 py-1 text-xs"
                      >
                        Cancel
                      </button>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </PageChrome>
  );
}
