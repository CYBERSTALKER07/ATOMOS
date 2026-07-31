"use client";

import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";

type Pref = {
  event_type: string;
  channel: string;
  enabled: boolean;
  quiet_from?: string;
  quiet_to?: string;
};

const DEFAULT_EVENTS = [
  { event_type: "cash_reconciliation.created", channel: "PUSH" },
  { event_type: "cash_reconciliation.escalation", channel: "PUSH" },
  { event_type: "credit_note.created", channel: "PUSH" },
  { event_type: "credit.score.updated", channel: "EMAIL" },
];

export default function NotificationPreferencesPage() {
  const [prefs, setPrefs] = useState<Pref[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await supplierFetch("/v1/user/notification-preferences");
      if (!res.ok) throw new Error(`prefs_${res.status}`);
      const body = (await res.json()) as { preferences?: Pref[] };
      const existing = body.preferences ?? [];
      if (existing.length === 0) {
        setPrefs(DEFAULT_EVENTS.map((e) => ({ ...e, enabled: true })));
      } else {
        setPrefs(existing);
      }
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

  const save = async () => {
    setSaved(false);
    try {
      const res = await supplierFetch("/v1/user/notification-preferences", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ preferences: prefs }),
      });
      if (!res.ok) throw new Error(`save_${res.status}`);
      setSaved(true);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_failed");
    }
  };

  return (
    <PageChrome title="Notification preferences" description="Channel and quiet-hour rules per event type." loading={loading} error={error}>
      <ul className="md-card divide-y">
        {prefs.map((p, idx) => (
          <li key={`${p.event_type}:${p.channel}`} className="p-4 flex flex-wrap gap-4 items-center">
            <span className="font-mono text-xs">{p.event_type}</span>
            <span className="text-xs">{p.channel}</span>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={p.enabled}
                onChange={(e) => {
                  const next = [...prefs];
                  next[idx] = { ...p, enabled: e.target.checked };
                  setPrefs(next);
                }}
              />
              Enabled
            </label>
            <input
              className="md-input w-24 text-xs"
              placeholder="quiet from"
              value={p.quiet_from ?? ""}
              onChange={(e) => {
                const next = [...prefs];
                next[idx] = { ...p, quiet_from: e.target.value };
                setPrefs(next);
              }}
            />
            <input
              className="md-input w-24 text-xs"
              placeholder="quiet to"
              value={p.quiet_to ?? ""}
              onChange={(e) => {
                const next = [...prefs];
                next[idx] = { ...p, quiet_to: e.target.value };
                setPrefs(next);
              }}
            />
          </li>
        ))}
      </ul>
      <button type="button" className="md-btn md-btn-filled mt-4" onClick={() => void save()}>
        Save preferences
      </button>
      {saved ? <p className="mt-2 text-sm text-emerald-700">Saved.</p> : null}
    </PageChrome>
  );
}
