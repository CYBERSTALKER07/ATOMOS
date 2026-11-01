import { createFactoryApi } from "./api";
import type {
  SearchAroundRequest,
  SearchAroundResponse,
  LotRecallRequest,
  RecallBlastRadiusReport,
  ActionExecutionRequest,
  ActionExecutionRecord,
  YardGateCheckEvent,
  AMLCheckRequest,
  AMLCheckResponse,
} from "@pegasusx/types";

const api = createFactoryApi();

export const foundryFactoryApi = {
  // --- Semantic Layer (Ontology) ---
  searchAround: (body: SearchAroundRequest): Promise<SearchAroundResponse> =>
    api.request<SearchAroundResponse>("/v1/foundry/ontology/search-around", "POST", { body }),

  lotRecall: (body: LotRecallRequest): Promise<RecallBlastRadiusReport> =>
    api.request<RecallBlastRadiusReport>("/v1/foundry/ontology/lot-recall", "POST", { body }),

  // --- Kinetic Actions ---
  executeAction: (body: ActionExecutionRequest): Promise<ActionExecutionRecord> =>
    api.request<ActionExecutionRecord>("/v1/foundry/kinetic/actions/execute", "POST", { body }),

  // --- Fleet / Yard Gate Events ---
  recordGateEvent: (body: YardGateCheckEvent): Promise<YardGateCheckEvent> =>
    api.request<YardGateCheckEvent>("/v1/foundry/fleet/gate-events", "POST", { body }),

  // --- Guards ---
  amlCheck: (body: AMLCheckRequest): Promise<AMLCheckResponse> =>
    api.request<AMLCheckResponse>("/v1/foundry/guards/aml-check", "POST", { body }),
};
