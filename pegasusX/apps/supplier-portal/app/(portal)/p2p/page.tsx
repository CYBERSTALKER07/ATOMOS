"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import { resolvePortalSessionRole } from "@/lib/connectors-auditor-api";
import { currentAccountingPeriod } from "@/lib/finance-api";
import { usePortalT } from "@/lib/i18n";
import {
  computePurchaseOrderPreviewTotals,
  createP2aApi,
  extractControlError,
  formatDurationSec,
  formatFinanceMinor,
  parseMajorToSafeMinor,
} from "@/lib/p2a-api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";
import type {
  GoodsReceipt,
  LegalEntity,
  ProcessInstance,
  ProcessMiningReport,
  ProcessStep,
  ProcessTemplate,
  PurchaseOrder,
  ThreeWayMatchResult,
  VendorInvoice,
} from "@pegasusx/types";

type P2PTab = "orders" | "receipts_invoices" | "three_way_match" | "process_mining";

const CURRENCIES = ["UZS", "USD", "EUR", "RUB"] as const;
const DOMAINS = ["ALL", "P2P", "O2C", "RETURNS"] as const;
const DEFAULT_TEMPLATE_STEPS_JSON = JSON.stringify(
  [
    { code: "CREATED", name: "Created", sequence: 1, sla_seconds: 3600 },
    { code: "APPROVED", name: "Approved", sequence: 2, sla_seconds: 14400 },
    { code: "RECEIVED", name: "Received", sequence: 3, sla_seconds: 86400 },
    { code: "MATCHED", name: "Matched", sequence: 4, sla_seconds: 43200 },
  ],
  null,
  2,
);

function normalizeTab(raw: string | null): P2PTab {
  switch ((raw || "").trim().toLowerCase()) {
    case "receipts":
    case "receipts_invoices":
    case "invoices":
      return "receipts_invoices";
    case "match":
    case "three_way_match":
    case "three-way-match":
      return "three_way_match";
    case "mining":
    case "process_mining":
    case "processes":
      return "process_mining";
    default:
      return "orders";
  }
}

function mapStatusBadgeState(status: string): string {
  switch ((status || "").toUpperCase()) {
    case "APPROVED":
    case "RECEIVED":
    case "CLOSED":
    case "MATCHED_3WAY":
    case "POSTED":
    case "COMPLETED":
    case "ACTIVE":
      return "COMPLETED";
    case "DISCREPANCY":
    case "CANCELLED":
      return "FAIL";
    case "PARTIAL_RECEIVED":
      return "PARTIAL";
    case "PENDING_APPROVAL":
    case "PENDING":
      return "PENDING_APPROVAL";
    default:
      return status || "DRAFT";
  }
}

interface DraftPOLine {
  sku: string;
  product_name: string;
  qty_ordered: string;
  unit_cost_major: string;
  vat_bps: string;
}

export default function P2PPage() {
  return (
    <Suspense fallback={null}>
      <P2PPageInner />
    </Suspense>
  );
}

function P2PPageInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const p2aApi = useMemo(() => createP2aApi(createSupplierApi()), []);

  const initialTab = normalizeTab(searchParams.get("tab"));
  const initialEntity = (searchParams.get("entity") || "").trim();

  const [roleResolved, setRoleResolved] = useState(false);
  const [isReadOnlyVisitor, setIsReadOnlyVisitor] = useState(false);
  const [activeTab, setActiveTab] = useState<P2PTab>(initialTab);

  // Loaded data
  const [entities, setEntities] = useState<LegalEntity[]>([]);
  const [purchaseOrders, setPurchaseOrders] = useState<PurchaseOrder[]>([]);
  const [receipts, setReceipts] = useState<GoodsReceipt[]>([]);
  const [vendorInvoices, setVendorInvoices] = useState<VendorInvoice[]>([]);
  const [templates, setTemplates] = useState<ProcessTemplate[]>([]);
  const [instances, setInstances] = useState<ProcessInstance[]>([]);
  const [mining, setMining] = useState<ProcessMiningReport | null>(null);

  // Mining filters
  const [miningDomain, setMiningDomain] = useState<string>("ALL");
  const [miningEntity, setMiningEntity] = useState<string>(initialEntity);
  const [miningFrom, setMiningFrom] = useState<string>("");
  const [miningTo, setMiningTo] = useState<string>("");

  // Create PO form
  const [poVendorId, setPoVendorId] = useState("");
  const [poVendorName, setPoVendorName] = useState("");
  const [poVendorTin, setPoVendorTin] = useState("");
  const [poEntityId, setPoEntityId] = useState(initialEntity);
  const [poCurrency, setPoCurrency] = useState("UZS");
  const [poTemplateCode, setPoTemplateCode] = useState("P2P-STD-UZ");
  const [poApprovalId, setPoApprovalId] = useState("");
  const [poLines, setPoLines] = useState<DraftPOLine[]>([
    { sku: "", product_name: "", qty_ordered: "1", unit_cost_major: "", vat_bps: "1200" },
  ]);

  // Post Goods Receipt form
  const [recPoId, setRecPoId] = useState("");
  const [recWarehouseId, setRecWarehouseId] = useState("");
  const [recPeriodId, setRecPeriodId] = useState(() => currentAccountingPeriod());
  const [recQtyBySku, setRecQtyBySku] = useState<Record<string, string>>({});

  // Create Vendor Invoice form
  const [invPoId, setInvPoId] = useState("");
  const [invReceiptId, setInvReceiptId] = useState("");
  const [invEsfDocId, setInvEsfDocId] = useState("");
  const [invVendorId, setInvVendorId] = useState("");
  const [invCurrency, setInvCurrency] = useState("UZS");
  const [invSubtotalMajor, setInvSubtotalMajor] = useState("");
  const [invVatMajor, setInvVatMajor] = useState("");
  const [invTotalMajor, setInvTotalMajor] = useState("");

  // 3-Way Match form & outcome
  const [matchInvoiceId, setMatchInvoiceId] = useState("");
  const [matchReceiptId, setMatchReceiptId] = useState("");
  const [matchEsfDocId, setMatchEsfDocId] = useState("");
  const [matchPeriodId, setMatchPeriodId] = useState(() => currentAccountingPeriod());
  const [matchToleranceBps, setMatchToleranceBps] = useState("100");
  const [matchToleranceMinor, setMatchToleranceMinor] = useState("0");
  const [matchOverride, setMatchOverride] = useState(false);
  const [matchApprovalId, setMatchApprovalId] = useState("");
  const [lastMatchResult, setLastMatchResult] = useState<ThreeWayMatchResult | null>(null);

  // Process Template & Instance forms
  const [tplCode, setTplCode] = useState("");
  const [tplName, setTplName] = useState("");
  const [tplDomain, setTplDomain] = useState("P2P");
  const [tplStepsJson, setTplStepsJson] = useState(DEFAULT_TEMPLATE_STEPS_JSON);
  const [instTemplateCode, setInstTemplateCode] = useState("P2P-STD-UZ");
  const [instObjectType, setInstObjectType] = useState("PurchaseOrder");
  const [instObjectId, setInstObjectId] = useState("");
  const [instNextStep, setInstNextStep] = useState("");

  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [pendingPoApprovalId, setPendingPoApprovalId] = useState<string | null>(null);

  const syncRoleFromToken = useCallback(() => {
    const token = getSupplierToken() || readTokenFromCookie();
    if (!token) {
      setIsReadOnlyVisitor(false);
      setRoleResolved(true);
      return;
    }
    const roleInfo = resolvePortalSessionRole(decodeJwtPayload(token));
    setIsReadOnlyVisitor(roleInfo.isAuditor || roleInfo.isAgent);
    setRoleResolved(true);
  }, []);

  useEffect(() => {
    syncRoleFromToken();
  }, [syncRoleFromToken]);

  const loadAll = useCallback(
    async (silent = false) => {
      syncRoleFromToken();
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);

      const results = await Promise.allSettled([
        p2aApi.listPurchaseOrders({ limit: 50 }),
        p2aApi.listGoodsReceipts({ limit: 50 }),
        p2aApi.listVendorInvoices({ limit: 50 }),
        p2aApi.listProcessTemplates(),
        p2aApi.listProcessInstances({ limit: 50 }),
        p2aApi.getProcessMining({
          domain: miningDomain !== "ALL" ? miningDomain : undefined,
          entity: miningEntity.trim() || undefined,
          from: miningFrom ? `${miningFrom}T00:00:00Z` : undefined,
          to: miningTo ? `${miningTo}T23:59:59Z` : undefined,
        }),
        p2aApi.listEntities().catch(() => ({ tenant_id: "", count: 0, entities: [] as LegalEntity[] })),
      ]);

      const [poRes, recRes, invRes, tplRes, instRes, minRes, entRes] = results;
      let firstErr: string | null = null;
      const captureErr = (r: PromiseSettledResult<unknown>) => {
        if (r.status === "rejected" && !firstErr) {
          firstErr = extractControlError(
            r.reason,
            tRef.current("supplier_portal.p2p.load_failed"),
          ).message;
        }
      };

      if (poRes.status === "fulfilled") {
        const pos = poRes.value.purchase_orders ?? [];
        setPurchaseOrders(pos);
        if (pos.length > 0) {
          setRecPoId((prev) => prev || pos[0].po_id);
          setInvPoId((prev) => prev || pos[0].po_id);
        }
      } else {
        captureErr(poRes);
      }

      if (recRes.status === "fulfilled") {
        setReceipts(recRes.value.receipts ?? []);
      } else {
        captureErr(recRes);
      }

      if (invRes.status === "fulfilled") {
        const invs = invRes.value.vendor_invoices ?? [];
        setVendorInvoices(invs);
        if (invs.length > 0) {
          setMatchInvoiceId((prev) => prev || invs[0].vendor_invoice_id);
        }
      } else {
        captureErr(invRes);
      }

      if (tplRes.status === "fulfilled") {
        setTemplates(tplRes.value.templates ?? []);
      }
      if (instRes.status === "fulfilled") {
        setInstances(instRes.value.instances ?? []);
      }
      if (minRes.status === "fulfilled") {
        setMining(minRes.value);
      }
      if (entRes.status === "fulfilled") {
        setEntities(entRes.value.entities ?? []);
      }

      if (firstErr) {
        setError(firstErr);
      }
      setLoading(false);
      setRefreshing(false);
    },
    [miningDomain, miningEntity, miningFrom, miningTo, p2aApi, syncRoleFromToken],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const canMutate = roleResolved && !isReadOnlyVisitor;

  const selectedReceiptPo = useMemo(
    () => purchaseOrders.find((p) => p.po_id === recPoId) ?? null,
    [purchaseOrders, recPoId],
  );

  useEffect(() => {
    if (!selectedReceiptPo) return;
    const defaults: Record<string, string> = {};
    for (const line of selectedReceiptPo.lines ?? []) {
      const remaining = Math.max(0, line.qty_ordered - line.qty_received);
      defaults[line.sku] = String(remaining > 0 ? remaining : line.qty_ordered);
    }
    setRecQtyBySku(defaults);
  }, [selectedReceiptPo]);

  const poPreview = useMemo(() => {
    const parsedLines = poLines.map((l) => {
      const qty = Number.parseInt(l.qty_ordered.trim() || "0", 10);
      const unitCostMinor = parseMajorToSafeMinor(l.unit_cost_major) ?? 0;
      const vatBps = Number.parseInt(l.vat_bps.trim() || "0", 10);
      return {
        qtyOrdered: Number.isFinite(qty) && qty > 0 ? qty : 0,
        unitCostMinor: unitCostMinor >= 0 ? unitCostMinor : 0,
        vatBps: Number.isFinite(vatBps) && vatBps >= 0 ? vatBps : 0,
      };
    });
    return computePurchaseOrderPreviewTotals(parsedLines);
  }, [poLines]);

  const handleCreatePO = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !poVendorId.trim()) return;
    const lines = poLines
      .map((l) => {
        const qty = Number.parseInt(l.qty_ordered.trim(), 10);
        const costMinor = parseMajorToSafeMinor(l.unit_cost_major);
        const vatBps = Number.parseInt(l.vat_bps.trim() || "1200", 10);
        return {
          sku: l.sku.trim(),
          product_name: l.product_name.trim() || undefined,
          qty_ordered: Number.isFinite(qty) ? qty : 0,
          unit_cost_minor: costMinor ?? -1,
          vat_bps: Number.isFinite(vatBps) ? vatBps : 1200,
        };
      })
      .filter((l) => l.sku !== "" && l.qty_ordered > 0 && l.unit_cost_minor >= 0);

    if (lines.length === 0) {
      setError(t("supplier_portal.p2p.action_failed"));
      return;
    }

    setBusyAction("create-po");
    setError(null);
    setNotice(null);
    setPendingPoApprovalId(null);
    try {
      const res = await p2aApi.createPurchaseOrder({
        vendor_id: poVendorId.trim(),
        vendor_name: poVendorName.trim() || undefined,
        vendor_tin: poVendorTin.trim() || undefined,
        entity_id: poEntityId.trim() || undefined,
        currency: poCurrency,
        template_code: poTemplateCode.trim() || undefined,
        approval_id: poApprovalId.trim() || undefined,
        lines,
      });
      setPoVendorId("");
      setPoVendorName("");
      setPoVendorTin("");
      setPoLines([
        { sku: "", product_name: "", qty_ordered: "1", unit_cost_major: "", vat_bps: "1200" },
      ]);
      if (res.approval_required || res.purchase_order.status === "PENDING_APPROVAL") {
        const reqId = res.approval_request_id || res.purchase_order.approval_request_id || "";
        setPendingPoApprovalId(reqId || null);
        setNotice(
          t("supplier_portal.p2p.po_approval_pending", {
            poId: res.purchase_order.po_id,
            requestId: reqId || "—",
          }),
        );
      } else {
        setNotice(
          t("supplier_portal.p2p.po_created", {
            poId: res.purchase_order.po_id,
            status: res.purchase_order.status,
          }),
        );
      }
      await loadAll(true);
    } catch (err) {
      setError(extractControlError(err, t("supplier_portal.p2p.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleApprovePO = async (po: PurchaseOrder) => {
    if (!canMutate) return;
    setBusyAction(`approve-po:${po.po_id}`);
    setError(null);
    setNotice(null);
    try {
      const approved = await p2aApi.approvePurchaseOrder(po.po_id, {
        approval_id: po.approval_request_id || undefined,
      });
      setNotice(
        t("supplier_portal.p2p.po_approved", {
          poId: approved.po_id,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractControlError(err, t("supplier_portal.p2p.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handlePostReceipt = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !recPoId.trim() || !recWarehouseId.trim() || !selectedReceiptPo) return;
    const lines = (selectedReceiptPo.lines ?? [])
      .map((l) => {
        const qty = Number.parseInt((recQtyBySku[l.sku] ?? "0").trim(), 10);
        return {
          line_num: l.line_num,
          sku: l.sku,
          qty_received: Number.isFinite(qty) ? qty : 0,
        };
      })
      .filter((l) => l.qty_received > 0);
    if (lines.length === 0) {
      setError(t("supplier_portal.p2p.action_failed"));
      return;
    }

    setBusyAction("post-receipt");
    setError(null);
    setNotice(null);
    try {
      const res = await p2aApi.postGoodsReceipt({
        po_id: recPoId.trim(),
        entity_id: selectedReceiptPo.entity_id || undefined,
        warehouse_id: recWarehouseId.trim(),
        period_id: recPeriodId.trim() || undefined,
        lines,
      });
      setNotice(
        t("supplier_portal.p2p.receipt_posted", {
          receiptId: res.receipt.receipt_id,
          journalId: res.journal_entry_id || res.receipt.journal_entry_id || "—",
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractControlError(err, t("supplier_portal.p2p.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateInvoice = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !invPoId.trim()) return;
    const subMinor = parseMajorToSafeMinor(invSubtotalMajor);
    const vatMinor = parseMajorToSafeMinor(invVatMajor || "0");
    const totMinor = parseMajorToSafeMinor(invTotalMajor);
    if (subMinor === null || vatMinor === null || totMinor === null) {
      setError(t("supplier_portal.p2p.action_failed"));
      return;
    }

    const po = purchaseOrders.find((p) => p.po_id === invPoId);
    setBusyAction("create-invoice");
    setError(null);
    setNotice(null);
    try {
      const created = await p2aApi.createVendorInvoice({
        po_id: invPoId.trim(),
        receipt_id: invReceiptId.trim() || undefined,
        esf_doc_id: invEsfDocId.trim() || undefined,
        entity_id: po?.entity_id || undefined,
        vendor_id: invVendorId.trim() || po?.vendor_id || undefined,
        currency: invCurrency || po?.currency || "UZS",
        subtotal_minor: subMinor,
        vat_minor: vatMinor,
        total_minor: totMinor,
      });
      setInvSubtotalMajor("");
      setInvVatMajor("");
      setInvTotalMajor("");
      setMatchInvoiceId(created.vendor_invoice_id);
      setNotice(
        t("supplier_portal.p2p.invoice_created", {
          invoiceId: created.vendor_invoice_id,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractControlError(err, t("supplier_portal.p2p.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleRunThreeWayMatch = async (e?: React.FormEvent, overrideInvoiceId?: string) => {
    if (e) e.preventDefault();
    if (!canMutate) return;
    const targetInvId = (overrideInvoiceId || matchInvoiceId).trim();
    if (!targetInvId) return;

    const inv = vendorInvoices.find((v) => v.vendor_invoice_id === targetInvId);
    const tolBps = Number.parseInt(matchToleranceBps.trim() || "0", 10);
    const tolMinor = Number.parseInt(matchToleranceMinor.trim() || "0", 10);

    setBusyAction(`match:${targetInvId}`);
    setError(null);
    setNotice(null);
    try {
      const result = await p2aApi.runThreeWayMatch({
        vendor_invoice_id: targetInvId,
        po_id: inv?.po_id || undefined,
        receipt_id: matchReceiptId.trim() || inv?.receipt_id || undefined,
        esf_doc_id: matchEsfDocId.trim() || inv?.esf_doc_id || undefined,
        period_id: matchPeriodId.trim() || undefined,
        tolerance_bps: Number.isFinite(tolBps) && tolBps >= 0 ? tolBps : 0,
        tolerance_minor: Number.isFinite(tolMinor) && tolMinor >= 0 ? tolMinor : 0,
        override_discrepancy: matchOverride,
        approval_id: matchApprovalId.trim() || undefined,
      });
      setLastMatchResult(result);
      setActiveTab("three_way_match");
      await loadAll(true);
    } catch (err) {
      setError(extractControlError(err, t("supplier_portal.p2p.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateTemplate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !tplCode.trim() || !tplName.trim()) return;
    let parsedSteps: ProcessStep[];
    try {
      const raw = JSON.parse(tplStepsJson);
      if (!Array.isArray(raw) || raw.length === 0) {
        setError(t("supplier_portal.p2p.action_failed"));
        return;
      }
      parsedSteps = raw as ProcessStep[];
    } catch {
      setError(t("supplier_portal.p2p.action_failed"));
      return;
    }

    setBusyAction("create-template");
    setError(null);
    setNotice(null);
    try {
      const saved = await p2aApi.createProcessTemplate({
        code: tplCode.trim(),
        name: tplName.trim(),
        domain: tplDomain,
        steps: parsedSteps,
      });
      setTplCode("");
      setTplName("");
      setNotice(t("supplier_portal.p2p.template_created", { code: saved.code }));
      await loadAll(true);
    } catch (err) {
      setError(extractControlError(err, t("supplier_portal.p2p.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleStartOrAdvanceInstance = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !instObjectId.trim()) return;
    setBusyAction("advance-instance");
    setError(null);
    setNotice(null);
    try {
      const inst = await p2aApi.startOrAdvanceProcessInstance({
        template_code: instTemplateCode.trim() || undefined,
        object_type: instObjectType.trim() || "PurchaseOrder",
        object_id: instObjectId.trim(),
        next_step: instNextStep.trim() || undefined,
      });
      setInstNextStep("");
      setNotice(
        t("supplier_portal.p2p.instance_updated", {
          instanceId: inst.instance_id,
          step: inst.current_step,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractControlError(err, t("supplier_portal.p2p.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const matchedInvoicesCount = vendorInvoices.filter(
    (inv) => inv.match_status === "MATCHED_3WAY",
  ).length;
  const bottleneckLabel =
    mining && !mining.empty && mining.bottleneck_stage
      ? `${mining.bottleneck_stage.from_step} -> ${mining.bottleneck_stage.to_step}`
      : "—";

  return (
    <PageChrome
      icon="orders"
      title={t("supplier_portal.p2p.title")}
      description={t("supplier_portal.p2p.subtitle")}
      loading={loading && purchaseOrders.length === 0 && templates.length === 0 && !error}
      skeletonVariant="dashboard"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/approvals" as Route} className="portal-btn portal-btn--outline">
            <Icon name="verified" size={16} />
            <span>{t("portal.nav.approvals")}</span>
          </Link>
          <Link href={"/finance/consolidation" as Route} className="portal-btn portal-btn--outline">
            <Icon name="ledger" size={16} />
            <span>{t("portal.nav.consolidation")}</span>
          </Link>
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
      {roleResolved && isReadOnlyVisitor ? (
        <div
          data-testid="p2p-read-only-banner"
          className="portal-alert portal-alert--info mb-6"
          role="status"
        >
          <p className="portal-alert-text text-xs">
            {t("supplier_portal.auditor.safe_mode_badge")} —{" "}
            {t("supplier_portal.p2p.read_only_notice")}
          </p>
        </div>
      ) : null}

      {error ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <p className="portal-alert-text font-semibold">{error}</p>
        </div>
      ) : null}

      {notice ? (
        <div className="portal-alert portal-alert--info mb-6" role="status">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="portal-alert-text">{notice}</p>
            {pendingPoApprovalId ? (
              <Link
                href={`/approvals?requestId=${encodeURIComponent(pendingPoApprovalId)}` as Route}
                className="text-xs font-semibold underline"
              >
                {t("supplier_portal.p2p.view_approvals_link")}
              </Link>
            ) : null}
          </div>
        </div>
      ) : null}

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.p2p.kpi_pos")}
          value={purchaseOrders.length}
          sub={`${purchaseOrders.filter((p) => p.status === "PENDING_APPROVAL").length} pending`}
        />
        <KpiStatCard
          label={t("supplier_portal.p2p.kpi_receipts")}
          value={receipts.length}
        />
        <KpiStatCard
          label={t("supplier_portal.p2p.kpi_matched_invoices")}
          value={`${matchedInvoicesCount} / ${vendorInvoices.length}`}
        />
        <KpiStatCard
          label={t("supplier_portal.p2p.kpi_bottleneck")}
          value={bottleneckLabel}
          sub={
            mining && !mining.empty && mining.bottleneck_stage
              ? `P95 ${formatDurationSec(mining.bottleneck_stage.p95_duration_sec)} · SLA ${mining.sla_breach_count}`
              : undefined
          }
        />
      </KpiStatGrid>

      <div className="mt-6 mb-4 flex flex-wrap gap-2" role="tablist">
        {(
          [
            ["orders", "supplier_portal.p2p.tab_orders"],
            ["receipts_invoices", "supplier_portal.p2p.tab_receipts_invoices"],
            ["three_way_match", "supplier_portal.p2p.tab_three_way_match"],
            ["process_mining", "supplier_portal.p2p.tab_process_mining"],
          ] as const
        ).map(([tabKey, labelKey]) => (
          <button
            key={tabKey}
            type="button"
            role="tab"
            aria-selected={activeTab === tabKey}
            onClick={() => setActiveTab(tabKey)}
            className={`portal-btn ${
              activeTab === tabKey ? "portal-btn--primary" : "portal-btn--outline"
            }`}
          >
            {t(labelKey)}
          </button>
        ))}
      </div>

      {activeTab === "orders" ? (
        <div className="space-y-6">
          {canMutate ? (
            <PortalSection icon="plus" title={t("supplier_portal.p2p.po_form_title")}>
              <form
                data-testid="create-po-form"
                onSubmit={(e) => void handleCreatePO(e)}
                className="space-y-4"
              >
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  <div>
                    <label htmlFor="po-vendor-id" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.vendor_id_label")}
                    </label>
                    <PortalInput
                      id="po-vendor-id"
                      required
                      value={poVendorId}
                      onChange={(e) => setPoVendorId(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="po-vendor-name" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.vendor_name_label")}
                    </label>
                    <PortalInput
                      id="po-vendor-name"
                      value={poVendorName}
                      onChange={(e) => setPoVendorName(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="po-vendor-tin" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.vendor_tin_label")}
                    </label>
                    <PortalInput
                      id="po-vendor-tin"
                      value={poVendorTin}
                      onChange={(e) => setPoVendorTin(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="po-entity-id" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.entity_label")}
                    </label>
                    {entities.length > 0 ? (
                      <PortalSelect
                        id="po-entity-id"
                        value={poEntityId}
                        onChange={(e) => setPoEntityId(e.target.value)}
                      >
                        <option value="">—</option>
                        {entities.map((ent) => (
                          <option key={ent.entity_id} value={ent.entity_id}>
                            {ent.name} ({ent.entity_id})
                          </option>
                        ))}
                      </PortalSelect>
                    ) : (
                      <PortalInput
                        id="po-entity-id"
                        value={poEntityId}
                        onChange={(e) => setPoEntityId(e.target.value)}
                      />
                    )}
                  </div>
                  <div>
                    <label htmlFor="po-currency" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.currency_label")}
                    </label>
                    <PortalSelect
                      id="po-currency"
                      value={poCurrency}
                      onChange={(e) => setPoCurrency(e.target.value)}
                    >
                      {CURRENCIES.map((c) => (
                        <option key={c} value={c}>
                          {c}
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="po-template-code" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.template_code_label")}
                    </label>
                    <PortalInput
                      id="po-template-code"
                      value={poTemplateCode}
                      onChange={(e) => setPoTemplateCode(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="po-approval-id" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.approval_id_label")}
                    </label>
                    <PortalInput
                      id="po-approval-id"
                      value={poApprovalId}
                      onChange={(e) => setPoApprovalId(e.target.value)}
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  {poLines.map((line, idx) => (
                    <div
                      key={idx}
                      className="grid grid-cols-1 sm:grid-cols-6 gap-2 items-end rounded-lg border p-2.5"
                      style={{ borderColor: "var(--desk-border)" }}
                    >
                      <div>
                        <label className="portal-label block mb-1">
                          {t("supplier_portal.p2p.sku_label")}
                        </label>
                        <PortalInput
                          required
                          aria-label={`${t("supplier_portal.p2p.sku_label")} ${idx + 1}`}
                          value={line.sku}
                          onChange={(e) =>
                            setPoLines((prev) =>
                              prev.map((item, i) =>
                                i === idx ? { ...item, sku: e.target.value } : item,
                              ),
                            )
                          }
                        />
                      </div>
                      <div>
                        <label className="portal-label block mb-1">
                          {t("supplier_portal.p2p.product_name_label")}
                        </label>
                        <PortalInput
                          aria-label={`${t("supplier_portal.p2p.product_name_label")} ${idx + 1}`}
                          value={line.product_name}
                          onChange={(e) =>
                            setPoLines((prev) =>
                              prev.map((item, i) =>
                                i === idx ? { ...item, product_name: e.target.value } : item,
                              ),
                            )
                          }
                        />
                      </div>
                      <div>
                        <label className="portal-label block mb-1">
                          {t("supplier_portal.p2p.qty_label")}
                        </label>
                        <PortalInput
                          type="number"
                          min={1}
                          required
                          aria-label={`${t("supplier_portal.p2p.qty_label")} ${idx + 1}`}
                          value={line.qty_ordered}
                          onChange={(e) =>
                            setPoLines((prev) =>
                              prev.map((item, i) =>
                                i === idx ? { ...item, qty_ordered: e.target.value } : item,
                              ),
                            )
                          }
                        />
                      </div>
                      <div>
                        <label className="portal-label block mb-1">
                          {t("supplier_portal.p2p.unit_cost_label")}
                        </label>
                        <PortalInput
                          required
                          aria-label={`${t("supplier_portal.p2p.unit_cost_label")} ${idx + 1}`}
                          placeholder="500.00"
                          value={line.unit_cost_major}
                          onChange={(e) =>
                            setPoLines((prev) =>
                              prev.map((item, i) =>
                                i === idx ? { ...item, unit_cost_major: e.target.value } : item,
                              ),
                            )
                          }
                        />
                      </div>
                      <div>
                        <label className="portal-label block mb-1">
                          {t("supplier_portal.p2p.vat_bps_label")}
                        </label>
                        <PortalInput
                          type="number"
                          min={0}
                          aria-label={`${t("supplier_portal.p2p.vat_bps_label")} ${idx + 1}`}
                          value={line.vat_bps}
                          onChange={(e) =>
                            setPoLines((prev) =>
                              prev.map((item, i) =>
                                i === idx ? { ...item, vat_bps: e.target.value } : item,
                              ),
                            )
                          }
                        />
                      </div>
                      <div>
                        {poLines.length > 1 ? (
                          <button
                            type="button"
                            onClick={() =>
                              setPoLines((prev) => prev.filter((_, i) => i !== idx))
                            }
                            className="portal-btn portal-btn--outline w-full"
                          >
                            {t("supplier_portal.p2p.remove_line_btn")}
                          </button>
                        ) : null}
                      </div>
                    </div>
                  ))}
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <button
                      type="button"
                      onClick={() =>
                        setPoLines((prev) => [
                          ...prev,
                          {
                            sku: "",
                            product_name: "",
                            qty_ordered: "1",
                            unit_cost_major: "",
                            vat_bps: "1200",
                          },
                        ])
                      }
                      className="portal-btn portal-btn--outline"
                    >
                      {t("supplier_portal.p2p.add_line_btn")}
                    </button>
                    <div className="flex flex-wrap gap-4 font-mono text-xs">
                      <span>
                        {t("supplier_portal.p2p.preview_subtotal")}:{" "}
                        <strong>{formatFinanceMinor(poPreview.subtotalMinor, poCurrency)}</strong>
                      </span>
                      <span>
                        {t("supplier_portal.p2p.preview_vat")}:{" "}
                        <strong>{formatFinanceMinor(poPreview.vatMinor, poCurrency)}</strong>
                      </span>
                      <span>
                        {t("supplier_portal.p2p.preview_total")}:{" "}
                        <strong>{formatFinanceMinor(poPreview.totalMinor, poCurrency)}</strong>
                      </span>
                    </div>
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={busyAction !== null || !poVendorId.trim()}
                  className="portal-btn portal-btn--primary"
                >
                  {t("supplier_portal.p2p.create_po_btn")}
                </button>
              </form>
            </PortalSection>
          ) : null}

          <PortalSection icon="orders" title={t("supplier_portal.p2p.po_table_title")}>
            {purchaseOrders.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.p2p.no_pos")}
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
                      }}
                    >
                      <th className="py-2 px-2">{t("supplier_portal.p2p.col_po_id")}</th>
                      <th className="py-2 px-2">{t("supplier_portal.p2p.col_vendor")}</th>
                      <th className="py-2 px-2">{t("supplier_portal.p2p.col_lines")}</th>
                      <th className="py-2 px-2 text-right">{t("supplier_portal.p2p.col_amounts")}</th>
                      <th className="py-2 px-2">{t("supplier_portal.p2p.col_status")}</th>
                      <th className="py-2 px-2">{t("supplier_portal.p2p.col_created_by")}</th>
                      {canMutate ? (
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.p2p.col_actions")}
                        </th>
                      ) : null}
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {purchaseOrders.map((po) => (
                      <tr key={po.po_id}>
                        <td className="py-2 px-2 font-mono">
                          <div className="font-semibold">{po.po_id}</div>
                          {po.entity_id ? (
                            <div
                              className="text-[11px]"
                              style={{ color: "var(--desk-text-secondary)" }}
                            >
                              {po.entity_id}
                            </div>
                          ) : null}
                        </td>
                        <td className="py-2 px-2">
                          <div className="font-semibold">{po.vendor_name || po.vendor_id}</div>
                          <div
                            className="font-mono text-[11px]"
                            style={{ color: "var(--desk-text-secondary)" }}
                          >
                            {po.vendor_id} {po.vendor_tin ? `· ${po.vendor_tin}` : ""}
                          </div>
                        </td>
                        <td className="py-2 px-2 font-mono">
                          {(po.lines ?? []).map((l) => (
                            <div key={`${po.po_id}:${l.line_num}`}>
                              {l.sku} ({l.qty_received}/{l.qty_ordered})
                            </div>
                          ))}
                        </td>
                        <td className="py-2 px-2 font-mono text-right">
                          <div>
                            {formatFinanceMinor(po.subtotal_minor, po.currency)} +{" "}
                            {formatFinanceMinor(po.vat_minor, po.currency)}
                          </div>
                          <div className="font-semibold">
                            {formatFinanceMinor(po.total_minor, po.currency)}
                          </div>
                        </td>
                        <td className="py-2 px-2">
                          <StatusBadge
                            state={mapStatusBadgeState(po.status)}
                            label={po.status}
                          />
                          {po.approval_request_id ? (
                            <div className="font-mono text-[11px] mt-1">
                              {po.approval_request_id}
                            </div>
                          ) : null}
                        </td>
                        <td className="py-2 px-2">{po.created_by}</td>
                        {canMutate ? (
                          <td className="py-2 px-2 text-right">
                            {po.status === "PENDING_APPROVAL" ? (
                              <button
                                type="button"
                                disabled={busyAction !== null}
                                onClick={() => void handleApprovePO(po)}
                                className="rounded border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                              >
                                {t("supplier_portal.p2p.approve_po_btn")}
                              </button>
                            ) : (
                              <span style={{ color: "var(--desk-text-tertiary)" }}>—</span>
                            )}
                          </td>
                        ) : null}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "receipts_invoices" ? (
        <div className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {canMutate ? (
              <PortalSection
                icon="inventory"
                title={t("supplier_portal.p2p.receipt_form_title")}
              >
                <form
                  data-testid="post-goods-receipt-form"
                  onSubmit={(e) => void handlePostReceipt(e)}
                  className="space-y-3"
                >
                  <div>
                    <label htmlFor="rec-po-select" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.select_po_label")}
                    </label>
                    <PortalSelect
                      id="rec-po-select"
                      value={recPoId}
                      onChange={(e) => setRecPoId(e.target.value)}
                    >
                      <option value="">—</option>
                      {purchaseOrders.map((po) => (
                        <option key={po.po_id} value={po.po_id}>
                          {po.po_id} · {po.vendor_name || po.vendor_id} ({po.status})
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label htmlFor="rec-wh-input" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.warehouse_id_label")}
                      </label>
                      <PortalInput
                        id="rec-wh-input"
                        required
                        value={recWarehouseId}
                        onChange={(e) => setRecWarehouseId(e.target.value)}
                      />
                    </div>
                    <div>
                      <label htmlFor="rec-period-input" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.period_id_label")}
                      </label>
                      <PortalInput
                        id="rec-period-input"
                        type="month"
                        value={recPeriodId}
                        onChange={(e) => setRecPeriodId(e.target.value)}
                      />
                    </div>
                  </div>
                  {selectedReceiptPo ? (
                    <div className="space-y-2">
                      {(selectedReceiptPo.lines ?? []).map((line) => (
                        <div
                          key={`${selectedReceiptPo.po_id}:${line.line_num}`}
                          className="flex items-center justify-between gap-2 text-xs rounded border p-2"
                          style={{ borderColor: "var(--desk-border)" }}
                        >
                          <span className="font-mono">
                            {line.sku} ({line.qty_received}/{line.qty_ordered})
                          </span>
                          <label className="flex items-center gap-2">
                            <span>{t("supplier_portal.p2p.qty_received_label")}</span>
                            <PortalInput
                              type="number"
                              min={0}
                              className="w-24"
                              value={recQtyBySku[line.sku] ?? "0"}
                              onChange={(e) =>
                                setRecQtyBySku((prev) => ({
                                  ...prev,
                                  [line.sku]: e.target.value,
                                }))
                              }
                            />
                          </label>
                        </div>
                      ))}
                    </div>
                  ) : null}
                  <button
                    type="submit"
                    disabled={busyAction !== null || !recPoId || !recWarehouseId.trim()}
                    className="portal-btn portal-btn--primary w-full"
                  >
                    {t("supplier_portal.p2p.post_receipt_btn")}
                  </button>
                </form>
              </PortalSection>
            ) : null}

            {canMutate ? (
              <PortalSection
                icon="manifests"
                title={t("supplier_portal.p2p.invoice_form_title")}
              >
                <form
                  data-testid="create-vendor-invoice-form"
                  onSubmit={(e) => void handleCreateInvoice(e)}
                  className="space-y-3"
                >
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label htmlFor="inv-po-select" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.select_po_label")}
                      </label>
                      <PortalSelect
                        id="inv-po-select"
                        value={invPoId}
                        onChange={(e) => setInvPoId(e.target.value)}
                      >
                        <option value="">—</option>
                        {purchaseOrders.map((po) => (
                          <option key={po.po_id} value={po.po_id}>
                            {po.po_id} · {po.vendor_name || po.vendor_id}
                          </option>
                        ))}
                      </PortalSelect>
                    </div>
                    <div>
                      <label htmlFor="inv-receipt-select" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.receipt_id_optional_label")}
                      </label>
                      <PortalSelect
                        id="inv-receipt-select"
                        value={invReceiptId}
                        onChange={(e) => setInvReceiptId(e.target.value)}
                      >
                        <option value="">—</option>
                        {receipts.map((rec) => (
                          <option key={rec.receipt_id} value={rec.receipt_id}>
                            {rec.receipt_id} ({rec.po_id})
                          </option>
                        ))}
                      </PortalSelect>
                    </div>
                    <div>
                      <label htmlFor="inv-esf-input" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.esf_doc_id_label")}
                      </label>
                      <PortalInput
                        id="inv-esf-input"
                        value={invEsfDocId}
                        onChange={(e) => setInvEsfDocId(e.target.value)}
                      />
                    </div>
                    <div>
                      <label htmlFor="inv-vendor-input" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.vendor_id_label")}
                      </label>
                      <PortalInput
                        id="inv-vendor-input"
                        value={invVendorId}
                        onChange={(e) => setInvVendorId(e.target.value)}
                      />
                    </div>
                    <div>
                      <label htmlFor="inv-currency-select" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.currency_label")}
                      </label>
                      <PortalSelect
                        id="inv-currency-select"
                        value={invCurrency}
                        onChange={(e) => setInvCurrency(e.target.value)}
                      >
                        {CURRENCIES.map((c) => (
                          <option key={c} value={c}>
                            {c}
                          </option>
                        ))}
                      </PortalSelect>
                    </div>
                    <div>
                      <label htmlFor="inv-subtotal-input" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.subtotal_label")}
                      </label>
                      <PortalInput
                        id="inv-subtotal-input"
                        required
                        placeholder="1000.00"
                        value={invSubtotalMajor}
                        onChange={(e) => setInvSubtotalMajor(e.target.value)}
                      />
                    </div>
                    <div>
                      <label htmlFor="inv-vat-input" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.vat_label")}
                      </label>
                      <PortalInput
                        id="inv-vat-input"
                        placeholder="120.00"
                        value={invVatMajor}
                        onChange={(e) => setInvVatMajor(e.target.value)}
                      />
                    </div>
                    <div>
                      <label htmlFor="inv-total-input" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.total_label")}
                      </label>
                      <PortalInput
                        id="inv-total-input"
                        required
                        placeholder="1120.00"
                        value={invTotalMajor}
                        onChange={(e) => setInvTotalMajor(e.target.value)}
                      />
                    </div>
                  </div>
                  <button
                    type="submit"
                    disabled={busyAction !== null || !invPoId || !invSubtotalMajor || !invTotalMajor}
                    className="portal-btn portal-btn--primary w-full"
                  >
                    {t("supplier_portal.p2p.create_invoice_btn")}
                  </button>
                </form>
              </PortalSection>
            ) : null}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <PortalSection icon="inventory" title={t("supplier_portal.p2p.receipts_table_title")}>
              {receipts.length === 0 ? (
                <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.p2p.no_receipts")}
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
                        }}
                      >
                        <th className="py-2 px-2">{t("supplier_portal.p2p.col_receipt_id")}</th>
                        <th className="py-2 px-2">{t("supplier_portal.p2p.col_warehouse")}</th>
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.p2p.col_received_total")}
                        </th>
                        <th className="py-2 px-2">{t("supplier_portal.p2p.col_status")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {receipts.map((rec) => (
                        <tr key={rec.receipt_id}>
                          <td className="py-2 px-2 font-mono">
                            <div className="font-semibold">{rec.receipt_id}</div>
                            <div
                              className="text-[11px]"
                              style={{ color: "var(--desk-text-secondary)" }}
                            >
                              {rec.po_id}
                            </div>
                          </td>
                          <td className="py-2 px-2 font-mono">{rec.warehouse_id}</td>
                          <td className="py-2 px-2 font-mono text-right">
                            {formatFinanceMinor(rec.total_received_minor, "UZS")}
                          </td>
                          <td className="py-2 px-2">
                            <StatusBadge state={mapStatusBadgeState(rec.status)} label={rec.status} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </PortalSection>

            <PortalSection icon="manifests" title={t("supplier_portal.p2p.invoices_table_title")}>
              {vendorInvoices.length === 0 ? (
                <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.p2p.no_invoices")}
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
                        }}
                      >
                        <th className="py-2 px-2">{t("supplier_portal.p2p.col_invoice_id")}</th>
                        <th className="py-2 px-2 text-right">{t("supplier_portal.p2p.total_label")}</th>
                        <th className="py-2 px-2">{t("supplier_portal.p2p.col_match_status")}</th>
                        <th className="py-2 px-2 text-right">{t("supplier_portal.p2p.col_variance")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {vendorInvoices.map((inv) => (
                        <tr key={inv.vendor_invoice_id}>
                          <td className="py-2 px-2 font-mono">
                            <div className="font-semibold">{inv.vendor_invoice_id}</div>
                            <div
                              className="text-[11px]"
                              style={{ color: "var(--desk-text-secondary)" }}
                            >
                              {inv.po_id} {inv.esf_doc_id ? `· ESF ${inv.esf_doc_id}` : ""}
                            </div>
                          </td>
                          <td className="py-2 px-2 font-mono text-right">
                            {formatFinanceMinor(inv.total_minor, inv.currency)}
                          </td>
                          <td className="py-2 px-2">
                            <StatusBadge
                              state={mapStatusBadgeState(inv.match_status)}
                              label={inv.match_status}
                            />
                          </td>
                          <td className="py-2 px-2 font-mono text-right">
                            {formatFinanceMinor(inv.variance_minor, inv.currency)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </PortalSection>
          </div>
        </div>
      ) : null}

      {activeTab === "three_way_match" ? (
        <div className="space-y-6">
          {canMutate ? (
            <PortalSection icon="reconcile" title={t("supplier_portal.p2p.match_form_title")}>
              <form
                data-testid="three-way-match-form"
                onSubmit={(e) => void handleRunThreeWayMatch(e)}
                className="space-y-3"
              >
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  <div>
                    <label htmlFor="match-invoice-select" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.select_invoice_label")}
                    </label>
                    <PortalSelect
                      id="match-invoice-select"
                      value={matchInvoiceId}
                      onChange={(e) => setMatchInvoiceId(e.target.value)}
                    >
                      <option value="">—</option>
                      {vendorInvoices.map((inv) => (
                        <option key={inv.vendor_invoice_id} value={inv.vendor_invoice_id}>
                          {inv.vendor_invoice_id} · {inv.po_id} ({inv.match_status})
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="match-receipt-select" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.receipt_id_optional_label")}
                    </label>
                    <PortalSelect
                      id="match-receipt-select"
                      value={matchReceiptId}
                      onChange={(e) => setMatchReceiptId(e.target.value)}
                    >
                      <option value="">—</option>
                      {receipts.map((r) => (
                        <option key={r.receipt_id} value={r.receipt_id}>
                          {r.receipt_id} ({r.po_id})
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="match-esf-input" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.esf_doc_id_label")}
                    </label>
                    <PortalInput
                      id="match-esf-input"
                      value={matchEsfDocId}
                      onChange={(e) => setMatchEsfDocId(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="match-period-input" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.period_id_label")}
                    </label>
                    <PortalInput
                      id="match-period-input"
                      type="month"
                      value={matchPeriodId}
                      onChange={(e) => setMatchPeriodId(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="match-tol-bps" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.tolerance_bps_label")}
                    </label>
                    <PortalInput
                      id="match-tol-bps"
                      type="number"
                      min={0}
                      value={matchToleranceBps}
                      onChange={(e) => setMatchToleranceBps(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="match-tol-minor" className="portal-label block mb-1">
                      {t("supplier_portal.p2p.tolerance_minor_label")}
                    </label>
                    <PortalInput
                      id="match-tol-minor"
                      type="number"
                      min={0}
                      value={matchToleranceMinor}
                      onChange={(e) => setMatchToleranceMinor(e.target.value)}
                    />
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-4">
                  <label className="flex items-center gap-2 text-xs">
                    <input
                      type="checkbox"
                      checked={matchOverride}
                      onChange={(e) => setMatchOverride(e.target.checked)}
                    />
                    <span>{t("supplier_portal.p2p.override_discrepancy_label")}</span>
                  </label>
                  {matchOverride ? (
                    <PortalInput
                      placeholder={t("supplier_portal.p2p.approval_id_label")}
                      value={matchApprovalId}
                      onChange={(e) => setMatchApprovalId(e.target.value)}
                    />
                  ) : null}
                  <button
                    type="submit"
                    disabled={busyAction !== null || !matchInvoiceId}
                    className="portal-btn portal-btn--primary"
                  >
                    {t("supplier_portal.p2p.run_match_btn")}
                  </button>
                </div>
              </form>
            </PortalSection>
          ) : null}

          {lastMatchResult ? (
            <PortalSection
              icon="verified"
              title={t("supplier_portal.p2p.match_result_title")}
            >
              <div
                data-testid="three-way-match-outcome"
                className="rounded-xl border p-4 space-y-2 text-xs"
                style={{
                  borderColor: lastMatchResult.matched
                    ? "var(--desk-success)"
                    : "var(--desk-danger)",
                  background: lastMatchResult.matched
                    ? "var(--desk-success-soft)"
                    : "var(--desk-danger-soft)",
                }}
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-semibold text-sm">
                    {lastMatchResult.matched
                      ? t("supplier_portal.p2p.match_matched")
                      : t("supplier_portal.p2p.match_discrepancy")}
                  </span>
                  <StatusBadge
                    state={mapStatusBadgeState(lastMatchResult.match_status)}
                    label={lastMatchResult.match_status}
                  />
                </div>
                <div className="font-mono">
                  {t("supplier_portal.p2p.col_variance")}:{" "}
                  {formatFinanceMinor(
                    lastMatchResult.variance_minor,
                    lastMatchResult.vendor_invoice.currency,
                  )}
                  {lastMatchResult.variance_reason
                    ? ` · ${t("supplier_portal.p2p.variance_reason_label")}: ${lastMatchResult.variance_reason}`
                    : ""}
                </div>
                <div className="font-mono">
                  {t("supplier_portal.p2p.gl_entry_label")}:{" "}
                  {lastMatchResult.journal_entry_id || "—"} ·{" "}
                  {t("supplier_portal.p2p.verified_by_label")}:{" "}
                  {lastMatchResult.verified_by || "—"}
                </div>
              </div>
            </PortalSection>
          ) : null}

          <PortalSection icon="manifests" title={t("supplier_portal.p2p.invoices_table_title")}>
            {vendorInvoices.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.p2p.no_invoices")}
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
                      }}
                    >
                      <th className="py-2 px-2">{t("supplier_portal.p2p.col_invoice_id")}</th>
                      <th className="py-2 px-2 text-right">{t("supplier_portal.p2p.total_label")}</th>
                      <th className="py-2 px-2">{t("supplier_portal.p2p.col_match_status")}</th>
                      <th className="py-2 px-2 text-right">{t("supplier_portal.p2p.col_variance")}</th>
                      <th className="py-2 px-2">{t("supplier_portal.p2p.variance_reason_label")}</th>
                      {canMutate ? (
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.p2p.col_actions")}
                        </th>
                      ) : null}
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {vendorInvoices.map((inv) => (
                      <tr key={inv.vendor_invoice_id}>
                        <td className="py-2 px-2 font-mono">{inv.vendor_invoice_id}</td>
                        <td className="py-2 px-2 font-mono text-right">
                          {formatFinanceMinor(inv.total_minor, inv.currency)}
                        </td>
                        <td className="py-2 px-2">
                          <StatusBadge
                            state={mapStatusBadgeState(inv.match_status)}
                            label={inv.match_status}
                          />
                        </td>
                        <td className="py-2 px-2 font-mono text-right">
                          {formatFinanceMinor(inv.variance_minor, inv.currency)}
                        </td>
                        <td className="py-2 px-2">{inv.variance_reason || "—"}</td>
                        {canMutate ? (
                          <td className="py-2 px-2 text-right">
                            <button
                              type="button"
                              disabled={busyAction !== null}
                              onClick={() =>
                                void handleRunThreeWayMatch(undefined, inv.vendor_invoice_id)
                              }
                              className="rounded border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                            >
                              {t("supplier_portal.p2p.run_match_btn")}
                            </button>
                          </td>
                        ) : null}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "process_mining" ? (
        <div className="space-y-6">
          <PortalSection
            icon="analytics"
            title={t("supplier_portal.p2p.mining_title")}
            description={t("supplier_portal.p2p.mining_desc")}
          >
            <div className="grid grid-cols-1 sm:grid-cols-4 gap-3 mb-4">
              <div>
                <label htmlFor="mining-domain-select" className="portal-label block mb-1">
                  {t("supplier_portal.p2p.domain_filter_label")}
                </label>
                <PortalSelect
                  id="mining-domain-select"
                  value={miningDomain}
                  onChange={(e) => setMiningDomain(e.target.value)}
                >
                  {DOMAINS.map((d) => (
                    <option key={d} value={d}>
                      {d === "ALL" ? t("supplier_portal.p2p.all_domains") : d}
                    </option>
                  ))}
                </PortalSelect>
              </div>
              <div>
                <label htmlFor="mining-entity-input" className="portal-label block mb-1">
                  {t("supplier_portal.p2p.entity_label")}
                </label>
                <PortalInput
                  id="mining-entity-input"
                  value={miningEntity}
                  onChange={(e) => setMiningEntity(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="mining-from-input" className="portal-label block mb-1">
                  {t("supplier_portal.p2p.from_date_label")}
                </label>
                <PortalInput
                  id="mining-from-input"
                  type="date"
                  value={miningFrom}
                  onChange={(e) => setMiningFrom(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="mining-to-input" className="portal-label block mb-1">
                  {t("supplier_portal.p2p.to_date_label")}
                </label>
                <PortalInput
                  id="mining-to-input"
                  type="date"
                  value={miningTo}
                  onChange={(e) => setMiningTo(e.target.value)}
                />
              </div>
            </div>

            {!mining || mining.empty || (mining.edges ?? []).length === 0 ? (
              <div
                data-testid="process-mining-empty"
                className="rounded-xl border p-4 text-xs"
                style={{
                  borderColor: "var(--desk-border)",
                  background: "var(--desk-surface-subtle)",
                  color: "var(--desk-text-secondary)",
                }}
              >
                {t("supplier_portal.p2p.mining_empty")}
              </div>
            ) : (
              <div data-testid="process-mining-results" className="space-y-4">
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
                  <div className="rounded-lg border p-3" style={{ borderColor: "var(--desk-border)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.p2p.kpi_total_cases")}
                    </div>
                    <div className="font-mono font-bold text-base mt-1">{mining.total_cases}</div>
                  </div>
                  <div className="rounded-lg border p-3" style={{ borderColor: "var(--desk-border)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.p2p.kpi_total_transitions")}
                    </div>
                    <div className="font-mono font-bold text-base mt-1">
                      {mining.total_transitions}
                    </div>
                  </div>
                  <div className="rounded-lg border p-3" style={{ borderColor: "var(--desk-border)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.p2p.kpi_sla_breaches")}
                    </div>
                    <div className="font-mono font-bold text-base mt-1">
                      {mining.sla_breach_count}
                    </div>
                  </div>
                  <div className="rounded-lg border p-3" style={{ borderColor: "var(--desk-border)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.p2p.kpi_top_bottleneck")}
                    </div>
                    <div className="font-mono font-bold text-sm mt-1">{bottleneckLabel}</div>
                  </div>
                </div>

                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse text-xs">
                    <thead>
                      <tr
                        className="border-b uppercase"
                        style={{
                          borderColor: "var(--desk-border)",
                          color: "var(--desk-text-tertiary)",
                        }}
                      >
                        <th className="py-2 px-2">{t("supplier_portal.p2p.col_step_edge")}</th>
                        <th className="py-2 px-2">{t("supplier_portal.p2p.col_domain")}</th>
                        <th className="py-2 px-2 text-right">{t("supplier_portal.p2p.col_cases")}</th>
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.p2p.col_transitions")}
                        </th>
                        <th className="py-2 px-2 text-right">{t("supplier_portal.p2p.col_avg_dur")}</th>
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.p2p.col_median_dur")}
                        </th>
                        <th className="py-2 px-2 text-right">{t("supplier_portal.p2p.col_p95_dur")}</th>
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.p2p.col_sla_target")}
                        </th>
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.p2p.col_sla_breaches")}
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {mining.edges.map((edge, idx) => (
                        <tr key={`${edge.domain}:${edge.from_step}:${edge.to_step}:${idx}`}>
                          <td className="py-2 px-2 font-mono font-semibold">
                            {edge.from_step} → {edge.to_step}
                          </td>
                          <td className="py-2 px-2 font-mono">{edge.domain}</td>
                          <td className="py-2 px-2 font-mono text-right">{edge.case_count}</td>
                          <td className="py-2 px-2 font-mono text-right">
                            {edge.transition_count}
                          </td>
                          <td className="py-2 px-2 font-mono text-right">
                            {formatDurationSec(edge.avg_duration_sec)}
                          </td>
                          <td className="py-2 px-2 font-mono text-right">
                            {formatDurationSec(edge.median_duration_sec)}
                          </td>
                          <td className="py-2 px-2 font-mono text-right font-semibold">
                            {formatDurationSec(edge.p95_duration_sec)}
                          </td>
                          <td className="py-2 px-2 font-mono text-right">
                            {formatDurationSec(edge.sla_target_sec)}
                          </td>
                          <td className="py-2 px-2 font-mono text-right">
                            {edge.sla_breach_count}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </PortalSection>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <PortalSection icon="catalog" title={t("supplier_portal.p2p.templates_title")}>
              <div className="space-y-4">
                {templates.length === 0 ? (
                  <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.p2p.no_templates")}
                  </p>
                ) : (
                  <div className="space-y-2">
                    {templates.map((tpl) => (
                      <div
                        key={`${tpl.template_id}:${tpl.code}`}
                        className="rounded-lg border p-3 text-xs space-y-1"
                        style={{ borderColor: "var(--desk-border)" }}
                      >
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <span className="font-mono font-semibold">
                            {tpl.code} · {tpl.name}
                          </span>
                          <div className="flex items-center gap-1.5">
                            <StatusBadge state="ACCENT" label={tpl.domain} />
                            {tpl.built_in ? (
                              <StatusBadge
                                state="PASS"
                                label={t("supplier_portal.p2p.built_in_badge")}
                              />
                            ) : null}
                          </div>
                        </div>
                        <div className="font-mono text-[11px]" style={{ color: "var(--desk-text-secondary)" }}>
                          {(tpl.steps ?? [])
                            .map((s) => `${s.code} (${formatDurationSec(s.sla_seconds)})`)
                            .join(" → ")}
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {canMutate ? (
                  <form
                    data-testid="create-process-template-form"
                    onSubmit={(e) => void handleCreateTemplate(e)}
                    className="space-y-3 pt-3 border-t"
                    style={{ borderColor: "var(--desk-border)" }}
                  >
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                      <div>
                        <label htmlFor="tpl-code-input" className="portal-label block mb-1">
                          {t("supplier_portal.p2p.template_code_input")}
                        </label>
                        <PortalInput
                          id="tpl-code-input"
                          required
                          value={tplCode}
                          onChange={(e) => setTplCode(e.target.value)}
                        />
                      </div>
                      <div>
                        <label htmlFor="tpl-name-input" className="portal-label block mb-1">
                          {t("supplier_portal.p2p.template_name_input")}
                        </label>
                        <PortalInput
                          id="tpl-name-input"
                          required
                          value={tplName}
                          onChange={(e) => setTplName(e.target.value)}
                        />
                      </div>
                      <div>
                        <label htmlFor="tpl-domain-select" className="portal-label block mb-1">
                          {t("supplier_portal.p2p.template_domain_input")}
                        </label>
                        <PortalSelect
                          id="tpl-domain-select"
                          value={tplDomain}
                          onChange={(e) => setTplDomain(e.target.value)}
                        >
                          <option value="P2P">P2P</option>
                          <option value="O2C">O2C</option>
                          <option value="RETURNS">RETURNS</option>
                        </PortalSelect>
                      </div>
                    </div>
                    <div>
                      <label htmlFor="tpl-steps-json" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.template_steps_json_label")}
                      </label>
                      <textarea
                        id="tpl-steps-json"
                        rows={4}
                        className="portal-input w-full font-mono text-xs"
                        value={tplStepsJson}
                        onChange={(e) => setTplStepsJson(e.target.value)}
                      />
                    </div>
                    <button
                      type="submit"
                      disabled={busyAction !== null || !tplCode.trim() || !tplName.trim()}
                      className="portal-btn portal-btn--primary"
                    >
                      {t("supplier_portal.p2p.create_template_btn")}
                    </button>
                  </form>
                ) : null}
              </div>
            </PortalSection>

            <PortalSection icon="dispatch" title={t("supplier_portal.p2p.instances_title")}>
              <div className="space-y-4">
                {canMutate ? (
                  <form
                    data-testid="process-instance-form"
                    onSubmit={(e) => void handleStartOrAdvanceInstance(e)}
                    className="grid grid-cols-1 sm:grid-cols-2 gap-2"
                  >
                    <div>
                      <label htmlFor="inst-tpl-input" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.template_code_label")}
                      </label>
                      <PortalInput
                        id="inst-tpl-input"
                        value={instTemplateCode}
                        onChange={(e) => setInstTemplateCode(e.target.value)}
                      />
                    </div>
                    <div>
                      <label htmlFor="inst-obj-type" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.instance_object_type_label")}
                      </label>
                      <PortalInput
                        id="inst-obj-type"
                        value={instObjectType}
                        onChange={(e) => setInstObjectType(e.target.value)}
                      />
                    </div>
                    <div>
                      <label htmlFor="inst-obj-id" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.instance_object_id_label")}
                      </label>
                      <PortalInput
                        id="inst-obj-id"
                        required
                        value={instObjectId}
                        onChange={(e) => setInstObjectId(e.target.value)}
                      />
                    </div>
                    <div>
                      <label htmlFor="inst-next-step" className="portal-label block mb-1">
                        {t("supplier_portal.p2p.instance_next_step_label")}
                      </label>
                      <PortalInput
                        id="inst-next-step"
                        value={instNextStep}
                        onChange={(e) => setInstNextStep(e.target.value)}
                      />
                    </div>
                    <div className="sm:col-span-2">
                      <button
                        type="submit"
                        disabled={busyAction !== null || !instObjectId.trim()}
                        className="portal-btn portal-btn--primary"
                      >
                        {t("supplier_portal.p2p.start_advance_btn")}
                      </button>
                    </div>
                  </form>
                ) : null}

                {instances.length === 0 ? (
                  <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.p2p.no_instances")}
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
                          }}
                        >
                          <th className="py-2 px-2">{t("supplier_portal.p2p.col_instance_id")}</th>
                          <th className="py-2 px-2">{t("supplier_portal.p2p.col_object")}</th>
                          <th className="py-2 px-2">{t("supplier_portal.p2p.col_current_step")}</th>
                          <th className="py-2 px-2">{t("supplier_portal.p2p.col_status")}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                        {instances.map((inst) => (
                          <tr key={inst.instance_id}>
                            <td className="py-2 px-2 font-mono">{inst.instance_id}</td>
                            <td className="py-2 px-2 font-mono">
                              {inst.object_type}:{inst.object_id}
                            </td>
                            <td className="py-2 px-2 font-mono">{inst.current_step}</td>
                            <td className="py-2 px-2">
                              <StatusBadge
                                state={mapStatusBadgeState(inst.status)}
                                label={inst.status}
                              />
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </PortalSection>
          </div>
        </div>
      ) : null}
    </PageChrome>
  );
}
