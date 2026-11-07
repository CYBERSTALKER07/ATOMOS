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
import type { ApiClient } from "../index";

export interface FoundryClientCaller {
  request<T>(
    path: string,
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    options?: {
      body?: unknown;
      headers?: Record<string, string>;
      idempotencyKey?: string;
      requiresAuth?: boolean;
    },
  ): Promise<T>;
}

export class FoundryApi {
  constructor(private readonly client: FoundryClientCaller) {}

  // --- Semantic Layer (Ontology) ---

  async upsertOntologyObject(req: OntologyObject): Promise<OntologyObject> {
    return this.client.request<OntologyObject>("/v1/foundry/ontology/objects", "POST", { body: req });
  }

  async getOntologyObject(cellId: string, objectType: string, objectId: string): Promise<OntologyObject> {
    return this.client.request<OntologyObject>(
      `/v1/foundry/ontology/objects/${encodeURIComponent(cellId)}/${encodeURIComponent(objectType)}/${encodeURIComponent(objectId)}`,
      "GET",
    );
  }

  async linkOntologyObjects(req: OntologyGraphEdge): Promise<OntologyGraphEdge> {
    return this.client.request<OntologyGraphEdge>("/v1/foundry/ontology/links", "POST", { body: req });
  }

  async searchAround(req: SearchAroundRequest): Promise<SearchAroundResponse> {
    return this.client.request<SearchAroundResponse>("/v1/foundry/ontology/search-around", "POST", { body: req });
  }

  async triggerLotRecall(req: LotRecallRequest): Promise<RecallBlastRadiusReport> {
    return this.client.request<RecallBlastRadiusReport>("/v1/foundry/ontology/lot-recall", "POST", { body: req });
  }

  // --- Kinetic Layer (Action Execution) ---

  async executeKineticAction(req: ActionExecutionRequest): Promise<ActionExecutionRecord> {
    return this.client.request<ActionExecutionRecord>("/v1/foundry/kinetic/actions/execute", "POST", {
      body: req,
      idempotencyKey: req.idempotency_key,
    });
  }

  // --- Dynamic Layer (Scenario Sandbox) ---

  async applyScenarioPatch(scenarioId: string, req: ScenarioPatchRequest): Promise<ScenarioPatch> {
    return this.client.request<ScenarioPatch>(
      `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/patches`,
      "POST",
      { body: req },
    );
  }

  async resolveScenarioObject(
    scenarioId: string,
    cellId: string,
    objectType: string,
    objectId: string,
  ): Promise<OntologyObject> {
    return this.client.request<OntologyObject>(
      `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/resolve/${encodeURIComponent(cellId)}/${encodeURIComponent(objectType)}/${encodeURIComponent(objectId)}`,
      "GET",
    );
  }

  async commitScenario(scenarioId: string): Promise<ScenarioCommitResult> {
    return this.client.request<ScenarioCommitResult>(
      `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/commit`,
      "POST",
    );
  }

  // --- Role 7 Depot Cashier & Vault Operations ---

  async startCashierSession(req: StartCashierSessionRequest): Promise<CashierSessionView> {
    return this.client.request<CashierSessionView>("/v1/foundry/cashier/sessions", "POST", { body: req });
  }

  async submitBlindCount(sessionId: string, req: BlindCountRequest): Promise<CashierSessionView> {
    return this.client.request<CashierSessionView>(
      `/v1/foundry/cashier/sessions/${encodeURIComponent(sessionId)}/blind-count`,
      "POST",
      { body: req },
    );
  }

  async reclassifyTender(req: RRNTenderReclassificationRequest): Promise<CashierSessionView> {
    return this.client.request<CashierSessionView>("/v1/foundry/cashier/reclassify-tender", "POST", { body: req });
  }

  async executeCITHandover(req: CITHandoverRequest): Promise<DepotVaultLedgerEntry> {
    return this.client.request<DepotVaultLedgerEntry>("/v1/foundry/cashier/vault/cit-handover", "POST", { body: req });
  }

  // --- Role 9 Fleet, Yard & Maintenance Operations ---

  async submitDVIR(req: FleetDVIRInspection): Promise<FleetDVIRInspection> {
    return this.client.request<FleetDVIRInspection>("/v1/foundry/fleet/dvir", "POST", { body: req });
  }

  async recordGateEvent(req: YardGateCheckEvent): Promise<YardGateCheckEvent> {
    return this.client.request<YardGateCheckEvent>("/v1/foundry/fleet/gate-events", "POST", { body: req });
  }

  async executeMidRouteRescue(req: MidRouteRescueRequest): Promise<MidRouteRescueResult> {
    return this.client.request<MidRouteRescueResult>("/v1/foundry/fleet/mid-route-rescue", "POST", { body: req });
  }

  async evaluatePredictiveReefer(req: PredictiveReeferRequest): Promise<PredictiveReeferResponse> {
    return this.client.request<PredictiveReeferResponse>("/v1/foundry/fleet/predictive-reefer", "POST", { body: req });
  }

  async checkDispatchAuthorized(vehicleId: string): Promise<DispatchCheckResponse> {
    return this.client.request<DispatchCheckResponse>(
      `/v1/foundry/fleet/vehicles/${encodeURIComponent(vehicleId)}/dispatch-check`,
      "GET",
    );
  }

  // --- 12 Cross-Role Pre-Execution Guards ---

  async reconcileSplitTender(req: SplitTenderRejectionInput): Promise<SplitTenderReconciliationResult> {
    return this.client.request<SplitTenderReconciliationResult>(
      "/v1/foundry/guards/split-tender-reconcile",
      "POST",
      { body: req },
    );
  }

  async checkAML(req: AMLCheckRequest): Promise<AMLCheckResponse> {
    return this.client.request<AMLCheckResponse>("/v1/foundry/guards/aml-check", "POST", { body: req });
  }

  async issueOfflineLease(req: OfflineLeaseIssueRequest): Promise<OfflineQuotaLease> {
    return this.client.request<OfflineQuotaLease>("/v1/foundry/guards/offline-lease/issue", "POST", { body: req });
  }

  async validateOfflineLease(req: OfflineLeaseValidateRequest): Promise<OfflineLeaseValidateResponse> {
    return this.client.request<OfflineLeaseValidateResponse>("/v1/foundry/guards/offline-lease/validate", "POST", {
      body: req,
    });
  }

  async checkFiscalPreflight(req: FiscalPreflightRequest): Promise<FiscalPreflightResponse> {
    return this.client.request<FiscalPreflightResponse>("/v1/foundry/guards/fiscal-preflight", "POST", { body: req });
  }

  async checkSlicePayoutEligibility(req: SlicePayoutEligibilityRequest): Promise<SlicePayoutEligibilityResponse> {
    return this.client.request<SlicePayoutEligibilityResponse>(
      "/v1/foundry/guards/slice-payout-eligibility",
      "POST",
      { body: req },
    );
  }

  async createClawbackSlice(req: ClawbackSliceRequest): Promise<SettlementSliceRecord> {
    return this.client.request<SettlementSliceRecord>("/v1/foundry/guards/clawback-slice", "POST", { body: req });
  }

  async validate3DLoadAndRTI(req: Load3DRTIRequest): Promise<ReturnableContainerBalance> {
    return this.client.request<ReturnableContainerBalance>("/v1/foundry/guards/load-3d-rti", "POST", { body: req });
  }

  async evaluateFXCorridor(req: FXCorridorRequest): Promise<FXCorridorResponse> {
    return this.client.request<FXCorridorResponse>("/v1/foundry/guards/fx-corridor", "POST", { body: req });
  }

  async registerToteState(req: PickToteState): Promise<PickToteState> {
    return this.client.request<PickToteState>("/v1/foundry/guards/tote-state", "POST", { body: req });
  }

  async takeoverAbandonedTote(req: ToteTakeoverRequest): Promise<PickToteState> {
    return this.client.request<PickToteState>("/v1/foundry/guards/tote-takeover", "POST", { body: req });
  }
}

export function createFoundryApi(client: FoundryClientCaller | ApiClient): FoundryApi {
  return new FoundryApi(client as FoundryClientCaller);
}
