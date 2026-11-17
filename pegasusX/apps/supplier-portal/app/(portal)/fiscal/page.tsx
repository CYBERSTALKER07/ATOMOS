"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { currentAccountingPeriod } from "@/lib/finance-api";
import {
  createControlAccountingApi,
  evaluateVatTieOut,
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  parseMajorToSafeMinor,
  type EsfDetailResponse,
  type EsfDocument,
  type EsfStatusHistoryItem,
  type MissingIkpuItem,
  type VatSummaryReport,
} from "@/lib/control-accounting-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";

type FiscalTab = "registry" | "vat" | "ikpu";

const PERIOD_RE = /^\d{4}-(0[1-9]|1[0-2])$/;
const DIRECTIONS = ["ALL", "OUTBOUND", "INBOUND"] as const;
const ESF_STATUSES = ["ALL", "DRAFT", "SUBMITTED", "ACCEPTED", "REJECTED", "CANCELLED", "FAILED"] as const;

function normalizeTab(raw: string | null): FiscalTab {
  switch ((raw || "").trim().toLowerCase()) {
    case "vat":
      return "vat";
    case "ikpu":
    case "mxik":
      return "ikpu";
    default:
      return "registry";
  }
}

function shortHash(hash?: string): string {
  const clean = (hash || "").trim();
  if (!clean) return "—";
  if (clean.length <= 16) return clean;
  return `${clean.slice(0, 8)}…${clean.slice(-6)}`;
}

function formatIso(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

export default function FiscalEsfConsolePage() {
  return (
    <Suspense fallback={null}>
      <FiscalEsfConsoleInner />
    </Suspense>
  );
}

function FiscalEsfConsoleInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const controlApi = useMemo(() => createControlAccountingApi(createSupplierApi()), []);

  const initialTab = normalizeTab(searchParams.get("tab"));
  const initialDirection = (searchParams.get("direction") || "ALL").trim().toUpperCase();
  const initialPeriod = (searchParams.get("period") || "").trim();
  const initialEsfId = (searchParams.get("esfId") || "").trim();
  const entityParam = (searchParams.get("entity") || "").trim();

  const [activeTab, setActiveTab] = useState<FiscalTab>(initialTab);
  const [period, setPeriod] = useState<string>(
    PERIOD_RE.test(initialPeriod) ? initialPeriod : currentAccountingPeriod(),
  );
  const [directionFilter, setDirectionFilter] = useState<string>(
    DIRECTIONS.includes(initialDirection as (typeof DIRECTIONS)[number]) ? initialDirection : "ALL",
  );
  const [statusFilter, setStatusFilter] = useState<string>("ALL");
  const [orderIdFilter, setOrderIdFilter] = useState<string>("");

  const [documents, setDocuments] = useState<EsfDocument[]>([]);
  const [selectedEsfId, setSelectedEsfId] = useState<string>(initialEsfId);
  const [selectedDetail, setSelectedDetail] = useState<EsfDetailResponse | null>(null);
  const [detailLoading, setDetailLoading] = useState<boolean>(false);
  const [transitionReason, setTransitionReason] = useState<string>("");

  const [vatSummary, setVatSummary] = useState<VatSummaryReport | null>(null);
  const [missingIkpu, setMissingIkpu] = useState<MissingIkpuItem[]>([]);

  // Outbound ESF creation form
  const [showOutboundForm, setShowOutboundForm] = useState<boolean>(false);
  const [outOrderId, setOutOrderId] = useState<string>("");
  const [outOperator, setOutOperator] = useState<string>("direct");
  const [outCounterpartyTin, setOutCounterpartyTin] = useState<string>("");
  const [outCounterpartyName, setOutCounterpartyName] = useState<string>("");
  const [outSubmitNow, setOutSubmitNow] = useState<boolean>(true);

  // Inbound ESF creation form
  const [showInboundForm, setShowInboundForm] = useState<boolean>(false);
  const [inPoId, setInPoId] = useState<string>("");
  const [inCounterpartyTin, setInCounterpartyTin] = useState<string>("");
  const [inCounterpartyName, setInCounterpartyName] = useState<string>("");
  const [inOperatorRef, setInOperatorRef] = useState<string>("");
  const [inTotalMajor, setInTotalMajor] = useState<string>("");
  const [inVatMajor, setInVatMajor] = useState<string>("");
  const [inVatRateBps, setInVatRateBps] = useState<string>("1200");

  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorCode, setErrorCode] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    const paramTab = searchParams.get("tab");
    if (paramTab) setActiveTab(normalizeTab(paramTab));
    const esfParam = (searchParams.get("esfId") || "").trim();
    if (esfParam) setSelectedEsfId(esfParam);
  }, [searchParams]);

  const loadAll = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      setErrorCode(null);

      const cleanPeriod = PERIOD_RE.test(period.trim()) ? period.trim() : undefined;

      const results = await Promise.allSettled([
        controlApi.listEsfDocuments({
          entity_id: entityParam || undefined,
          direction: directionFilter !== "ALL" ? directionFilter : undefined,
          status: statusFilter !== "ALL" ? statusFilter : undefined,
          period_id: cleanPeriod,
          order_id: orderIdFilter.trim() || undefined,
          limit: 100,
        }),
        controlApi.getVatSummary({
          entity_id: entityParam || undefined,
          period_id: cleanPeriod,
        }),
        controlApi.listMissingIkpu({
          entity_id: entityParam || undefined,
        }),
      ]);

      if (results[0].status === "fulfilled") {
        const docs = results[0].value.documents ?? [];
        setDocuments(docs);
        if (!selectedEsfId && docs.length > 0) {
          setSelectedEsfId(docs[0].esf_id);
        }
      }
      if (results[1].status === "fulfilled") {
        setVatSummary(results[1].value);
      }
      if (results[2].status === "fulfilled") {
        setMissingIkpu(results[2].value.items ?? []);
      }

      const firstErr = results.find((r) => r.status === "rejected");
      if (firstErr && firstErr.status === "rejected") {
        const parsed = extractControlError(firstErr.reason, tRef.current("supplier_portal.fiscal.load_failed"));
        setError(parsed.message);
        setErrorCode(parsed.code ?? null);
      }

      setLoading(false);
      setRefreshing(false);
    },
    [controlApi, directionFilter, entityParam, orderIdFilter, period, selectedEsfId, statusFilter],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const loadDetail = useCallback(
    async (esfId: string) => {
      const clean = esfId.trim();
      if (!clean) {
        setSelectedDetail(null);
        return;
      }
      setDetailLoading(true);
      try {
        const res = await controlApi.getEsfDocument(clean, {
          entity_id: entityParam || undefined,
        });
        setSelectedDetail(res);
      } catch (err) {
        const parsed = extractControlError(err, tRef.current("supplier_portal.fiscal.load_failed"));
        setError(parsed.message);
        setErrorCode(parsed.code ?? null);
      } finally {
        setDetailLoading(false);
      }
    },
    [controlApi, entityParam],
  );

  useEffect(() => {
    if (selectedEsfId) {
      void loadDetail(selectedEsfId);
    } else {
      setSelectedDetail(null);
    }
  }, [loadDetail, selectedEsfId]);

  const activeDoc: EsfDocument | null = useMemo(() => {
    if (selectedDetail?.document && selectedDetail.document.esf_id === selectedEsfId) {
      return selectedDetail.document;
    }
    return documents.find((d) => d.esf_id === selectedEsfId) ?? null;
  }, [documents, selectedDetail, selectedEsfId]);

  const timeline: EsfStatusHistoryItem[] = selectedDetail?.history ?? [];

  const handleResend = async (esfId: string) => {
    setBusyAction(`resend-${esfId}`);
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.resendEsfDocument(esfId, {
        entity_id: entityParam || undefined,
      });
      const alreadySelected = selectedEsfId === res.esf_id;
      setSelectedEsfId(res.esf_id);
      setNotice(
        t("supplier_portal.fiscal.resend_success", {
          esfId: res.esf_id,
          status: res.status,
        }),
      );
      await Promise.all([
        loadAll(true),
        alreadySelected ? loadDetail(res.esf_id) : Promise.resolve(),
      ]);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.fiscal.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleTransitionStatus = async (action: "ACCEPT" | "REJECT" | "CANCEL") => {
    if (!activeDoc) return;
    setBusyAction(`status-${action}`);
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.transitionEsfStatus(activeDoc.esf_id, {
        entity_id: entityParam || undefined,
        action,
        reason: transitionReason.trim() || undefined,
      });
      setSelectedDetail(res);
      setTransitionReason("");
      setNotice(
        t("supplier_portal.fiscal.status_updated", {
          esfId: res.document.esf_id,
          status: res.document.status,
        }),
      );
      await loadAll(true);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.fiscal.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateOutbound = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!outOrderId.trim()) {
      setError(t("supplier_portal.fiscal.order_id_required"));
      return;
    }
    setBusyAction("create-outbound");
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.createOutboundEsf({
        entity_id: entityParam || undefined,
        order_id: outOrderId.trim(),
        operator: outOperator.trim() || undefined,
        counterparty_tin: outCounterpartyTin.trim() || undefined,
        counterparty_name: outCounterpartyName.trim() || undefined,
        submit: outSubmitNow,
      });
      const alreadySelected = selectedEsfId === res.esf_id;
      setSelectedEsfId(res.esf_id);
      setOutOrderId("");
      setShowOutboundForm(false);
      setNotice(
        t("supplier_portal.fiscal.outbound_created", {
          esfId: res.esf_id,
          status: res.status,
        }),
      );
      await Promise.all([
        loadAll(true),
        alreadySelected ? loadDetail(res.esf_id) : Promise.resolve(),
      ]);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.fiscal.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleRecordInbound = async (e: React.FormEvent) => {
    e.preventDefault();
    const totalMinor = parseMajorToSafeMinor(inTotalMajor, false);
    const vatMinor = parseMajorToSafeMinor(inVatMajor, false);
    const rateBps = Number.parseInt(inVatRateBps, 10);
    if (
      !inCounterpartyTin.trim() ||
      !inCounterpartyName.trim() ||
      !inOperatorRef.trim() ||
      totalMinor === null ||
      vatMinor === null ||
      !Number.isInteger(rateBps) ||
      rateBps < 0
    ) {
      setError(t("supplier_portal.fiscal.inbound_fields_invalid"));
      return;
    }
    setBusyAction("create-inbound");
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.recordInboundEsf({
        entity_id: entityParam || undefined,
        po_id: inPoId.trim() || undefined,
        counterparty_tin: inCounterpartyTin.trim(),
        counterparty_name: inCounterpartyName.trim(),
        total_minor: totalMinor,
        vat_minor: vatMinor,
        vat_rate_bps: rateBps,
        operator_ref: inOperatorRef.trim(),
        period_id: PERIOD_RE.test(period.trim()) ? period.trim() : undefined,
      });
      const alreadySelected = selectedEsfId === res.esf_id;
      setSelectedEsfId(res.esf_id);
      setInPoId("");
      setInOperatorRef("");
      setInTotalMajor("");
      setInVatMajor("");
      setShowInboundForm(false);
      setNotice(
        t("supplier_portal.fiscal.inbound_recorded", {
          esfId: res.esf_id,
        }),
      );
      await Promise.all([
        loadAll(true),
        alreadySelected ? loadDetail(res.esf_id) : Promise.resolve(),
      ]);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.fiscal.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const vatTieOut = useMemo(() => evaluateVatTieOut(vatSummary), [vatSummary]);
  const vatCurrency = vatSummary?.currency || "UZS";

  return (
    <PageChrome
      icon="manifests"
      title={t("supplier_portal.fiscal.title")}
      description={t("supplier_portal.fiscal.subtitle")}
      loading={loading && documents.length === 0 && !vatSummary && !error}
      skeletonVariant="table"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => {
              setShowOutboundForm((v) => !v);
              setShowInboundForm(false);
            }}
            className="portal-btn portal-btn--outline"
          >
            <Icon name="plus" size={16} />
            <span>{t("supplier_portal.fiscal.btn_issue_outbound")}</span>
          </button>
          <button
            type="button"
            onClick={() => {
              setShowInboundForm((v) => !v);
              setShowOutboundForm(false);
            }}
            className="portal-btn portal-btn--outline"
          >
            <Icon name="plus" size={16} />
            <span>{t("supplier_portal.fiscal.btn_record_inbound")}</span>
          </button>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadAll(true)}
            className="portal-btn portal-btn--primary"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{refreshing ? t("common.status.loading") : t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      {error ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <div>
            <p className="portal-alert-text font-semibold">{error}</p>
            {errorCode ? (
              <p className="text-xs font-mono mt-1" style={{ color: "var(--desk-danger)" }}>
                {errorCode}
              </p>
            ) : null}
          </div>
        </div>
      ) : null}

      {notice ? (
        <div className="portal-alert portal-alert--info mb-6">
          <p className="portal-alert-text">{notice}</p>
        </div>
      ) : null}

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.fiscal.kpi_esf_docs")}
          value={documents.length}
          sub={
            vatSummary
              ? `${vatSummary.outbound_count} OUT · ${vatSummary.inbound_count} IN · ${vatSummary.rejected_count} REJ`
              : period
          }
        />
        <KpiStatCard
          label={t("supplier_portal.fiscal.kpi_esf_net_vat")}
          value={vatSummary ? formatFinanceMinor(vatSummary.esf_net_vat_minor, vatCurrency) : "—"}
          sub={
            vatSummary
              ? `OUT ${formatFinanceMinor(vatSummary.esf_output_vat_minor, vatCurrency)} · IN ${formatFinanceMinor(vatSummary.esf_input_vat_minor, vatCurrency)}`
              : undefined
          }
        />
        <KpiStatCard
          label={t("supplier_portal.fiscal.kpi_gl_net_vat")}
          value={vatSummary ? formatFinanceMinor(vatSummary.gl_net_vat_minor, vatCurrency) : "—"}
          sub={
            vatSummary
              ? `OUT ${formatFinanceMinor(vatSummary.gl_output_vat_minor, vatCurrency)} · IN ${formatFinanceMinor(vatSummary.gl_input_vat_minor, vatCurrency)}`
              : undefined
          }
        />
        <KpiStatCard
          label={t("supplier_portal.fiscal.kpi_vat_tieout")}
          value={
            vatSummary ? (
              <div className="flex flex-wrap items-center gap-2">
                <span>
                  {formatFinanceMinor(vatTieOut.diffMinor, vatCurrency, { signDisplay: "always" })}
                </span>
                <StatusBadge
                  state={vatTieOut.isZeroDiff ? "PASS" : "FAIL"}
                  label={
                    vatTieOut.isZeroDiff
                      ? t("supplier_portal.fiscal.diff_minor_zero_badge")
                      : t("supplier_portal.fiscal.diff_minor_mismatch_badge")
                  }
                />
              </div>
            ) : (
              "—"
            )
          }
          sub={t("supplier_portal.fiscal.kpi_missing_ikpu", { count: missingIkpu.length })}
        />
      </KpiStatGrid>

      {showOutboundForm ? (
        <div className="mt-6">
          <PortalSection
            icon="plus"
            title={t("supplier_portal.fiscal.outbound_form_title")}
            description={t("supplier_portal.fiscal.outbound_form_desc")}
          >
            <form onSubmit={(e) => void handleCreateOutbound(e)} className="grid grid-cols-1 md:grid-cols-5 gap-3 items-end">
              <div>
                <label htmlFor="out-esf-order-id" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.order_id_label")}
                </label>
                <PortalInput
                  id="out-esf-order-id"
                  type="text"
                  placeholder="ord-..."
                  value={outOrderId}
                  onChange={(e) => setOutOrderId(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="out-esf-operator" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.operator_label")}
                </label>
                <PortalSelect
                  id="out-esf-operator"
                  value={outOperator}
                  onChange={(e) => setOutOperator(e.target.value)}
                >
                  <option value="direct">direct (Soliq EHF)</option>
                  <option value="didox">didox</option>
                  <option value="faktura">faktura</option>
                </PortalSelect>
              </div>
              <div>
                <label htmlFor="out-esf-tin" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.counterparty_tin_label")}
                </label>
                <PortalInput
                  id="out-esf-tin"
                  type="text"
                  placeholder="301234567"
                  value={outCounterpartyTin}
                  onChange={(e) => setOutCounterpartyTin(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="out-esf-name" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.counterparty_name_label")}
                </label>
                <PortalInput
                  id="out-esf-name"
                  type="text"
                  value={outCounterpartyName}
                  onChange={(e) => setOutCounterpartyName(e.target.value)}
                />
              </div>
              <div className="flex flex-col gap-2">
                <label className="flex items-center gap-2 text-xs cursor-pointer">
                  <input
                    type="checkbox"
                    checked={outSubmitNow}
                    onChange={(e) => setOutSubmitNow(e.target.checked)}
                  />
                  <span>{t("supplier_portal.fiscal.submit_immediately")}</span>
                </label>
                <button
                  type="submit"
                  disabled={busyAction !== null}
                  className="portal-btn portal-btn--primary"
                >
                  <span>{t("supplier_portal.fiscal.submit_outbound_button")}</span>
                </button>
              </div>
            </form>
          </PortalSection>
        </div>
      ) : null}

      {showInboundForm ? (
        <div className="mt-6">
          <PortalSection
            icon="plus"
            title={t("supplier_portal.fiscal.inbound_form_title")}
            description={t("supplier_portal.fiscal.inbound_form_desc")}
          >
            <form onSubmit={(e) => void handleRecordInbound(e)} className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
              <div>
                <label htmlFor="in-esf-po-id" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.po_id_label")}
                </label>
                <PortalInput
                  id="in-esf-po-id"
                  type="text"
                  placeholder="po-..."
                  value={inPoId}
                  onChange={(e) => setInPoId(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="in-esf-tin" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.counterparty_tin_label")}
                </label>
                <PortalInput
                  id="in-esf-tin"
                  type="text"
                  placeholder="301234567"
                  value={inCounterpartyTin}
                  onChange={(e) => setInCounterpartyTin(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="in-esf-name" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.counterparty_name_label")}
                </label>
                <PortalInput
                  id="in-esf-name"
                  type="text"
                  value={inCounterpartyName}
                  onChange={(e) => setInCounterpartyName(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="in-esf-ref" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.operator_ref_label")}
                </label>
                <PortalInput
                  id="in-esf-ref"
                  type="text"
                  placeholder="SOLIQ-IN-..."
                  value={inOperatorRef}
                  onChange={(e) => setInOperatorRef(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="in-esf-total" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.total_major_label")}
                </label>
                <PortalInput
                  id="in-esf-total"
                  type="text"
                  placeholder="1120000"
                  value={inTotalMajor}
                  onChange={(e) => setInTotalMajor(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="in-esf-vat" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.vat_major_label")}
                </label>
                <PortalInput
                  id="in-esf-vat"
                  type="text"
                  placeholder="120000"
                  value={inVatMajor}
                  onChange={(e) => setInVatMajor(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="in-esf-bps" className="portal-label block mb-1">
                  {t("supplier_portal.fiscal.vat_rate_bps_label")}
                </label>
                <PortalInput
                  id="in-esf-bps"
                  type="number"
                  min={0}
                  value={inVatRateBps}
                  onChange={(e) => setInVatRateBps(e.target.value)}
                />
              </div>
              <div>
                <button
                  type="submit"
                  disabled={busyAction !== null}
                  className="portal-btn portal-btn--primary w-full"
                >
                  <span>{t("supplier_portal.fiscal.submit_inbound_button")}</span>
                </button>
              </div>
            </form>
          </PortalSection>
        </div>
      ) : null}

      <div className="mt-6 mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-2" role="tablist" aria-label={t("supplier_portal.fiscal.tabs_label")}>
          {(
            [
              ["registry", "supplier_portal.fiscal.tab_registry"],
              ["vat", "supplier_portal.fiscal.tab_vat_summary"],
              ["ikpu", "supplier_portal.fiscal.tab_missing_ikpu"],
            ] as const
          ).map(([tabKey, labelKey]) => {
            const active = activeTab === tabKey;
            return (
              <button
                key={tabKey}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => setActiveTab(tabKey)}
                className={`portal-btn ${active ? "portal-btn--primary" : "portal-btn--outline"}`}
              >
                <span>{t(labelKey)}</span>
              </button>
            );
          })}
        </div>

        <div className="flex items-center gap-2">
          <label htmlFor="fiscal-period-input" className="portal-label">
            {t("supplier_portal.fiscal.period_label")}
          </label>
          <PortalInput
            id="fiscal-period-input"
            type="month"
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
          />
        </div>
      </div>

      {activeTab === "registry" ? (
        <div className="grid grid-cols-1 xl:grid-cols-5 gap-6">
          <div className="xl:col-span-3">
            <PortalSection
              icon="manifests"
              title={t("supplier_portal.fiscal.registry_title")}
              description={t("supplier_portal.fiscal.registry_desc")}
              actions={
                <div className="flex flex-wrap gap-2">
                  <PortalSelect
                    id="esf-direction-filter"
                    aria-label={t("supplier_portal.fiscal.direction_filter_label")}
                    value={directionFilter}
                    onChange={(e) => setDirectionFilter(e.target.value)}
                  >
                    {DIRECTIONS.map((dir) => (
                      <option key={dir} value={dir}>
                        {dir === "ALL" ? t("supplier_portal.fiscal.all_directions") : dir}
                      </option>
                    ))}
                  </PortalSelect>
                  <PortalSelect
                    id="esf-status-filter"
                    aria-label={t("supplier_portal.fiscal.status_filter_label")}
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value)}
                  >
                    {ESF_STATUSES.map((st) => (
                      <option key={st} value={st}>
                        {st === "ALL" ? t("supplier_portal.fiscal.all_statuses") : st}
                      </option>
                    ))}
                  </PortalSelect>
                  <PortalInput
                    id="esf-order-filter"
                    aria-label={t("supplier_portal.fiscal.order_id_label")}
                    type="text"
                    placeholder={t("supplier_portal.fiscal.order_filter_placeholder")}
                    value={orderIdFilter}
                    onChange={(e) => setOrderIdFilter(e.target.value)}
                  />
                </div>
              }
            >
              {documents.length === 0 ? (
                <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.fiscal.no_documents")}
                </p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse text-xs">
                    <thead>
                      <tr
                        className="border-b uppercase"
                        style={{
                          borderColor: "var(--desk-border)",
                          color: "var(--desk-text-tertiary)",
                          background: "var(--desk-surface-subtle)",
                        }}
                      >
                        <th className="py-2.5 px-3">{t("supplier_portal.fiscal.col_esf")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.fiscal.col_direction")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.fiscal.col_counterparty")}</th>
                        <th className="py-2.5 px-3 text-right">{t("supplier_portal.fiscal.col_total_vat")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.fiscal.col_status")}</th>
                        <th className="py-2.5 px-3 text-right">{t("supplier_portal.fiscal.col_actions")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {documents.map((doc) => {
                        const selected = doc.esf_id === selectedEsfId;
                        const currency = doc.currency || vatCurrency;
                        return (
                          <tr
                            key={doc.esf_id}
                            onClick={() => setSelectedEsfId(doc.esf_id)}
                            className="cursor-pointer transition-colors hover:bg-[var(--desk-surface-subtle)]"
                            style={selected ? { background: "var(--desk-surface-subtle)" } : undefined}
                          >
                            <td className="py-2.5 px-3">
                              <div className="font-mono font-semibold">{doc.esf_id}</div>
                              <div className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                                {doc.order_id || doc.po_id || doc.operator_ref || "—"}
                              </div>
                            </td>
                            <td className="py-2.5 px-3">
                              <StatusBadge state={doc.direction} />
                              <div className="font-mono text-[11px] mt-0.5" style={{ color: "var(--desk-text-tertiary)" }}>
                                {doc.operator}
                              </div>
                            </td>
                            <td className="py-2.5 px-3">
                              <div className="font-medium">{doc.counterparty_name}</div>
                              <div className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                                TIN {doc.counterparty_tin}
                              </div>
                            </td>
                            <td className="py-2.5 px-3 font-mono text-right">
                              <div className="font-medium">{formatFinanceMinor(doc.total_minor, currency)}</div>
                              <div className="text-[11px]" style={{ color: "var(--desk-text-secondary)" }}>
                                VAT {formatFinanceMinor(doc.vat_minor, currency)} ({formatRateBps(doc.vat_rate_bps)})
                              </div>
                            </td>
                            <td className="py-2.5 px-3">
                              <StatusBadge state={doc.status} />
                            </td>
                            <td className="py-2.5 px-3 text-right" onClick={(e) => e.stopPropagation()}>
                              <button
                                type="button"
                                disabled={busyAction !== null || doc.direction === "INBOUND"}
                                onClick={() => void handleResend(doc.esf_id)}
                                className="rounded-lg border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                                style={{ borderColor: "var(--desk-border)" }}
                              >
                                {t("supplier_portal.fiscal.resend_button")}
                              </button>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </PortalSection>
          </div>

          <div className="xl:col-span-2">
            <PortalSection
              icon="reconcile"
              title={t("supplier_portal.fiscal.timeline_title")}
              description={activeDoc ? activeDoc.esf_id : t("supplier_portal.fiscal.select_esf_hint")}
            >
              {detailLoading && !activeDoc ? (
                <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("common.status.loading")}
                </p>
              ) : !activeDoc ? (
                <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.fiscal.select_esf_hint")}
                </p>
              ) : (
                <div className="space-y-4">
                  <div
                    className="rounded-xl border p-3 text-xs space-y-1.5"
                    style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="font-mono font-semibold">{activeDoc.esf_id}</span>
                      <div className="flex items-center gap-1.5">
                        <StatusBadge state={activeDoc.direction} />
                        <StatusBadge state={activeDoc.status} />
                      </div>
                    </div>
                    <div className="flex justify-between gap-2">
                      <span style={{ color: "var(--desk-text-tertiary)" }}>Operator Ref</span>
                      <span className="font-mono">{activeDoc.operator_ref || "—"}</span>
                    </div>
                    <div className="flex justify-between gap-2">
                      <span style={{ color: "var(--desk-text-tertiary)" }}>Payload Hash</span>
                      <span className="font-mono" title={activeDoc.payload_hash}>
                        {shortHash(activeDoc.payload_hash)}
                      </span>
                    </div>
                    {activeDoc.signed_payload_uri ? (
                      <div className="flex justify-between gap-2">
                        <span style={{ color: "var(--desk-text-tertiary)" }}>Signed URI</span>
                        <span className="font-mono truncate max-w-[220px]">{activeDoc.signed_payload_uri}</span>
                      </div>
                    ) : null}
                    {activeDoc.failure_reason ? (
                      <div className="text-xs font-medium" style={{ color: "var(--desk-danger)" }}>
                        {activeDoc.failure_reason}
                      </div>
                    ) : null}
                  </div>

                  {selectedDetail?.gl_reversal_entry_id || selectedDetail?.vat_reversed_minor ? (
                    <div
                      className="rounded-xl border p-3 text-xs space-y-1"
                      style={{ borderColor: "var(--desk-warning)", background: "var(--desk-warning-soft)" }}
                    >
                      <div className="font-semibold">{t("supplier_portal.fiscal.reversal_notice")}</div>
                      <div className="font-mono">
                        VAT Reversed: {formatFinanceMinor(selectedDetail.vat_reversed_minor ?? 0, vatCurrency)}
                        {selectedDetail.gl_reversal_entry_id ? ` · GL ${selectedDetail.gl_reversal_entry_id}` : ""}
                        {selectedDetail.credit_note_id ? ` · CN ${selectedDetail.credit_note_id}` : ""}
                      </div>
                    </div>
                  ) : null}

                  <div className="space-y-2 border-t pt-3" style={{ borderColor: "var(--desk-border)" }}>
                    <label htmlFor="esf-transition-reason" className="portal-label block">
                      {t("supplier_portal.fiscal.transition_reason_label")}
                    </label>
                    <PortalInput
                      id="esf-transition-reason"
                      type="text"
                      placeholder={t("supplier_portal.fiscal.transition_reason_placeholder")}
                      value={transitionReason}
                      onChange={(e) => setTransitionReason(e.target.value)}
                    />
                    <div className="flex flex-wrap gap-2">
                      <button
                        type="button"
                        disabled={busyAction !== null || activeDoc.direction === "INBOUND"}
                        onClick={() => void handleResend(activeDoc.esf_id)}
                        className="portal-btn portal-btn--primary"
                      >
                        <Icon name="refresh" size={16} />
                        <span>{t("supplier_portal.fiscal.resend_button")}</span>
                      </button>
                      <button
                        type="button"
                        disabled={busyAction !== null}
                        onClick={() => void handleTransitionStatus("ACCEPT")}
                        className="portal-btn portal-btn--outline"
                      >
                        <span>{t("supplier_portal.fiscal.accept_button")}</span>
                      </button>
                      <button
                        type="button"
                        disabled={busyAction !== null}
                        onClick={() => void handleTransitionStatus("REJECT")}
                        className="portal-btn portal-btn--outline"
                      >
                        <span>{t("supplier_portal.fiscal.reject_button")}</span>
                      </button>
                      <button
                        type="button"
                        disabled={busyAction !== null}
                        onClick={() => void handleTransitionStatus("CANCEL")}
                        className="portal-btn portal-btn--ghost"
                      >
                        <span>{t("supplier_portal.fiscal.cancel_button")}</span>
                      </button>
                    </div>
                  </div>

                  <div className="space-y-2 border-t pt-3" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="text-xs font-semibold">
                      {t("supplier_portal.fiscal.status_history_title")} ({timeline.length})
                    </div>
                    {timeline.length === 0 ? (
                      <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.fiscal.no_history")}
                      </p>
                    ) : (
                      <ol className="space-y-2">
                        {timeline.map((ev) => (
                          <li
                            key={ev.history_id}
                            className="rounded-lg border p-2.5 text-xs space-y-1"
                            style={{ borderColor: "var(--desk-border)" }}
                          >
                            <div className="flex flex-wrap items-center justify-between gap-2">
                              <div className="flex items-center gap-1.5 font-mono">
                                {ev.from_status ? (
                                  <>
                                    <span>{ev.from_status}</span>
                                    <span>→</span>
                                  </>
                                ) : null}
                                <StatusBadge state={ev.to_status} />
                              </div>
                              <span className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                                {ev.actor_id}
                              </span>
                            </div>
                            {ev.reason ? (
                              <p style={{ color: "var(--desk-text-secondary)" }}>{ev.reason}</p>
                            ) : null}
                            <div className="text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                              {formatIso(ev.created_at)}
                            </div>
                          </li>
                        ))}
                      </ol>
                    )}
                  </div>
                </div>
              )}
            </PortalSection>
          </div>
        </div>
      ) : null}

      {activeTab === "vat" ? (
        <PortalSection
          icon="reconcile"
          title={t("supplier_portal.fiscal.vat_summary_title")}
          description={t("supplier_portal.fiscal.vat_summary_desc", { period })}
          actions={
            vatSummary ? (
              <StatusBadge
                state={vatTieOut.isZeroDiff ? "PASS" : "FAIL"}
                label={
                  vatTieOut.isZeroDiff
                    ? t("supplier_portal.fiscal.diff_minor_zero_badge")
                    : t("supplier_portal.fiscal.diff_minor_mismatch_badge")
                }
              />
            ) : null
          }
        >
          {!vatSummary ? (
            <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.fiscal.vat_summary_empty")}
            </p>
          ) : (
            <div className="space-y-6">
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
                      <th className="py-2.5 px-3">{t("supplier_portal.fiscal.vat_col_leg")}</th>
                      <th className="py-2.5 px-3 text-right">{t("supplier_portal.fiscal.vat_col_esf")}</th>
                      <th className="py-2.5 px-3 text-right">{t("supplier_portal.fiscal.vat_col_gl")}</th>
                      <th className="py-2.5 px-3 text-right">{t("supplier_portal.fiscal.vat_col_diff")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y font-mono" style={{ borderColor: "var(--desk-border)" }}>
                    <tr>
                      <td className="py-2.5 px-3 font-sans font-medium">
                        {t("supplier_portal.fiscal.vat_row_output")}
                      </td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatSummary.esf_output_vat_minor, vatCurrency)}
                      </td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatSummary.gl_output_vat_minor, vatCurrency)}
                      </td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatTieOut.outputDiffMinor, vatCurrency, { signDisplay: "always" })}
                      </td>
                    </tr>
                    <tr>
                      <td className="py-2.5 px-3 font-sans font-medium">
                        {t("supplier_portal.fiscal.vat_row_input")}
                      </td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatSummary.esf_input_vat_minor, vatCurrency)}
                      </td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatSummary.gl_input_vat_minor, vatCurrency)}
                      </td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatTieOut.inputDiffMinor, vatCurrency, { signDisplay: "always" })}
                      </td>
                    </tr>
                    <tr className="font-semibold" style={{ background: "var(--desk-surface-subtle)" }}>
                      <td className="py-2.5 px-3 font-sans">{t("supplier_portal.fiscal.vat_row_net")}</td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatSummary.esf_net_vat_minor, vatCurrency)}
                      </td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatSummary.gl_net_vat_minor, vatCurrency)}
                      </td>
                      <td className="py-2.5 px-3 text-right">
                        {formatFinanceMinor(vatTieOut.diffMinor, vatCurrency, { signDisplay: "always" })}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {vatSummary.by_rate && vatSummary.by_rate.length > 0 ? (
                <div>
                  <h3 className="text-xs font-semibold uppercase tracking-wider mb-2" style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.fiscal.vat_by_rate_title")}
                  </h3>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse text-xs font-mono">
                      <thead>
                        <tr className="border-b uppercase" style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}>
                          <th className="py-2 px-3">{t("supplier_portal.fiscal.vat_col_rate")}</th>
                          <th className="py-2 px-3 text-right">{t("supplier_portal.fiscal.vat_col_out_base")}</th>
                          <th className="py-2 px-3 text-right">{t("supplier_portal.fiscal.vat_col_out_vat")}</th>
                          <th className="py-2 px-3 text-right">{t("supplier_portal.fiscal.vat_col_in_base")}</th>
                          <th className="py-2 px-3 text-right">{t("supplier_portal.fiscal.vat_col_in_vat")}</th>
                          <th className="py-2 px-3 text-right">{t("supplier_portal.fiscal.vat_col_docs")}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                        {vatSummary.by_rate.map((row) => (
                          <tr key={row.vat_rate_bps}>
                            <td className="py-2 px-3 font-semibold">{formatRateBps(row.vat_rate_bps)}</td>
                            <td className="py-2 px-3 text-right">{formatFinanceMinor(row.outbound_base_minor, vatCurrency)}</td>
                            <td className="py-2 px-3 text-right">{formatFinanceMinor(row.outbound_vat_minor, vatCurrency)}</td>
                            <td className="py-2 px-3 text-right">{formatFinanceMinor(row.inbound_base_minor, vatCurrency)}</td>
                            <td className="py-2 px-3 text-right">{formatFinanceMinor(row.inbound_vat_minor, vatCurrency)}</td>
                            <td className="py-2 px-3 text-right">{row.doc_count}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              ) : null}
            </div>
          )}
        </PortalSection>
      ) : null}

      {activeTab === "ikpu" ? (
        <PortalSection
          icon="warning"
          title={t("supplier_portal.fiscal.missing_ikpu_title")}
          description={t("supplier_portal.fiscal.missing_ikpu_desc")}
          actions={
            <Link href={"/catalog" as Route} className="portal-btn portal-btn--outline">
              <Icon name="catalog" size={16} />
              <span>{t("portal.nav.catalog")}</span>
            </Link>
          }
        >
          {missingIkpu.length === 0 ? (
            <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.fiscal.missing_ikpu_empty")}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr
                    className="border-b uppercase"
                    style={{
                      borderColor: "var(--desk-border)",
                      color: "var(--desk-text-tertiary)",
                      background: "var(--desk-surface-subtle)",
                    }}
                  >
                    <th className="py-2.5 px-3">{t("supplier_portal.fiscal.ikpu_col_product")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.fiscal.ikpu_col_barcode")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.fiscal.ikpu_col_ikpu")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.fiscal.ikpu_col_package")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.fiscal.ikpu_col_marking")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.fiscal.ikpu_col_missing")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {missingIkpu.map((item) => (
                    <tr key={item.product_id}>
                      <td className="py-2.5 px-3">
                        <div className="font-medium">{item.name}</div>
                        <div className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                          {item.product_id}
                        </div>
                      </td>
                      <td className="py-2.5 px-3 font-mono">{item.barcode || "—"}</td>
                      <td className="py-2.5 px-3 font-mono">{item.ikpu_code || "—"}</td>
                      <td className="py-2.5 px-3 font-mono">{item.package_code || "—"}</td>
                      <td className="py-2.5 px-3 font-mono">{item.marking_code || "—"}</td>
                      <td className="py-2.5 px-3">
                        <div className="flex flex-wrap gap-1">
                          {(item.missing_fields ?? []).map((f) => (
                            <StatusBadge key={f} state="WARN" label={f} />
                          ))}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </PortalSection>
      ) : null}
    </PageChrome>
  );
}
