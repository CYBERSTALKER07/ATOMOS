"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Loader2, Clock, AlertTriangle } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
import { moneyCurrency, sessionPackCurrency } from "@/lib/payment-catalog";

type TimeEntry = {
  entry_id: string;
  user_id: string;
  location_id: string;
  status: string;
  clock_in_at?: string;
  clock_out_at?: string;
  auto_closed?: boolean;
};

type Shift = {
  shift_id: string;
  location_id: string;
  register_id?: string;
  status: string;
  opening_float_minor: number;
  closing_cash_minor?: number;
  expected_cash_minor?: number;
  variance_minor?: number;
  currency: string;
  linked_pos_session_id?: string;
  opened_at?: string;
  closed_at?: string;
};

function formatMoney(minor: number, currency?: string) {
  return `${(minor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })} ${moneyCurrency(currency)}`;
}

export default function ShiftsPage() {
  const t = usePortalT();
  const [clockedIn, setClockedIn] = useState(false);
  const [openEntry, setOpenEntry] = useState<TimeEntry | null>(null);
  const [entries, setEntries] = useState<TimeEntry[]>([]);
  const [shifts, setShifts] = useState<Shift[]>([]);
  const [floatMinor, setFloatMinor] = useState("0");
  const [closingCash, setClosingCash] = useState("0");
  const [registerId, setRegisterId] = useState("");
  const [banner, setBanner] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const [timeRes, shiftRes, regRes] = await Promise.all([
        apiFetch("/v1/retailer/time/entries"),
        apiFetch("/v1/retailer/shifts"),
        apiFetch("/v1/retailer/registers"),
      ]);
      if (timeRes.ok) {
        const json = (await timeRes.json()) as {
          items?: TimeEntry[];
          open_entry?: TimeEntry;
          clocked_in?: boolean;
        };
        setEntries(json.items ?? []);
        setClockedIn(Boolean(json.clocked_in));
        setOpenEntry(json.open_entry?.entry_id ? json.open_entry : null);
      }
      if (shiftRes.ok) {
        const json = (await shiftRes.json()) as { items?: Shift[] };
        setShifts(json.items ?? []);
      }
      if (regRes.ok) {
        const json = (await regRes.json()) as {
          items?: { register_id: string }[];
        };
        if (!registerId && json.items?.[0]) {
          setRegisterId(json.items[0].register_id);
        }
      }
    } catch {
      /* ignore */
    }
  }, [registerId]);

  useEffect(() => {
    void load();
  }, [load]);

  const clockIn = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/time/clock-in", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error((json as { error?: string }).error || "clock_in_failed");
      }
      setBanner("Clocked in (SHIFTS pack auto-enabled if needed)");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.clock_in_failed"));
    } finally {
      setBusy(false);
    }
  };

  const clockOut = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/time/clock-out", {
        method: "POST",
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error((json as { error?: string }).error || "clock_out_failed");
      }
      setBanner("Clocked out");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.clock_out_failed"));
    } finally {
      setBusy(false);
    }
  };

  const openShift = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/shifts", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `shift-${Date.now()}`,
        },
        body: JSON.stringify({
          register_id: registerId || undefined,
          opening_float_minor: Number(floatMinor) || 0,
          currency: sessionPackCurrency(),
        }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error((json as { error?: string }).error || "open_shift_failed");
      }
      setBanner("Shift opened — open POS when ready");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.open_shift_failed"));
    } finally {
      setBusy(false);
    }
  };

  const closeShift = async (shiftId: string) => {
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch(`/v1/retailer/shifts/${shiftId}/close`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          closing_cash_minor: Number(closingCash) || 0,
        }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error((json as { error?: string }).error || "close_shift_failed");
      }
      const v = (json as Shift).variance_minor ?? 0;
      setBanner(`Shift closed. Variance: ${formatMoney(v)}`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.close_shift_failed"));
    } finally {
      setBusy(false);
    }
  };

  const openShifts = shifts.filter((s) => s.status === "OPEN");

  return (
    <PageChrome
      title={t("retailer_desktop.shifts.text.shifts_and_time")}
      description={t("retailer_desktop.residual.text.clock_in_out_open_cash_shifts_reconcile_float_when_shifts_is_on_")}
    >
      <div className="mx-auto flex max-w-3xl flex-col gap-6 p-4">
        {banner && (
          <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm">
            {banner}
          </div>
        )}
        {error && (
          <div className="flex items-center gap-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-600 dark:text-red-400">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            {error}
          </div>
        )}

        <section className="rounded-xl border border-border bg-card p-4">
          <div className="mb-3 flex items-center gap-2">
            <Clock className="h-5 w-5 text-muted-foreground" />
            <h2 className="font-semibold">{t("retailer_desktop.shifts.text.time_clock")}</h2>
          </div>
          <p className="mb-3 text-sm text-muted-foreground">
            {clockedIn && openEntry
              ? `Clocked in since ${openEntry.clock_in_at ?? "—"} at ${openEntry.location_id}`
              : "Not clocked in"}
          </p>
          <div className="flex flex-wrap gap-2">
            {!clockedIn ? (
              <button
                type="button"
                disabled={busy}
                onClick={() => void clockIn()}
                className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
              >
                {busy && <Loader2 className="h-4 w-4 animate-spin" />}
                Clock in
              </button>
            ) : (
              <button
                type="button"
                disabled={busy}
                onClick={() => void clockOut()}
                className="inline-flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium disabled:opacity-50"
              >
                {busy && <Loader2 className="h-4 w-4 animate-spin" />}
                Clock out
              </button>
            )}
          </div>
        </section>

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 font-semibold">{t("retailer_desktop.shifts.text.cash_shift")}</h2>
          <div className="mb-3 grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-sm">
              Opening float (minor)
              <input
                className="rounded-md border border-border bg-background px-3 py-2"
                value={floatMinor}
                onChange={(e) => setFloatMinor(e.target.value)}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm">
              Closing cash (minor)
              <input
                className="rounded-md border border-border bg-background px-3 py-2"
                value={closingCash}
                onChange={(e) => setClosingCash(e.target.value)}
              />
            </label>
          </div>
          <button
            type="button"
            disabled={busy || !clockedIn}
            onClick={() => void openShift()}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Open shift
          </button>
          {!clockedIn && (
            <p className="mt-2 text-xs text-muted-foreground">
              Clock in first when require_clock_in is enabled.
            </p>
          )}

          {openShifts.length > 0 && (
            <ul className="mt-4 space-y-2">
              {openShifts.map((s) => (
                <li
                  key={s.shift_id}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border px-3 py-2 text-sm"
                >
                  <span>
                    Open · float {formatMoney(s.opening_float_minor, s.currency)}
                    {s.register_id ? ` · reg ${s.register_id.slice(0, 8)}` : ""}
                  </span>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => void closeShift(s.shift_id)}
                    className="rounded-md border border-border px-3 py-1 text-xs font-medium"
                  >
                    Close with counted cash
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 font-semibold">{t("retailer_desktop.shifts.text.recent_shifts")}</h2>
          {shifts.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("retailer_desktop.shifts.text.no_shifts_yet")}</p>
          ) : (
            <ul className="space-y-2 text-sm">
              {shifts.slice(0, 20).map((s) => (
                <li
                  key={s.shift_id}
                  className="flex flex-wrap justify-between gap-2 border-b border-border/50 py-2 last:border-0"
                >
                  <span>
                    {s.status} · {s.opened_at?.slice(0, 16) ?? "—"}
                  </span>
                  <span className="text-muted-foreground">
                    {s.variance_minor != null
                      ? `var ${formatMoney(s.variance_minor, s.currency)}`
                      : formatMoney(s.opening_float_minor, s.currency)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 font-semibold">{t("retailer_desktop.shifts.text.time_entries")}</h2>
          {entries.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("retailer_desktop.shifts.text.no_entries_yet")}</p>
          ) : (
            <ul className="space-y-2 text-sm">
              {entries.slice(0, 20).map((e) => (
                <li key={e.entry_id} className="border-b border-border/50 py-2 last:border-0">
                  {e.status} · {e.user_id} · in {e.clock_in_at?.slice(0, 16) ?? "—"}
                  {e.clock_out_at ? ` → out ${e.clock_out_at.slice(0, 16)}` : ""}
                  {e.auto_closed ? " (auto)" : ""}
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </PageChrome>
  );
}
