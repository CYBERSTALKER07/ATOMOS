import { createSupplierApi } from "./api";
import { ApiError } from "@pegasusx/api-core";
import type {
  OntologyObject,
  SearchAroundRequest,
  SearchAroundResponse,
  LotRecallRequest,
  RecallBlastRadiusReport,
  ScenarioPatch,
  ScenarioPatchRequest,
  ScenarioCommitResult,
  AMLCheckRequest,
  AMLCheckResponse,
  ClawbackSliceRequest,
  SettlementSliceRecord,
  FXCorridorRequest,
  FXCorridorResponse,
} from "@pegasusx/types";

const api = createSupplierApi();

export const foundrySupplierApi = {
  // --- Semantic Layer (Ontology) ---
  searchAround: (body: SearchAroundRequest): Promise<SearchAroundResponse> =>
    api.request<SearchAroundResponse>("/v1/foundry/ontology/search-around", "POST", { body }),

  lotRecall: (body: LotRecallRequest): Promise<RecallBlastRadiusReport> =>
    api.request<RecallBlastRadiusReport>("/v1/foundry/ontology/lot-recall", "POST", { body }),

  // --- Dynamic Layer (Scenario Sandbox) ---
  applyScenarioPatch: (scenarioId: string, body: ScenarioPatchRequest): Promise<ScenarioPatch> =>
    api.request<ScenarioPatch>(
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
    api.request<OntologyObject>(
      `/v1/foundry/scenarios/${encodeURIComponent(scenarioId)}/resolve/${encodeURIComponent(cellId)}/${encodeURIComponent(objectType)}/${encodeURIComponent(objectId)}`,
      "GET",
    ),

  commitScenario: async (scenarioId: string): Promise<ScenarioCommitResult> => {
    try {
      return await api.request<ScenarioCommitResult>(
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

  // --- Guards ---
  amlCheck: (body: AMLCheckRequest): Promise<AMLCheckResponse> =>
    api.request<AMLCheckResponse>("/v1/foundry/guards/aml-check", "POST", { body }),

  clawbackSlice: (body: ClawbackSliceRequest): Promise<SettlementSliceRecord> =>
    api.request<SettlementSliceRecord>("/v1/foundry/guards/clawback-slice", "POST", { body }),

  fxCorridor: (body: FXCorridorRequest): Promise<FXCorridorResponse> =>
    api.request<FXCorridorResponse>("/v1/foundry/guards/fx-corridor", "POST", { body }),
};
