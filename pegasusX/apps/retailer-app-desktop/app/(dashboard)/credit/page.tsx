"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  AlertTriangle,
  Banknote,
  CalendarClock,
  CheckCircle2,
  CreditCard,
  Loader2,
  RefreshCw,
  ShieldAlert,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { NasiyaOpeningBalancesPanel } from "@/components/credit/NasiyaOpeningBalancesPanel";
import { usePortalLocale } from "@/lib/i18n";
import { ApiError } from "@pegasusx/api-core";
import { NASIYA_MIGRATION_REFRESH_EVENTS } from "@pegasusx/ws-refresh-contract";
import { useOptionalWebSocket } from "@/lib/ws";
import type { MigrationOpeningBalanceInvoice } from "@pegasusx/types";
import {
  createArPromise,
  getArInvoices,
  getArPromises,
  getArStatement,
  getCreditRelationships,
  listRetailerNasiyaOpeningBalances,
  payArFifo,
  payArInvoice,
} from "@/lib/api";
import {
  formatWholeSom,
  minorToSomInput,
  somInputToMinor,
} from "@/lib/retailer-w2";
import {
  generateIdempotencyUuid,
  parseArApiErrorMessage,
  parseQuarterlyPromiseLimitError,
  resolveRelationshipGraceDays,
  resolveRelationshipTermsDays,
  sortArInvoicesOldestFirst,
  W3_CREDIT_COPY,
  type ArInvoice,
  type ArPaymentMethod,
  type ArPaymentPromise,
  type ArStatement,
  type CreditRelationship,
  type PayArFifoResponse,
  type QuarterlyPromiseLimitError,
} from "@/lib/retailer-w3";

type PayMode = "FIFO" | "EARMARK";

const PAYMENT_METHODS: ArPaymentMethod[] = [
  "WALLET",
  "CARD",
  "BANK_TRANSFER",
  "CASH",
];

function formatDateLabel(value?: string): string {
  if (!value) {
    return "—";
  }
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) {
    return value;
  }
  return new Date(parsed).toLocaleDateString();
}

export default function CreditPage() {
  const locale = usePortalLocale();
  const copy = W3_CREDIT_COPY[locale] ?? W3_CREDIT_COPY.uz;

  const [relationships, setRelationships] = useState<CreditRelationship[]>([]);
  const [selectedSupplierId, setSelectedSupplierId] = useState<string>("");
  const [statement, setStatement] = useState<ArStatement | null>(null);
  const [invoices, setInvoices] = useState<ArInvoice[]>([]);
  const [promises, setPromises] = useState<ArPaymentPromise[]>([]);
  const [quarterKeyOverride, setQuarterKeyOverride] = useState<string>("");
  const [promiseUsedOverride, setPromiseUsedOverride] = useState<boolean | null>(
    null,
  );
  const [maxExtensionDaysOverride, setMaxExtensionDaysOverride] = useState<
    number | null
  >(null);

  const [loading, setLoading] = useState<boolean>(true);
  const [detailLoading, setDetailLoading] = useState<boolean>(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [openingBalances, setOpeningBalances] = useState<MigrationOpeningBalanceInvoice[]>([]);
  const [openingLoading, setOpeningLoading] = useState<boolean>(false);
  const [openingError, setOpeningError] = useState<string | null>(null);
  const ws = useOptionalWebSocket();

  const [payMode, setPayMode] = useState<PayMode>("FIFO");
  const [earmarkInvoiceId, setEarmarkInvoiceId] = useState<string>("");
  const [fifoPriorityInvoiceId, setFifoPriorityInvoiceId] =
    useState<string>("");
  const [amountSom, setAmountSom] = useState<string>("");
  const [paymentMethod, setPaymentMethod] =
    useState<ArPaymentMethod>("BANK_TRANSFER");
  const [paymentReference, setPaymentReference] = useState<string>("");
  const [paying, setPaying] = useState<boolean>(false);
  const [payError, setPayError] = useState<string | null>(null);
  const [paySuccessMessage, setPaySuccessMessage] = useState<string | null>(
    null,
  );
  const [payReceipt, setPayReceipt] = useState<PayArFifoResponse | null>(null);

  const [extensionDays, setExtensionDays] = useState<string>("7");
  const [promiseInvoiceId, setPromiseInvoiceId] = useState<string>("");
  const [creatingPromise, setCreatingPromise] = useState<boolean>(false);
  const [promiseError, setPromiseError] = useState<string | null>(null);
  const [promiseSuccess, setPromiseSuccess] = useState<string | null>(null);
  const [quarterlyLimitError, setQuarterlyLimitError] =
    useState<QuarterlyPromiseLimitError | null>(null);

  const loadSupplierDetail = useCallback(async (supplierId: string) => {
    if (!supplierId) {
      setStatement(null);
      setInvoices([]);
      setPromises([]);
      return;
    }
    setDetailLoading(true);
    try {
      const [stmtRes, invRes, promRes] = await Promise.allSettled([
        getArStatement(supplierId, 50),
        getArInvoices("OPEN,PARTIAL", 50, supplierId),
        getArPromises(supplierId),
      ]);

      const nextStatement =
        stmtRes.status === "fulfilled" ? stmtRes.value : null;
      const statementInvoices = Array.isArray(nextStatement?.invoices)
        ? nextStatement.invoices
        : [];
      const fallbackInvoices =
        invRes.status === "fulfilled" ? invRes.value : [];
      const mergedInvoices =
        statementInvoices.length > 0 ? statementInvoices : fallbackInvoices;

      const statementPromises = Array.isArray(nextStatement?.promises)
        ? nextStatement.promises
        : [];
      const promisesEnvelope =
        promRes.status === "fulfilled" ? promRes.value : null;
      const fallbackPromises = promisesEnvelope?.promises ?? [];
      const mergedPromises =
        statementPromises.length > 0 ? statementPromises : fallbackPromises;

      setStatement(nextStatement);
      setInvoices(sortArInvoicesOldestFirst(mergedInvoices));
      setPromises(mergedPromises);
      setQuarterKeyOverride(promisesEnvelope?.current_quarter_key ?? "");
      setPromiseUsedOverride(
        typeof promisesEnvelope?.self_serve_promise_used === "boolean"
          ? promisesEnvelope.self_serve_promise_used
          : null,
      );
      setMaxExtensionDaysOverride(
        typeof promisesEnvelope?.max_extension_days === "number"
          ? promisesEnvelope.max_extension_days
          : null,
      );

      if (
        stmtRes.status === "rejected" &&
        invRes.status === "rejected" &&
        promRes.status === "rejected"
      ) {
        setLoadError(
          parseArApiErrorMessage(stmtRes.reason, copy.loadError),
        );
      }
    } finally {
      setDetailLoading(false);
    }
  }, [copy.loadError]);

  const loadOpeningBalances = useCallback(async () => {
    setOpeningLoading(true);
    setOpeningError(null);
    try {
      const res = await listRetailerNasiyaOpeningBalances();
      setOpeningBalances(Array.isArray(res.invoices) ? res.invoices : []);
    } catch (err) {
      if (err instanceof ApiError && (err.status === 404 || err.status === 503)) {
        setOpeningBalances([]);
        return;
      }
      setOpeningError(
        err instanceof Error ? err.message : copy.loadError,
      );
    } finally {
      setOpeningLoading(false);
    }
  }, [copy.loadError]);

  const loadAll = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [rels] = await Promise.all([
        getCreditRelationships(),
        loadOpeningBalances(),
      ]);
      setRelationships(rels);
      const nextSupplierId =
        selectedSupplierId &&
        rels.some((r) => r.supplier_id === selectedSupplierId)
          ? selectedSupplierId
          : (rels[0]?.supplier_id ?? "");
      setSelectedSupplierId(nextSupplierId);
      if (nextSupplierId) {
        await loadSupplierDetail(nextSupplierId);
      } else {
        const invs = await getArInvoices("OPEN,PARTIAL", 50);
        setStatement(null);
        setInvoices(sortArInvoicesOldestFirst(invs));
        setPromises([]);
      }
    } catch (err) {
      setLoadError(parseArApiErrorMessage(err, copy.loadError));
    } finally {
      setLoading(false);
    }
  }, [copy.loadError, loadOpeningBalances, loadSupplierDetail, selectedSupplierId]);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useEffect(() => {
    if (!ws) return;
    const unsubs = Array.from(NASIYA_MIGRATION_REFRESH_EVENTS).map((evt) =>
      ws.subscribe(evt, () => {
        void loadAll();
      }),
    );
    return () => {
      for (const unsub of unsubs) unsub();
    };
  }, [ws, loadAll]);

  const activeRelationship = useMemo(
    () =>
      relationships.find((r) => r.supplier_id === selectedSupplierId) ??
      relationships[0] ??
      null,
    [relationships, selectedSupplierId],
  );

  const sortedInvoices = useMemo(
    () => sortArInvoicesOldestFirst(invoices),
    [invoices],
  );

  useEffect(() => {
    if (sortedInvoices.length === 0) {
      setEarmarkInvoiceId("");
      return;
    }
    if (!sortedInvoices.some((inv) => inv.invoice_id === earmarkInvoiceId)) {
      setEarmarkInvoiceId(sortedInvoices[0].invoice_id);
    }
  }, [earmarkInvoiceId, sortedInvoices]);

  const totalOpenMinor = useMemo(() => {
    if (statement && typeof statement.total_open_minor === "number") {
      return statement.total_open_minor;
    }
    return sortedInvoices.reduce((acc, inv) => acc + inv.balance_minor, 0);
  }, [sortedInvoices, statement]);

  const totalDisputedMinor = useMemo(() => {
    if (statement && typeof statement.total_disputed_minor === "number") {
      return statement.total_disputed_minor;
    }
    return sortedInvoices.reduce(
      (acc, inv) => acc + (inv.disputed_minor ?? 0),
      0,
    );
  }, [sortedInvoices, statement]);

  const totalUndisputedMinor = useMemo(() => {
    if (statement && typeof statement.total_undisputed_minor === "number") {
      return statement.total_undisputed_minor;
    }
    return sortedInvoices.reduce(
      (acc, inv) =>
        acc +
        (typeof inv.undisputed_minor === "number"
          ? inv.undisputed_minor
          : Math.max(0, inv.balance_minor - (inv.disputed_minor ?? 0))),
      0,
    );
  }, [sortedInvoices, statement]);

  const advanceBalanceMinor = statement?.advance_balance_minor ?? 0;
  const activeCurrency = statement?.currency || "UZS";
  const currentQuarterKey =
    statement?.current_quarter_key || quarterKeyOverride || "";
  const selfServePromiseUsed =
    statement?.self_serve_promise_used ?? promiseUsedOverride ?? false;
  const maxExtensionDays =
    statement?.max_extension_days ?? maxExtensionDaysOverride ?? 14;
  const isPrepayLocked = Boolean(activeRelationship?.prepay_only_locked);

  async function handleSelectSupplier(supplierId: string) {
    setSelectedSupplierId(supplierId);
    setPayError(null);
    setPaySuccessMessage(null);
    setPayReceipt(null);
    setPromiseError(null);
    setPromiseSuccess(null);
    setQuarterlyLimitError(null);
    await loadSupplierDetail(supplierId);
  }

  function handleSelectInvoiceForPay(inv: ArInvoice) {
    const undisputed =
      typeof inv.undisputed_minor === "number"
        ? inv.undisputed_minor
        : Math.max(0, inv.balance_minor - (inv.disputed_minor ?? 0));
    const suggestedMinor = undisputed > 0 ? undisputed : inv.balance_minor;
    setPayMode("EARMARK");
    setEarmarkInvoiceId(inv.invoice_id);
    if (suggestedMinor > 0) {
      setAmountSom(minorToSomInput(suggestedMinor));
    }
    setPayError(null);
    setPaySuccessMessage(null);
  }

  async function handleSubmitPayment(event: React.FormEvent) {
    event.preventDefault();
    setPayError(null);
    setPaySuccessMessage(null);
    setPayReceipt(null);

    const amountMinor = somInputToMinor(amountSom);
    if (amountMinor === null || amountMinor <= 0) {
      setPayError(copy.invalidAmount);
      return;
    }
    if (!selectedSupplierId && payMode === "FIFO") {
      setPayError(copy.emptySuppliers);
      return;
    }

    const idempotencyKey = generateIdempotencyUuid();
    const ref = paymentReference.trim() || undefined;
    setPaying(true);
    try {
      if (payMode === "EARMARK") {
        if (!earmarkInvoiceId) {
          setPayError(copy.selectInvoiceLabel);
          return;
        }
        await payArInvoice(
          earmarkInvoiceId,
          {
            amount_minor: amountMinor,
            payment_method: paymentMethod,
            payment_reference: ref,
          },
          idempotencyKey,
        );
        setPaySuccessMessage(
          `${copy.paymentSuccessTitle}: ${formatWholeSom(amountMinor)} (${earmarkInvoiceId})`,
        );
      } else {
        const result = await payArFifo(
          {
            supplier_id: selectedSupplierId,
            amount_minor: amountMinor,
            currency: activeCurrency,
            payment_method: paymentMethod,
            payment_reference: ref,
            earmark_invoice_id: fifoPriorityInvoiceId || undefined,
          },
          idempotencyKey,
        );
        setPayReceipt(result);
        setPaySuccessMessage(copy.paymentSuccessTitle);
      }
      setAmountSom("");
      setPaymentReference("");
      await loadAll();
    } catch (err) {
      setPayError(parseArApiErrorMessage(err, copy.paymentError));
    } finally {
      setPaying(false);
    }
  }

  async function handleCreatePromise(event: React.FormEvent) {
    event.preventDefault();
    if (!selectedSupplierId) {
      return;
    }
    const parsedDays = Number.parseInt(extensionDays.trim(), 10);
    const clampedDays =
      Number.isFinite(parsedDays) && parsedDays > 0
        ? Math.min(parsedDays, maxExtensionDays > 0 ? maxExtensionDays : 30)
        : 7;

    setCreatingPromise(true);
    setPromiseError(null);
    setPromiseSuccess(null);
    setQuarterlyLimitError(null);
    try {
      await createArPromise(
        {
          supplier_id: selectedSupplierId,
          invoice_id: promiseInvoiceId || undefined,
          extension_days: clampedDays,
        },
        generateIdempotencyUuid(),
      );
      setPromiseSuccess(copy.promiseCreated);
      await loadSupplierDetail(selectedSupplierId);
    } catch (err) {
      const limitErr = parseQuarterlyPromiseLimitError(err);
      if (limitErr) {
        setQuarterlyLimitError(limitErr);
      } else {
        setPromiseError(parseArApiErrorMessage(err, copy.promiseError));
      }
    } finally {
      setCreatingPromise(false);
    }
  }

  return (
    <PageChrome
      title={copy.title}
      description={copy.subtitle}
      actions={
        <button
          type="button"
          onClick={() => void loadAll()}
          disabled={loading || detailLoading}
          className="inline-flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2 text-xs font-medium text-foreground hover:bg-muted/60 disabled:opacity-50"
        >
          <RefreshCw
            className={`h-3.5 w-3.5 ${loading || detailLoading ? "animate-spin" : ""}`}
          />
          {copy.refresh}
        </button>
      }
    >
      {loadError ? (
        <div
          role="alert"
          className="mb-4 flex items-center gap-2 rounded-xl border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-700 dark:text-red-300"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span>{loadError}</span>
        </div>
      ) : null}

      <NasiyaOpeningBalancesPanel
        invoices={openingBalances}
        loading={openingLoading}
        error={openingError}
        onChanged={loadAll}
      />

      {loading && relationships.length === 0 ? (
        <div className="flex items-center gap-2 rounded-xl border border-border bg-card p-6 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>{copy.loading}</span>
        </div>
      ) : null}

      {!loading && relationships.length === 0 ? (
        <div className="rounded-xl border border-border bg-card p-6 text-sm text-muted-foreground">
          {copy.emptySuppliers}
        </div>
      ) : null}

      {relationships.length > 0 ? (
        <section className="mb-6">
          <h2 className="mb-3 text-sm font-semibold text-foreground">
            {copy.suppliersHeading}
          </h2>
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {relationships.map((rel) => {
              const isSelected = rel.supplier_id === selectedSupplierId;
              const termsDays = resolveRelationshipTermsDays(rel);
              const graceDays = resolveRelationshipGraceDays(rel);
              return (
                <button
                  key={rel.supplier_id}
                  type="button"
                  onClick={() => void handleSelectSupplier(rel.supplier_id)}
                  className={`rounded-xl border p-4 text-left transition ${
                    isSelected
                      ? "border-primary bg-primary/5 ring-1 ring-primary"
                      : "border-border bg-card hover:bg-muted/40"
                  }`}
                >
                  <div className="mb-3 flex flex-wrap items-start justify-between gap-2">
                    <div>
                      <div className="text-sm font-semibold text-foreground">
                        {rel.supplier_name || rel.supplier_id}
                      </div>
                      {rel.supplier_name ? (
                        <div className="font-mono text-[11px] text-muted-foreground">
                          {rel.supplier_id}
                        </div>
                      ) : null}
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {rel.on_hold ? (
                        <span className="rounded-full bg-red-500/15 px-2.5 py-0.5 text-[11px] font-semibold text-red-700 dark:text-red-300">
                          {copy.onHold}
                        </span>
                      ) : null}
                      {rel.prepay_only_locked ? (
                        <span className="rounded-full bg-amber-500/15 px-2.5 py-0.5 text-[11px] font-semibold text-amber-700 dark:text-amber-300">
                          {copy.prepayOnly}
                        </span>
                      ) : null}
                      {!rel.on_hold && !rel.prepay_only_locked ? (
                        <span className="rounded-full bg-emerald-500/15 px-2.5 py-0.5 text-[11px] font-medium text-emerald-700 dark:text-emerald-300">
                          {copy.activeBadge}
                        </span>
                      ) : null}
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-2 text-xs">
                    <div>
                      <div className="text-muted-foreground">
                        {copy.available}
                      </div>
                      <div className="font-mono font-semibold text-foreground tabular-nums">
                        {formatWholeSom(rel.available_credit_minor)}
                      </div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">
                        {copy.balanceDue}
                      </div>
                      <div className="font-mono font-semibold text-foreground tabular-nums">
                        {formatWholeSom(rel.current_balance_minor)}
                      </div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">{copy.limit}</div>
                      <div className="font-mono font-medium text-foreground tabular-nums">
                        {formatWholeSom(rel.credit_limit_minor)}
                      </div>
                    </div>
                  </div>

                  <div className="mt-3 flex flex-wrap items-center gap-3 border-t border-border/70 pt-2 text-[11px] text-muted-foreground">
                    <span>
                      {copy.terms}:{" "}
                      <strong className="text-foreground">
                        {termsDays} {copy.daysUnit}
                      </strong>
                    </span>
                    <span>
                      {copy.graceDays}:{" "}
                      <strong className="text-foreground">
                        {graceDays} {copy.daysUnit}
                      </strong>
                    </span>
                  </div>
                </button>
              );
            })}
          </div>
        </section>
      ) : null}

      {selectedSupplierId ? (
        <div className="space-y-6">
          <section className="rounded-xl border border-border bg-card p-5">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <CreditCard className="h-4 w-4 text-primary" />
                <h2 className="text-sm font-semibold text-foreground">
                  {copy.statementHeading}
                </h2>
              </div>
              {currentQuarterKey ? (
                <span className="rounded-full bg-muted px-2.5 py-0.5 font-mono text-xs text-muted-foreground">
                  {copy.quarterLabel}: {currentQuarterKey}
                </span>
              ) : null}
            </div>

            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-lg border border-border bg-background/60 p-3">
                <div className="text-xs text-muted-foreground">
                  {copy.totalOpen}
                </div>
                <div className="mt-1 font-mono text-base font-semibold tabular-nums">
                  {formatWholeSom(totalOpenMinor)}
                </div>
              </div>
              <div className="rounded-lg border border-border bg-background/60 p-3">
                <div className="text-xs text-muted-foreground">
                  {copy.undisputed}
                </div>
                <div className="mt-1 font-mono text-base font-semibold tabular-nums">
                  {formatWholeSom(totalUndisputedMinor)}
                </div>
              </div>
              <div className="rounded-lg border border-border bg-background/60 p-3">
                <div className="text-xs text-muted-foreground">
                  {copy.disputed}
                </div>
                <div className="mt-1 font-mono text-base font-semibold text-amber-700 dark:text-amber-300 tabular-nums">
                  {formatWholeSom(totalDisputedMinor)}
                </div>
              </div>
              <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-3">
                <div className="text-xs text-emerald-700 dark:text-emerald-300">
                  {copy.advanceBalance}
                </div>
                <div className="mt-1 font-mono text-base font-semibold text-emerald-800 dark:text-emerald-200 tabular-nums">
                  {formatWholeSom(advanceBalanceMinor)}
                </div>
              </div>
            </div>
          </section>

          <div className="grid gap-6 lg:grid-cols-2">
            <section className="rounded-xl border border-border bg-card p-5">
              <div className="mb-3 flex items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <Banknote className="h-4 w-4 text-primary" />
                  <h2 className="text-sm font-semibold text-foreground">
                    {copy.payHeading}
                  </h2>
                </div>
                {totalUndisputedMinor > 0 ? (
                  <button
                    type="button"
                    onClick={() =>
                      setAmountSom(minorToSomInput(totalUndisputedMinor))
                    }
                    className="rounded-md border border-border px-2 py-1 text-[11px] font-medium text-foreground hover:bg-muted/60"
                  >
                    {copy.useUndisputedFill}
                  </button>
                ) : null}
              </div>

              <div className="mb-3 flex gap-2 rounded-lg bg-muted/60 p-1">
                <button
                  type="button"
                  onClick={() => {
                    setPayMode("FIFO");
                    setPayError(null);
                  }}
                  className={`flex-1 rounded-md px-3 py-1.5 text-xs font-medium transition ${
                    payMode === "FIFO"
                      ? "bg-card text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {copy.payModeFifo}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setPayMode("EARMARK");
                    setPayError(null);
                  }}
                  className={`flex-1 rounded-md px-3 py-1.5 text-xs font-medium transition ${
                    payMode === "EARMARK"
                      ? "bg-card text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {copy.payModeEarmark}
                </button>
              </div>

              <p className="mb-4 text-xs text-muted-foreground">
                {payMode === "FIFO"
                  ? copy.payModeFifoHint
                  : copy.payModeEarmarkHint}
              </p>

              <form onSubmit={(e) => void handleSubmitPayment(e)} className="space-y-3">
                {payMode === "EARMARK" ? (
                  <div>
                    <label
                      htmlFor="earmark-invoice-select"
                      className="mb-1 block text-xs font-medium text-foreground"
                    >
                      {copy.selectInvoiceLabel}
                    </label>
                    <select
                      id="earmark-invoice-select"
                      value={earmarkInvoiceId}
                      onChange={(e) => setEarmarkInvoiceId(e.target.value)}
                      className="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs text-foreground"
                    >
                      {sortedInvoices.map((inv) => (
                        <option key={inv.invoice_id} value={inv.invoice_id}>
                          {inv.order_id || inv.invoice_id} —{" "}
                          {formatWholeSom(inv.balance_minor)}
                        </option>
                      ))}
                    </select>
                  </div>
                ) : sortedInvoices.length > 0 ? (
                  <div>
                    <label
                      htmlFor="fifo-earmark-select"
                      className="mb-1 block text-xs font-medium text-muted-foreground"
                    >
                      {copy.promiseInvoiceOptional}
                    </label>
                    <select
                      id="fifo-earmark-select"
                      value={fifoPriorityInvoiceId}
                      onChange={(e) => setFifoPriorityInvoiceId(e.target.value)}
                      className="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs text-foreground"
                    >
                      <option value="">{copy.payModeFifo}</option>
                      {sortedInvoices.map((inv) => (
                        <option key={inv.invoice_id} value={inv.invoice_id}>
                          {inv.order_id || inv.invoice_id} (
                          {formatWholeSom(inv.balance_minor)})
                        </option>
                      ))}
                    </select>
                  </div>
                ) : null}

                <div className="grid gap-3 sm:grid-cols-2">
                  <div>
                    <label
                      htmlFor="ar-pay-amount-som"
                      className="mb-1 block text-xs font-medium text-foreground"
                    >
                      {copy.amountLabel}
                    </label>
                    <input
                      id="ar-pay-amount-som"
                      type="text"
                      inputMode="numeric"
                      value={amountSom}
                      onChange={(e) => setAmountSom(e.target.value)}
                      placeholder={copy.amountPlaceholder}
                      className="w-full rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm text-foreground"
                    />
                  </div>
                  <div>
                    <label
                      htmlFor="ar-pay-method"
                      className="mb-1 block text-xs font-medium text-foreground"
                    >
                      {copy.methodLabel}
                    </label>
                    <select
                      id="ar-pay-method"
                      value={paymentMethod}
                      onChange={(e) =>
                        setPaymentMethod(e.target.value as ArPaymentMethod)
                      }
                      className="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs text-foreground"
                    >
                      {PAYMENT_METHODS.map((method) => (
                        <option key={method} value={method}>
                          {copy.methods[method]}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                <div>
                  <label
                    htmlFor="ar-pay-reference"
                    className="mb-1 block text-xs font-medium text-foreground"
                  >
                    {copy.referenceLabel}
                  </label>
                  <input
                    id="ar-pay-reference"
                    type="text"
                    value={paymentReference}
                    onChange={(e) => setPaymentReference(e.target.value)}
                    placeholder={copy.referencePlaceholder}
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs text-foreground"
                  />
                </div>

                {payError ? (
                  <div
                    role="alert"
                    className="rounded-lg border border-red-500/40 bg-red-500/10 px-3 py-2 text-xs text-red-700 dark:text-red-300"
                  >
                    {payError}
                  </div>
                ) : null}

                {paySuccessMessage ? (
                  <div className="rounded-lg border border-emerald-500/40 bg-emerald-500/10 p-3 text-xs text-emerald-800 dark:text-emerald-200">
                    <div className="flex items-center gap-1.5 font-semibold">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      <span>{paySuccessMessage}</span>
                    </div>
                    {payReceipt ? (
                      <div className="mt-2 space-y-1">
                        <div>
                          {copy.paidSummary}:{" "}
                          <span className="font-mono font-semibold">
                            {formatWholeSom(payReceipt.paid_minor)}
                          </span>
                        </div>
                        {payReceipt.advance_added_minor > 0 ? (
                          <div>
                            {copy.advanceAddedSummary}:{" "}
                            <span className="font-mono font-semibold">
                              {formatWholeSom(payReceipt.advance_added_minor)}
                            </span>
                          </div>
                        ) : null}
                      </div>
                    ) : null}
                  </div>
                ) : null}

                <button
                  type="submit"
                  disabled={
                    paying || (payMode === "EARMARK" && sortedInvoices.length === 0)
                  }
                  className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground hover:opacity-90 disabled:opacity-50"
                >
                  {paying ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : null}
                  {paying
                    ? copy.paying
                    : payMode === "FIFO"
                      ? copy.submitPayFifo
                      : copy.submitPayEarmark}
                </button>
              </form>
            </section>

            <section className="rounded-xl border border-border bg-card p-5">
              <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <CalendarClock className="h-4 w-4 text-primary" />
                  <h2 className="text-sm font-semibold text-foreground">
                    {copy.promisesHeading}
                  </h2>
                </div>
                <span
                  className={`rounded-full px-2.5 py-0.5 text-[11px] font-medium ${
                    selfServePromiseUsed
                      ? "bg-amber-500/15 text-amber-700 dark:text-amber-300"
                      : "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
                  }`}
                >
                  {selfServePromiseUsed
                    ? copy.promiseUsedBadge
                    : copy.promiseAvailableBadge}
                </span>
              </div>

              {quarterlyLimitError ? (
                <div
                  role="alert"
                  className="mb-3 flex items-start gap-2 rounded-lg border border-amber-500/50 bg-amber-500/10 p-3 text-xs text-amber-800 dark:text-amber-200"
                >
                  <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0" />
                  <div>
                    <div className="font-semibold">
                      {copy.quarterlyLimitTitle}
                      {quarterlyLimitError.quarterKey
                        ? ` (${quarterlyLimitError.quarterKey})`
                        : ""}
                    </div>
                    <div className="mt-0.5">{copy.quarterlyLimitDetail}</div>
                  </div>
                </div>
              ) : null}

              {isPrepayLocked ? (
                <div className="mb-3 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-200">
                  {copy.promisePrepayBlocked}
                </div>
              ) : null}

              <form onSubmit={(e) => void handleCreatePromise(e)} className="space-y-3">
                <div className="grid gap-3 sm:grid-cols-2">
                  <div>
                    <label
                      htmlFor="ar-extension-days"
                      className="mb-1 block text-xs font-medium text-foreground"
                    >
                      {copy.extensionDaysLabel}
                    </label>
                    <input
                      id="ar-extension-days"
                      type="number"
                      min={1}
                      max={maxExtensionDays > 0 ? maxExtensionDays : 30}
                      value={extensionDays}
                      onChange={(e) => setExtensionDays(e.target.value)}
                      className="w-full rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm text-foreground"
                    />
                  </div>
                  <div>
                    <label
                      htmlFor="ar-promise-invoice"
                      className="mb-1 block text-xs font-medium text-foreground"
                    >
                      {copy.promiseInvoiceOptional}
                    </label>
                    <select
                      id="ar-promise-invoice"
                      value={promiseInvoiceId}
                      onChange={(e) => setPromiseInvoiceId(e.target.value)}
                      className="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs text-foreground"
                    >
                      <option value="">—</option>
                      {sortedInvoices.map((inv) => (
                        <option key={inv.invoice_id} value={inv.invoice_id}>
                          {inv.order_id || inv.invoice_id} —{" "}
                          {formatWholeSom(inv.balance_minor)}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                {promiseError ? (
                  <div
                    role="alert"
                    className="rounded-lg border border-red-500/40 bg-red-500/10 px-3 py-2 text-xs text-red-700 dark:text-red-300"
                  >
                    {promiseError}
                  </div>
                ) : null}

                {promiseSuccess ? (
                  <div className="rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-800 dark:text-emerald-200">
                    {promiseSuccess}
                  </div>
                ) : null}

                <button
                  type="submit"
                  disabled={creatingPromise || isPrepayLocked}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-border bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted/60 disabled:opacity-50"
                >
                  {creatingPromise ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : null}
                  {creatingPromise
                    ? copy.requestingExtension
                    : copy.requestExtension}
                </button>
              </form>

              <div className="mt-4 border-t border-border/70 pt-3">
                {promises.length === 0 ? (
                  <div className="text-xs text-muted-foreground">
                    {copy.noPromises}
                  </div>
                ) : (
                  <div className="space-y-2">
                    {promises.map((prom) => (
                      <div
                        key={prom.promise_id}
                        className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border/80 bg-background/50 px-3 py-2 text-xs"
                      >
                        <div>
                          <div className="font-mono font-medium text-foreground">
                            {prom.quarter_key} · {formatDateLabel(prom.due_at)}
                          </div>
                          {prom.invoice_id ? (
                            <div className="text-[11px] text-muted-foreground">
                              {prom.invoice_id}
                            </div>
                          ) : null}
                        </div>
                        <div className="text-right">
                          <div className="font-mono font-semibold tabular-nums">
                            {formatWholeSom(prom.promised_minor)}
                          </div>
                          <div className="text-[11px] uppercase text-muted-foreground">
                            {prom.status}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </section>
          </div>

          <section className="rounded-xl border border-border bg-card p-5">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <h2 className="text-sm font-semibold text-foreground">
                {copy.invoicesHeading}
              </h2>
              <span className="text-xs text-muted-foreground">
                {copy.oldestFirstNote}
              </span>
            </div>

            {sortedInvoices.length === 0 ? (
              <div className="text-xs text-muted-foreground">
                {copy.noOpenInvoices}
              </div>
            ) : (
              <div className="space-y-2.5">
                {sortedInvoices.map((inv) => {
                  const disputedMinor = inv.disputed_minor ?? 0;
                  const undisputedMinor =
                    typeof inv.undisputed_minor === "number"
                      ? inv.undisputed_minor
                      : Math.max(0, inv.balance_minor - disputedMinor);
                  return (
                    <div
                      key={inv.invoice_id}
                      className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border bg-background/60 p-3.5"
                    >
                      <div className="space-y-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-mono text-xs font-semibold text-foreground">
                            {inv.order_id
                              ? `${copy.invoiceOrderPrefix} #${inv.order_id}`
                              : inv.invoice_id}
                          </span>
                          <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-semibold uppercase text-muted-foreground">
                            {inv.status}
                          </span>
                          {inv.aging_bucket ? (
                            <span className="rounded-full bg-amber-500/15 px-2 py-0.5 font-mono text-[10px] font-medium text-amber-800 dark:text-amber-200">
                              {copy.agingLabel}: {inv.aging_bucket}
                            </span>
                          ) : null}
                          {disputedMinor > 0 ? (
                            <span className="rounded-full bg-red-500/15 px-2 py-0.5 font-mono text-[10px] font-medium text-red-700 dark:text-red-300">
                              {copy.disputedTag}: {formatWholeSom(disputedMinor)}
                            </span>
                          ) : null}
                        </div>
                        <div className="flex flex-wrap items-center gap-3 text-[11px] text-muted-foreground">
                          <span>
                            {copy.dueLabel}: {formatDateLabel(inv.due_at)}
                          </span>
                          <span>
                            {copy.undisputedTag}:{" "}
                            <strong className="font-mono text-foreground">
                              {formatWholeSom(undisputedMinor)}
                            </strong>
                          </span>
                          {typeof inv.dunning_step === "number" &&
                          inv.dunning_step > 0 ? (
                            <span>
                              {copy.dunningLabel}: #{inv.dunning_step}
                            </span>
                          ) : null}
                        </div>
                      </div>

                      <div className="flex items-center gap-3">
                        <div className="text-right font-mono text-sm font-semibold tabular-nums">
                          {formatWholeSom(inv.balance_minor)}
                        </div>
                        <button
                          type="button"
                          onClick={() => handleSelectInvoiceForPay(inv)}
                          className="rounded-lg border border-border bg-card px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted/60"
                        >
                          {copy.payThisInvoice}
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </section>
        </div>
      ) : null}
    </PageChrome>
  );
}
