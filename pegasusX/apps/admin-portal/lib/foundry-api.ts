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

const base = () =>
  (process.env.NEXT_PUBLIC_BACKEND_BASE_URL || "http://localhost:8080").replace(/\/$/, "");

async function req<T>(token: string, method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const res = await fetch(base() + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
  });

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const j = await res.json() as { error?: string };
      if (j?.error) msg = j.error;
    } catch {
      /* non-json */
    }
    throw new Error(msg);
  }
  return (await res.json()) as T;
}

export const foundryApi = {
  // --- Semantic Layer (Ontology) ---
  upsertObject: (token: string, body: OntologyObject) =>
    req<OntologyObject>(token, "POST", "/v1/foundry/ontology/objects", body),

  getObject: (token: string, cellId: string, objectType: string, objectId: string) =>
    req<OntologyObject>(
      token,
      "GET",
      `/v1/foundry/ontology/objects/${encodeURIComponent(cellId)}/${encodeURIComponent(objectType)}/${encodeURIComponent(objectId)}`,
    ),

  linkObjects: (token: string, body: OntologyGraphEdge) =>
    req<OntologyGraphEdge>(token, "POST", "/v1/foundry/ontology/links", body),

  searchAround: (token: string, body: SearchAroundRequest) =>
    req<SearchAroundResponse>(token, "POST", "/v1/foundry/ontology/search-around", body),

  lotRecall: (token: string, body: LotRecallRequest) =>
    req<RecallBlastRadiusReport>(token, "POST", "/v1/foundry/ontology/lot-recall", body),

  // --- Kinetic Layer ---
  executeAction: (token: string, body: ActionExecutionRequest) =>
    req<ActionExecutionRecord>(token, "POST", "/v1/foundry/kinetic/actions/execute", body),

  // --- Dynamic Layer (Scenario Sandbox) ---
  applyScenarioPatch: (token: string, scenarioId: string, body: ScenarioPatchRequest) =>
    req<ScenarioPatch>(token, "POST", `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/patches`, body),

  resolveScenarioObject: (token: string, scenarioId: string, cellId: string, objectType: string, objectId: string) =>
    req<OntologyObject>(
      token,
      "GET",
      `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/resolve/${encodeURIComponent(cellId)}/${encodeURIComponent(objectType)}/${encodeURIComponent(objectId)}`,
    ),

  commitScenario: (token: string, scenarioId: string) =>
    req<ScenarioCommitResult>(token, "POST", `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/commit`),

  // --- Role 7 Depot Cashier & Vault Operations ---
  startCashierSession: (token: string, body: StartCashierSessionRequest) =>
    req<CashierSessionView>(token, "POST", "/v1/foundry/cashier/sessions", body),

  submitBlindCount: (token: string, sessionId: string, body: BlindCountRequest) =>
    req<CashierSessionView>(
      token,
      "POST",
      `/v1/foundry/cashier/sessions/${encodeURIComponent(sessionId)}/blind-count`,
      body,
    ),

  reclassifyTender: (token: string, body: RRNTenderReclassificationRequest) =>
    req<CashierSessionView>(token, "POST", "/v1/foundry/cashier/reclassify-tender", body),

  citVaultHandover: (token: string, body: CITHandoverRequest) =>
    req<DepotVaultLedgerEntry>(token, "POST", "/v1/foundry/cashier/vault/cit-handover", body),

  // --- Role 9 Fleet, Yard & Maintenance Operations ---
  submitDVIR: (token: string, body: FleetDVIRInspection) =>
    req<FleetDVIRInspection>(token, "POST", "/v1/foundry/fleet/dvir", body),

  recordGateEvent: (token: string, body: YardGateCheckEvent) =>
    req<YardGateCheckEvent>(token, "POST", "/v1/foundry/fleet/gate-events", body),

  midRouteRescue: (token: string, body: MidRouteRescueRequest) =>
    req<MidRouteRescueResult>(token, "POST", "/v1/foundry/fleet/mid-route-rescue", body),

  predictiveReefer: (token: string, body: PredictiveReeferRequest) =>
    req<PredictiveReeferResponse>(token, "POST", "/v1/foundry/fleet/predictive-reefer", body),

  dispatchCheck: (token: string, vehicleId: string) =>
    req<DispatchCheckResponse>(
      token,
      "GET",
      `/v1/foundry/fleet/vehicles/${encodeURIComponent(vehicleId)}/dispatch-check`,
    ),

  // --- 12 Cross-Role Pre-Execution Guards ---
  splitTenderReconcile: (token: string, body: SplitTenderRejectionInput) =>
    req<SplitTenderReconciliationResult>(token, "POST", "/v1/foundry/guards/split-tender-reconcile", body),

  amlCheck: (token: string, body: AMLCheckRequest) =>
    req<AMLCheckResponse>(token, "POST", "/v1/foundry/guards/aml-check", body),

  issueOfflineLease: (token: string, body: OfflineLeaseIssueRequest) =>
    req<OfflineQuotaLease>(token, "POST", "/v1/foundry/guards/offline-lease/issue", body),

  validateOfflineLease: (token: string, body: OfflineLeaseValidateRequest) =>
    req<OfflineLeaseValidateResponse>(token, "POST", "/v1/foundry/guards/offline-lease/validate", body),

  fiscalPreflight: (token: string, body: FiscalPreflightRequest) =>
    req<FiscalPreflightResponse>(token, "POST", "/v1/foundry/guards/fiscal-preflight", body),

  slicePayoutEligibility: (token: string, body: SlicePayoutEligibilityRequest) =>
    req<SlicePayoutEligibilityResponse>(token, "POST", "/v1/foundry/guards/slice-payout-eligibility", body),

  clawbackSlice: (token: string, body: ClawbackSliceRequest) =>
    req<SettlementSliceRecord>(token, "POST", "/v1/foundry/guards/clawback-slice", body),

  load3DRTI: (token: string, body: Load3DRTIRequest) =>
    req<ReturnableContainerBalance>(token, "POST", "/v1/foundry/guards/load-3d-rti", body),

  fxCorridor: (token: string, body: FXCorridorRequest) =>
    req<FXCorridorResponse>(token, "POST", "/v1/foundry/guards/fx-corridor", body),

  registerToteState: (token: string, body: PickToteState) =>
    req<PickToteState>(token, "POST", "/v1/foundry/guards/tote-state", body),

  toteTakeover: (token: string, body: ToteTakeoverRequest) =>
    req<PickToteState>(token, "POST", "/v1/foundry/guards/tote-takeover", body),
};
