"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useMemo, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
import { ExceptionsList } from "@/components/exceptions/ExceptionsList";
import { PlaybookRunsPanel } from "@/components/exceptions/PlaybookRunsPanel";
import {
  EXCEPTION_HANDLER_GROUPS,
  formatWholeSom,
  HANDLER_META,
  resolveHandlerGroup,
  sumMoneyAtRiskMinor,
  type ExceptionHandlerGroup,
  type SupplierExceptionDeskRow,
} from "@/lib/exception-desk";

export default function ExceptionsPage() {
  const t = usePortalT();
  const [exceptions, setExceptions] = useState<SupplierExceptionDeskRow[]>([]);
  const [activeHandler, setActiveHandler] = useState<ExceptionHandlerGroup | "ALL">("ALL");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const qs = activeHandler !== "ALL" ? `?handler=${encodeURIComponent(activeHandler)}` : "";
      const res = await supplierFetch(`/v1/supplier/exceptions${qs}`);
      if (!res.ok) {
        throw new Error(`exceptions_${res.status}`);
      }
      const resp = (await res.json()) as { exceptions?: SupplierExceptionDeskRow[] };
      setExceptions(resp.exceptions ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_exceptions_failed"));
    } finally {
      setLoading(false);
    }
  }, [activeHandler, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const handlerStats = useMemo(() => {
    const stats = new Map<ExceptionHandlerGroup, { count: number; riskMinor: number }>();
    for (const id of EXCEPTION_HANDLER_GROUPS) {
      stats.set(id, { count: 0, riskMinor: 0 });
    }
    for (const row of exceptions) {
      const id = resolveHandlerGroup(row);
      const entry = stats.get(id)!;
      entry.count += 1;
      entry.riskMinor += row.money_at_risk_minor ?? 0;
    }
    return stats;
  }, [exceptions]);

  const totalRiskMinor = useMemo(() => sumMoneyAtRiskMinor(exceptions), [exceptions]);

  return (
    <PageChrome
      title={t("portal.nav.exceptions")}
      description="15-Minute Money-at-Risk Exception Desk — triaged by handler group, severity tier, and total so'm exposure."
      icon="warning"
      loading={loading}
      error={error}
      empty={!loading && exceptions.length === 0 && activeHandler === "ALL"}
      emptyMessage={t("supplier_portal.residual.text.no_open_exceptions_escalations_appear_here_when_operators_raise_")}
    >
      <div className="mb-5 grid grid-cols-1 md:grid-cols-3 gap-3">
        <div className="md-card p-4">
          <div className="text-xs uppercase tracking-wide text-[var(--color-md-outline)]">Total Money at Risk</div>
          <div className="mt-1 text-xl font-mono font-bold text-[var(--desk-text-primary)]">
            {formatWholeSom(totalRiskMinor)}
          </div>
          <div className="mt-1 text-xs text-[var(--color-md-outline)]">
            {exceptions.length} open item(s) across 6 handlers
          </div>
        </div>
        <div className="md-card p-4 md:col-span-2 flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => setActiveHandler("ALL")}
            aria-pressed={activeHandler === "ALL"}
            className={`px-3 py-1.5 rounded-full text-xs font-medium border transition ${
              activeHandler === "ALL"
                ? "bg-[var(--color-md-primary)] text-white border-[var(--color-md-primary)]"
                : "border-[var(--desk-border)] text-[var(--desk-text-primary)]"
            }`}
          >
            All Handlers ({exceptions.length})
          </button>
          {EXCEPTION_HANDLER_GROUPS.map((handlerId) => {
            const meta = HANDLER_META[handlerId];
            const st = handlerStats.get(handlerId) ?? { count: 0, riskMinor: 0 };
            const selected = activeHandler === handlerId;
            return (
              <button
                key={handlerId}
                type="button"
                onClick={() => setActiveHandler(handlerId)}
                aria-pressed={selected}
                className={`px-3 py-1.5 rounded-full text-xs font-medium border transition ${
                  selected
                    ? "bg-[var(--color-md-primary)] text-white border-[var(--color-md-primary)]"
                    : "border-[var(--desk-border)] text-[var(--desk-text-primary)]"
                }`}
              >
                {meta.label} ({st.count} · {formatWholeSom(st.riskMinor)})
              </button>
            );
          })}
        </div>
      </div>

      <section className="mb-6">
        <h2 className="mb-2 md-typescale-title-medium">{t("supplier_portal.exceptions.text.recommended_playbooks")}</h2>
        <PlaybookRunsPanel />
      </section>

      {!loading && exceptions.length === 0 && activeHandler !== "ALL" ? (
        <div className="md-card p-6 text-sm text-[var(--color-md-outline)]">
          No open exceptions under {HANDLER_META[activeHandler].label}.{" "}
          <button type="button" className="underline text-[var(--color-md-primary)]" onClick={() => setActiveHandler("ALL")}>
            Show all handlers
          </button>
        </div>
      ) : (
        <ExceptionsList
          exceptions={exceptions}
          activeHandler={activeHandler}
          onResolved={() => void load()}
        />
      )}

      <div className="mt-6 flex flex-wrap gap-4 md-typescale-body-medium">
        <Link href={"/exceptions/claims" as Route} className="text-[var(--color-md-primary)] underline">
          Claims / chargebacks
        </Link>
        <Link href={"/exceptions/shop-closed" as Route} className="text-[var(--color-md-primary)] underline">
          Shop closed queue
        </Link>
        <Link href={"/exceptions/early-complete" as Route} className="text-[var(--color-md-primary)] underline">
          Early route complete
        </Link>
        <Link href={"/manifest-exceptions" as Route} className="text-[var(--color-md-primary)] underline">
          Manifest gate exceptions
        </Link>
        <Link href={"/treasury/cash-reconciliations" as Route} className="text-[var(--color-md-primary)] underline">
          Cash discrepancies
        </Link>
        <Link href={"/finance/credit-notes" as Route} className="text-[var(--color-md-primary)] underline">
          Credit notes
        </Link>
        <Link href={"/operations" as Route} className="text-[var(--color-md-primary)] underline">
          Operations
        </Link>
        <Link href={"/orders" as Route} className="text-[var(--color-md-primary)] underline">
          Orders
        </Link>
      </div>
    </PageChrome>
  );
}
