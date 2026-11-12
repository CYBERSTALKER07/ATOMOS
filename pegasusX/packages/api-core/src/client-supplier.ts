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
  CreateRetailerReceiveSessionRequest,
  RetailerReceiveSession,
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
  FxRateRow,
  FxRatesListResponse,
  FxRateUpsertInput,
  FxRateUpsertResponse,
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
  SupplierServicePolicy,
  PromiseEvaluationRequest,
  PromiseEvaluationResult,
  OrderServicePromiseSnapshot,
  ProximityUnlockRequest,
  ProximityUnlockResponse,
  PartialOffloadRequest,
  PartialOffloadResponse,
  CreditDeliveryRequest,
  SupplierCreditProgram,
  RetailerPaymentTerms,
  ArInvoice,
  AgingSummaryResponse,
  DelinquencyLockStatus,
  RetailerPayInvoiceRequest,
  WriteOffRequest,
  CashBagSummaryResponse,
  CashBagTurnInRequest,
  CashReconciliation,
  PartnerDeadLetterAttempt,
  PartnerEdiDocument,
  PartnerExportJob,
  PartnerIssuedKey,
  PartnerCoaMap,
  PartnerSftpConfig,
  PartnerAs2Config,
  PartnerWebhookSubscription,
  PartnerApiKeyMeta,
  PartnerWebhookCreated,
  PartnerAvailabilityRow,
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
  SupplierReplenishmentPolicyPatch,
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
  LaborZoneCapacity,
  LaborDriverScore,
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
  PlanningScenarioCloneInput,
  PlanningScenarioCompareResult,
  PlanningScenarioListResponse,
  PlanningSignalIngestStatus,
  PlanningSAndOPSnapshot,
  SupplierKnowledgeGraph,
  GovernedAgentInvocation,
  GovernedAgentInvocationResponse,
  SeasonalOverrideInput,
  SeasonalOverrideRow,
  SeasonalEstimateResult,
  SeasonalTemplatesResponse,
  SupplierCRMListResponse,
  SupplierCRMRetailerDetail,
  SupplyRequestQCResponse,
  SupplyRequestQCRequest,
  NetworkModeResponse,
  NetworkModeUpdateRequest,
  NetworkModeUpdateResponse,
  PullMatrixResponse,
  KillSwitchRequest,
  KillSwitchResponse,
  LoyaltyProgram,
  LoyaltyTierView,
  LoyaltyLedgerResponse,
  EntityResolutionResolveRequest,
  EntityResolutionResolveResponse,
  EntityResolutionExplainRequest,
  EntityResolutionExplainResponse,
  PredictivePushResponse,
  PayoutRailInfo,
  PayoutBatch,
  PayoutBatchListResponse,
  PayoutBatchGenerateRequest,
  PayoutBatchGenerateResponse,
  PayoutMarkPaidResponse,
  PayoutDispatchResponse,
  SupplierPayoutPolicy,
  SupplierPayoutPolicyPatch,
  ForecastAccuracyResponse,
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
  ManifestShipUnitsResponse,
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
  WarehouseCoverageRequest,
  WarehouseCoverageResponse,
  WarehouseOpsLocationResponse,
  WarehouseOpsPaymentConfigResponse,
  WarehouseOpsSupplyFactoryResponse,
  WarehousePinsResponse,
  SupplierRegionsResponse,
  SupplierRegionsReplaceRequest,
  SupplierPaymentCatalogResponse,
  RetailerPaymentCatalogResponse,
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
  WarehouseBinLocation,
  WarehouseBinListResponse,
  WarehouseBinCreateRequest,
  StockLot,
  StockLotListResponse,
  StockLotPutawayRequest,
  StockLotPutawayResponse,
  QuarantineLotRequest,
  ReleaseLotRequest,
  LotGenealogyView,
  InitiateRecallRequest,
  RecallCampaignView,
  PickWave,
  PickWaveListResponse,
  PickWaveCreateRequest,
  PickTaskConfirmRequest,
  CycleCount,
  CycleCountListResponse,
  CycleCountCreateRequest,
  CycleCountSubmitRequest,
  InventoryAdjustment,
  InventoryAdjustmentListResponse,
  WarehouseFleetLiveMapResponse,
  FactoryFleetLiveMapResponse,
  WarehouseFleetVehicleDetailResponse,
  WarehouseFleetVehicleListResponse,
  WarehouseOpsDashboardResponse,
  WarehouseOpsFinancialsResponse,
  WarehouseOpsSettings,
  WarehouseOpsSettingsPatchRequest,
  CheckoutPreviewResponse,
  OrderCurrencyOptions,
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
import { claimFileKey, claimApproveKey, claimRejectKey } from "../idempotency";
import { type ConditionalGet } from "../conditional-get";
import { ApiClientBase, ApiError, appendQuery, extractErrorMessage, parseResponsePayload } from "./client-base";

export class SupplierApiClient extends ApiClientBase {
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

  async listManifestShipUnits(
    manifestId: string,
    role: "warehouse" | "payload" | "payloader" | "supplier" = "warehouse",
  ): Promise<ManifestShipUnitsResponse> {
    const mid = encodeURIComponent(manifestId);
    const path =
      role === "supplier"
        ? `/v1/supplier/manifests/${mid}/ship-units`
        : role === "payload" || role === "payloader"
          ? `/v1/${role}/manifests/${mid}/ship-units`
          : `/v1/warehouse/manifests/${mid}/ship-units`;
    return this.request<ManifestShipUnitsResponse>(path, "GET");
  }

  /** Downloads GS1-128 ZPL as text/plain (caller may save as .zpl). */
  async renderManifestLabelsZpl(
    manifestId: string,
    opts: { orderId?: string; role?: "warehouse" | "payload" | "payloader" | "supplier" } = {},
  ): Promise<string> {
    const mid = encodeURIComponent(manifestId);
    const role = opts.role ?? "warehouse";
    const path =
      role === "supplier"
        ? `/v1/supplier/manifests/${mid}/labels`
        : role === "payload" || role === "payloader"
          ? `/v1/${role}/manifests/${mid}/labels`
          : `/v1/warehouse/manifests/${mid}/labels`;
    const body = opts.orderId ? { order_id: opts.orderId } : {};
    return this.requestText(path, "POST", { body });
  }

  async getSupplierTopology(): Promise<SupplierTopologyResponse> {
    return this.request<SupplierTopologyResponse>("/v1/supplier/topology", "GET");
  }

  async updateSupplierTopology(request: SupplierTopologyUpdateRequest): Promise<SupplierTopologyResponse> {
    return this.request<SupplierTopologyResponse>("/v1/supplier/topology", "PUT", { body: request });
  }

  async getSupplierPaymentCatalog(): Promise<SupplierPaymentCatalogResponse> {
    return this.request<SupplierPaymentCatalogResponse>("/v1/supplier/payment-catalog", "GET");
  }

  async getRetailerPaymentCatalog(): Promise<RetailerPaymentCatalogResponse> {
    return this.request<RetailerPaymentCatalogResponse>("/v1/retailer/payment-catalog", "GET");
  }

  async getSupplierRegions(): Promise<SupplierRegionsResponse> {
    return this.request<SupplierRegionsResponse>("/v1/supplier/regions", "GET");
  }

  async replaceSupplierRegions(request: SupplierRegionsReplaceRequest): Promise<SupplierRegionsResponse> {
    return this.request<SupplierRegionsResponse>("/v1/supplier/regions", "PUT", { body: request });
  }

  async getWarehouseCoverage(warehouseId: string): Promise<WarehouseCoverageResponse> {
    return this.request<WarehouseCoverageResponse>(
      `/v1/supplier/warehouses/${encodeURIComponent(warehouseId)}/coverage`,
      "GET",
    );
  }

  async replaceWarehouseCoverage(
    warehouseId: string,
    request: WarehouseCoverageRequest,
  ): Promise<WarehouseCoverageResponse> {
    return this.request<WarehouseCoverageResponse>(
      `/v1/supplier/warehouses/${encodeURIComponent(warehouseId)}/coverage`,
      "PUT",
      { body: request },
    );
  }

  async getWarehousePins(warehouseId: string): Promise<WarehousePinsResponse> {
    return this.request<WarehousePinsResponse>(
      `/v1/supplier/warehouses/${encodeURIComponent(warehouseId)}/pins`,
      "GET",
    );
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

  async getSupplierDashboardConditional(ifNoneMatch?: string): Promise<ConditionalGet<SupplierDashboardResponse>> {
    return this.requestConditional<SupplierDashboardResponse>("/v1/supplier/dashboard", "GET", {
      ifNoneMatch,
    });
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

  /** §8.4 persisted WAPE / bias / tracking signal from ForecastAccuracyDaily. */
  async getSupplierDemandAccuracy(params?: {
    days?: number;
    warehouse_id?: string;
    product_id?: string;
  }): Promise<import("@pegasusx/types").SupplierDemandAccuracyResponse> {
    const q = new URLSearchParams();
    if (params?.days != null) q.set("days", String(params.days));
    if (params?.warehouse_id) q.set("warehouse_id", params.warehouse_id);
    if (params?.product_id) q.set("product_id", params.product_id);
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request<import("@pegasusx/types").SupplierDemandAccuracyResponse>(
      `/v1/supplier/analytics/demand/accuracy${suffix}`,
      "GET",
    );
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

  async sealAllSupplierManifests(idempotencyKey: string): Promise<{ sealed_count: number; errors?: unknown[] }> {
    return this.request<{ sealed_count: number; errors?: unknown[] }>("/v1/supplier/manifests/seal-all", "POST", {
      idempotencyKey,
    });
  }

  async sealAllPayloaderManifests(idempotencyKey: string): Promise<{ sealed_count: number; errors?: unknown[] }> {
    return this.request<{ sealed_count: number; errors?: unknown[] }>("/v1/payloader/manifests/seal-all", "POST", {
      idempotencyKey,
    });
  }

  async setFactoryStaffPassword(
    staffId: string,
    body: { pin?: string; password?: string },
    idempotencyKey: string,
  ): Promise<{ staff_id: string; password_set: boolean }> {
    return this.request(`/v1/factory/staff/${encodeURIComponent(staffId)}/set-password`, "POST", {
      body,
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
  async getSupplierNegotiationsPending(params?: {
    limit?: number;
    offset?: number;
  }): Promise<NegotiationPendingResponse> {
    const q = new URLSearchParams();
    if (params?.limit != null) q.set("limit", String(params.limit));
    if (params?.offset != null) q.set("offset", String(params.offset));
    const qs = q.toString();
    return this.request<NegotiationPendingResponse>(
      `/v1/supplier/negotiations/pending${qs ? `?${qs}` : ""}`,
      "GET",
    );
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

  async patchSupplierReplenishmentPolicies(
    patch: SupplierReplenishmentPolicyPatch,
  ): Promise<SupplierReplenishmentPolicy> {
    return this.request<SupplierReplenishmentPolicy>(
      "/v1/supplier/replenishment/policies",
      "PATCH",
      { body: patch },
    );
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

  async getSupplierAgingSummary(params?: { supplier_id?: string }): Promise<AgingSummaryResponse> {
    const q = new URLSearchParams();
    if (params?.supplier_id) q.set("supplier_id", params.supplier_id);
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/supplier/ar/aging-summary${suffix}`, "GET");
  }

  async writeOffInvoice(invoiceId: string, body: WriteOffRequest): Promise<ArInvoice> {
    return this.request(
      `/v1/supplier/ar/invoices/${encodeURIComponent(invoiceId)}/write-off`,
      "POST",
      { body },
    );
  }

  async retailerPayInvoice(
    invoiceId: string,
    body: RetailerPayInvoiceRequest,
    options: { idempotencyKey?: string } = {},
  ): Promise<ArInvoice> {
    return this.request(
      `/v1/retailer/ar/invoices/${encodeURIComponent(invoiceId)}/pay`,
      "POST",
      { body, idempotencyKey: options.idempotencyKey },
    );
  }

  async checkRetailerDelinquencyLock(params?: {
    retailer_id?: string;
    supplier_id?: string;
  }): Promise<DelinquencyLockStatus> {
    const q = new URLSearchParams();
    if (params?.retailer_id) q.set("retailer_id", params.retailer_id);
    if (params?.supplier_id) q.set("supplier_id", params.supplier_id);
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/retailer/ar/delinquency-lock${suffix}`, "GET");
  }

  async getCashBagSummary(): Promise<CashBagSummaryResponse> {
    return this.request("/v1/fleet/driver/cash-bag/summary", "GET");
  }

  async turnInCashBag(body: CashBagTurnInRequest): Promise<CashReconciliation> {
    return this.request("/v1/fleet/driver/cash-bag/turn-in", "POST", { body });
  }

  async listWarehouseOpsCashReconciliations(params?: { status?: string }): Promise<{ reconciliations: CashReconciliation[] }> {
    const q = new URLSearchParams();
    if (params?.status) q.set("status", params.status);
    const suffix = q.toString() ? `?${q.toString()}` : "";
    return this.request(`/v1/warehouse/ops/cash-reconciliations${suffix}`, "GET");
  }

  async acceptWarehouseOpsCashReconciliation(id: string, body?: { finance_note?: string }): Promise<CashReconciliation> {
    return this.request(
      `/v1/warehouse/ops/cash-reconciliations/${encodeURIComponent(id)}/accept`,
      "POST",
      { body: body || {} },
    );
  }

  async disputeWarehouseOpsCashReconciliation(id: string, body?: { finance_note?: string }): Promise<CashReconciliation> {
    return this.request(
      `/v1/warehouse/ops/cash-reconciliations/${encodeURIComponent(id)}/dispute`,
      "POST",
      { body: body || {} },
    );
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

  async adminDisableCreditProgram(
    supplierId: string,
    body: { ticket_id: string; reason: string },
  ): Promise<StatusResponse> {
    return this.request(
      `/v1/admin/credit-program/${encodeURIComponent(supplierId)}/disable`,
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


  async listLaborZoneCapacity(date?: string): Promise<{ zones: LaborZoneCapacity[] } | LaborZoneCapacity> {
    const q = date ? `?date=${encodeURIComponent(date)}` : "";
    return this.request<{ zones: LaborZoneCapacity[] } | LaborZoneCapacity>(`/v1/labor-capacity/zone-capacity${q}`, "GET");
  }

  async getLaborDriverScore(driverId: string): Promise<LaborDriverScore> {
    return this.request<LaborDriverScore>(
      `/v1/labor-capacity/driver-score/${encodeURIComponent(driverId)}`,
      "GET",
    );
  }

  async setLaborDriverAvailability(body: {
    driverId: string;
    date: string;
    availableHours: number;
    zoneH3?: string;
    status: string;
  }): Promise<{ status: string }> {
    return this.request<{ status: string }>("/v1/labor-capacity/driver-availability", "POST", { body });
  }

  async getSupplierMEIONetworkSummary(): Promise<SupplierMEIONetworkSummary> {
    return this.request<SupplierMEIONetworkSummary>("/v1/supplier/meio/network-summary", "GET");
  }
}
