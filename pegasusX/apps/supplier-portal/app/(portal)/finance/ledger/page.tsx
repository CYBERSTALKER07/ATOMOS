"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import {
  CANONICAL_GL_ACCOUNTS,
  createSupplierFinanceApi,
  evaluateManualJournalLines,
  formatFinanceMinor,
  type AccountLedgerRow,
  type GlDirection,
  type JournalEntryRecord,
  type ManualDraftLine,
} from "@/lib/finance-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import { ObjectHistoryPanel } from "@/components/audit/ObjectHistoryPanel";
import Icon from "@/components/Icon";

const PERIOD_RE = /^\d{4}-(0[1-9]|1[0-2])$/;

export default function GlLedgerPage() {
  return (
    <Suspense fallback={null}>
      <GlLedgerPageInner />
    </Suspense>
  );
}

function GlLedgerPageInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const financeApi = useMemo(() => createSupplierFinanceApi(createSupplierApi()), []);

  const initialAccount = (searchParams.get("account") || "AR:RETAILER").trim().toUpperCase();
  const initialPeriod = (searchParams.get("period") || "").trim();
  const initialEntryId = (searchParams.get("entryId") || "").trim();

  const [accountCode, setAccountCode] = useState<string>(initialAccount);
  const [period, setPeriod] = useState<string>(initialPeriod);
  const [limit, setLimit] = useState<number>(50);

  useEffect(() => {
    const nextAcct = (searchParams.get("account") || "").trim().toUpperCase();
    const nextPeriod = (searchParams.get("period") || "").trim();
    if (nextAcct) setAccountCode(nextAcct);
    if (nextPeriod) setPeriod(nextPeriod);
  }, [searchParams]);
  const [rows, setRows] = useState<AccountLedgerRow[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  // Journal entry inspector + reversal state
  const [inspectedEntry, setInspectedEntry] = useState<JournalEntryRecord | null>(null);
  const [inspectLoading, setInspectLoading] = useState<boolean>(false);
  const [inspectError, setInspectError] = useState<string | null>(null);
  const [reverseReason, setReverseReason] = useState<string>("");
  const [reversing, setReversing] = useState<boolean>(false);
  const [inspectorTab, setInspectorTab] = useState<"postings" | "history">("postings");

  // Balanced manual journal entry state
  const [manualOpen, setManualOpen] = useState<boolean>(false);
  const [manualSourceId, setManualSourceId] = useState<string>(() => `man-${Date.now()}`);
  const [manualCurrency, setManualCurrency] = useState<string>("UZS");
  const [manualMemo, setManualMemo] = useState<string>("");
  const [manualLines, setManualLines] = useState<ManualDraftLine[]>([
    { account_code: initialAccount, direction: "DEBIT", amount_major: "", description: "" },
    { account_code: "EQUITY:OPENING_BALANCE", direction: "CREDIT", amount_major: "", description: "" },
  ]);
  const [submittingManual, setSubmittingManual] = useState<boolean>(false);

  const accountOptions = useMemo(() => {
    const known = new Set(CANONICAL_GL_ACCOUNTS.map((a) => a.account_code));
    if (accountCode && !known.has(accountCode)) {
      return [
        { account_code: accountCode, class: "ASSET" as const, name: accountCode, is_control: false },
        ...CANONICAL_GL_ACCOUNTS,
      ];
    }
    return CANONICAL_GL_ACCOUNTS;
  }, [accountCode]);

  const loadLedger = useCallback(
    async (silent = false) => {
      const cleanPeriod = period.trim();
      if (cleanPeriod && !PERIOD_RE.test(cleanPeriod)) {
        return;
      }
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      try {
        const res = await financeApi.getAccountLedger(accountCode, {
          period: cleanPeriod || undefined,
          limit,
        });
        setRows(res.items ?? []);
      } catch (err) {
        setError(err instanceof Error ? err.message : tRef.current("supplier_portal.finance.gl_ledger.load_failed"));
        setRows([]);
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [accountCode, financeApi, limit, period],
  );

  const inspectEntry = useCallback(
    async (entryId: string) => {
      const cleanId = entryId.trim();
      if (!cleanId) return;
      setInspectLoading(true);
      setInspectError(null);
      try {
        const rec = await financeApi.getJournalEntry(cleanId);
        setInspectedEntry(rec);
      } catch (err) {
        setInspectError(err instanceof Error ? err.message : tRef.current("supplier_portal.finance.gl_ledger.load_failed"));
        setInspectedEntry(null);
      } finally {
        setInspectLoading(false);
      }
    },
    [financeApi],
  );

  useEffect(() => {
    void loadLedger();
  }, [loadLedger]);

  useEffect(() => {
    if (initialEntryId) {
      void inspectEntry(initialEntryId);
    }
  }, [initialEntryId, inspectEntry]);

  useSupplierSessionReconcile(() => {
    void loadLedger(true);
  });

  const balanceSummary = useMemo(
    () => evaluateManualJournalLines(manualLines, manualCurrency),
    [manualLines, manualCurrency],
  );

  const handleSubmitManual = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!balanceSummary.balanced || !manualMemo.trim() || !manualSourceId.trim()) {
      return;
    }
    setSubmittingManual(true);
    setError(null);
    setNotice(null);
    try {
      const res = await financeApi.postManualJournal({
        source_type: "MANUAL",
        source_id: manualSourceId.trim(),
        memo: manualMemo.trim(),
        currency: manualCurrency,
        period_id: PERIOD_RE.test(period.trim()) ? period.trim() : undefined,
        postings: balanceSummary.validLines,
      });
      setNotice(
        t("supplier_portal.finance.gl_ledger.manual_posted", { entryId: res.entry_id }),
      );
      setManualMemo("");
      setManualSourceId(`man-${Date.now()}`);
      setManualLines([
        { account_code: accountCode, direction: "DEBIT", amount_major: "", description: "" },
        { account_code: "EQUITY:OPENING_BALANCE", direction: "CREDIT", amount_major: "", description: "" },
      ]);
      setManualOpen(false);
      await loadLedger(true);
      await inspectEntry(res.entry_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.finance.gl_ledger.load_failed"));
    } finally {
      setSubmittingManual(false);
    }
  };

  const handleReverse = async () => {
    if (!inspectedEntry) return;
    setReversing(true);
    setInspectError(null);
    setNotice(null);
    try {
      const res = await financeApi.reverseJournalEntry(inspectedEntry.entry_id, {
        reason: reverseReason.trim() || undefined,
      });
      setNotice(
        t("supplier_portal.finance.gl_ledger.reverse_success", { entryId: res.entry_id }),
      );
      setReverseReason("");
      await loadLedger(true);
      await inspectEntry(res.entry_id);
    } catch (err) {
      setInspectError(err instanceof Error ? err.message : t("supplier_portal.finance.gl_ledger.load_failed"));
    } finally {
      setReversing(false);
    }
  };

  return (
    <PageChrome
      icon="ledger"
      title={t("supplier_portal.finance.gl_ledger.title")}
      description={t("supplier_portal.finance.gl_ledger.subtitle")}
      loading={loading && rows.length === 0 && !error}
      skeletonVariant="table"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/finance/trial-balance" as Route} className="portal-btn portal-btn--outline">
            <Icon name="arrowBack" size={16} />
            <span>{t("portal.nav.trial_balance")}</span>
          </Link>
          <button
            type="button"
            onClick={() => setManualOpen((v) => !v)}
            className="portal-btn portal-btn--primary"
          >
            <Icon name="plus" size={16} />
            <span>
              {manualOpen
                ? t("supplier_portal.finance.gl_ledger.cancel_manual_button")
                : t("supplier_portal.finance.gl_ledger.post_manual_button")}
            </span>
          </button>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadLedger(true)}
            className="portal-btn portal-btn--outline"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      {error ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <p className="portal-alert-text">{error}</p>
        </div>
      ) : null}

      {notice ? (
        <div className="portal-alert portal-alert--info mb-6">
          <p className="portal-alert-text">{notice}</p>
        </div>
      ) : null}

      <div className="desk-card p-4 mb-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
          <div>
            <label htmlFor="gl-account-select" className="portal-label block mb-2">
              {t("supplier_portal.finance.gl_ledger.account_code_label")}
            </label>
            <PortalSelect
              id="gl-account-select"
              value={accountCode}
              onChange={(e) => setAccountCode(e.target.value)}
            >
              {accountOptions.map((acct) => (
                <option key={acct.account_code} value={acct.account_code}>
                  {acct.account_code} — {acct.name}
                </option>
              ))}
            </PortalSelect>
          </div>

          <div>
            <label htmlFor="gl-period-input" className="portal-label block mb-2">
              {t("supplier_portal.finance.gl_ledger.period_label")}
            </label>
            <div className="flex gap-2">
              <PortalInput
                id="gl-period-input"
                type="month"
                value={period}
                onChange={(e) => setPeriod(e.target.value)}
              />
              {period ? (
                <button
                  type="button"
                  onClick={() => setPeriod("")}
                  className="portal-btn portal-btn--ghost px-3"
                >
                  {t("supplier_portal.finance.gl_ledger.close_inspector")}
                </button>
              ) : null}
            </div>
          </div>

          <div>
            <label htmlFor="gl-limit-select" className="portal-label block mb-2">
              {t("supplier_portal.finance.gl_ledger.limit_label")}
            </label>
            <PortalSelect
              id="gl-limit-select"
              value={String(limit)}
              onChange={(e) => setLimit(Number(e.target.value) || 50)}
            >
              <option value="25">25</option>
              <option value="50">50</option>
              <option value="100">100</option>
              <option value="200">200</option>
            </PortalSelect>
          </div>
        </div>
      </div>

      {manualOpen ? (
        <div className="mb-6">
          <PortalSection
            icon="edit"
            title={t("supplier_portal.finance.gl_ledger.manual_title")}
            description={t("supplier_portal.finance.gl_ledger.manual_subtitle")}
          >
            <form onSubmit={(e) => void handleSubmitManual(e)} className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label htmlFor="manual-source-id" className="portal-label block mb-2">
                    {t("supplier_portal.finance.gl_ledger.source_id_label")}
                  </label>
                  <PortalInput
                    id="manual-source-id"
                    value={manualSourceId}
                    onChange={(e) => setManualSourceId(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label htmlFor="manual-currency" className="portal-label block mb-2">
                    {t("supplier_portal.finance.gl_ledger.currency_label")}
                  </label>
                  <PortalSelect
                    id="manual-currency"
                    value={manualCurrency}
                    onChange={(e) => setManualCurrency(e.target.value)}
                  >
                    <option value="UZS">UZS</option>
                    <option value="KZT">KZT</option>
                    <option value="USD">USD</option>
                  </PortalSelect>
                </div>
                <div>
                  <label htmlFor="manual-memo" className="portal-label block mb-2">
                    {t("supplier_portal.finance.gl_ledger.memo_label")}
                  </label>
                  <PortalInput
                    id="manual-memo"
                    value={manualMemo}
                    onChange={(e) => setManualMemo(e.target.value)}
                    required
                  />
                </div>
              </div>

              <div className="space-y-3">
                {manualLines.map((line, idx) => (
                  <div
                    key={idx}
                    className="grid grid-cols-1 md:grid-cols-12 gap-3 items-end p-3 rounded-xl border"
                    style={{
                      borderColor: "var(--desk-border)",
                      background: "var(--desk-surface-subtle)",
                    }}
                  >
                    <div className="md:col-span-4">
                      <label htmlFor={`manual-line-acct-${idx}`} className="portal-label block mb-1">
                        {t("supplier_portal.finance.gl_ledger.account_code_label")}
                      </label>
                      <PortalSelect
                        id={`manual-line-acct-${idx}`}
                        value={line.account_code}
                        onChange={(e) =>
                          setManualLines((prev) =>
                            prev.map((item, i) => (i === idx ? { ...item, account_code: e.target.value } : item)),
                          )
                        }
                      >
                        {accountOptions.map((acct) => (
                          <option key={acct.account_code} value={acct.account_code}>
                            {acct.account_code} — {acct.name}
                          </option>
                        ))}
                      </PortalSelect>
                    </div>

                    <div className="md:col-span-2">
                      <label htmlFor={`manual-line-dir-${idx}`} className="portal-label block mb-1">
                        {t("supplier_portal.finance.gl_ledger.direction")}
                      </label>
                      <PortalSelect
                        id={`manual-line-dir-${idx}`}
                        value={line.direction}
                        onChange={(e) =>
                          setManualLines((prev) =>
                            prev.map((item, i) =>
                              i === idx ? { ...item, direction: e.target.value as GlDirection } : item,
                            ),
                          )
                        }
                      >
                        <option value="DEBIT">{t("supplier_portal.finance.gl_ledger.direction_debit")}</option>
                        <option value="CREDIT">{t("supplier_portal.finance.gl_ledger.direction_credit")}</option>
                      </PortalSelect>
                    </div>

                    <div className="md:col-span-3">
                      <label htmlFor={`manual-line-amt-${idx}`} className="portal-label block mb-1">
                        {t("supplier_portal.finance.gl_ledger.amount_major_label")}
                      </label>
                      <PortalInput
                        id={`manual-line-amt-${idx}`}
                        type="text"
                        inputMode="decimal"
                        placeholder="0.00"
                        value={line.amount_major}
                        onChange={(e) =>
                          setManualLines((prev) =>
                            prev.map((item, i) => (i === idx ? { ...item, amount_major: e.target.value } : item)),
                          )
                        }
                      />
                    </div>

                    <div className="md:col-span-2">
                      <label htmlFor={`manual-line-desc-${idx}`} className="portal-label block mb-1">
                        {t("supplier_portal.finance.gl_ledger.description_label")}
                      </label>
                      <PortalInput
                        id={`manual-line-desc-${idx}`}
                        type="text"
                        value={line.description ?? ""}
                        onChange={(e) =>
                          setManualLines((prev) =>
                            prev.map((item, i) => (i === idx ? { ...item, description: e.target.value } : item)),
                          )
                        }
                      />
                    </div>

                    <div className="md:col-span-1 flex justify-end">
                      <button
                        type="button"
                        disabled={manualLines.length <= 2}
                        onClick={() =>
                          setManualLines((prev) => prev.filter((_, i) => i !== idx))
                        }
                        className="portal-btn portal-btn--ghost px-3"
                      >
                        {t("supplier_portal.finance.gl_ledger.remove_line")}
                      </button>
                    </div>
                  </div>
                ))}
              </div>

              <div className="flex flex-wrap items-center justify-between gap-4 pt-2 border-t" style={{ borderColor: "var(--desk-border)" }}>
                <div className="flex flex-wrap items-center gap-4 text-sm">
                  <button
                    type="button"
                    onClick={() =>
                      setManualLines((prev) => [
                        ...prev,
                        { account_code: accountCode, direction: "DEBIT", amount_major: "", description: "" },
                      ])
                    }
                    className="portal-btn portal-btn--outline"
                  >
                    <Icon name="plus" size={14} />
                    <span>{t("supplier_portal.finance.gl_ledger.add_line")}</span>
                  </button>
                  <span className="font-mono">
                    {t("supplier_portal.finance.gl_ledger.debit_total")}:{" "}
                    <strong>{formatFinanceMinor(balanceSummary.debitMinor, manualCurrency)}</strong>
                  </span>
                  <span className="font-mono">
                    {t("supplier_portal.finance.gl_ledger.credit_total")}:{" "}
                    <strong>{formatFinanceMinor(balanceSummary.creditMinor, manualCurrency)}</strong>
                  </span>
                  <span
                    className="text-xs font-semibold"
                    style={{
                      color: balanceSummary.balanced ? "var(--desk-text-primary)" : "var(--desk-text-secondary)",
                    }}
                  >
                    {balanceSummary.balanced
                      ? t("supplier_portal.finance.gl_ledger.balance_check_ok")
                      : t("supplier_portal.finance.gl_ledger.balance_check_err")}
                  </span>
                </div>

                <button
                  type="submit"
                  disabled={!balanceSummary.balanced || !manualMemo.trim() || !manualSourceId.trim() || submittingManual}
                  className="portal-btn portal-btn--primary"
                >
                  {submittingManual
                    ? t("supplier_portal.finance.gl_ledger.submitting")
                    : t("supplier_portal.finance.gl_ledger.submit_manual")}
                </button>
              </div>
            </form>
          </PortalSection>
        </div>
      ) : null}

      {(inspectLoading || inspectError || inspectedEntry) ? (
        <div className="mb-6">
          <PortalSection
            icon="search"
            title={
              inspectedEntry
                ? `${t("supplier_portal.finance.gl_ledger.inspector_title")} · ${inspectedEntry.entry_id}`
                : t("supplier_portal.finance.gl_ledger.inspector_title")
            }
            description={
              inspectedEntry
                ? `${inspectedEntry.source_type}:${inspectedEntry.source_id} · ${inspectedEntry.period_id}`
                : undefined
            }
            actions={
              <div className="flex flex-wrap items-center gap-2">
                {inspectedEntry ? (
                  <div className="inline-flex rounded-lg border p-0.5 text-xs" style={{ borderColor: "var(--desk-border)" }}>
                    <button
                      type="button"
                      onClick={() => setInspectorTab("postings")}
                      className={`rounded-md px-2.5 py-1 font-medium transition ${
                        inspectorTab === "postings"
                          ? "bg-[var(--desk-accent)] text-[var(--desk-accent-on)]"
                          : "text-[var(--desk-text-secondary)]"
                      }`}
                    >
                      {t("supplier_portal.audit.tab_postings")}
                    </button>
                    <button
                      type="button"
                      onClick={() => setInspectorTab("history")}
                      className={`rounded-md px-2.5 py-1 font-medium transition ${
                        inspectorTab === "history"
                          ? "bg-[var(--desk-accent)] text-[var(--desk-accent-on)]"
                          : "text-[var(--desk-text-secondary)]"
                      }`}
                    >
                      {t("supplier_portal.audit.tab_audit_history")}
                    </button>
                  </div>
                ) : null}
                <button
                  type="button"
                  onClick={() => {
                    setInspectedEntry(null);
                    setInspectError(null);
                    setInspectorTab("postings");
                  }}
                  className="portal-btn portal-btn--ghost"
                >
                  {t("supplier_portal.finance.gl_ledger.close_inspector")}
                </button>
              </div>
            }
          >
            {inspectLoading ? (
              <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                {t("common.status.loading")}
              </p>
            ) : null}

            {inspectError ? (
              <div className="portal-alert portal-alert--error" role="alert">
                <p className="portal-alert-text">{inspectError}</p>
              </div>
            ) : null}

            {inspectedEntry && inspectorTab === "history" ? (
              <ObjectHistoryPanel
                objectType="journal_entry"
                fallbackObjectTypes={["journal", "JournalEntry"]}
                objectId={inspectedEntry.entry_id}
                compact
                autoVerifyChain
              />
            ) : null}

            {inspectedEntry && inspectorTab === "postings" ? (
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs">
                  <div className="p-3 rounded-xl border" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>{t("supplier_portal.finance.gl_ledger.posted_by")}</div>
                    <div className="font-mono font-semibold mt-1">{inspectedEntry.posted_by || "system"}</div>
                  </div>
                  <div className="p-3 rounded-xl border" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>{t("supplier_portal.finance.gl_ledger.prev_hash")}</div>
                    <div className="font-mono truncate mt-1" title={inspectedEntry.prev_hash}>
                      {inspectedEntry.prev_hash}
                    </div>
                  </div>
                  <div className="p-3 rounded-xl border" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>{t("supplier_portal.finance.gl_ledger.entry_hash")}</div>
                    <div className="font-mono truncate mt-1" title={inspectedEntry.entry_hash}>
                      {inspectedEntry.entry_hash}
                    </div>
                  </div>
                </div>

                {inspectedEntry.reverses_entry_id ? (
                  <p className="text-xs font-mono" style={{ color: "var(--desk-text-secondary)" }}>
                    {t("supplier_portal.finance.gl_ledger.reverses_entry")}: {inspectedEntry.reverses_entry_id}
                  </p>
                ) : null}

                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse text-xs">
                    <thead>
                      <tr className="border-b uppercase" style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}>
                        <th className="py-2 px-3">#</th>
                        <th className="py-2 px-3">{t("supplier_portal.finance.trial_balance.account_code")}</th>
                        <th className="py-2 px-3">{t("supplier_portal.finance.gl_ledger.direction")}</th>
                        <th className="py-2 px-3 text-right">{t("supplier_portal.finance.gl_ledger.amount")}</th>
                        <th className="py-2 px-3">{t("supplier_portal.finance.gl_ledger.description_label")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {(inspectedEntry.postings ?? []).map((p) => (
                        <tr key={`${p.entry_id}:${p.line}`}>
                          <td className="py-2 px-3 font-mono">{p.line}</td>
                          <td className="py-2 px-3 font-mono font-semibold">{p.account_code}</td>
                          <td className="py-2 px-3 font-mono">{p.direction}</td>
                          <td className="py-2 px-3 font-mono text-right">
                            {formatFinanceMinor(p.amount_minor, p.currency || inspectedEntry.currency)}
                          </td>
                          <td className="py-2 px-3" style={{ color: "var(--desk-text-secondary)" }}>
                            {p.description || inspectedEntry.memo || "—"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {!inspectedEntry.reverses_entry_id ? (
                  <div className="flex flex-wrap items-center gap-3 pt-3 border-t" style={{ borderColor: "var(--desk-border)" }}>
                    <PortalInput
                      id="reverse-reason-input"
                      aria-label={t("supplier_portal.finance.gl_ledger.reverse_reason_placeholder")}
                      placeholder={t("supplier_portal.finance.gl_ledger.reverse_reason_placeholder")}
                      value={reverseReason}
                      onChange={(e) => setReverseReason(e.target.value)}
                      className="max-w-md"
                    />
                    <button
                      type="button"
                      disabled={reversing}
                      onClick={() => void handleReverse()}
                      className="portal-btn portal-btn--outline"
                    >
                      {reversing
                        ? t("supplier_portal.finance.gl_ledger.reversing")
                        : t("supplier_portal.finance.gl_ledger.reverse_button")}
                    </button>
                  </div>
                ) : null}
              </div>
            ) : null}
          </PortalSection>
        </div>
      ) : null}

      <PortalSection
        icon="ledger"
        title={`${t("supplier_portal.finance.gl_ledger.postings_title")} · ${accountCode}`}
        description={t("supplier_portal.finance.gl_ledger.postings_desc")}
      >
        {rows.length === 0 ? (
          <div className="py-10 text-center text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
            {t("supplier_portal.finance.gl_ledger.empty")}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-sm">
              <thead>
                <tr
                  className="border-b text-xs uppercase tracking-wider"
                  style={{
                    borderColor: "var(--desk-border)",
                    color: "var(--desk-text-tertiary)",
                    background: "var(--desk-surface-subtle)",
                  }}
                >
                  <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.gl_ledger.occurred_at")}</th>
                  <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.gl_ledger.entry_id")}</th>
                  <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.gl_ledger.source")}</th>
                  <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.gl_ledger.memo_label")}</th>
                  <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.gl_ledger.direction")}</th>
                  <th className="py-3 px-4 font-semibold text-right">{t("supplier_portal.finance.gl_ledger.amount")}</th>
                  <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.gl_ledger.hash")}</th>
                  <th className="py-3 px-4 font-semibold text-right">{t("supplier_portal.finance.gl_ledger.actions")}</th>
                </tr>
              </thead>
              <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                {rows.map((row) => (
                  <tr key={`${row.entry_id}:${row.line}`} className="hover:bg-[var(--desk-surface-subtle)] transition-colors">
                    <td className="py-3 px-4 text-xs whitespace-nowrap" style={{ color: "var(--desk-text-secondary)" }}>
                      {new Date(row.occurred_at).toLocaleString()}
                    </td>
                    <td className="py-3 px-4 font-mono text-xs font-semibold">
                      {row.entry_id}
                    </td>
                    <td className="py-3 px-4 text-xs">
                      <div className="font-mono">{row.source_type}</div>
                      <div className="font-mono" style={{ color: "var(--desk-text-tertiary)" }}>
                        {row.order_id || row.source_id}
                      </div>
                    </td>
                    <td className="py-3 px-4 text-xs" style={{ color: "var(--desk-text-primary)" }}>
                      {row.description || row.memo || "—"}
                    </td>
                    <td className="py-3 px-4">
                      <span
                        className="inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-mono font-semibold"
                        style={{
                          borderColor: "var(--desk-border)",
                          background: row.direction === "DEBIT" ? "var(--desk-surface-subtle)" : "var(--desk-surface)",
                          color: "var(--desk-text-primary)",
                        }}
                      >
                        {row.direction}
                      </span>
                    </td>
                    <td className="py-3 px-4 font-mono font-semibold text-right whitespace-nowrap">
                      {formatFinanceMinor(row.amount_minor, row.currency)}
                    </td>
                    <td className="py-3 px-4 font-mono text-xs" title={row.entry_hash} style={{ color: "var(--desk-text-tertiary)" }}>
                      {row.entry_hash.slice(0, 12)}…
                    </td>
                    <td className="py-3 px-4 text-right">
                      <button
                        type="button"
                        onClick={() => void inspectEntry(row.entry_id)}
                        className="rounded-lg border px-2.5 py-1 text-xs font-semibold hover:bg-[var(--desk-surface-subtle)]"
                        style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-primary)" }}
                      >
                        {t("supplier_portal.finance.gl_ledger.inspect_entry")}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </PortalSection>
    </PageChrome>
  );
}
