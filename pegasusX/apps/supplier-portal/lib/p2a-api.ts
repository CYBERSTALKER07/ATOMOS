import { createSupplierApi } from "@/lib/api";
import {
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  normalizeMinorBigInt,
  parseMajorToSafeMinor,
  type ApprovalInboxResponse,
  type ControlErrorDetail,
} from "@/lib/control-accounting-api";
import type { FinanceRequestTransport } from "@/lib/finance-api";
import type {
  ApprovePurchaseOrderRequest,
  BankCreditSummary,
  BankPackManifest,
  ConnectorPushRequest,
  ConnectorPushResponse,
  ConsolidationReport,
  CreateIntercompanyTransferRequest,
  CreateIntercompanyTransferResponse,
  CreateLegalEntityRequest,
  CreateProcessTemplateRequest,
  CreatePurchaseOrderRequest,
  CreatePurchaseOrderResponse,
  CreateVendorInvoiceRequest,
  ExportPack,
  FxRevaluationRunsListResponse,
  GoodsReceiptsListResponse,
  IntercompanyTransfersListResponse,
  LegalEntitiesListResponse,
  LegalEntity,
  LegalEntityResponse,
  P2AConnectorProvider,
  PostGoodsReceiptRequest,
  PostGoodsReceiptResponse,
  ProcessInstance,
  ProcessInstanceResponse,
  ProcessInstancesListResponse,
  ProcessMiningReport,
  ProcessTemplate,
  ProcessTemplateResponse,
  ProcessTemplatesListResponse,
  PurchaseOrder,
  PurchaseOrderDetailResponse,
  PurchaseOrdersListResponse,
  RunFxRevaluationRequest,
  RunFxRevaluationResponse,
  StartOrAdvanceProcessInstanceRequest,
  SubmitConnectorPushApprovalRequest,
  SubmitConnectorPushApprovalResponse,
  ThreeWayMatchRequest,
  ThreeWayMatchResult,
  UpdateLegalEntityRequest,
  VendorInvoice,
  VendorInvoiceResponse,
  VendorInvoicesListResponse,
} from "@pegasusx/types";

export {
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  normalizeMinorBigInt,
  parseMajorToSafeMinor,
  type ControlErrorDetail,
};

const BIGINT_ZERO = BigInt(0);
const BIGINT_SIXTY = BigInt(60);
const BIGINT_HOUR = BigInt(3600);
const BIGINT_DAY = BigInt(86400);
const BIGINT_HALF_BPS = BigInt(5000);
const BIGINT_BPS = BigInt(10000);
const BIGINT_DEFAULT_FX_SCALE = BigInt(100000000);

function withQuery(
  path: string,
  query: Record<string, string | number | boolean | undefined>,
): string {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && String(v).trim() !== "") {
      params.set(k, String(v).trim());
    }
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

function makeIdempotencyKey(prefix: string, explicit?: string): string {
  const clean = explicit?.trim();
  if (clean) return clean;
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}`;
}

function groupIntegerDigits(digits: string): string {
  if (digits.length <= 3) return digits;
  const parts: string[] = [];
  for (let i = digits.length; i > 0; i -= 3) {
    parts.push(digits.slice(Math.max(0, i - 3), i));
  }
  parts.reverse();
  return parts.join(",");
}

/**
 * Formats an integer-scaled FX rate (`rate_scaled / scale`) using pure BigInt math.
 */
export function formatFxScaledRate(
  rateScaled: number | bigint | null | undefined,
  scale: number | bigint | null | undefined = BIGINT_DEFAULT_FX_SCALE,
  inverse = false,
): string {
  const r = normalizeMinorBigInt(rateScaled);
  const s = normalizeMinorBigInt(scale ?? BIGINT_DEFAULT_FX_SCALE);
  if (r <= BIGINT_ZERO || s <= BIGINT_ZERO) {
    return "—";
  }
  const whole = r / s;
  const rem = r % s;
  const scaleStr = s.toString();
  const decimals = Math.max(0, scaleStr.length - 1);
  let formatted = groupIntegerDigits(whole.toString());
  if (decimals > 0 && rem > BIGINT_ZERO) {
    const padded = rem.toString().padStart(decimals, "0");
    const trimmed = padded.replace(/0+$/, "");
    const visibleFrac = trimmed.length < 2 ? padded.slice(0, Math.min(2, decimals)) : trimmed;
    if (visibleFrac) {
      formatted = `${formatted}.${visibleFrac}`;
    }
  }
  return inverse ? `${formatted} (inv)` : formatted;
}

/**
 * Formats integer seconds into compact human-readable durations without float math.
 */
export function formatDurationSec(seconds: number | bigint | null | undefined): string {
  const raw = normalizeMinorBigInt(seconds);
  const safe = raw < BIGINT_ZERO ? BIGINT_ZERO : raw;
  if (safe < BIGINT_SIXTY) {
    return `${safe.toString()}s`;
  }
  if (safe < BIGINT_HOUR) {
    const mins = safe / BIGINT_SIXTY;
    const secs = safe % BIGINT_SIXTY;
    return secs === BIGINT_ZERO ? `${mins.toString()}m` : `${mins.toString()}m ${secs.toString()}s`;
  }
  if (safe < BIGINT_DAY) {
    const hours = safe / BIGINT_HOUR;
    const mins = (safe % BIGINT_HOUR) / BIGINT_SIXTY;
    return mins === BIGINT_ZERO ? `${hours.toString()}h` : `${hours.toString()}h ${mins.toString()}m`;
  }
  const days = safe / BIGINT_DAY;
  const hours = (safe % BIGINT_DAY) / BIGINT_HOUR;
  return hours === BIGINT_ZERO ? `${days.toString()}d` : `${days.toString()}d ${hours.toString()}h`;
}

/**
 * Computes exact BigInt PO subtotal, VAT, and total minor units for live form previews.
 */
export function computePurchaseOrderPreviewTotals(
  lines: ReadonlyArray<{
    qtyOrdered: number | bigint;
    unitCostMinor: number | bigint;
    vatBps: number | bigint;
  }>,
): { subtotalMinor: bigint; vatMinor: bigint; totalMinor: bigint } {
  let subtotalMinor = BIGINT_ZERO;
  let vatMinor = BIGINT_ZERO;
  for (const line of lines) {
    const qty = normalizeMinorBigInt(line.qtyOrdered);
    const cost = normalizeMinorBigInt(line.unitCostMinor);
    const bps = normalizeMinorBigInt(line.vatBps);
    if (qty <= BIGINT_ZERO || cost < BIGINT_ZERO) continue;
    const lineNet = qty * cost;
    const lineVat = bps > BIGINT_ZERO ? (lineNet * bps + BIGINT_HALF_BPS) / BIGINT_BPS : BIGINT_ZERO;
    subtotalMinor += lineNet;
    vatMinor += lineVat;
  }
  return {
    subtotalMinor,
    vatMinor,
    totalMinor: subtotalMinor + vatMinor,
  };
}

/**
 * Extracts `bank_credit_summary` from a BANK ExportPack's hydrated manifest or raw manifest_json.
 */
export function extractBankCreditSummary(pack: ExportPack | null | undefined): BankCreditSummary | null {
  if (!pack) return null;
  const manifest = pack.manifest as BankPackManifest | undefined;
  if (manifest?.bank_credit_summary && typeof manifest.bank_credit_summary === "object") {
    return manifest.bank_credit_summary;
  }
  if (typeof pack.manifest_json === "string" && pack.manifest_json.trim() !== "") {
    try {
      const parsed = JSON.parse(pack.manifest_json) as BankPackManifest;
      if (parsed?.bank_credit_summary && typeof parsed.bank_credit_summary === "object") {
        return parsed.bank_credit_summary;
      }
    } catch {
      return null;
    }
  }
  return null;
}

function unwrapLegalEntity(res: LegalEntityResponse): LegalEntity {
  if (res && typeof res === "object" && "entity" in res && res.entity) {
    return res.entity;
  }
  return res as LegalEntity;
}

function unwrapPurchaseOrder(res: PurchaseOrderDetailResponse): PurchaseOrder {
  if (res && typeof res === "object" && "purchase_order" in res && res.purchase_order) {
    return res.purchase_order;
  }
  return res as PurchaseOrder;
}

function unwrapVendorInvoice(res: VendorInvoiceResponse): VendorInvoice {
  if (res && typeof res === "object" && "vendor_invoice" in res && res.vendor_invoice) {
    return res.vendor_invoice;
  }
  return res as VendorInvoice;
}

function unwrapProcessTemplate(res: ProcessTemplateResponse): ProcessTemplate {
  if (res && typeof res === "object" && "template" in res && res.template) {
    return res.template;
  }
  return res as ProcessTemplate;
}

function unwrapProcessInstance(res: ProcessInstanceResponse): ProcessInstance {
  if (res && typeof res === "object" && "instance" in res && res.instance) {
    return res.instance;
  }
  return res as ProcessInstance;
}

export function createP2aApi(client?: FinanceRequestTransport) {
  const transport = client ?? createSupplierApi();

  return {
    // ── §6.4 Multi-Entity, Intercompany Transfers, FX Revaluation & Consolidation ──
    listEntities: (query: { status?: string } = {}): Promise<LegalEntitiesListResponse> =>
      transport.request<LegalEntitiesListResponse>(withQuery("/v1/org/entities", query), "GET"),

    createEntity: async (
      body: CreateLegalEntityRequest,
      idempotencyKey?: string,
    ): Promise<LegalEntity> => {
      const res = await transport.request<LegalEntityResponse>("/v1/org/entities", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("org-entity-create", idempotencyKey),
      });
      return unwrapLegalEntity(res);
    },

    updateEntity: async (
      entityID: string,
      body: UpdateLegalEntityRequest,
      idempotencyKey?: string,
    ): Promise<LegalEntity> => {
      const cleanId = encodeURIComponent(entityID.trim());
      const res = await transport.request<LegalEntityResponse>(
        `/v1/org/entities/${cleanId}`,
        "PATCH",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`org-entity-patch-${entityID.trim()}`, idempotencyKey),
        },
      );
      return unwrapLegalEntity(res);
    },

    listIntercompanyTransfers: (
      query: { period?: string; entity?: string; limit?: number } = {},
    ): Promise<IntercompanyTransfersListResponse> =>
      transport.request<IntercompanyTransfersListResponse>(
        withQuery("/v1/finance/intercompany-transfers", query),
        "GET",
      ),

    createIntercompanyTransfer: (
      body: CreateIntercompanyTransferRequest,
      idempotencyKey?: string,
    ): Promise<CreateIntercompanyTransferResponse> => {
      const resolvedKey = makeIdempotencyKey(
        "ic-transfer",
        idempotencyKey ?? body.idempotency_key,
      );
      return transport.request<CreateIntercompanyTransferResponse>(
        "/v1/finance/intercompany-transfers",
        "POST",
        {
          body: { ...body, idempotency_key: resolvedKey },
          idempotencyKey: resolvedKey,
        },
      );
    },

    listFxRevaluations: (
      query: { period?: string; entity?: string; limit?: number } = {},
    ): Promise<FxRevaluationRunsListResponse> =>
      transport.request<FxRevaluationRunsListResponse>(
        withQuery("/v1/finance/fx-revaluation", query),
        "GET",
      ),

    runFxRevaluation: (
      body: RunFxRevaluationRequest,
      idempotencyKey?: string,
    ): Promise<RunFxRevaluationResponse> => {
      const resolvedKey = makeIdempotencyKey(
        "fx-revaluation",
        idempotencyKey ?? body.idempotency_key,
      );
      return transport.request<RunFxRevaluationResponse>("/v1/finance/fx-revaluation", "POST", {
        body: { ...body, idempotency_key: resolvedKey },
        idempotencyKey: resolvedKey,
      });
    },

    getConsolidation: (query: {
      period?: string;
      currency?: string;
      entities?: string | string[];
    } = {}): Promise<ConsolidationReport> => {
      const entitiesParam = Array.isArray(query.entities)
        ? query.entities.filter(Boolean).join(",")
        : query.entities;
      return transport.request<ConsolidationReport>(
        withQuery("/v1/finance/consolidation", {
          period: query.period,
          currency: query.currency,
          entities: entitiesParam,
        }),
        "GET",
      );
    },

    // ── §6.6 Process Templates, Process Instances & Process Mining ──
    listProcessTemplates: (
      query: { domain?: string; status?: string } = {},
    ): Promise<ProcessTemplatesListResponse> =>
      transport.request<ProcessTemplatesListResponse>(
        withQuery("/v1/processes/templates", query),
        "GET",
      ),

    createProcessTemplate: async (
      body: CreateProcessTemplateRequest,
      idempotencyKey?: string,
    ): Promise<ProcessTemplate> => {
      const res = await transport.request<ProcessTemplateResponse>(
        "/v1/processes/templates",
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey("process-template", idempotencyKey),
        },
      );
      return unwrapProcessTemplate(res);
    },

    listProcessInstances: (
      query: { object_type?: string; object_id?: string; entity?: string; limit?: number } = {},
    ): Promise<ProcessInstancesListResponse> =>
      transport.request<ProcessInstancesListResponse>(
        withQuery("/v1/processes/instances", query),
        "GET",
      ),

    startOrAdvanceProcessInstance: async (
      body: StartOrAdvanceProcessInstanceRequest,
      idempotencyKey?: string,
    ): Promise<ProcessInstance> => {
      const res = await transport.request<ProcessInstanceResponse>(
        "/v1/processes/instances",
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey("process-instance", idempotencyKey),
        },
      );
      return unwrapProcessInstance(res);
    },

    getProcessMining: (
      query: { domain?: string; entity?: string; from?: string; to?: string } = {},
    ): Promise<ProcessMiningReport> =>
      transport.request<ProcessMiningReport>(withQuery("/v1/processes/mining", query), "GET"),

    // ── §6.6 Procure-to-Pay (POs, Goods Receipts, Vendor Invoices, 3-Way Match) ──
    listPurchaseOrders: (
      query: { entity?: string; status?: string; vendor_id?: string; limit?: number } = {},
    ): Promise<PurchaseOrdersListResponse> =>
      transport.request<PurchaseOrdersListResponse>(
        withQuery("/v1/p2p/purchase-orders", query),
        "GET",
      ),

    getPurchaseOrder: async (poID: string): Promise<PurchaseOrder> => {
      const cleanId = encodeURIComponent(poID.trim());
      const res = await transport.request<PurchaseOrderDetailResponse>(
        `/v1/p2p/purchase-orders/${cleanId}`,
        "GET",
      );
      return unwrapPurchaseOrder(res);
    },

    createPurchaseOrder: (
      body: CreatePurchaseOrderRequest,
      idempotencyKey?: string,
    ): Promise<CreatePurchaseOrderResponse> => {
      const resolvedKey = makeIdempotencyKey(
        "p2p-po-create",
        idempotencyKey ?? body.idempotency_key,
      );
      return transport.request<CreatePurchaseOrderResponse>("/v1/p2p/purchase-orders", "POST", {
        body: { ...body, idempotency_key: resolvedKey },
        idempotencyKey: resolvedKey,
      });
    },

    approvePurchaseOrder: async (
      poID: string,
      body: ApprovePurchaseOrderRequest = {},
      idempotencyKey?: string,
    ): Promise<PurchaseOrder> => {
      const cleanId = encodeURIComponent(poID.trim());
      const res = await transport.request<PurchaseOrderDetailResponse>(
        `/v1/p2p/purchase-orders/${cleanId}/approve`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`p2p-po-approve-${poID.trim()}`, idempotencyKey),
        },
      );
      return unwrapPurchaseOrder(res);
    },

    listGoodsReceipts: (
      query: { po_id?: string; entity?: string; limit?: number } = {},
    ): Promise<GoodsReceiptsListResponse> =>
      transport.request<GoodsReceiptsListResponse>(withQuery("/v1/p2p/receipts", query), "GET"),

    postGoodsReceipt: (
      body: PostGoodsReceiptRequest,
      idempotencyKey?: string,
    ): Promise<PostGoodsReceiptResponse> => {
      const resolvedKey = makeIdempotencyKey(
        "p2p-receipt-post",
        idempotencyKey ?? body.idempotency_key,
      );
      return transport.request<PostGoodsReceiptResponse>("/v1/p2p/receipts", "POST", {
        body: { ...body, idempotency_key: resolvedKey },
        idempotencyKey: resolvedKey,
      });
    },

    listVendorInvoices: (
      query: { po_id?: string; match_status?: string; entity?: string; limit?: number } = {},
    ): Promise<VendorInvoicesListResponse> =>
      transport.request<VendorInvoicesListResponse>(
        withQuery("/v1/p2p/vendor-invoices", query),
        "GET",
      ),

    createVendorInvoice: async (
      body: CreateVendorInvoiceRequest,
      idempotencyKey?: string,
    ): Promise<VendorInvoice> => {
      const resolvedKey = makeIdempotencyKey(
        "p2p-vendor-invoice",
        idempotencyKey ?? body.idempotency_key,
      );
      const res = await transport.request<VendorInvoiceResponse>("/v1/p2p/vendor-invoices", "POST", {
        body: { ...body, idempotency_key: resolvedKey },
        idempotencyKey: resolvedKey,
      });
      return unwrapVendorInvoice(res);
    },

    runThreeWayMatch: (
      body: ThreeWayMatchRequest,
      idempotencyKey?: string,
    ): Promise<ThreeWayMatchResult> =>
      transport.request<ThreeWayMatchResult>("/v1/p2p/three-way-match", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey(
          `p2p-3way-${body.vendor_invoice_id.trim()}`,
          idempotencyKey,
        ),
      }),

    // ── §7.10 Approval-Gated 2-Way Connector Push ──
    listConnectorPushApprovals: (
      status = "APPROVED",
    ): Promise<ApprovalInboxResponse> =>
      transport.request<ApprovalInboxResponse>(
        withQuery("/v1/approvals/inbox", {
          object_type: "ConnectorPush",
          status,
          limit: 50,
        }),
        "GET",
      ),

    submitConnectorPushApproval: (
      body: SubmitConnectorPushApprovalRequest,
      idempotencyKey?: string,
    ): Promise<SubmitConnectorPushApprovalResponse> => {
      const resolvedKey = makeIdempotencyKey(
        `connector-push-approval-${body.object_id}`,
        idempotencyKey ?? body.idempotency_key,
      );
      return transport.request<SubmitConnectorPushApprovalResponse>(
        "/v1/approvals/requests",
        "POST",
        {
          body: {
            ...body,
            object_type: body.object_type || "ConnectorPush",
            action: body.action || "push",
            force_create: body.force_create ?? true,
            idempotency_key: resolvedKey,
          },
          idempotencyKey: resolvedKey,
        },
      );
    },

    pushConnector: (
      provider: P2AConnectorProvider | string,
      body: ConnectorPushRequest,
      idempotencyKey?: string,
    ): Promise<ConnectorPushResponse> => {
      const cleanProvider = encodeURIComponent(provider.trim());
      const resolvedKey = makeIdempotencyKey(
        `connector-push-${provider.trim()}`,
        idempotencyKey ?? body.idempotency_key,
      );
      return transport.request<ConnectorPushResponse>(
        `/v1/connectors/${cleanProvider}/push`,
        "POST",
        {
          body: { ...body, idempotency_key: resolvedKey },
          idempotencyKey: resolvedKey,
        },
      );
    },
  };
}

export type P2aApi = ReturnType<typeof createP2aApi>;
