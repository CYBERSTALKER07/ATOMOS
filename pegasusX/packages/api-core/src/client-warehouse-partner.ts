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
import { appendQuery } from "./client-base";
import { PlanningAndRetailerApiClient } from "./client-planning-retailer";

export class ApiClient extends PlanningAndRetailerApiClient {
  async listInventoryAdjustments(
    query: { warehouse_id?: string; status?: string } = {},
  ): Promise<InventoryAdjustmentListResponse> {
    return this.request<InventoryAdjustmentListResponse>(
      appendQuery("/v1/warehouse/ops/inventory-adjustments", query as Record<string, unknown>),
      "GET",
    );
  }

  async approveInventoryAdjustment(
    adjustmentId: string,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<InventoryAdjustment> {
    return this.request<InventoryAdjustment>(
      appendQuery(
        `/v1/warehouse/ops/inventory-adjustments/${encodeURIComponent(adjustmentId)}/approve`,
        query as Record<string, unknown>,
      ),
      "POST",
      { body: {}, idempotencyKey },
    );
  }

  async getWarehouseOpsCoverage(query: { warehouse_id?: string } = {}): Promise<WarehouseCoverageResponse> {
    return this.request<WarehouseCoverageResponse>(
      appendQuery("/v1/warehouse/ops/coverage", query as Record<string, unknown>),
      "GET",
    );
  }

  async getWarehouseOpsSupplyFactory(query: { warehouse_id?: string } = {}): Promise<WarehouseOpsSupplyFactoryResponse> {
    return this.request<WarehouseOpsSupplyFactoryResponse>(
      appendQuery("/v1/warehouse/ops/supply-factory", query as Record<string, unknown>),
      "GET",
    );
  }

  async getWarehouseOpsLocation(query: { warehouse_id?: string } = {}): Promise<WarehouseOpsLocationResponse> {
    return this.request<WarehouseOpsLocationResponse>(
      appendQuery("/v1/warehouse/ops/location", query as Record<string, unknown>),
      "GET",
    );
  }

  async getWarehouseOpsPaymentConfig(query: { warehouse_id?: string } = {}): Promise<WarehouseOpsPaymentConfigResponse> {
    return this.request<WarehouseOpsPaymentConfigResponse>(
      appendQuery("/v1/warehouse/ops/payment-config", query as Record<string, unknown>),
      "GET",
    );
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

  async getOrderCurrencies(): Promise<OrderCurrencyOptions> {
    return this.request<OrderCurrencyOptions>("/v1/order/currencies", "GET");
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

  async getAdminFxRates(limit = 100): Promise<FxRatesListResponse> {
    return this.request<FxRatesListResponse>(appendQuery("/v1/admin/fx-rates", { limit }), "GET");
  }

  async putAdminFxRate(request: FxRateUpsertInput): Promise<FxRateUpsertResponse> {
    return this.request<FxRateUpsertResponse>("/v1/admin/fx-rates", "PUT", { body: request });
  }

  async getSupplierFxRates(limit = 100): Promise<FxRatesListResponse> {
    return this.request<FxRatesListResponse>(appendQuery("/v1/supplier/fx-rates", { limit }), "GET");
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
      idempotencyKey: idempotencyKey ?? claimApproveKey(claimId, request),
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
      idempotencyKey: idempotencyKey ?? claimRejectKey(claimId, request.resolution_note ?? request.reason),
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

  /** GET /v1/supplier/service-policy */
  async getSupplierServicePolicy(): Promise<SupplierServicePolicy> {
    return this.request<SupplierServicePolicy>("/v1/supplier/service-policy", "GET");
  }

  /** PUT /v1/supplier/service-policy */
  async putSupplierServicePolicy(
    body: Partial<SupplierServicePolicy>,
    idempotencyKey?: string,
  ): Promise<SupplierServicePolicy> {
    return this.request<SupplierServicePolicy>("/v1/supplier/service-policy", "PUT", {
      body,
      idempotencyKey,
    });
  }

  /** GET /v1/retailer/service-promise */
  async evaluateServicePromise(
    query: PromiseEvaluationRequest,
  ): Promise<PromiseEvaluationResult> {
    return this.request<PromiseEvaluationResult>(
      appendQuery("/v1/retailer/service-promise", query as unknown as Record<string, unknown>),
      "GET",
    );
  }

  /** GET /v1/supplier/service-promises/breaches */
  async listBreachedServicePromises(limit: number = 50): Promise<{ breaches: OrderServicePromiseSnapshot[]; total: number }> {
    return this.request<{ breaches: OrderServicePromiseSnapshot[]; total: number }>(
      appendQuery("/v1/supplier/service-promises/breaches", { limit }),
      "GET",
    );
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

  // ── Partner Integration Layer (/partner/v1 + admin key issue) ─────────────

  async issuePartnerApiKey(body: {
    tenant_type?: string;
    tenant_id?: string;
    scopes?: string[];
  }): Promise<PartnerIssuedKey> {
    return this.request<PartnerIssuedKey>("/v1/admin/partner-keys", "POST", { body });
  }

  async listPartnerApiKeys(): Promise<{ keys: PartnerApiKeyMeta[] }> {
    return this.request("/v1/admin/partner-keys", "GET");
  }

  async revokePartnerApiKey(keyId: string): Promise<{ ok: boolean }> {
    return this.request(`/v1/admin/partner-keys/${encodeURIComponent(keyId)}/revoke`, "POST", { body: {} });
  }

  /** Machine-auth partner call — pass full `pxk_…` secret; does not use session JWT. */
  async partnerCreateOrder(
    apiKey: string,
    body: Record<string, unknown>,
    idempotencyKey: string,
  ): Promise<{ order_id: string; status: string; total_minor: number; currency: string }> {
    return this.request("/partner/v1/orders", "POST", {
      body,
      idempotencyKey,
      requiresAuth: false,
      headers: { Authorization: `Bearer ${apiKey}` },
    });
  }

  async partnerGetOrder(apiKey: string, orderId: string): Promise<unknown> {
    return this.request(`/partner/v1/orders/${encodeURIComponent(orderId)}`, "GET", {
      requiresAuth: false,
      headers: { Authorization: `Bearer ${apiKey}` },
    });
  }

  async partnerListCatalog(
    apiKey: string,
    query: { supplier_id?: string; retailer_id?: string; category_id?: string } = {},
  ): Promise<{ products: unknown[] }> {
    return this.request(appendQuery("/partner/v1/catalog", query as Record<string, unknown>), "GET", {
      requiresAuth: false,
      headers: { Authorization: `Bearer ${apiKey}` },
    });
  }

  async partnerInventoryAvailability(
    apiKey: string,
    query: { supplier_id?: string; retailer_id?: string; product_ids?: string } = {},
  ): Promise<{ availability: PartnerAvailabilityRow[] }> {
    return this.request(
      appendQuery("/partner/v1/inventory/availability", query as Record<string, unknown>),
      "GET",
      { requiresAuth: false, headers: { Authorization: `Bearer ${apiKey}` } },
    );
  }

  async partnerCreateWebhook(
    apiKey: string,
    body: { url: string; event_types?: string[] },
  ): Promise<PartnerWebhookCreated> {
    return this.request("/partner/v1/webhooks", "POST", {
      body,
      requiresAuth: false,
      headers: { Authorization: `Bearer ${apiKey}` },
    });
  }

  async partnerPingWebhook(apiKey: string, subscriptionId: string): Promise<{ ok: boolean }> {
    return this.request(`/partner/v1/webhooks/${encodeURIComponent(subscriptionId)}/ping`, "POST", {
      body: {},
      requiresAuth: false,
      headers: { Authorization: `Bearer ${apiKey}` },
    });
  }

  // Supplier JWT convenience aliases (ADMIN session = supplier portal)
  async listSupplierPartnerKeys(): Promise<{ keys: PartnerApiKeyMeta[] }> {
    return this.request("/v1/supplier/partner-keys", "GET");
  }

  async issueSupplierPartnerKey(body: {
    scopes?: string[];
  } = {}): Promise<PartnerIssuedKey> {
    return this.request<PartnerIssuedKey>("/v1/supplier/partner-keys", "POST", { body });
  }

  async revokeSupplierPartnerKey(keyId: string): Promise<{ ok: boolean }> {
    return this.request(`/v1/supplier/partner-keys/${encodeURIComponent(keyId)}/revoke`, "POST", {
      body: {},
    });
  }

  async listSupplierPartnerWebhooks(): Promise<{ subscriptions: PartnerWebhookSubscription[] }> {
    return this.request("/v1/supplier/partner-webhooks", "GET");
  }

  async createSupplierPartnerWebhook(body: {
    url: string;
    event_types?: string[];
  }): Promise<PartnerWebhookCreated> {
    return this.request("/v1/supplier/partner-webhooks", "POST", { body });
  }

  async deactivateSupplierPartnerWebhook(subscriptionId: string): Promise<{ ok: boolean }> {
    return this.request(
      `/v1/supplier/partner-webhooks/${encodeURIComponent(subscriptionId)}`,
      "DELETE",
    );
  }

  async pingSupplierPartnerWebhook(subscriptionId: string): Promise<{ ok: boolean }> {
    return this.request(
      `/v1/supplier/partner-webhooks/${encodeURIComponent(subscriptionId)}/ping`,
      "POST",
      { body: {} },
    );
  }

  async listSupplierPartnerDeadLetter(): Promise<{ attempts: PartnerDeadLetterAttempt[] }> {
    return this.request("/v1/supplier/partner-webhooks/dead-letter", "GET");
  }

  async replaySupplierPartnerDeadLetter(
    attemptId: string,
  ): Promise<{ ok: boolean; attempt_id: string; status: string }> {
    return this.request(
      `/v1/supplier/partner-webhooks/dead-letter/${encodeURIComponent(attemptId)}/replay`,
      "POST",
      { body: {} },
    );
  }

  async createSupplierPartnerExport(body: {
    resource: string;
    format?: string;
    from?: string;
    to?: string;
  }): Promise<PartnerExportJob> {
    return this.request("/v1/supplier/partner-exports", "POST", { body });
  }

  async listSupplierPartnerExports(): Promise<{ jobs: PartnerExportJob[] }> {
    return this.request("/v1/supplier/partner-exports", "GET");
  }

  async getSupplierPartnerExport(jobId: string): Promise<PartnerExportJob> {
    return this.request(`/v1/supplier/partner-exports/${encodeURIComponent(jobId)}`, "GET");
  }

  async getSupplierPartnerSftp(): Promise<PartnerSftpConfig> {
    return this.request("/v1/supplier/partner-sftp", "GET");
  }

  async putSupplierPartnerSftp(body: {
    host?: string;
    port?: number;
    username?: string;
    secret_ref?: string;
    remote_dir?: string;
    inbound_dir?: string;
    outbound_dir?: string;
    archive_dir?: string;
    edi_enabled?: boolean;
  }): Promise<{ ok: boolean }> {
    return this.request("/v1/supplier/partner-sftp", "PUT", { body });
  }

  async getSupplierPartnerAs2(): Promise<PartnerAs2Config> {
    return this.request("/v1/supplier/partner-as2", "GET");
  }

  async putSupplierPartnerAs2(body: {
    as2_enabled?: boolean;
    our_as2_id?: string;
    partner_as2_id?: string;
    partner_url?: string;
    our_cert_secret_ref?: string;
    our_key_secret_ref?: string;
    partner_cert_secret_ref?: string;
    sign_required?: boolean;
    encrypt_required?: boolean;
  }): Promise<PartnerAs2Config> {
    return this.request("/v1/supplier/partner-as2", "PUT", { body });
  }

  async getSupplierPartnerCoa(): Promise<PartnerCoaMap> {
    return this.request("/v1/supplier/partner-coa", "GET");
  }

  async putSupplierPartnerCoa(body: {
    account_ar?: string;
    account_revenue?: string;
    account_bank_cash?: string;
  }): Promise<PartnerCoaMap> {
    return this.request("/v1/supplier/partner-coa", "PUT", { body });
  }

  async listSupplierPartnerEdiDocuments(): Promise<{ documents: PartnerEdiDocument[] }> {
    return this.request("/v1/supplier/partner-edi/documents", "GET");
  }

  async getSupplierPartnerEdiDocument(documentId: string): Promise<PartnerEdiDocument> {
    return this.request(
      `/v1/supplier/partner-edi/documents/${encodeURIComponent(documentId)}`,
      "GET",
    );
  }

  async replaySupplierPartnerEdiDocument(documentId: string): Promise<PartnerEdiDocument> {
    return this.request(
      `/v1/supplier/partner-edi/documents/${encodeURIComponent(documentId)}/replay`,
      "POST",
      { body: {} },
    );
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
}
