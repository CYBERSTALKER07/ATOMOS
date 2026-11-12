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
import { SupplierApiClient } from "./client-supplier";

export class PlanningAndRetailerApiClient extends SupplierApiClient {
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

  async listPlanningScenarios(): Promise<PlanningScenarioListResponse> {
    return this.request<PlanningScenarioListResponse>("/v1/supplier/planning/scenarios", "GET");
  }

  async clonePlanningScenario(
    scenarioId: string,
    request: PlanningScenarioCloneInput,
    idempotencyKey: string,
  ): Promise<PlanningScenarioResult> {
    return this.request<PlanningScenarioResult>(`/v1/supplier/planning/scenarios/${scenarioId}/clone`, "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async comparePlanningScenarios(scenarioIds: [string, string], idempotencyKey: string): Promise<PlanningScenarioCompareResult> {
    return this.request<PlanningScenarioCompareResult>("/v1/supplier/planning/scenarios/compare", "POST", {
      body: { scenario_ids: scenarioIds },
      idempotencyKey,
    });
  }

  async publishPlanningScenario(scenarioId: string, idempotencyKey: string): Promise<PlanningScenarioResult> {
    return this.request<PlanningScenarioResult>(`/v1/supplier/planning/scenarios/${scenarioId}/publish`, "POST", {
      body: {},
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

  async getForecastAccuracy(opts: {
    supplierId: string;
    days?: number;
    warehouseId?: string;
    productId?: string;
  }): Promise<ForecastAccuracyResponse> {
    return this.request<ForecastAccuracyResponse>(
      appendQuery("/v1/admin/planning/accuracy", {
        supplier_id: opts.supplierId,
        days: opts.days ?? 28,
        warehouse_id: opts.warehouseId || undefined,
        product_id: opts.productId || undefined,
      }),
      "GET",
    );
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

  async estimateSeasonalMultipliers(
    persistDrafts = false,
    idempotencyKey?: string,
  ): Promise<SeasonalEstimateResult> {
    return this.request<SeasonalEstimateResult>("/v1/supplier/planning/seasonal-estimate", "POST", {
      body: { persist_drafts: persistDrafts },
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

  async getSupplierCRMRetailers(): Promise<SupplierCRMListResponse> {
    return this.request<SupplierCRMListResponse>("/v1/supplier/crm/retailers", "GET");
  }

  async getSupplierCRMRetailer(retailerId: string): Promise<SupplierCRMRetailerDetail> {
    return this.request<SupplierCRMRetailerDetail>(
      `/v1/supplier/crm/retailers/${encodeURIComponent(retailerId)}`,
      "GET",
    );
  }

  async getSupplierLoyaltyProgram(): Promise<LoyaltyProgram> {
    return this.request<LoyaltyProgram>("/v1/supplier/loyalty/program", "GET");
  }

  async patchSupplierLoyaltyProgram(request: LoyaltyProgram, idempotencyKey: string): Promise<LoyaltyProgram> {
    return this.request<LoyaltyProgram>("/v1/supplier/loyalty/program", "PATCH", {
      body: request,
      idempotencyKey,
    });
  }

  async getRetailerLoyaltyTier(): Promise<LoyaltyTierView> {
    return this.request<LoyaltyTierView>("/v1/retailer/loyalty/tier", "GET");
  }

  async getRetailerLoyaltyLedger(): Promise<LoyaltyLedgerResponse> {
    return this.request<LoyaltyLedgerResponse>("/v1/retailer/loyalty/ledger", "GET");
  }

  async resolveSupplierEntity(request: EntityResolutionResolveRequest): Promise<EntityResolutionResolveResponse> {
    return this.request<EntityResolutionResolveResponse>("/v1/supplier/entity-resolution/resolve", "POST", {
      body: request,
    });
  }

  async explainSupplierEntity(request: EntityResolutionExplainRequest): Promise<EntityResolutionExplainResponse> {
    return this.request<EntityResolutionExplainResponse>("/v1/supplier/entity-resolution/explain", "POST", {
      body: request,
    });
  }

  async postPlanningPredictivePush(idempotencyKey: string): Promise<PredictivePushResponse> {
    return this.request<PredictivePushResponse>("/v1/supplier/planning/predictive-push", "POST", {
      body: {},
      idempotencyKey,
    });
  }

  async getNetworkMode(): Promise<NetworkModeResponse> {
    return this.request<NetworkModeResponse>("/v1/supplier/network-mode", "GET");
  }

  async putNetworkMode(
    request: NetworkModeUpdateRequest,
    idempotencyKey: string,
  ): Promise<NetworkModeUpdateResponse> {
    return this.request<NetworkModeUpdateResponse>("/v1/supplier/network-mode", "PUT", {
      body: request,
      idempotencyKey,
    });
  }

  async postPlanningPullMatrix(idempotencyKey: string): Promise<PullMatrixResponse> {
    return this.request<PullMatrixResponse>("/v1/supplier/planning/pull-matrix", "POST", {
      body: {},
      idempotencyKey,
    });
  }

  async postPlanningKillSwitch(
    request: KillSwitchRequest,
    idempotencyKey: string,
  ): Promise<KillSwitchResponse> {
    return this.request<KillSwitchResponse>("/v1/supplier/planning/kill-switch", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getFactorySupplyRequestQC(requestId: string): Promise<SupplyRequestQCResponse> {
    return this.request<SupplyRequestQCResponse>(
      `/v1/factory/supply-requests/${encodeURIComponent(requestId)}/qc`,
      "GET",
    );
  }

  async postFactorySupplyRequestQC(
    requestId: string,
    body: SupplyRequestQCRequest,
    idempotencyKey: string,
  ): Promise<SupplyRequestQCResponse> {
    return this.request<SupplyRequestQCResponse>(
      `/v1/factory/supply-requests/${encodeURIComponent(requestId)}/qc`,
      "POST",
      { body, idempotencyKey },
    );
  }

  async getWarehouseSupplyRequestQC(requestId: string): Promise<SupplyRequestQCResponse> {
    return this.request<SupplyRequestQCResponse>(
      `/v1/warehouse/supply-requests/${encodeURIComponent(requestId)}/qc`,
      "GET",
    );
  }

  async getPayoutRail(): Promise<PayoutRailInfo> {
    return this.request<PayoutRailInfo>("/v1/supplier/payouts/rail", "GET");
  }

  async getPayoutPolicy(): Promise<SupplierPayoutPolicy> {
    return this.request<SupplierPayoutPolicy>("/v1/supplier/payout-policy", "GET");
  }

  async patchPayoutPolicy(body: SupplierPayoutPolicyPatch): Promise<SupplierPayoutPolicy> {
    return this.request<SupplierPayoutPolicy>("/v1/supplier/payout-policy", "PATCH", { body });
  }

  async listPayoutBatches(): Promise<PayoutBatchListResponse> {
    return this.request<PayoutBatchListResponse>("/v1/supplier/payouts/batches", "GET");
  }

  async getPayoutBatch(batchId: string): Promise<PayoutBatch> {
    return this.request<PayoutBatch>(
      `/v1/supplier/payouts/batches/${encodeURIComponent(batchId)}`,
      "GET",
    );
  }

  async generatePayoutBatch(
    request: PayoutBatchGenerateRequest,
    idempotencyKey: string,
  ): Promise<PayoutBatchGenerateResponse> {
    return this.request<PayoutBatchGenerateResponse>("/v1/supplier/payouts/batches", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async exportPayoutBatch(batchId: string): Promise<string> {
    return this.requestText(
      `/v1/supplier/payouts/batches/${encodeURIComponent(batchId)}/export`,
      "POST",
      { body: {} },
    );
  }

  async dispatchPayoutBatch(batchId: string, live: boolean): Promise<PayoutDispatchResponse> {
    return this.request<PayoutDispatchResponse>(
      `/v1/supplier/payouts/batches/${encodeURIComponent(batchId)}/dispatch`,
      "POST",
      { body: { live } },
    );
  }

  async markPayoutBatchPaid(batchId: string): Promise<PayoutMarkPaidResponse> {
    return this.request<PayoutMarkPaidResponse>(
      `/v1/supplier/payouts/batches/${encodeURIComponent(batchId)}/mark-paid`,
      "POST",
      { body: {} },
    );
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

  async getWarehouseOpsDashboardConditional(
    query: { warehouse_id?: string } = {},
    ifNoneMatch?: string,
  ): Promise<ConditionalGet<WarehouseOpsDashboardResponse>> {
    return this.requestConditional<WarehouseOpsDashboardResponse>(
      appendQuery("/v1/warehouse/ops/dashboard", query as Record<string, unknown>),
      "GET",
      { ifNoneMatch },
    );
  }

  async getWarehouseFleetLiveMap(query: { warehouse_id?: string } = {}): Promise<WarehouseFleetLiveMapResponse> {
    return this.request<WarehouseFleetLiveMapResponse>(appendQuery("/v1/warehouse/ops/fleet/live-map", query as Record<string, unknown>), "GET");
  }

  async getFactoryFleetLiveMap(query: { factory_id?: string } = {}): Promise<FactoryFleetLiveMapResponse> {
    return this.request<FactoryFleetLiveMapResponse>(appendQuery("/v1/factory/fleet/live-map", query as Record<string, unknown>), "GET");
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

  async createRetailerReceiveSession(
    request: CreateRetailerReceiveSessionRequest,
    options: { idempotencyKey?: string } = {},
  ): Promise<RetailerReceiveSession> {
    return this.request<RetailerReceiveSession>(
      "/v1/retailer/stock/receive-sessions",
      "POST",
      {
        body: request,
        idempotencyKey: options.idempotencyKey,
      },
    );
  }

  async confirmRetailerReceiveSession(
    sessionId: string,
    options: { idempotencyKey?: string } = {},
  ): Promise<RetailerReceiveSession> {
    return this.request<RetailerReceiveSession>(
      `/v1/retailer/stock/receive-sessions/${encodeURIComponent(sessionId)}/confirm`,
      "POST",
      {
        idempotencyKey: options.idempotencyKey,
      },
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

  async listWarehouseBins(query: { warehouse_id?: string } = {}): Promise<WarehouseBinListResponse> {
    return this.request<WarehouseBinListResponse>(appendQuery("/v1/warehouse/ops/bins", query as Record<string, unknown>), "GET");
  }

  async createWarehouseBin(
    request: WarehouseBinCreateRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseBinLocation> {
    return this.request<WarehouseBinLocation>(
      appendQuery("/v1/warehouse/ops/bins", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getWarehouseBin(locationId: string, query: { warehouse_id?: string } = {}): Promise<WarehouseBinLocation> {
    return this.request<WarehouseBinLocation>(
      appendQuery(`/v1/warehouse/ops/bins/${encodeURIComponent(locationId)}`, query as Record<string, unknown>),
      "GET",
    );
  }

  async patchWarehouseBin(
    locationId: string,
    patch: Partial<WarehouseBinCreateRequest> & { is_active?: boolean },
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseBinLocation> {
    return this.request<WarehouseBinLocation>(
      appendQuery(`/v1/warehouse/ops/bins/${encodeURIComponent(locationId)}`, query as Record<string, unknown>),
      "PATCH",
      { body: patch, idempotencyKey },
    );
  }

  async listStockLots(
    query: { warehouse_id?: string; product_id?: string; location_id?: string; status?: string } = {},
  ): Promise<StockLotListResponse> {
    return this.request<StockLotListResponse>(appendQuery("/v1/warehouse/ops/lots", query as Record<string, unknown>), "GET");
  }

  async getStockLot(lotId: string, query: { warehouse_id?: string } = {}): Promise<StockLot> {
    return this.request<StockLot>(
      appendQuery(`/v1/warehouse/ops/lots/${encodeURIComponent(lotId)}`, query as Record<string, unknown>),
      "GET",
    );
  }

  async putawayStockLot(
    request: StockLotPutawayRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<StockLotPutawayResponse> {
    return this.request<StockLotPutawayResponse>(
      appendQuery("/v1/warehouse/ops/lots/putaway", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async quarantineStockLot(
    lotId: string,
    request: QuarantineLotRequest = {},
    query: { warehouse_id?: string } = {},
  ): Promise<StockLot> {
    return this.request<StockLot>(
      appendQuery(`/v1/warehouse/ops/lots/${encodeURIComponent(lotId)}/quarantine`, query as Record<string, unknown>),
      "POST",
      { body: request },
    );
  }

  async releaseStockLot(
    lotId: string,
    request: ReleaseLotRequest = {},
    query: { warehouse_id?: string } = {},
  ): Promise<StockLot> {
    return this.request<StockLot>(
      appendQuery(`/v1/warehouse/ops/lots/${encodeURIComponent(lotId)}/release`, query as Record<string, unknown>),
      "POST",
      { body: request },
    );
  }

  async traceStockLot(
    lotId: string,
    query: { warehouse_id?: string } = {},
  ): Promise<LotGenealogyView> {
    return this.request<LotGenealogyView>(
      appendQuery(`/v1/warehouse/ops/lots/${encodeURIComponent(lotId)}/trace`, query as Record<string, unknown>),
      "GET",
    );
  }

  async initiateRecallCampaign(
    request: InitiateRecallRequest,
    options: { idempotencyKey?: string } = {},
  ): Promise<RecallCampaignView> {
    return this.request<RecallCampaignView>(
      "/v1/supplier/recalls",
      "POST",
      { body: request, idempotencyKey: options.idempotencyKey },
    );
  }

  async listRecallCampaigns(
    query: { status?: string } = {},
  ): Promise<{ campaigns: RecallCampaignView[] }> {
    return this.request<{ campaigns: RecallCampaignView[] }>(
      appendQuery("/v1/supplier/recalls", query as Record<string, unknown>),
      "GET",
    );
  }

  async getRecallCampaign(campaignId: string): Promise<RecallCampaignView> {
    return this.request<RecallCampaignView>(
      `/v1/supplier/recalls/${encodeURIComponent(campaignId)}`,
      "GET",
    );
  }

  async listPickWaves(
    query: { warehouse_id?: string; manifest_id?: string; status?: string } = {},
  ): Promise<PickWaveListResponse> {
    return this.request<PickWaveListResponse>(
      appendQuery("/v1/warehouse/ops/pick-waves", query as Record<string, unknown>),
      "GET",
    );
  }

  async createPickWave(
    request: PickWaveCreateRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<PickWave> {
    return this.request<PickWave>(
      appendQuery("/v1/warehouse/ops/pick-waves", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getPickWave(waveId: string, query: { warehouse_id?: string } = {}): Promise<PickWave> {
    return this.request<PickWave>(
      appendQuery(`/v1/warehouse/ops/pick-waves/${encodeURIComponent(waveId)}`, query as Record<string, unknown>),
      "GET",
    );
  }

  async confirmPickTask(
    waveId: string,
    taskId: string,
    request: PickTaskConfirmRequest = {},
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<PickWave> {
    return this.request<PickWave>(
      appendQuery(
        `/v1/warehouse/ops/pick-waves/${encodeURIComponent(waveId)}/tasks/${encodeURIComponent(taskId)}/confirm`,
        query as Record<string, unknown>,
      ),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async waivePickShorts(
    waveId: string,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<PickWave> {
    return this.request<PickWave>(
      appendQuery(
        `/v1/warehouse/ops/pick-waves/${encodeURIComponent(waveId)}/waive-shorts`,
        query as Record<string, unknown>,
      ),
      "POST",
      { body: {}, idempotencyKey },
    );
  }

  async listCycleCounts(
    query: { warehouse_id?: string; status?: string } = {},
  ): Promise<CycleCountListResponse> {
    return this.request<CycleCountListResponse>(
      appendQuery("/v1/warehouse/ops/cycle-counts", query as Record<string, unknown>),
      "GET",
    );
  }

  async createCycleCount(
    request: CycleCountCreateRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<CycleCount> {
    return this.request<CycleCount>(
      appendQuery("/v1/warehouse/ops/cycle-counts", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getCycleCount(countId: string, query: { warehouse_id?: string } = {}): Promise<CycleCount> {
    return this.request<CycleCount>(
      appendQuery(`/v1/warehouse/ops/cycle-counts/${encodeURIComponent(countId)}`, query as Record<string, unknown>),
      "GET",
    );
  }

  async submitCycleCount(
    countId: string,
    request: CycleCountSubmitRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<CycleCount> {
    return this.request<CycleCount>(
      appendQuery(
        `/v1/warehouse/ops/cycle-counts/${encodeURIComponent(countId)}/submit`,
        query as Record<string, unknown>,
      ),
      "POST",
      { body: request, idempotencyKey },
    );
  }
}
