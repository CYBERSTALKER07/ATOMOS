import type { SupplierSettingsResponse, RecommendReassignRequest, RecommendReassignResponse, ApplyReassignRequest, StatusResponse, 
  ConfirmAIOrderRequest,
  ConfirmPreorderRequest,
  EditPreorderRequest,
  AcceptDeliveryProposalRequest,
  RejectDeliveryProposalRequest,
  RejectPreorderRequest,
  ProposeDeliveryDateRequest,
  PaymentChargebackRequest,
  PaymentChargebackResponse,
  PaymentChargebackReversalRequest,
  PaymentChargebackReversalResponse,
  SupplierClaimsListResponse,
  ApproveClaimRequest,
  ApproveClaimResponse,
  RejectClaimRequest,
  Claim,
  ClaimChargebacksQuery,
  ClaimChargebacksResponse,
  RejectAIOrderRequest,
  RetailerAIPredictionsResponse,
  RetailerOrderLifecycleResponse,
  PaymentLedgerQuery,
  PaymentLedgerResponse,
  ReconciliationMismatchQuery,
  ReconciliationMismatchResponse,
  RetailerActiveFulfillmentResponse,
  RetailerPendingPaymentsResponse,
  RetailerProfileResponse,
  RetailerPricingRuleResponse,
  RetailerStockCountCommitRequest,
  RetailerStockCountCommitResponse,
  RetailerStockCountVersionResponse,
  RetailerProfileUpdateRequest,
  CreateRetailerPriceOverrideRequest,
  CreateRetailerPriceOverrideResponse,
  RetailerPriceOverridesResponse,
  RetailerSupplierPreference,
  RetailerTrackingResponse,
  PulseResponse,
  ExceptionMapResponse,
  BroadcastTemplatesResponse,
  BroadcastTemplate,
  RetailerOverridePreview,
  SupplyFulfillOptions,
  SettlementAuthorityQuery,
  SettlementAuthorityResponse,
  SupplierBillingSetupRequest,
  SupplierBillingSetupResponse,
  SupplierBusinessSetupRequest,
  SupplierBusinessSetupResponse,
  SupplierConfigureRequest,
  SupplierConfigureResponse,
  SupplierDashboardResponse,
  SupplierExceptionsResponse,
  ShopClosedActiveResponse,
  ShopClosedResolveRequest,
  ShopClosedReportRequest,
  ShopClosedReportResponse,
  SupplierReturnPolicy,
  WarehouseReturnPolicy,
  ProximityUnlockRequest,
  ProximityUnlockResponse,
  PartialOffloadRequest,
  PartialOffloadResponse,
  CreditDeliveryRequest,
  SupplierCreditProgram,
  RetailerPaymentTerms,
  ArInvoice,
  ComplianceDashboardResponse,
  ComplianceExportBucket,
  NegotiationPendingResponse,
  NegotiationResolveRequest,
  NegotiationResolveResponse,
  PaymentBypassRequest,
  PaymentBypassResponse,
  SupplierEmpathyAdoption,
  SupplierBroadcastRequest,
  SupplierBroadcastResponse,
  SupplierReplenishmentTriggerResponse,
  SupplierFleetOrderRow,
  SupplierFleetLiveMapResponse,
  SupplierLoginRequest,
  SupplierLoginResponse,
  SupplierManifestsResponse,
  SupplierManifestDetail,
  SupplierManifestExceptionsResponse,
  SupplierManifestInjectOrderRequest,
  SupplierManifestSealResponse,
  SupplierSupplyLanesResponse,
  SupplierAIRecommendationDecisionRequest,
  SupplierAIRecommendationDecisionResponse,
  SupplierAIRecommendationsQuery,
  SupplierActivityResponse,
  SupplierAIRecommendationsResponse,
  SupplierAnalyticsVelocityResponse,
  SupplierAnalyticsRevenueResponse,
  SupplierDemandSummaryResponse,
  SupplierDemandHistoryResponse,
  SupplierMEIONetworkSummary,
  SupplierReplenishmentPolicy,
  SupplierReplenishmentTraceabilityResponse,
  ReorderSuggestionsListResponse,
  ReorderSuggestionDismissRequest,
  ReorderSuggestionCreateDraftRequest,
  ReorderSuggestionBulkCreateDraftsRequest,
  ReorderSuggestionBulkCreateDraftsResponse,
  CashReconciliationRow,
  CashReconciliationsListResponse,
  CashReconciliationActionRequest,
  CreditNoteRow,
  CreditNotesListResponse,
  CreateCreditNoteRequest,
  TwinOpsRouteView,
  TwinVehicleInventoryRow,
  ControlTowerZoneOverride,
  ControlTowerZoneOverridesResponse,
  ControlTowerZoneOverrideRequest,
  ControlTowerPlaybookRunsResponse,
  ScoredExceptionsResponse,
  ControlTowerPlaybooksResponse,
  RetailerSegmentsResponse,
  SkuClassesResponse,
  SegmentationBootstrapResult,
  SetRetailerSegmentInput,
  SetSkuClassInput,
  PlanningScenarioInput,
  PlanningScenarioResult,
  PlanningSignalIngestStatus,
  PlanningSAndOPSnapshot,
  SupplierKnowledgeGraph,
  GovernedAgentInvocation,
  GovernedAgentInvocationResponse,
  SeasonalOverrideInput,
  SeasonalOverrideRow,
  SeasonalTemplatesResponse,
  PlanningSignalIngestInput,
  SparsityGateResult,
  PromoSimulateInput,
  PromoSimulateResult,
  PromoPerformanceResult,
  SupplierInventoryImportResult,
  SupplierImportApplyResponse,
  SupplierImportIngestResponse,
  SupplierImportMappingResponse,
  SupplierImportSession,
  SupplierImportSessionCreateResponse,
  SupplierDispatchPreview,
  SupplierDispatchExecuteRequest,
  SupplierDispatchExecuteResponse,
  SupplierOrgMemberCreateRequest,
  SupplierOrgMemberUpdateRequest,
  SupplierOrgMembersResponse,
  SupplierOrdersResponse,
  SupplierOrder,
  AssignOrderRequest,
  AssignOrderResponse,
  OrderStatusPatchRequest,
  OrderStatusPatchResponse,
  SupplierFleetDriverCreateRequest,
  SupplierFleetDriversResponse,
  SupplierFleetVehicleCreateRequest,
  SupplierFleetVehiclesResponse,
  SupplierProfile,
  SupplierProfileUpdateRequest,
  SupplierPricingRule,
  SupplierPricingRuleUpdateRequest,
  SupplierPromotion,
  SupplierPromotionUpsertRequest,
  SupplierPromotionsResponse,
  SupplierRegisterRequest,
  SupplierRegisterResponse,
  SupplierInventoryResponse,
  SupplierTopologyResponse,
  SupplierTopologyUpdateRequest,
  WarehouseDispatchLock,
  WarehouseDispatchLockAcquireRequest,
  WarehouseDispatchLockReleaseResponse,
  WarehouseDispatchLocksResponse,
  WarehouseDispatchExecuteRequest,
  WarehouseDispatchExecuteResponse,
  WarehouseDispatchPreview,
  WarehouseDemandForecastResponse,
  WarehouseEmergencyTransferRequest,
  WarehouseForceReceiveRequest,
  WarehouseInventoryResponse,
  WarehouseFleetLiveMapResponse,
  WarehouseFleetVehicleDetailResponse,
  WarehouseFleetVehicleListResponse,
  WarehouseOpsDashboardResponse,
  WarehouseOpsFinancialsResponse,
  WarehouseOpsSettings,
  WarehouseOpsSettingsPatchRequest,
  CheckoutPreviewResponse,
  WarehouseOrderDetail,
  OrderTimelineResponse,
  WarehouseOrderMutationRequest,
  WarehouseOrderMutationResponse,
  WarehouseOrdersResponse,
  WarehousePreorderEditRequest,
  WarehousePreordersResponse,
  WarehouseDispatchSettingsPatchRequest,
  WarehouseDispatchSettingsResponse,
  WarehouseReplenishmentInsightActionResponse,
  WarehouseReplenishmentInsightsResponse,
  WarehouseSupplyRequest,
  WarehouseSupplyRequestsResponse,
  WarehouseTransferMutationResponse,
  WarehouseDispatchRunsResponse,
  WarehouseDispatchRun,
  WarehouseOpsBoardResponse,
  WarehouseOpsExceptionsResponse,
} from "@pegasusx/types";

export { reconnectDelayMs, parseRetryAfterSeconds, retryAfterSecondsFromResponse } from "./reconnect";
export type { ReconnectBackoffOptions } from "./reconnect";
export {
  driverDeliverKey,
  driverOffloadKey,
  driverCompleteKey,
  driverCollectCashKey,
  driverFiscalRetryKey,
  adminForceCompleteKey,
  driverAvailabilityKey,
  retailerCheckoutKey,
  retailerUnifiedCheckoutKey,
  retailerCardCheckoutKey,
  retailerCashCheckoutKey,
  retailerPosSaleKey,
  retailerSupplierAddKey,
  retailerSupplierRemoveKey,
  supplierDispatchKey,
  warehouseDispatchKey,
  warehouseOrderDelayKey,
  warehouseOrderRejectKey,
  warehouseOrderOverflowKey,
  warehouseDispatchLockAcquireKey,
  warehouseDispatchLockReleaseKey,
  payloadStartLoadingKey,
  payloadSupplierStartLoadingKey,
  payloadSealKey,
  payloadOrderSealKey,
  payloadInjectKey,
  payloadSupplierInjectKey,
  payloadSealCompletedKey,
  payloadRecommendReassignKey,
  payloadFleetReassignKey,
  payloadApplyReassignKey,
  payloadSupplierSealManifestKey,
  supplierManifestSealKey,
  supplierManifestSealAllKey,
  payloadSealAllKey,
  supplierManifestStartLoadingKey,
  supplierManifestInjectKey,
  supplierVetOrderKey,
  supplierImportCreateKey,
  supplierImportIngestKey,
  supplierImportApproveKey,
  supplierImportApplyKey,
  supplierOrgMemberCreateKey,
  supplierOrgMemberUpdateKey,
  supplierOrgMemberDeactivateKey,
  supplierFleetDriverCreateKey,
  supplierFleetVehicleCreateKey,
  supplierChargebackKey,
  supplierChargebackReversalKey,
  supplierTopologyPutKey,
  supplierReplenishmentTriggerKey,
  supplierBroadcastKey,
  supplierPaymentBypassKey,
  supplierApproveEarlyCompleteKey,
  supplierNegotiationResolveKey,
  driverRequestEarlyCompleteKey,
  driverRouteReorderKey,
  driverConfirmPaymentBypassKey,
  driverBypassOffloadKey,
  driverReportShopClosedKey,
  driverProximityUnlockKey,
  driverPartialOffloadKey,
  supplierShopClosedResolveKey,
  supplierResolveReturnKey,
  supplierProfileUpdateKey,
  supplierConfigureKey,
  supplierBusinessSetupKey,
  supplierPricingRulePatchKey,
  supplierInventoryAdjustKey,
  supplierRetailerPriceOverrideCreateKey,
  supplierRetailerPriceOverrideDeleteKey,
  supplierPromotionCreateKey,
  supplierPromotionUpdateKey,
  supplierPromotionDeactivateKey,
  driverSupplyTransferArriveKey,
  warehouseCreateSupplyRequestKey,
  warehouseSupplyRequestTransitionKey,
  driverDepartKey,
  driverReturnCompleteKey,
  driverSyncBatchKey,
  driverMarkArrivedKey,
  driverSplitPaymentKey,
  driverCreditDeliveryKey,
  driverMissingItemsKey,
  payloadMissingItemsKey,
  driverReportDamageKey,
  driverNegotiateKey,
  retailerOrderCreateKey,
  retailerConfirmCashKey,
  retailerCancelKey,
  retailerRequestCancelKey,
  retailerShopClosedResponseKey,
  retailerConfirmPreorderKey,
  retailerAcceptDeliveryProposalKey,
  retailerRejectDeliveryProposalKey,
  retailerRejectPreorderKey,
  warehouseOrderProposeDeliveryKey,
  retailerConfirmAIKey,
  retailerRejectAIKey,
  retailerEditPreorderKey,
  retailerSetupKey,
  retailerProfileUpdateKey,
  adminOrderAssignKey,
  adminOrderStatusPatchKey,
  warehouseEmergencyTransferKey,
  warehouseForceReceiveKey,
  warehouseReceiveTransferKey,
  warehouseCreateDriverKey,
  warehouseCreateStaffKey,
  warehouseCreateVehicleKey,
  warehouseAdjustInventoryKey,
  warehouseAssignDriverVehicleKey,
  warehouseUpdateVehicleKey,
  warehouseInventoryPolicyKey,
  warehouseReplenishmentInsightActionKey,
  warehouseDispatchSettingsKey,
  warehouseOpsSettingsKey,
  supplierReturnPolicyPutKey,
  warehouseReturnPolicyPutKey,
  warehouseOpsLocationKey,
  warehouseBroadcastKey,
  warehouseBroadcastTemplateCreateKey,
  warehouseBroadcastTemplateDeleteKey,
  factoryManifestStartLoadingKey,
  factoryManifestSealKey,
  factoryManifestDispatchKey,
  factoryManifestCompleteKey,
  factoryBatchDispatchKey,
  factoryManifestRebalanceKey,
  factoryManifestCancelTransferKey,
  factoryManifestCancelKey,
  factoryTransferCreateKey,
  factoryTransferTransitionKey,
  factorySupplyRequestTransitionKey,
  factorySupplyRequestAcceptKey,
  factoryOpsLocationKey,
  warehouseInboundScanKey,
  warehouseInboundConfirmKey,
  payloadInboundScanKey,
  payloadInboundConfirmKey,
  payloadManifestExceptionKey,
} from "./idempotency";
export {
  SESSION_RECONCILE_ENDPOINTS,
  reconcileSession,
} from "./session-reconcile";
export type { SessionReconcileRole, SessionReconcileEndpoint, SessionReconcileOptions, SessionReconcileResult } from "./session-reconcile";
export { usePolling } from "./usePolling";
export type { UsePollingOptions } from "./usePolling";

export interface ApiClientConfig {
  baseUrl: string;
  getAuthToken?: () => string | null;
  fetchImpl?: typeof fetch;
}

interface RequestOptions {
  body?: unknown;
  rawBody?: string;
  idempotencyKey?: string;
  requiresAuth?: boolean;
  headers?: HeadersInit;
}

export class ApiError extends Error {
  public readonly status: number;
  public readonly payload: unknown;

  constructor(message: string, status: number, payload: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

export class ApiClient {
  constructor(public readonly config: ApiClientConfig) {}

  async registerSupplier(request: SupplierRegisterRequest, idempotencyKey: string): Promise<SupplierRegisterResponse> {
    return this.request<SupplierRegisterResponse>("/v1/auth/supplier/register", "POST", {
      body: request,
      idempotencyKey,
      requiresAuth: false,
    });
  }

  async loginSupplier(request: SupplierLoginRequest): Promise<SupplierLoginResponse> {
    return this.request<SupplierLoginResponse>("/v1/auth/supplier/login", "POST", {
      body: request,
      requiresAuth: false,
    });
  }

  async configureSupplier(
    request: SupplierConfigureRequest,
    idempotencyKey?: string,
  ): Promise<SupplierConfigureResponse> {
    return this.request<SupplierConfigureResponse>("/v1/supplier/configure", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async configureSupplierBilling(
    request: SupplierBillingSetupRequest,
    idempotencyKey: string,
  ): Promise<SupplierBillingSetupResponse> {
    return this.request<SupplierBillingSetupResponse>("/v1/supplier/billing/setup", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async setupSupplierBusiness(
    request: SupplierBusinessSetupRequest,
    idempotencyKey?: string,
  ): Promise<SupplierBusinessSetupResponse> {
    return this.request<SupplierBusinessSetupResponse>("/v1/supplier/business/setup", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierProfile(): Promise<SupplierProfile> {
    return this.request<SupplierProfile>("/v1/supplier/profile", "GET");
  }

  async updateSupplierProfile(
    request: SupplierProfileUpdateRequest,
    idempotencyKey: string,
  ): Promise<SupplierProfile> {
    return this.request<SupplierProfile>("/v1/supplier/profile", "PUT", { body: request, idempotencyKey });
  }

  async getSupplierTopology(): Promise<SupplierTopologyResponse> {
    return this.request<SupplierTopologyResponse>("/v1/supplier/topology", "GET");
  }

  async updateSupplierTopology(request: SupplierTopologyUpdateRequest): Promise<SupplierTopologyResponse> {
    return this.request<SupplierTopologyResponse>("/v1/supplier/topology", "PUT", { body: request });
  }

  async getSupplierInventory(): Promise<SupplierInventoryResponse> {
    return this.request<SupplierInventoryResponse>("/v1/supplier/inventory", "GET");
  }

  async getSupplierOrgMembers(): Promise<SupplierOrgMembersResponse> {
    return this.request<SupplierOrgMembersResponse>("/v1/supplier/org/members", "GET");
  }

  async createSupplierOrgMember(
    request: SupplierOrgMemberCreateRequest,
    idempotencyKey: string,
  ): Promise<SupplierOrgMembersResponse> {
    return this.request<SupplierOrgMembersResponse>("/v1/supplier/org/members", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async updateSupplierOrgMember(
    userId: string,
    request: SupplierOrgMemberUpdateRequest,
    idempotencyKey: string,
  ): Promise<SupplierOrgMembersResponse> {
    return this.request<SupplierOrgMembersResponse>(`/v1/supplier/org/members/${encodeURIComponent(userId)}`, "PATCH", {
      body: request,
      idempotencyKey,
    });
  }

  async deactivateSupplierOrgMember(
    userId: string,
    idempotencyKey: string,
  ): Promise<SupplierOrgMembersResponse> {
    return this.request<SupplierOrgMembersResponse>(`/v1/supplier/org/members/${encodeURIComponent(userId)}`, "DELETE", {
      idempotencyKey,
    });
  }

  async getSupplierFleetDrivers(): Promise<SupplierFleetDriversResponse> {
    return this.request<SupplierFleetDriversResponse>("/v1/supplier/fleet/drivers", "GET");
  }

  async createSupplierFleetDriver(
    request: SupplierFleetDriverCreateRequest,
    idempotencyKey: string,
  ): Promise<SupplierFleetDriversResponse> {
    return this.request<SupplierFleetDriversResponse>("/v1/supplier/fleet/drivers", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierFleetVehicles(): Promise<SupplierFleetVehiclesResponse> {
    return this.request<SupplierFleetVehiclesResponse>("/v1/supplier/fleet/vehicles", "GET");
  }

  async createSupplierFleetVehicle(
    request: SupplierFleetVehicleCreateRequest,
    idempotencyKey: string,
  ): Promise<SupplierFleetVehiclesResponse> {
    return this.request<SupplierFleetVehiclesResponse>("/v1/supplier/fleet/vehicles", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierPricingRule(): Promise<SupplierPricingRule> {
    return this.request<SupplierPricingRule>("/v1/supplier/pricing/rules", "GET");
  }

  async listRetailerPriceOverrides(params?: {
    retailer_id?: string;
    product_id?: string;
  }): Promise<RetailerPriceOverridesResponse> {
    const query = new URLSearchParams();
    if (params?.retailer_id) query.set("retailer_id", params.retailer_id);
    if (params?.product_id) query.set("product_id", params.product_id);
    const suffix = query.toString();
    const path = suffix
      ? `/v1/supplier/pricing/retailer-overrides?${suffix}`
      : "/v1/supplier/pricing/retailer-overrides";
    return this.request<RetailerPriceOverridesResponse>(path, "GET");
  }

  async createRetailerPriceOverride(
    request: CreateRetailerPriceOverrideRequest,
    idempotencyKey: string,
  ): Promise<CreateRetailerPriceOverrideResponse> {
    return this.request<CreateRetailerPriceOverrideResponse>(
      "/v1/supplier/pricing/retailer-overrides",
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async deleteRetailerPriceOverride(
    overrideId: string,
    idempotencyKey: string,
  ): Promise<{ status: string; override_id: string }> {
    return this.request<{ status: string; override_id: string }>(
      `/v1/supplier/pricing/retailer-overrides/${overrideId}`,
      "DELETE",
      { idempotencyKey },
    );
  }

  async previewRetailerPriceOverride(body: {
    retailer_id?: string;
    product_id?: string;
    sku_id?: string;
    proposed_price: number;
  }): Promise<RetailerOverridePreview> {
    return this.request<RetailerOverridePreview>(
      "/v1/supplier/pricing/retailer-overrides/preview",
      "POST",
      { body },
    );
  }

  async getSupplierExceptionMap(query: { window_hours?: number } = {}): Promise<ExceptionMapResponse> {
    return this.request<ExceptionMapResponse>(
      appendQuery("/v1/supplier/ops/exception-map", query as Record<string, unknown>),
      "GET",
    );
  }

  async getFactorySupplyFulfillOptions(requestId: string): Promise<SupplyFulfillOptions> {
    return this.request<SupplyFulfillOptions>(
      `/v1/factory/supply-requests/${requestId}/fulfill-options`,
      "GET",
    );
  }

  async listSupplierPromotions(): Promise<SupplierPromotionsResponse> {
    return this.request<SupplierPromotionsResponse>("/v1/supplier/promotions", "GET");
  }

  async createSupplierPromotion(
    request: SupplierPromotionUpsertRequest,
    idempotencyKey: string,
  ): Promise<SupplierPromotion> {
    return this.request<SupplierPromotion>("/v1/supplier/promotions", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async updateSupplierPromotion(
    promotionId: string,
    request: SupplierPromotionUpsertRequest,
    idempotencyKey: string,
  ): Promise<SupplierPromotion> {
    return this.request<SupplierPromotion>(`/v1/supplier/promotions/${promotionId}`, "PATCH", {
      body: request,
      idempotencyKey,
    });
  }

  async deactivateSupplierPromotion(
    promotionId: string,
    idempotencyKey: string,
  ): Promise<{ status: string }> {
    return this.request<{ status: string }>(`/v1/supplier/promotions/${promotionId}`, "DELETE", {
      idempotencyKey,
    });
  }

  async getSupplierOrders(
    query: {
      limit?: number;
      offset?: number;
      status?: string;
      filter?: "ACTIVE" | "COMPLETED" | "CANCELLED" | "RETURNS";
    } = {},
  ): Promise<SupplierOrdersResponse> {
    return this.request<SupplierOrdersResponse>(appendQuery("/v1/supplier/orders", query as Record<string, unknown>), "GET");
  }

  async vetSupplierOrder(
    request: { order_id: string; decision: string; note?: string },
    idempotencyKey?: string,
  ): Promise<{ order: SupplierOrder }> {
    return this.request<{ order: SupplierOrder }>("/v1/supplier/orders/vet", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** ADMIN / WAREHOUSE_ADMIN / FACTORY_ADMIN — assign driver to order. */
  async assignOrder(
    orderId: string,
    request: AssignOrderRequest,
    idempotencyKey: string,
  ): Promise<AssignOrderResponse> {
    return this.request<AssignOrderResponse>(`/v1/orders/${encodeURIComponent(orderId)}/assign`, "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** ADMIN / RETAILER — patch canonical order status. */
  async patchOrderStatus(
    orderId: string,
    request: OrderStatusPatchRequest,
    idempotencyKey: string,
  ): Promise<OrderStatusPatchResponse> {
    return this.request<OrderStatusPatchResponse>(`/v1/order/${encodeURIComponent(orderId)}/status`, "PATCH", {
      body: request,
      idempotencyKey,
    });
  }

  /** ADMIN / WAREHOUSE_ADMIN — audited force-complete past fiscal failure (ADR-009). */
  async forceCompleteOrder(
    orderId: string,
    request: { reason_code: string },
    idempotencyKey: string,
  ): Promise<{ order_id: string; state: string; message?: string; attempt_id?: string }> {
    return this.request(`/v1/order/${encodeURIComponent(orderId)}/force-complete`, "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** DRIVER / ADMIN / WAREHOUSE_ADMIN — retry fiscal when FISCAL_FAILED. */
  async retryFiscal(
    orderId: string,
    idempotencyKey: string,
  ): Promise<{ order_id: string; state: string; attempt_id?: string; message?: string }> {
    return this.request(`/v1/order/${encodeURIComponent(orderId)}/fiscal/retry`, "POST", {
      body: {},
      idempotencyKey,
    });
  }

  /** Role-scoped Pegasus settlement receipt metadata (html_url / pdf_url for branded doc). */
  async getOrderReceipt(
    role: "retailer" | "supplier" | "warehouse",
    orderId: string,
  ): Promise<{
    receipt_id: string;
    html_url?: string;
    pdf_url?: string;
    qr_url?: string;
    party_copy?: string;
    legal_class?: string;
    tax_ofd?: boolean;
    amount_minor?: number;
    currency?: string;
  }> {
    return this.request(
      `/v1/${role}/orders/${encodeURIComponent(orderId)}/receipt?format=json`,
      "GET",
    );
  }

  /** Fetch branded receipt HTML or PDF bytes with auth (opens via blob URL in portals). */
  async fetchOrderReceiptBlob(
    role: "retailer" | "supplier" | "warehouse",
    orderId: string,
    format: "html" | "pdf",
  ): Promise<Blob> {
    const path = `/v1/${role}/orders/${encodeURIComponent(orderId)}/receipt?format=${format}`;
    const headers = this.buildHeaders(undefined, true, undefined, true);
    const fetchImpl = this.config.fetchImpl ?? fetch;
    const response = await fetchImpl(this.resolveURL(path), { method: "GET", headers });
    if (!response.ok) {
      const payload = await parseResponsePayload(response);
      throw new ApiError(extractErrorMessage(response.status, payload), response.status, payload);
    }
    return response.blob();
  }

  async getSupplierDispatchPreview(query: { warehouse_id?: string } = {}): Promise<SupplierDispatchPreview> {
    return this.request<SupplierDispatchPreview>(
      appendQuery("/v1/supplier/dispatch/preview", query as Record<string, unknown>),
      "GET",
    );
  }

  async createSupplierDispatchPreview(
    request: Record<string, unknown>,
  ): Promise<SupplierDispatchPreview> {
    return this.request<SupplierDispatchPreview>("/v1/supplier/dispatch/preview", "POST", {
      body: request,
    });
  }

  async executeSupplierDispatch(
    request: SupplierDispatchExecuteRequest = { mode: "AUTO" },
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<SupplierDispatchExecuteResponse> {
    return this.request<SupplierDispatchExecuteResponse>(
      appendQuery("/v1/supplier/dispatch/execute", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getSupplierActivity(query: { limit?: number } = {}): Promise<SupplierActivityResponse> {
    return this.request<SupplierActivityResponse>(
      appendQuery("/v1/supplier/activity", query as Record<string, unknown>),
      "GET",
    );
  }

  async getSupplierDashboard(): Promise<SupplierDashboardResponse> {
    return this.request<SupplierDashboardResponse>("/v1/supplier/dashboard", "GET");
  }

  async getSupplierAnalyticsVelocity(): Promise<SupplierAnalyticsVelocityResponse> {
    return this.request<SupplierAnalyticsVelocityResponse>("/v1/supplier/analytics/velocity", "GET");
  }

  async getSupplierAnalyticsRevenue(): Promise<SupplierAnalyticsRevenueResponse> {
    return this.request<SupplierAnalyticsRevenueResponse>("/v1/supplier/analytics/revenue", "GET");
  }

  async getSupplierDemandToday(query?: {
    granularity?: "macro" | "regional" | "micro";
    warehouse_id?: string;
    region_id?: string;
    retailer_id?: string;
  }): Promise<SupplierDemandSummaryResponse> {
    const params = new URLSearchParams();
    if (query?.granularity) params.set("granularity", query.granularity);
    if (query?.warehouse_id) params.set("warehouse_id", query.warehouse_id);
    if (query?.region_id) params.set("region_id", query.region_id);
    if (query?.retailer_id) params.set("retailer_id", query.retailer_id);
    const suffix = params.toString() ? `?${params.toString()}` : "";
    return this.request<SupplierDemandSummaryResponse>(`/v1/supplier/analytics/demand/today${suffix}`, "GET");
  }

  async getSupplierDemandHistory(): Promise<SupplierDemandHistoryResponse> {
    return this.request<SupplierDemandHistoryResponse>("/v1/supplier/analytics/demand/history", "GET");
  }

  /** B4.4 STORE_POS flywheel DEMAND_SIGNAL feed (not planning multipliers). */
  async getSupplierDemandFlywheel(params?: {
    days?: number;
    limit?: number;
    sku?: string;
  }): Promise<import("@pegasusx/types").FlywheelDemandFeedResponse> {
    const q = new URLSearchParams();
    if (params?.days != null) q.set("days", String(params.days));
    if (params?.limit != null) q.set("limit", String(params.limit));
    if (params?.sku) q.set("sku", params.sku);
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request<import("@pegasusx/types").FlywheelDemandFeedResponse>(
      `/v1/supplier/analytics/demand/flywheel${suffix}`,
      "GET",
    );
  }

  async getDemandSignals(): Promise<{ signals: import("@pegasusx/types").DemandSignal[] }> {
    return this.request<{ signals: import("@pegasusx/types").DemandSignal[] }>("/v1/demand/signals", "GET");
  }

  async createDemandSignal(payload: import("@pegasusx/types").CreateSignalRequest): Promise<import("@pegasusx/types").DemandSignal> {
    return this.request<import("@pegasusx/types").DemandSignal>("/v1/demand/signals", "POST", { body: payload });
  }

  async importSupplierInventoryCSV(
    csvBody: string,
    idempotencyKey: string,
  ): Promise<SupplierInventoryImportResult> {
    return this.request<SupplierInventoryImportResult>("/v1/supplier/inventory/import", "POST", {
      rawBody: csvBody,
      idempotencyKey,
      headers: { "Content-Type": "text/csv" },
    });
  }

  async createSupplierImportSession(
    fileName: string,
    fileSizeBytes: number,
    idempotencyKey: string,
  ): Promise<SupplierImportSessionCreateResponse> {
    return this.request<SupplierImportSessionCreateResponse>("/v1/supplier/inventory/imports", "POST", {
      body: { file_name: fileName, file_size_bytes: fileSizeBytes },
      idempotencyKey,
    });
  }

  async ingestSupplierImportSession(
    sessionId: string,
    csvBody: string,
    idempotencyKey: string,
  ): Promise<SupplierImportIngestResponse> {
    return this.request<SupplierImportIngestResponse>(
      `/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}/ingest`,
      "POST",
      {
        rawBody: csvBody,
        idempotencyKey,
        headers: { "Content-Type": "text/csv" },
      },
    );
  }

  async getSupplierImportSession(sessionId: string): Promise<SupplierImportSession> {
    return this.request<SupplierImportSession>(
      `/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}`,
      "GET",
    );
  }

  async getSupplierImportMapping(sessionId: string): Promise<SupplierImportMappingResponse> {
    return this.request<SupplierImportMappingResponse>(
      `/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}/mapping`,
      "GET",
    );
  }

  async approveSupplierImportSession(
    sessionId: string,
    idempotencyKey: string,
  ): Promise<{ session_id: string; status: string }> {
    return this.request(`/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}/approve`, "POST", {
      idempotencyKey,
    });
  }

  async applySupplierImportSession(
    sessionId: string,
    idempotencyKey: string,
  ): Promise<SupplierImportApplyResponse> {
    return this.request<SupplierImportApplyResponse>(
      `/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}/apply`,
      "POST",
      { idempotencyKey },
    );
  }

  async updateSupplierInventory(
    request: Record<string, unknown>,
  ): Promise<void> {
    return this.request<void>("/v1/supplier/inventory", "PATCH", {
      body: request,
    });
  }

  async getSupplierInventoryAudit(): Promise<unknown> {
    return this.request<unknown>("/v1/supplier/inventory/audit", "GET");
  }

  async getSupplierManifests(): Promise<SupplierManifestsResponse> {
    return this.request<SupplierManifestsResponse>("/v1/supplier/manifests", "GET");
  }

  async getSupplierManifestDetail(manifestId: string): Promise<SupplierManifestDetail> {
    return this.request<SupplierManifestDetail>(`/v1/supplier/manifests/${encodeURIComponent(manifestId)}`, "GET");
  }

  async startSupplierManifestLoading(manifestId: string, idempotencyKey: string): Promise<{ status?: string; manifest_id?: string; state?: string }> {
    return this.request(`/v1/supplier/manifests/${encodeURIComponent(manifestId)}/start-loading`, "POST", {
      idempotencyKey,
    });
  }

  async injectSupplierManifestOrder(
    manifestId: string,
    request: SupplierManifestInjectOrderRequest,
    idempotencyKey: string,
  ): Promise<{ status?: string; manifest_id?: string; order_id?: string }> {
    return this.request(`/v1/supplier/manifests/${encodeURIComponent(manifestId)}/inject-order`, "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async sealSupplierManifest(manifestId: string, idempotencyKey: string): Promise<SupplierManifestSealResponse> {
    return this.request<SupplierManifestSealResponse>(`/v1/supplier/manifests/${encodeURIComponent(manifestId)}/seal`, "POST", {
      idempotencyKey,
    });
  }

  async sealAllSupplierManifests(idempotencyKey: string): Promise<{ sealed_count: number; errors?: any[] }> {
    return this.request<{ sealed_count: number; errors?: any[] }>("/v1/supplier/manifests/seal-all", "POST", {
      idempotencyKey,
    });
  }

  async sealAllPayloaderManifests(idempotencyKey: string): Promise<{ sealed_count: number; errors?: any[] }> {
    return this.request<{ sealed_count: number; errors?: any[] }>("/v1/payloader/manifests/seal-all", "POST", {
      idempotencyKey,
    });
  }

  async getSupplierManifestExceptions(params?: { escalated?: boolean }): Promise<SupplierManifestExceptionsResponse> {
    const query = new URLSearchParams();
    if (params?.escalated) query.set("escalated", "true");
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return this.request<SupplierManifestExceptionsResponse>(`/v1/supplier/manifest-exceptions${suffix}`, "GET");
  }

  async getSupplierSupplyLanes(): Promise<SupplierSupplyLanesResponse> {
    return this.request<SupplierSupplyLanesResponse>("/v1/supplier/supply-lanes", "GET");
  }

  async getSupplierSettings(): Promise<SupplierSettingsResponse> {
    return this.request<SupplierSettingsResponse>("/v1/supplier/settings", "GET");
  }

  async postSupplierRecommendReassign(
    request: RecommendReassignRequest,
    idempotencyKey?: string,
  ): Promise<RecommendReassignResponse> {
    return this.request<RecommendReassignResponse>("/v1/supplier/recommend-reassign", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async postSupplierApplyReassign(
    request: ApplyReassignRequest,
    idempotencyKey?: string,
  ): Promise<StatusResponse> {
    return this.request<StatusResponse>("/v1/supplier/reassign-order", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierExceptions(): Promise<SupplierExceptionsResponse> {
    return this.request<SupplierExceptionsResponse>("/v1/supplier/exceptions", "GET");
  }

  async getSupplierShopClosedActive(params?: { limit?: number; offset?: number }): Promise<ShopClosedActiveResponse> {
    const query = new URLSearchParams();
    if (params?.limit != null) query.set("limit", String(params.limit));
    if (params?.offset != null) query.set("offset", String(params.offset));
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return this.request<ShopClosedActiveResponse>(`/v1/supplier/shop-closed/active${suffix}`, "GET");
  }

  async resolveSupplierShopClosed(
    request: ShopClosedResolveRequest,
    idempotencyKey: string,
  ): Promise<{ status?: string; bypass_token?: string; queued?: boolean }> {
    return this.request("/v1/supplier/shop-closed/resolve", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** Driver: mark shop closed at stop (POST /v1/delivery/shop-closed). */
  async reportShopClosed(
    request: ShopClosedReportRequest,
    idempotencyKey: string,
  ): Promise<ShopClosedReportResponse> {
    return this.request<ShopClosedReportResponse>("/v1/delivery/shop-closed", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** Driver: unlock payment modes when physically at stop. */
  async proximityUnlock(
    request: ProximityUnlockRequest,
    idempotencyKey: string,
  ): Promise<ProximityUnlockResponse> {
    return this.request<ProximityUnlockResponse>("/v1/delivery/proximity-unlock", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** Driver: line-level partial offload (qty math enforced server-side). */
  async partialOffload(
    request: PartialOffloadRequest,
    idempotencyKey: string,
  ): Promise<PartialOffloadResponse> {
    return this.request<PartialOffloadResponse>("/v1/delivery/partial-offload", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** Driver: leave delivery on credit (requires proximity unlock or force_bypass_token). */
  async creditDelivery(
    request: CreditDeliveryRequest,
    idempotencyKey: string,
  ): Promise<{ status?: string; order_id?: string }> {
    return this.request("/v1/delivery/credit-delivery", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** Supplier compliance / fiscal audit dashboard (combined buckets). */
  async getComplianceDashboard(params?: { limit?: number }): Promise<ComplianceDashboardResponse> {
    const q = new URLSearchParams();
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request<ComplianceDashboardResponse>(`/v1/compliance/dashboard${suffix}`, "GET");
  }

  async getComplianceFiscalOpen(params?: { limit?: number }): Promise<{ data: ComplianceDashboardResponse["open_fiscal"]; count: number }> {
    const q = new URLSearchParams();
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/compliance/fiscal-open${suffix}`, "GET");
  }

  async getComplianceForceCompletes(params?: { limit?: number }): Promise<{ data: ComplianceDashboardResponse["force_completes"]; count: number }> {
    const q = new URLSearchParams();
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/compliance/force-completes${suffix}`, "GET");
  }

  async getComplianceClaimMismatches(params?: { limit?: number }): Promise<{ data: ComplianceDashboardResponse["claim_mismatches"]; count: number }> {
    const q = new URLSearchParams();
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/compliance/claim-mismatches${suffix}`, "GET");
  }

  async getComplianceCreditFreezes(params?: { limit?: number }): Promise<{ data: ComplianceDashboardResponse["credit_freezes"]; count: number }> {
    const q = new URLSearchParams();
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/compliance/credit-freezes${suffix}`, "GET");
  }

  /** JSON export of one or all compliance buckets. For CSV use download URL with format=csv. */
  async getComplianceExport(params?: {
    format?: "json" | "csv";
    bucket?: ComplianceExportBucket;
    limit?: number;
  }): Promise<unknown> {
    const q = new URLSearchParams();
    q.set("format", params?.format ?? "json");
    if (params?.bucket) q.set("bucket", params.bucket);
    if (params?.limit != null) q.set("limit", String(params.limit));
    return this.request(`/v1/compliance/export?${q.toString()}`, "GET");
  }

  /**
   * Quantity negotiation is product-disabled ecosystem-wide.
   * Backend returns empty pending list; clients must not surface propose UX.
   * Independent of claims / shop-closed / partial-offload.
   */
  async getSupplierNegotiationsPending(_params?: { limit?: number; offset?: number }): Promise<NegotiationPendingResponse> {
    return { data: [] };
  }

  async approveSupplierEarlyComplete(
    request: Record<string, unknown>,
    idempotencyKey?: string,
  ): Promise<void> {
    return this.request<void>("/v1/supplier/route/approve-early-complete", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /**
   * Quantity negotiation is product-disabled ecosystem-wide (410 on live API).
   * Do not call — use claims / shop-closed / missing-items for delivery exceptions.
   */
  async resolveSupplierNegotiation(
    _request: NegotiationResolveRequest,
    _idempotencyKey: string,
  ): Promise<NegotiationResolveResponse> {
    throw new Error("quantity_negotiation_disabled");
  }

  async issueSupplierPaymentBypass(
    request: PaymentBypassRequest,
    idempotencyKey: string,
  ): Promise<PaymentBypassResponse> {
    return this.request<PaymentBypassResponse>("/v1/supplier/orders/payment-bypass", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierEmpathyAdoption(): Promise<SupplierEmpathyAdoption> {
    return this.request<SupplierEmpathyAdoption>("/v1/supplier/empathy/adoption", "GET");
  }

  async postSupplierBroadcast(
    request: SupplierBroadcastRequest,
    idempotencyKey: string,
  ): Promise<SupplierBroadcastResponse> {
    return this.request<SupplierBroadcastResponse>("/v1/supplier/broadcast", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async triggerSupplierReplenishment(): Promise<SupplierReplenishmentTriggerResponse> {
    return this.request<SupplierReplenishmentTriggerResponse>("/v1/supplier/replenishment/trigger", "POST");
  }

  async getSupplierReplenishmentPolicies(): Promise<SupplierReplenishmentPolicy> {
    return this.request<SupplierReplenishmentPolicy>("/v1/supplier/replenishment/policies", "GET");
  }

  async getSupplierReplenishmentTraceability(): Promise<SupplierReplenishmentTraceabilityResponse> {
    return this.request<SupplierReplenishmentTraceabilityResponse>("/v1/supplier/replenishment/traceability", "GET");
  }

  async listReorderSuggestions(params?: {
    retailerId?: string;
    status?: string;
    sku?: string;
    /** Server-side demand source filter: STORE_POS | WHOLESALE_HISTORY */
    source?: string;
  }): Promise<ReorderSuggestionsListResponse> {
    const q = new URLSearchParams();
    if (params?.retailerId) q.set("retailerId", params.retailerId);
    if (params?.status) q.set("status", params.status);
    if (params?.sku) q.set("sku", params.sku);
    if (params?.source) q.set("source", params.source);
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request<ReorderSuggestionsListResponse>(`/v1/replenishment/suggestions${suffix}`, "GET");
  }

  async dismissReorderSuggestion(request: ReorderSuggestionDismissRequest): Promise<StatusResponse> {
    return this.request<StatusResponse>("/v1/replenishment/suggestions/dismiss", "POST", { body: request });
  }

  async createDraftFromReorderSuggestion(
    request: ReorderSuggestionCreateDraftRequest,
    idempotencyKey?: string,
  ): Promise<{ order_id: string; status?: string }> {
    return this.request("/v1/replenishment/suggestions/create-draft", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async bulkCreateDraftsFromReorderSuggestions(
    request: ReorderSuggestionBulkCreateDraftsRequest,
    idempotencyKey?: string,
  ): Promise<ReorderSuggestionBulkCreateDraftsResponse> {
    return this.request<ReorderSuggestionBulkCreateDraftsResponse>("/v1/replenishment/suggestions/create-drafts", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async listCashReconciliations(params?: { status?: string; limit?: number }): Promise<CashReconciliationsListResponse> {
    const q = new URLSearchParams();
    if (params?.status) q.set("status", params.status);
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request<CashReconciliationsListResponse>(`/v1/supplier/cash-reconciliations${suffix}`, "GET");
  }

  async acceptCashReconciliation(id: string, request?: CashReconciliationActionRequest): Promise<StatusResponse> {
    return this.request<StatusResponse>(`/v1/supplier/cash-reconciliations/${encodeURIComponent(id)}/accept`, "POST", {
      body: request ?? {},
    });
  }

  async writeOffCashReconciliation(id: string, request?: CashReconciliationActionRequest): Promise<StatusResponse> {
    return this.request<StatusResponse>(`/v1/supplier/cash-reconciliations/${encodeURIComponent(id)}/write-off`, "POST", {
      body: request ?? {},
    });
  }

  async listCreditNotes(params?: { status?: string; limit?: number }): Promise<CreditNotesListResponse> {
    const q = new URLSearchParams();
    if (params?.status) q.set("status", params.status);
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request<CreditNotesListResponse>(`/v1/supplier/credit-notes${suffix}`, "GET");
  }

  async createCreditNote(request: CreateCreditNoteRequest): Promise<CreditNoteRow> {
    return this.request<CreditNoteRow>("/v1/supplier/credit-notes", "POST", { body: request });
  }

  async issueCreditNote(id: string): Promise<StatusResponse> {
    return this.request<StatusResponse>(`/v1/supplier/credit-notes/${encodeURIComponent(id)}/issue`, "POST", {});
  }

  async getCreditNoteOrderLines(orderId: string): Promise<{ lines: Array<{ order_line_id: string; sku: string; qty: number; gross_minor: number }> }> {
    return this.request(`/v1/supplier/credit-notes/order-lines?order_id=${encodeURIComponent(orderId)}`, "GET");
  }

  async getSupplierCreditProgram(): Promise<SupplierCreditProgram> {
    return this.request<SupplierCreditProgram>("/v1/supplier/credit-program", "GET");
  }

  async enableSupplierCreditProgram(body: {
    warning_ack: boolean;
    warning_ack_at?: string;
    global_terms_days?: number;
    global_grace_days?: number;
    global_default_limit_minor?: number;
    timezone?: string;
  }): Promise<SupplierCreditProgram> {
    return this.request<SupplierCreditProgram>("/v1/supplier/credit-program", "POST", { body });
  }

  async patchSupplierCreditProgramDefaults(body: {
    global_terms_days?: number;
    global_grace_days?: number;
    global_default_limit_minor?: number;
    timezone?: string;
  }): Promise<SupplierCreditProgram> {
    return this.request<SupplierCreditProgram>("/v1/supplier/credit-program/defaults", "PATCH", { body });
  }

  async listSupplierCreditRelationships(): Promise<{ relationships: RetailerPaymentTerms[] }> {
    return this.request("/v1/supplier/credit-relationships", "GET");
  }

  async enableRetailerCreditRelationship(
    retailerId: string,
    body: {
      warning_ack: boolean;
      warning_ack_at?: string;
      terms_days?: number;
      grace_period_days?: number;
      credit_limit_minor?: number;
      use_global_defaults?: boolean;
    },
  ): Promise<RetailerPaymentTerms> {
    return this.request(
      `/v1/supplier/credit-relationships/${encodeURIComponent(retailerId)}/enable`,
      "POST",
      { body },
    );
  }

  async listRetailerCreditRelationships(): Promise<{ relationships: RetailerPaymentTerms[] }> {
    return this.request("/v1/retailer/credit-relationships", "GET");
  }

  async listRetailerArInvoices(params?: { status?: string; limit?: number }): Promise<{ invoices: ArInvoice[] }> {
    const q = new URLSearchParams();
    if (params?.status) q.set("status", params.status);
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/retailer/ar/invoices${suffix}`, "GET");
  }

  async listSupplierArInvoices(params?: { status?: string }): Promise<{ invoices: ArInvoice[] }> {
    const q = new URLSearchParams();
    if (params?.status) q.set("status", params.status);
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/supplier/ar/invoices${suffix}`, "GET");
  }

  async adminDisableCreditRelationship(
    supplierId: string,
    retailerId: string,
    body: { ticket_id: string; reason: string },
  ): Promise<StatusResponse> {
    return this.request(
      `/v1/admin/credit-relationships/${encodeURIComponent(supplierId)}/${encodeURIComponent(retailerId)}/disable`,
      "POST",
      { body },
    );
  }

  async resolveSupplierException(
    kind: string,
    id: string,
    body?: { note?: string; credit_note_id?: string },
  ): Promise<StatusResponse> {
    return this.request<StatusResponse>(
      `/v1/supplier/exceptions/${encodeURIComponent(kind)}/${encodeURIComponent(id)}/resolve`,
      "POST",
      { body: body ?? {} },
    );
  }

  async getNotificationPreferences(): Promise<{ preferences: Array<{ event_type: string; channel: string; enabled: boolean; quiet_from?: string; quiet_to?: string }> }> {
    return this.request("/v1/user/notification-preferences", "GET");
  }

  async patchNotificationPreferences(preferences: Array<{ event_type: string; channel: string; enabled: boolean; quiet_from?: string; quiet_to?: string }>): Promise<StatusResponse> {
    return this.request<StatusResponse>("/v1/user/notification-preferences", "PATCH", { body: { preferences } });
  }

  async listRoutePerformance(params?: { limit?: number }): Promise<{ routes: Array<Record<string, unknown>> }> {
    const q = new URLSearchParams();
    if (params?.limit != null) q.set("limit", String(params.limit));
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/supplier/route-performance${suffix}`, "GET");
  }

  async listTwinActiveRoutes(params?: {
    zoneH3?: string;
    delayedOnly?: boolean;
    shopClosedOnly?: boolean;
    driverId?: string;
  }): Promise<TwinOpsRouteView[]> {
    const q = new URLSearchParams();
    if (params?.zoneH3) q.set("zoneH3", params.zoneH3);
    if (params?.delayedOnly) q.set("delayedOnly", "true");
    if (params?.shopClosedOnly) q.set("shopClosedOnly", "true");
    if (params?.driverId) q.set("driverId", params.driverId);
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request<TwinOpsRouteView[]>(`/v1/twin/routes/active${suffix}`, "GET");
  }

  async getTwinRoute(routeId: string): Promise<TwinOpsRouteView> {
    return this.request<TwinOpsRouteView>(`/v1/twin/routes/${encodeURIComponent(routeId)}`, "GET");
  }

  async getTwinRouteInventory(routeId: string): Promise<TwinVehicleInventoryRow[]> {
    return this.request<TwinVehicleInventoryRow[]>(
      `/v1/twin/routes/${encodeURIComponent(routeId)}/inventory`,
      "GET",
    );
  }

  async getSupplierMEIONetworkSummary(): Promise<SupplierMEIONetworkSummary> {
    return this.request<SupplierMEIONetworkSummary>("/v1/supplier/meio/network-summary", "GET");
  }

  async listControlTowerZoneOverrides(): Promise<ControlTowerZoneOverridesResponse> {
    return this.request<ControlTowerZoneOverridesResponse>("/v1/supplier/control-tower/zone-overrides", "GET");
  }

  async createControlTowerZoneOverride(
    request: ControlTowerZoneOverrideRequest,
    idempotencyKey: string,
  ): Promise<ControlTowerZoneOverride> {
    return this.request<ControlTowerZoneOverride>("/v1/supplier/control-tower/zone-overrides", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async listPlaybookRuns(status?: string, exceptionId?: string): Promise<ControlTowerPlaybookRunsResponse> {
    const params = new URLSearchParams();
    if (status) params.set("status", status);
    if (exceptionId) params.set("exception_id", exceptionId);
    const q = params.toString();
    return this.request<ControlTowerPlaybookRunsResponse>(
      `/v1/control-tower/runs${q ? `?${q}` : ""}`,
      "GET",
    );
  }

  async listScoredExceptions(limit?: number): Promise<ScoredExceptionsResponse> {
    const params = new URLSearchParams();
    if (limit != null) params.set("limit", String(limit));
    const q = params.toString();
    return this.request<ScoredExceptionsResponse>(
      `/v1/control-tower/exceptions/scored${q ? `?${q}` : ""}`,
      "GET",
    );
  }

  async listPlaybooks(): Promise<ControlTowerPlaybooksResponse> {
    return this.request<ControlTowerPlaybooksResponse>("/v1/control-tower/playbooks", "GET");
  }

  async deactivatePlaybook(playbookId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(`/v1/control-tower/playbooks/${playbookId}/deactivate`, "POST");
  }

  async approvePlaybookRun(runId: string, idempotencyKey: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(`/v1/control-tower/runs/${runId}/approve`, "POST", {
      idempotencyKey,
    });
  }

  async skipPlaybookRun(runId: string, idempotencyKey: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(`/v1/control-tower/runs/${runId}/skip`, "POST", {
      idempotencyKey,
    });
  }

  async listRetailerSegments(): Promise<RetailerSegmentsResponse> {
    return this.request<RetailerSegmentsResponse>("/v1/supplier/segmentation/retailers", "GET");
  }

  async setRetailerSegment(retailerId: string, body: SetRetailerSegmentInput): Promise<{ status: string }> {
    return this.request<{ status: string }>(`/v1/supplier/segmentation/retailers/${encodeURIComponent(retailerId)}`, "PATCH", {
      body,
    });
  }

  async listSkuClasses(): Promise<SkuClassesResponse> {
    return this.request<SkuClassesResponse>("/v1/supplier/segmentation/sku-classes", "GET");
  }

  async setSkuClass(sku: string, body: SetSkuClassInput): Promise<{ status: string }> {
    return this.request<{ status: string }>(`/v1/supplier/segmentation/sku-classes/${encodeURIComponent(sku)}`, "PATCH", {
      body,
    });
  }

  async bootstrapSegmentation(): Promise<SegmentationBootstrapResult> {
    return this.request<SegmentationBootstrapResult>("/v1/supplier/segmentation/bootstrap", "POST");
  }

  async runPlanningScenario(request: PlanningScenarioInput, idempotencyKey: string): Promise<PlanningScenarioResult> {
    return this.request<PlanningScenarioResult>("/v1/supplier/planning/scenarios/run", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getPlanningSAndOP(): Promise<PlanningSAndOPSnapshot> {
    return this.request<PlanningSAndOPSnapshot>("/v1/supplier/planning/s-and-op", "GET");
  }

  async getSupplierKnowledgeGraph(): Promise<SupplierKnowledgeGraph> {
    return this.request<SupplierKnowledgeGraph>("/v1/supplier/knowledge-graph", "GET");
  }

  async invokeGovernedPlanningAgent(
    request: GovernedAgentInvocation,
    idempotencyKey: string,
  ): Promise<GovernedAgentInvocationResponse> {
    return this.request<GovernedAgentInvocationResponse>("/v1/supplier/planning/agent/invoke", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSeasonalOverrides(): Promise<SeasonalTemplatesResponse> {
    return this.request<SeasonalTemplatesResponse>("/v1/supplier/planning/seasonal-overrides", "GET");
  }

  async createSeasonalOverride(
    request: SeasonalOverrideInput,
    idempotencyKey: string,
  ): Promise<SeasonalOverrideRow> {
    return this.request<SeasonalOverrideRow>("/v1/supplier/planning/seasonal-overrides", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async ingestPlanningSignal(
    request: PlanningSignalIngestInput,
    idempotencyKey: string,
  ): Promise<{ signal_id: string; status: string }> {
    return this.request<{ signal_id: string; status: string }>("/v1/supplier/planning/signals/ingest", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getPlanningSignalStatus(): Promise<PlanningSignalIngestStatus> {
    return this.request<PlanningSignalIngestStatus>("/v1/supplier/planning/signals/status", "GET");
  }

  async checkPlanningSparsity(retailerId: string): Promise<SparsityGateResult> {
    return this.request<SparsityGateResult>(`/v1/supplier/planning/sparsity/${encodeURIComponent(retailerId)}`, "GET");
  }

  async simulatePromotionPandL(
    request: PromoSimulateInput,
    idempotencyKey: string,
  ): Promise<PromoSimulateResult> {
    return this.request<PromoSimulateResult>("/v1/supplier/planning/promotions/simulate", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getPromotionPerformance(promotionId: string): Promise<PromoPerformanceResult> {
    return this.request<PromoPerformanceResult>(
      `/v1/supplier/planning/promotions/${encodeURIComponent(promotionId)}/performance`,
      "GET",
    );
  }

  async getSupplierFleetOrders(): Promise<SupplierFleetOrderRow[]> {
    return this.request<SupplierFleetOrderRow[]>("/v1/supplier/fleet/orders", "GET");
  }

  async getSupplierFleetLiveMap(): Promise<SupplierFleetLiveMapResponse> {
    return this.request<SupplierFleetLiveMapResponse>("/v1/supplier/fleet/live-map", "GET");
  }

  async getSupplierEarnings(): Promise<{
    currency: string;
    today_minor: number;
    week_minor: number;
    month_minor: number;
    authority_source?: string;
    authoritative?: boolean;
    updated_at: string;
  }> {
    return this.request("/v1/supplier/earnings", "GET");
  }

  async getSupplierAIRecommendations(query: SupplierAIRecommendationsQuery = {}): Promise<SupplierAIRecommendationsResponse> {
    return this.request<SupplierAIRecommendationsResponse>(appendQuery("/v1/supplier/ai/recommendations", query as Record<string, unknown>), "GET");
  }

  async recordSupplierAIRecommendationDecision(
    request: SupplierAIRecommendationDecisionRequest,
    idempotencyKey: string,
  ): Promise<SupplierAIRecommendationDecisionResponse> {
    return this.request<SupplierAIRecommendationDecisionResponse>("/v1/supplier/ai/recommendations", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async updateSupplierPricingRule(
    request: SupplierPricingRuleUpdateRequest,
    idempotencyKey: string,
  ): Promise<SupplierPricingRule> {
    return this.request<SupplierPricingRule>("/v1/supplier/pricing/rules", "PATCH", {
      body: request,
      idempotencyKey,
    });
  }

  async getRetailerSuppliers(): Promise<RetailerSupplierPreference[]> {
    return this.request<RetailerSupplierPreference[]>("/v1/retailer/suppliers", "GET");
  }

  async getRetailerPricingRule(): Promise<RetailerPricingRuleResponse> {
    return this.request<RetailerPricingRuleResponse>("/v1/retailer/pricing/rules", "GET");
  }

  async getRetailerProfile(): Promise<RetailerProfileResponse> {
    return this.request<RetailerProfileResponse>("/v1/retailer/profile", "GET");
  }

  async updateRetailerProfile(
    request: RetailerProfileUpdateRequest,
    idempotencyKey: string,
  ): Promise<RetailerProfileResponse> {
    return this.request<RetailerProfileResponse>("/v1/retailer/profile", "PUT", {
      body: request,
      idempotencyKey,
    });
  }

  async getRetailerTracking(): Promise<RetailerTrackingResponse> {
    return this.request<RetailerTrackingResponse>("/v1/retailer/tracking", "GET");
  }

  async getRetailerActiveFulfillment(): Promise<RetailerActiveFulfillmentResponse> {
    return this.request<RetailerActiveFulfillmentResponse>("/v1/retailer/active-fulfillment", "GET");
  }

  async getRetailerPendingPayments(): Promise<RetailerPendingPaymentsResponse> {
    return this.request<RetailerPendingPaymentsResponse>("/v1/retailer/pending-payments", "GET");
  }

  async getRetailerAIPredictions(limit?: number): Promise<RetailerAIPredictionsResponse> {
    const path = limit && limit > 0 ? appendQuery("/v1/retailer/ai/predictions", { limit }) : "/v1/retailer/ai/predictions";
    return this.request<RetailerAIPredictionsResponse>(path, "GET");
  }

  async confirmRetailerAIOrder(
    request: ConfirmAIOrderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/retailer/orders/confirm-ai", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async rejectRetailerAIOrder(
    request: RejectAIOrderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/retailer/orders/reject-ai", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async editRetailerPreorder(
    request: EditPreorderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/orders/edit-preorder", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async confirmRetailerPreorder(
    request: ConfirmPreorderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/orders/confirm-preorder", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async acceptDeliveryProposal(
    request: AcceptDeliveryProposalRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/orders/accept-delivery-proposal", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async rejectDeliveryProposal(
    request: RejectDeliveryProposalRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/orders/reject-delivery-proposal", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async rejectRetailerPreorder(
    request: RejectPreorderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/orders/reject-preorder", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async postWarehouseProposeDelivery(
    orderId: string,
    request: ProposeDeliveryDateRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>(
      appendQuery(`/v1/warehouse/ops/preorders/${orderId}/propose-delivery`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getWarehouseDemandForecast(query: { warehouse_id?: string; start_date?: string; days?: number } = {}): Promise<WarehouseDemandForecastResponse> {
    return this.request<WarehouseDemandForecastResponse>(appendQuery("/v1/warehouse/demand/forecast", query as Record<string, unknown>), "GET");
  }

  async getWarehouseOpsDashboard(query: { warehouse_id?: string } = {}): Promise<WarehouseOpsDashboardResponse> {
    return this.request<WarehouseOpsDashboardResponse>(appendQuery("/v1/warehouse/ops/dashboard", query as Record<string, unknown>), "GET");
  }

  async getWarehouseFleetLiveMap(query: { warehouse_id?: string } = {}): Promise<WarehouseFleetLiveMapResponse> {
    return this.request<WarehouseFleetLiveMapResponse>(appendQuery("/v1/warehouse/ops/fleet/live-map", query as Record<string, unknown>), "GET");
  }

  async getWarehousePulse(query: { warehouse_id?: string } = {}): Promise<PulseResponse> {
    return this.request<PulseResponse>(appendQuery("/v1/warehouse/ops/pulse", query as Record<string, unknown>), "GET");
  }

  async getSupplierPulse(): Promise<PulseResponse> {
    return this.request<PulseResponse>("/v1/supplier/pulse", "GET");
  }

  async getRetailerPulse(): Promise<PulseResponse> {
    return this.request<PulseResponse>("/v1/retailer/pulse", "GET");
  }

  async getStockCountVersion(query: {
    location_id: string;
    stock_bin?: string;
  }): Promise<RetailerStockCountVersionResponse> {
    return this.request<RetailerStockCountVersionResponse>(
      appendQuery("/v1/retailer/stock/counts/version", query as Record<string, unknown>),
      "GET",
    );
  }

  async commitStockCount(
    request: RetailerStockCountCommitRequest,
    options: { idempotencyKey?: string } = {},
  ): Promise<RetailerStockCountCommitResponse> {
    return this.request<RetailerStockCountCommitResponse>(
      "/v1/retailer/stock/counts/commit",
      "POST",
      {
        body: request,
        idempotencyKey: options.idempotencyKey,
      },
    );
  }

  async getFactoryPulse(): Promise<PulseResponse> {
    return this.request<PulseResponse>("/v1/factory/pulse", "GET");
  }

  async getDriverPulse(): Promise<PulseResponse> {
    return this.request<PulseResponse>("/v1/driver/pulse", "GET");
  }

  async getPayloaderPulse(): Promise<PulseResponse> {
    return this.request<PulseResponse>("/v1/payloader/pulse", "GET");
  }

  async getWarehouseDispatchRuns(query: { warehouse_id?: string } = {}): Promise<WarehouseDispatchRunsResponse> {
    return this.request<WarehouseDispatchRunsResponse>(appendQuery("/v1/warehouse/ops/dispatch/runs", query as Record<string, unknown>), "GET");
  }

  async getWarehouseDispatchRun(runId: string, query: { warehouse_id?: string } = {}): Promise<WarehouseDispatchRun> {
    return this.request<WarehouseDispatchRun>(appendQuery(`/v1/warehouse/ops/dispatch/runs/${runId}`, query as Record<string, unknown>), "GET");
  }

  async getWarehouseOpsBoard(query: { warehouse_id?: string; date?: string } = {}): Promise<WarehouseOpsBoardResponse> {
    return this.request<WarehouseOpsBoardResponse>(appendQuery("/v1/warehouse/ops/board", query as Record<string, unknown>), "GET");
  }

  async getWarehouseOpsExceptions(query: { warehouse_id?: string } = {}): Promise<WarehouseOpsExceptionsResponse> {
    return this.request<WarehouseOpsExceptionsResponse>(appendQuery("/v1/warehouse/ops/exceptions", query as Record<string, unknown>), "GET");
  }

  async getWarehouseInventory(query: { warehouse_id?: string } = {}): Promise<WarehouseInventoryResponse> {
    return this.request<WarehouseInventoryResponse>(appendQuery("/v1/warehouse/ops/inventory", query as Record<string, unknown>), "GET");
  }

  async getWarehouseOpsSettings(query: { warehouse_id?: string } = {}): Promise<WarehouseOpsSettings> {
    return this.request<WarehouseOpsSettings>(appendQuery("/v1/warehouse/ops/settings", query as Record<string, unknown>), "GET");
  }

  async patchWarehouseOpsSettings(
    request: WarehouseOpsSettingsPatchRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseOpsSettings> {
    return this.request<WarehouseOpsSettings>(
      appendQuery("/v1/warehouse/ops/settings", query as Record<string, unknown>),
      "PATCH",
      { body: request, idempotencyKey },
    );
  }

  async postCheckoutPreview(body: Record<string, unknown>): Promise<CheckoutPreviewResponse> {
    return this.request<CheckoutPreviewResponse>("/v1/checkout/preview", "POST", { body });
  }

  async getWarehouseOrders(query: { warehouse_id?: string; state?: string } = {}): Promise<WarehouseOrdersResponse> {
    return this.request<WarehouseOrdersResponse>(appendQuery("/v1/warehouse/ops/orders", query as Record<string, unknown>), "GET");
  }

  async getWarehouseOrder(orderId: string, query: { warehouse_id?: string } = {}): Promise<WarehouseOrderDetail> {
    return this.request<WarehouseOrderDetail>(
      appendQuery(`/v1/warehouse/ops/orders/${orderId}`, query as Record<string, unknown>),
      "GET",
    );
  }

  async getOrderTimeline(orderId: string): Promise<OrderTimelineResponse> {
    return this.request<OrderTimelineResponse>(`/v1/order/${orderId}/timeline`, "GET");
  }

  async getWarehousePreorders(query: { limit?: number; offset?: number; warehouse_id?: string } = {}): Promise<WarehousePreordersResponse> {
    return this.request<WarehousePreordersResponse>(
      appendQuery("/v1/warehouse/ops/preorders", query as Record<string, unknown>),
      "GET",
    );
  }

  async postWarehouseOrderProposeDelivery(
    orderId: string,
    request: ProposeDeliveryDateRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>(
      appendQuery(`/v1/warehouse/ops/orders/${orderId}/propose-delivery`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehousePreorderEdit(
    orderId: string,
    request: WarehousePreorderEditRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>(
      appendQuery(`/v1/warehouse/ops/preorders/${orderId}/edit`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getWarehouseStockCommitments(query: { sku_id?: string } = {}): Promise<{ items: unknown[]; skus: unknown[] }> {
    return this.request<{ items: unknown[]; skus: unknown[] }>(
      appendQuery("/v1/warehouse/ops/stock-commitments", query as Record<string, unknown>),
      "GET",
    );
  }

  async getWarehouseFleetVehicles(query: { warehouse_id?: string } = {}): Promise<WarehouseFleetVehicleListResponse> {
    return this.request<WarehouseFleetVehicleListResponse>(appendQuery("/v1/warehouse/ops/vehicles", query as Record<string, unknown>), "GET");
  }

  async getWarehouseFleetVehicle(vehicleId: string, query: { warehouse_id?: string } = {}): Promise<WarehouseFleetVehicleDetailResponse> {
    return this.request<WarehouseFleetVehicleDetailResponse>(
      appendQuery(`/v1/warehouse/ops/vehicles/${vehicleId}`, query as Record<string, unknown>),
      "GET",
    );
  }

  async previewWarehouseDispatch(
    query: { warehouse_id?: string } = {},
    body: { order_ids?: string[] } = {},
  ): Promise<WarehouseDispatchPreview> {
    return this.request<WarehouseDispatchPreview>(
      appendQuery("/v1/warehouse/ops/dispatch/preview", query as Record<string, unknown>),
      "POST",
      { body },
    );
  }

  async executeWarehouseDispatch(
    request: WarehouseDispatchExecuteRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<WarehouseDispatchExecuteResponse> {
    return this.request<WarehouseDispatchExecuteResponse>(appendQuery("/v1/warehouse/ops/dispatch/execute", query as Record<string, unknown>), "POST", {
      body: request,
      idempotencyKey,
    });
  }

  // ── Rescue Operations ──
  async postWarehouseProposeRescue(
    request: {
      broken_driver_id: string;
      rescue_driver_id: string;
      rescue_id: string;
    },
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      appendQuery(`/v1/warehouse/ops/dispatch/rescue/propose`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehousePreviewRescue(
    request: {
      broken_driver_id: string;
    },
    query: { warehouse_id?: string } = {},
  ): Promise<{
    broken_driver_id: string;
    pending_volume_vu: number;
    rescue_options: {
      driver_id: string;
      name: string;
      license_plate: string;
      truck_status: string;
      effective_capacity_vu: number;
      is_capacity_exceeded: boolean;
    }[];
  }> {
    return this.request(
      appendQuery(`/v1/warehouse/ops/dispatch/rescue/preview`, query as Record<string, unknown>),
      "POST",
      { body: request }
    );
  }

  async setupWarehouse(
    request: Record<string, unknown>,
    query: { warehouse_id?: string } = {},
  ): Promise<void> {
    return this.request<void>(appendQuery("/v1/warehouse/setup", query as Record<string, unknown>), "POST", {
      body: request,
    });
  }

  async getWarehouseSupplyRequests(query: { warehouse_id?: string } = {}): Promise<WarehouseSupplyRequestsResponse> {
    return this.request<WarehouseSupplyRequestsResponse>(appendQuery("/v1/warehouse/supply-requests", query as Record<string, unknown>), "GET");
  }

  async getWarehouseSupplyRequest(
    id: string,
    query: { warehouse_id?: string } = {},
  ): Promise<WarehouseSupplyRequest> {
    return this.request<WarehouseSupplyRequest>(appendQuery(`/v1/warehouse/supply-requests/${id}`, query as Record<string, unknown>), "GET");
  }

  async openWarehouseSupplyRequest(
    query: { warehouse_id?: string; start_date?: string; days?: number; requested_by?: string } = {},
  ): Promise<WarehouseSupplyRequest> {
    return this.request<WarehouseSupplyRequest>(appendQuery("/v1/warehouse/supply-requests", query as Record<string, unknown>), "POST");
  }

  async getWarehouseDispatchLocks(query: { warehouse_id?: string } = {}): Promise<WarehouseDispatchLocksResponse> {
    return this.request<WarehouseDispatchLocksResponse>(appendQuery("/v1/warehouse/dispatch-locks", query as Record<string, unknown>), "GET");
  }

  async acquireWarehouseDispatchLock(
    request: WarehouseDispatchLockAcquireRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseDispatchLock> {
    return this.request<WarehouseDispatchLock>(appendQuery("/v1/warehouse/dispatch-lock", query as Record<string, unknown>), "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async releaseWarehouseDispatchLock(
    query: { warehouse_id?: string; lock_id: string },
    idempotencyKey?: string,
  ): Promise<WarehouseDispatchLockReleaseResponse> {
    return this.request<WarehouseDispatchLockReleaseResponse>(appendQuery("/v1/warehouse/dispatch-lock", query as Record<string, unknown>), "DELETE", {
      idempotencyKey,
    });
  }

  async postWarehouseEmergencyTransfer(
    request: WarehouseEmergencyTransferRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<WarehouseTransferMutationResponse> {
    return this.request<WarehouseTransferMutationResponse>(
      appendQuery("/v1/warehouse/transfers/emergency", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehouseForceReceive(
    request: WarehouseForceReceiveRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<WarehouseTransferMutationResponse> {
    return this.request<WarehouseTransferMutationResponse>(
      appendQuery("/v1/warehouse/transfers/force-receive", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehouseReceiveTransfer(
    transferId: string,
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<WarehouseTransferMutationResponse> {
    return this.request<WarehouseTransferMutationResponse>(
      appendQuery(`/v1/warehouse/transfers/${encodeURIComponent(transferId)}/receive`, query as Record<string, unknown>),
      "POST",
      { idempotencyKey },
    );
  }

  async getWarehouseReplenishmentInsights(
    query: { warehouse_id?: string; limit?: number } = {},
  ): Promise<WarehouseReplenishmentInsightsResponse> {
    return this.request<WarehouseReplenishmentInsightsResponse>(
      appendQuery("/v1/warehouse/replenishment/insights", query as Record<string, unknown>),
      "GET",
    );
  }

  async postWarehouseReplenishmentInsightAction(
    insightId: string,
    action: "approve" | "dismiss",
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseReplenishmentInsightActionResponse> {
    return this.request<WarehouseReplenishmentInsightActionResponse>(
      appendQuery(`/v1/warehouse/replenishment/insights/${insightId}/${action}`, query as Record<string, unknown>),
      "POST",
      { idempotencyKey },
    );
  }

  async getWarehouseDispatchSettings(
    query: { warehouse_id?: string } = {},
  ): Promise<WarehouseDispatchSettingsResponse> {
    return this.request<WarehouseDispatchSettingsResponse>(
      appendQuery("/v1/warehouse/ops/dispatch/settings", query as Record<string, unknown>),
      "GET",
    );
  }

  async patchWarehouseDispatchSettings(
    request: WarehouseDispatchSettingsPatchRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      appendQuery("/v1/warehouse/ops/dispatch/settings", query as Record<string, unknown>),
      "PATCH",
      { body: request, idempotencyKey },
    );
  }

  async getWarehouseOpsFinancials(
    query: { warehouse_id?: string; period?: string } = {},
  ): Promise<WarehouseOpsFinancialsResponse> {
    return this.request<WarehouseOpsFinancialsResponse>(
      appendQuery("/v1/warehouse/ops/financials", query as Record<string, unknown>),
      "GET",
    );
  }

  async getWarehouseBroadcastTemplates(
    query: { warehouse_id?: string } = {},
  ): Promise<BroadcastTemplatesResponse> {
    return this.request<BroadcastTemplatesResponse>(
      appendQuery("/v1/warehouse/ops/broadcast/templates", query as Record<string, unknown>),
      "GET",
    );
  }

  async createWarehouseBroadcastTemplate(
    request: { title: string; body: string; default_role?: string; category?: string },
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<BroadcastTemplate> {
    return this.request<BroadcastTemplate>(
      appendQuery("/v1/warehouse/ops/broadcast/templates", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async deleteWarehouseBroadcastTemplate(
    templateId: string,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<{ status: string; template_id: string }> {
    return this.request<{ status: string; template_id: string }>(
      appendQuery(`/v1/warehouse/ops/broadcast/templates/${templateId}`, query as Record<string, unknown>),
      "DELETE",
      { body: {}, idempotencyKey },
    );
  }

  async postWarehouseBroadcast(
    request: { title: string; body: string; role?: string },
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<{ status: string; warehouse_id: string; supplier_id: string }> {
    return this.request<{ status: string; warehouse_id: string; supplier_id: string }>(
      appendQuery("/v1/warehouse/ops/broadcast", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async previewWarehouseRetailerPriceOverride(
    request: {
      retailer_id?: string;
      product_id?: string;
      sku_id?: string;
      proposed_price: number;
    },
    query: { warehouse_id?: string } = {},
  ): Promise<RetailerOverridePreview> {
    return this.request<RetailerOverridePreview>(
      appendQuery("/v1/warehouse/ops/pricing/retailer-overrides/preview", query as Record<string, unknown>),
      "POST",
      { body: request },
    );
  }

  async postWarehouseOrderDelay(
    orderId: string,
    request: WarehouseOrderMutationRequest = {},
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseOrderMutationResponse> {
    return this.request<WarehouseOrderMutationResponse>(
      appendQuery(`/v1/warehouse/ops/orders/${orderId}/delay`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehouseOrderReject(
    orderId: string,
    request: WarehouseOrderMutationRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseOrderMutationResponse> {
    return this.request<WarehouseOrderMutationResponse>(
      appendQuery(`/v1/warehouse/ops/orders/${orderId}/reject`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehousePreorderReject(
    orderId: string,
    request: WarehouseOrderMutationRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseOrderMutationResponse> {
    return this.request<WarehouseOrderMutationResponse>(
      appendQuery(`/v1/warehouse/ops/preorders/${orderId}/reject`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehouseOrderOverflow(
    orderId: string,
    request: WarehouseOrderMutationRequest = {},
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseOrderMutationResponse> {
    return this.request<WarehouseOrderMutationResponse>(
      appendQuery(`/v1/warehouse/ops/orders/${orderId}/overflow`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getPaymentLedger(query: PaymentLedgerQuery = {}): Promise<PaymentLedgerResponse> {
    return this.request<PaymentLedgerResponse>(appendQuery("/v1/payment/ledger", query as Record<string, unknown>), "GET");
  }

  async getPaymentSettlementAuthority(query: SettlementAuthorityQuery = {}): Promise<SettlementAuthorityResponse> {
    return this.request<SettlementAuthorityResponse>(appendQuery("/v1/payment/settlement/authority", query as Record<string, unknown>), "GET");
  }

  async getPaymentReconciliationMismatches(query: ReconciliationMismatchQuery = {}): Promise<ReconciliationMismatchResponse> {
    return this.request<ReconciliationMismatchResponse>(appendQuery("/v1/payment/reconciliation/mismatches", query as Record<string, unknown>), "GET");
  }

  async recordPaymentChargeback(
    request: PaymentChargebackRequest,
    idempotencyKey: string,
  ): Promise<PaymentChargebackResponse> {
    return this.request<PaymentChargebackResponse>("/v1/payment/chargeback", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async recordPaymentChargebackReversal(
    request: PaymentChargebackReversalRequest,
    idempotencyKey: string,
  ): Promise<PaymentChargebackReversalResponse> {
    return this.request<PaymentChargebackReversalResponse>("/v1/payment/chargeback/reversal", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** GET /v1/supplier/claims — supplier-scoped logistics claims queue. */
  async listSupplierClaims(query: {
    status?: string;
    limit?: number;
  } = {}): Promise<SupplierClaimsListResponse> {
    return this.request<SupplierClaimsListResponse>(
      appendQuery("/v1/supplier/claims", query as Record<string, unknown>),
      "GET",
    );
  }

  /** POST /v1/claims/{claimId}/approve */
  async approveClaim(
    claimId: string,
    request: ApproveClaimRequest = {},
    idempotencyKey?: string,
  ): Promise<ApproveClaimResponse> {
    return this.request<ApproveClaimResponse>(`/v1/claims/${encodeURIComponent(claimId)}/approve`, "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** POST /v1/claims/{claimId}/reject */
  async rejectClaim(
    claimId: string,
    request: RejectClaimRequest = {},
    idempotencyKey?: string,
  ): Promise<Claim> {
    return this.request<Claim>(`/v1/claims/${encodeURIComponent(claimId)}/reject`, "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** GET /v1/supplier/claim-chargebacks — claim-originated CHARGEBACK_RECORDED ledger rows. */
  async listSupplierClaimChargebacks(
    query: ClaimChargebacksQuery = {},
  ): Promise<ClaimChargebacksResponse> {
    return this.request<ClaimChargebacksResponse>(
      appendQuery("/v1/supplier/claim-chargebacks", query as Record<string, unknown>),
      "GET",
    );
  }

  /** GET /v1/supplier/return-policy */
  async getSupplierReturnPolicy(): Promise<SupplierReturnPolicy> {
    return this.request<SupplierReturnPolicy>("/v1/supplier/return-policy", "GET");
  }

  /** PUT /v1/supplier/return-policy */
  async putSupplierReturnPolicy(
    body: SupplierReturnPolicy,
    idempotencyKey?: string,
  ): Promise<SupplierReturnPolicy> {
    return this.request<SupplierReturnPolicy>("/v1/supplier/return-policy", "PUT", {
      body,
      idempotencyKey,
    });
  }

  /** GET /v1/warehouse/return-policy */
  async getWarehouseReturnPolicy(query: {
    warehouse_id?: string;
    supplier_id?: string;
  } = {}): Promise<WarehouseReturnPolicy> {
    return this.request<WarehouseReturnPolicy>(
      appendQuery("/v1/warehouse/return-policy", query as Record<string, unknown>),
      "GET",
    );
  }

  /** PUT /v1/warehouse/return-policy */
  async putWarehouseReturnPolicy(
    body: WarehouseReturnPolicy,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseReturnPolicy> {
    return this.request<WarehouseReturnPolicy>(
      appendQuery("/v1/warehouse/return-policy", query as Record<string, unknown>),
      "PUT",
      { body, idempotencyKey },
    );
  }

  private async request<TResponse>(
    path: string,
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    options: RequestOptions = {},
  ): Promise<TResponse> {
    const fetchImpl = this.config.fetchImpl ?? fetch;
    const requiresAuth = options.requiresAuth ?? true;
    const headers = this.buildHeaders(options.headers, requiresAuth, options.idempotencyKey, options.rawBody !== undefined);

    const init: RequestInit = {
      method,
      headers,
      credentials: "include",
    };
    if (options.rawBody !== undefined) {
      init.body = options.rawBody;
    } else if (options.body !== undefined) {
      init.body = JSON.stringify(options.body);
    }

    const response = await fetchImpl(this.resolveURL(path), init);
    const payload = await parseResponsePayload(response);
    if (!response.ok) {
      const message = extractErrorMessage(response.status, payload);
      throw new ApiError(message, response.status, payload);
    }

    return payload as TResponse;
  }

  // ── Fleet Telemetry & Capacity Command Center ─────────────────────────────────
  async getFleetDispatchOverview(
    role: "supplier" | "warehouse",
    query: { partner_id?: string; status_filter?: string; search_query?: string } = {},
  ): Promise<import("@pegasusx/types").FleetDispatchOverview> {
    return this.request<import("@pegasusx/types").FleetDispatchOverview>(
      appendQuery(`/v1/${role}/dispatch/tracking`, query as Record<string, unknown>),
      "GET"
    );
  }

  async getVehicleCapacity(vehicleId: string): Promise<import("@pegasusx/types").VehicleCapacityMetrics> {
    return this.request<import("@pegasusx/types").VehicleCapacityMetrics>(`/v1/payload/capacity/${vehicleId}`, "GET");
  }

  async getRouteTelemetry(routeId: string): Promise<import("@pegasusx/types").RouteTelemetryDetails> {
    return this.request<import("@pegasusx/types").RouteTelemetryDetails>(`/v1/driver/telemetry/${routeId}`, "GET");
  }

  async getShipmentProofPhotos(shipmentId: string): Promise<import("@pegasusx/types").PoDPhotoReport[]> {
    return this.request<import("@pegasusx/types").PoDPhotoReport[]>(`/v1/delivery/proof-photos/${shipmentId}`, "GET");
  }

  private buildHeaders(extra: HeadersInit | undefined, requiresAuth: boolean, idempotencyKey: string | undefined, hasRawBody: boolean): Headers {

    const headers = new Headers(extra);
    if (!hasRawBody) {
      headers.set("Content-Type", "application/json");
    }

    if (requiresAuth && this.config.getAuthToken) {
      const token = this.config.getAuthToken();
      if (token) {
        headers.set("Authorization", `Bearer ${token}`);
      }
    }
    if (idempotencyKey) {
      headers.set("Idempotency-Key", idempotencyKey);
    }

    return headers;
  }

  private resolveURL(path: string): string {
    if (/^https?:\/\//i.test(path)) {
      return path;
    }
    const base = this.config.baseUrl.endsWith("/") ? this.config.baseUrl : `${this.config.baseUrl}/`;
    const normalizedPath = path.startsWith("/") ? path.slice(1) : path;
    return new URL(normalizedPath, base).toString();
  }
}

async function parseResponsePayload(response: Response): Promise<unknown> {
  if (response.status === 204) {
    return undefined;
  }

  const contentType = response.headers.get("content-type") || "";
  if (contentType.toLowerCase().includes("application/json")) {
    try {
      return await response.json();
    } catch {
      return undefined;
    }
  }

  try {
    const text = await response.text();
    return text.length > 0 ? text : undefined;
  } catch {
    return undefined;
  }
}

function extractErrorMessage(status: number, payload: unknown): string {
  if (payload && typeof payload === "object" && "error" in payload) {
    const candidate = (payload as { error?: unknown }).error;
    if (typeof candidate === "string" && candidate.length > 0) {
      return candidate;
    }
  }
  if (typeof payload === "string" && payload.length > 0) {
    return payload;
  }
  return `request failed (${status})`;
}

function appendQuery(path: string, query: Record<string, unknown>): string {
  const entries = Object.entries(query).filter(([, value]) => value !== undefined && value !== null && `${value}`.trim() !== "");
  if (entries.length === 0) {
    return path;
  }
  const params = new URLSearchParams();
  for (const [key, value] of entries) {
    params.set(key, String(value));
  }
  return `${path}?${params.toString()}`;
}
