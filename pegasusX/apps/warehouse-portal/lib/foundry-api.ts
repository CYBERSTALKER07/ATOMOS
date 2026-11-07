import { warehouseApi } from "./warehouse-api";
import { ApiError } from "@pegasusx/api-core";
import type {
  OntologyObject,
  OntologyGraphEdge,
  SearchAroundRequest,
  SearchAroundResponse,
  LotRecallRequest,
  RecallBlastRadiusReport,
  ActionExecutionRequest,
  ActionExecutionRecord,
  ScenarioPatch,
  ScenarioPatchRequest,
  ScenarioCommitResult,
  StartCashierSessionRequest,
  CashierSessionView,
  BlindCountRequest,
  RRNTenderReclassificationRequest,
  CITHandoverRequest,
  DepotVaultLedgerEntry,
  FleetDVIRInspection,
  YardGateCheckEvent,
  PredictiveReeferRequest,
  PredictiveReeferResponse,
  DispatchCheckResponse,
  MidRouteRescueRequest,
  MidRouteRescueResult,
  SplitTenderRejectionInput,
  SplitTenderReconciliationResult,
  AMLCheckRequest,
  AMLCheckResponse,
  OfflineLeaseIssueRequest,
  OfflineQuotaLease,
  OfflineLeaseValidateRequest,
  OfflineLeaseValidateResponse,
  FiscalPreflightRequest,
  FiscalPreflightResponse,
  SlicePayoutEligibilityRequest,
  SlicePayoutEligibilityResponse,
  ClawbackSliceRequest,
  SettlementSliceRecord,
  Load3DRTIRequest,
  ReturnableContainerBalance,
  FXCorridorRequest,
  FXCorridorResponse,
  PickToteState,
  ToteTakeoverRequest,
} from "@pegasusx/types";

export const foundryWarehouseApi = {
  // --- Semantic Layer (Ontology) ---
  upsertObject: (body: OntologyObject): Promise<OntologyObject> =>
    warehouseApi.request<OntologyObject>("/v1/foundry/ontology/objects", "POST", { body }),

  getObject: (cellId: string, objectType: string, objectId: string): Promise<OntologyObject> =>
    warehouseApi.request<OntologyObject>(
      `/v1/foundry/ontology/objects/${encodeURIComponent(cellId)}/${encodeURIComponent(objectType)}/${encodeURIComponent(objectId)}`,
      "GET",
    ),

  linkObjects: (body: OntologyGraphEdge): Promise<OntologyGraphEdge> =>
    warehouseApi.request<OntologyGraphEdge>("/v1/foundry/ontology/links", "POST", { body }),

  searchAround: (body: SearchAroundRequest): Promise<SearchAroundResponse> =>
    warehouseApi.request<SearchAroundResponse>("/v1/foundry/ontology/search-around", "POST", { body }),

  lotRecall: (body: LotRecallRequest): Promise<RecallBlastRadiusReport> =>
    warehouseApi.request<RecallBlastRadiusReport>("/v1/foundry/ontology/lot-recall", "POST", { body }),

  // --- Kinetic Layer ---
  executeAction: (body: ActionExecutionRequest): Promise<ActionExecutionRecord> =>
    warehouseApi.request<ActionExecutionRecord>("/v1/foundry/kinetic/actions/execute", "POST", {
      body,
      idempotencyKey: body.idempotency_key,
    }),

  // --- Dynamic Layer (Scenario Sandbox) ---
  applyScenarioPatch: (scenarioId: string, body: ScenarioPatchRequest): Promise<ScenarioPatch> =>
    warehouseApi.request<ScenarioPatch>(
      `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/patches`,
      "POST",
      { body },
    ),

  resolveScenarioObject: (
    scenarioId: string,
    cellId: string,
    objectType: string,
    objectId: string,
  ): Promise<OntologyObject> =>
    warehouseApi.request<OntologyObject>(
      `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/resolve/${encodeURIComponent(cellId)}/${encodeURIComponent(objectType)}/${encodeURIComponent(objectId)}`,
      "GET",
    ),

  commitScenario: async (scenarioId: string): Promise<ScenarioCommitResult> => {
    try {
      return await warehouseApi.request<ScenarioCommitResult>(
        `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/commit`,
        "POST",
      );
    } catch (err) {
      if (err instanceof ApiError && err.status === 409 && err.payload) {
        return err.payload as ScenarioCommitResult;
      }
      throw err;
    }
  },

  // --- Role 7 Depot Cashier & Vault Operations ---
  startCashierSession: (body: StartCashierSessionRequest): Promise<CashierSessionView> =>
    warehouseApi.request<CashierSessionView>("/v1/foundry/cashier/sessions", "POST", { body }),

  submitBlindCount: async (sessionId: string, body: BlindCountRequest): Promise<CashierSessionView> => {
    try {
      return await warehouseApi.request<CashierSessionView>(
        `/v1/foundry/cashier/sessions/${encodeURIComponent(sessionId)}/blind-count`,
        "POST",
        { body },
      );
    } catch (err) {
      if (err instanceof ApiError && err.status === 409 && err.payload) {
        return err.payload as CashierSessionView;
      }
      throw err;
    }
  },

  reclassifyTender: (body: RRNTenderReclassificationRequest): Promise<CashierSessionView> =>
    warehouseApi.request<CashierSessionView>("/v1/foundry/cashier/reclassify-tender", "POST", { body }),

  citVaultHandover: (body: CITHandoverRequest): Promise<DepotVaultLedgerEntry> =>
    warehouseApi.request<DepotVaultLedgerEntry>("/v1/foundry/cashier/vault/cit-handover", "POST", { body }),

  // --- Role 9 Fleet, Yard & Maintenance Operations ---
  submitDVIR: async (body: FleetDVIRInspection): Promise<FleetDVIRInspection> => {
    try {
      return await warehouseApi.request<FleetDVIRInspection>("/v1/foundry/fleet/dvir", "POST", { body });
    } catch (err) {
      if (err instanceof ApiError && err.status === 409 && err.payload) {
        return err.payload as FleetDVIRInspection;
      }
      throw err;
    }
  },

  recordGateEvent: (body: YardGateCheckEvent): Promise<YardGateCheckEvent> =>
    warehouseApi.request<YardGateCheckEvent>("/v1/foundry/fleet/gate-events", "POST", { body }),

  midRouteRescue: (body: MidRouteRescueRequest): Promise<MidRouteRescueResult> =>
    warehouseApi.request<MidRouteRescueResult>("/v1/foundry/fleet/mid-route-rescue", "POST", { body }),

  predictiveReefer: async (body: PredictiveReeferRequest): Promise<PredictiveReeferResponse> => {
    try {
      return await warehouseApi.request<PredictiveReeferResponse>("/v1/foundry/fleet/predictive-reefer", "POST", { body });
    } catch (err) {
      if (err instanceof ApiError && err.status === 409 && err.payload) {
        return err.payload as PredictiveReeferResponse;
      }
      throw err;
    }
  },

  dispatchCheck: (vehicleId: string): Promise<DispatchCheckResponse> =>
    warehouseApi.request<DispatchCheckResponse>(
      `/v1/foundry/fleet/vehicles/${encodeURIComponent(vehicleId)}/dispatch-check`,
      "GET",
    ),

  // --- 12 Cross-Role Pre-Execution Guards ---
  splitTenderReconcile: (body: SplitTenderRejectionInput): Promise<SplitTenderReconciliationResult> =>
    warehouseApi.request<SplitTenderReconciliationResult>(
      "/v1/foundry/guards/split-tender-reconcile",
      "POST",
      { body },
    ),

  amlCheck: (body: AMLCheckRequest): Promise<AMLCheckResponse> =>
    warehouseApi.request<AMLCheckResponse>("/v1/foundry/guards/aml-check", "POST", { body }),

  issueOfflineLease: (body: OfflineLeaseIssueRequest): Promise<OfflineQuotaLease> =>
    warehouseApi.request<OfflineQuotaLease>("/v1/foundry/guards/offline-lease/issue", "POST", { body }),

  validateOfflineLease: (body: OfflineLeaseValidateRequest): Promise<OfflineLeaseValidateResponse> =>
    warehouseApi.request<OfflineLeaseValidateResponse>("/v1/foundry/guards/offline-lease/validate", "POST", { body }),

  fiscalPreflight: (body: FiscalPreflightRequest): Promise<FiscalPreflightResponse> =>
    warehouseApi.request<FiscalPreflightResponse>("/v1/foundry/guards/fiscal-preflight", "POST", { body }),

  slicePayoutEligibility: (body: SlicePayoutEligibilityRequest): Promise<SlicePayoutEligibilityResponse> =>
    warehouseApi.request<SlicePayoutEligibilityResponse>(
      "/v1/foundry/guards/slice-payout-eligibility",
      "POST",
      { body },
    ),

  clawbackSlice: (body: ClawbackSliceRequest): Promise<SettlementSliceRecord> =>
    warehouseApi.request<SettlementSliceRecord>("/v1/foundry/guards/clawback-slice", "POST", { body }),

  load3DRTI: (body: Load3DRTIRequest): Promise<ReturnableContainerBalance> =>
    warehouseApi.request<ReturnableContainerBalance>("/v1/foundry/guards/load-3d-rti", "POST", { body }),

  fxCorridor: (body: FXCorridorRequest): Promise<FXCorridorResponse> =>
    warehouseApi.request<FXCorridorResponse>("/v1/foundry/guards/fx-corridor", "POST", { body }),

  registerToteState: (body: PickToteState): Promise<PickToteState> =>
    warehouseApi.request<PickToteState>("/v1/foundry/guards/tote-state", "POST", { body }),

  toteTakeover: (body: ToteTakeoverRequest): Promise<PickToteState> =>
    warehouseApi.request<PickToteState>("/v1/foundry/guards/tote-takeover", "POST", { body }),
};
