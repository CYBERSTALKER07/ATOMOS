import type { ApiClient } from "@pegasusx/api-core";
import {
  MARKING_AML_INVESTIGATION,
  MARKING_DISCREPANT_QUARANTINE,
  MARKING_FINANCE_ONLY,
  MARKING_LOT_RECALL_QUARANTINE,
  MARKING_MICRO_SKIMMING_SUSPECT,
  MARKING_PII,
  MARKING_THERMAL_QUARANTINE_SPOILED,
  ONTOLOGY_LINK_TYPES,
  ONTOLOGY_OBJECT_TYPES,
  ONTOLOGY_SCHEMA_VERSION,
  ONTOLOGY_SECURITY_MARKINGS,
  type KnowledgeGraphEdge,
  type KnowledgeGraphNode,
  type OntologyGraphEdge,
  type OntologyLinkType,
  type OntologyNeighborNode,
  type OntologyNeighborResponse,
  type OntologyObject,
  type OntologyObjectChangedPayload,
  type OntologyObjectType,
  type OntologyObjectsResponse,
  type OntologyPropertiesByType,
  type OntologyRebuildRequest,
  type OntologyRebuildResponse,
  type OntologySecurityMarking,
  type OntologyTypeDescriptor,
  type OntologyTypesResponse,
  type SupplierKnowledgeGraph,
  type TypedOntologyObject,
} from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { formatMinorUnits } from "@/lib/money";

export {
  MARKING_AML_INVESTIGATION,
  MARKING_DISCREPANT_QUARANTINE,
  MARKING_FINANCE_ONLY,
  MARKING_LOT_RECALL_QUARANTINE,
  MARKING_MICRO_SKIMMING_SUSPECT,
  MARKING_PII,
  MARKING_THERMAL_QUARANTINE_SPOILED,
  ONTOLOGY_LINK_TYPES,
  ONTOLOGY_OBJECT_TYPES,
  ONTOLOGY_SCHEMA_VERSION,
  ONTOLOGY_SECURITY_MARKINGS,
};

export type {
  KnowledgeGraphEdge,
  KnowledgeGraphNode,
  OntologyGraphEdge,
  OntologyLinkType,
  OntologyNeighborNode,
  OntologyNeighborResponse,
  OntologyObject,
  OntologyObjectChangedPayload,
  OntologyObjectType,
  OntologyObjectsResponse,
  OntologyPropertiesByType,
  OntologyRebuildRequest,
  OntologyRebuildResponse,
  OntologySecurityMarking,
  OntologyTypeDescriptor,
  OntologyTypesResponse,
  SupplierKnowledgeGraph,
  TypedOntologyObject,
};

function withQuery(path: string, query: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  for (const [key, val] of Object.entries(query)) {
    if (val !== undefined && val !== null && String(val).trim() !== "") {
      params.set(key, String(val).trim());
    }
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

export interface ListOntologyObjectsParams {
  type?: OntologyObjectType | string;
  q?: string;
  cell_id?: string;
  marking?: OntologySecurityMarking | string;
  entity?: string;
  limit?: number;
  offset?: number;
}

export interface GetOntologyNeighborsParams {
  depth?: number;
  cell_id?: string;
  link_type?: OntologyLinkType | string;
  entity?: string;
}

export interface SupplierOntologyApi {
  /** GET /v1/ontology/types */
  listTypes: () => Promise<OntologyTypesResponse>;
  /** GET /v1/ontology/objects */
  listObjects: (params?: ListOntologyObjectsParams) => Promise<OntologyObjectsResponse>;
  /** GET /v1/ontology/objects/{type}/{id}/neighbors */
  getNeighbors: (
    objectType: string,
    objectId: string,
    params?: GetOntologyNeighborsParams,
  ) => Promise<OntologyNeighborResponse>;
  /** POST /v1/ontology/rebuild */
  rebuild: (input?: OntologyRebuildRequest, idempotencyKey?: string) => Promise<OntologyRebuildResponse>;
  /** GET /v1/supplier/knowledge-graph */
  getKnowledgeGraph: () => Promise<SupplierKnowledgeGraph>;
}

export function clampTraversalDepth(depth: number | undefined | null): 1 | 2 | 3 | 4 {
  if (typeof depth !== "number" || !Number.isFinite(depth)) {
    return 2;
  }
  const truncated = Math.trunc(depth);
  if (truncated <= 1) return 1;
  if (truncated === 2) return 2;
  if (truncated === 3) return 3;
  return 4;
}

export function createSupplierOntologyApi(client?: ApiClient): SupplierOntologyApi {
  const api = client ?? createSupplierApi();

  return {
    listTypes() {
      return api.request<OntologyTypesResponse>("/v1/ontology/types", "GET");
    },

    listObjects(params = {}) {
      return api.request<OntologyObjectsResponse>(
        withQuery("/v1/ontology/objects", {
          type: params.type,
          q: params.q,
          cell_id: params.cell_id,
          marking: params.marking,
          entity: params.entity,
          limit: params.limit,
          offset: params.offset,
        }),
        "GET",
      );
    },

    getNeighbors(objectType, objectId, params = {}) {
      const ot = encodeURIComponent(objectType.trim());
      const oid = encodeURIComponent(objectId.trim());
      const depth = params.depth !== undefined ? clampTraversalDepth(params.depth) : undefined;
      return api.request<OntologyNeighborResponse>(
        withQuery(`/v1/ontology/objects/${ot}/${oid}/neighbors`, {
          depth,
          cell_id: params.cell_id,
          link_type: params.link_type,
          entity: params.entity,
        }),
        "GET",
      );
    },

    rebuild(input = {}, idempotencyKey) {
      return api.request<OntologyRebuildResponse>("/v1/ontology/rebuild", "POST", {
        body: input,
        idempotencyKey,
      });
    },

    getKnowledgeGraph() {
      return api.request<SupplierKnowledgeGraph>("/v1/supplier/knowledge-graph", "GET");
    },
  };
}

export type SecurityMarkingTone = "finance" | "pii" | "quarantine" | "neutral";

export function securityMarkingTone(marking: string): SecurityMarkingTone {
  const norm = marking.trim().toUpperCase();
  if (norm === MARKING_FINANCE_ONLY || norm.includes("FINANCE")) {
    return "finance";
  }
  if (norm === MARKING_PII || norm.includes("PII")) {
    return "pii";
  }
  if (
    norm.includes("QUARANTINE") ||
    norm.includes("AML") ||
    norm.includes("SKIMMING") ||
    norm.includes("RECALL")
  ) {
    return "quarantine";
  }
  return "neutral";
}

export function formatOntologyMarkingLabel(marking: string): string {
  const trimmed = marking.trim();
  if (!trimmed) return "";
  return trimmed.startsWith("MARKING:") ? trimmed.slice("MARKING:".length) : trimmed;
}

export function isMinorUnitPropertyKey(key: string): boolean {
  return key.endsWith("_minor");
}

export function formatOntologyPropertyValue(key: string, value: unknown, currency = "UZS"): string {
  if (value === null || value === undefined || value === "") {
    return "—";
  }
  if (isMinorUnitPropertyKey(key)) {
    if (typeof value === "bigint") {
      return formatMinorUnits(value, { currency, minimumFractionDigits: 0, maximumFractionDigits: 2 });
    }
    if (typeof value === "number" && Number.isInteger(value)) {
      return formatMinorUnits(BigInt(value), {
        currency,
        minimumFractionDigits: 0,
        maximumFractionDigits: 2,
      });
    }
    if (typeof value === "string" && /^-?\d+$/.test(value.trim())) {
      return formatMinorUnits(BigInt(value.trim()), {
        currency,
        minimumFractionDigits: 0,
        maximumFractionDigits: 2,
      });
    }
  }
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

export function extractObjectPrimaryLabel(obj: Pick<OntologyObject, "object_id" | "properties">): string {
  const props = obj.properties ?? {};
  const candidateKeys = [
    "name",
    "legal_name",
    "sku",
    "lot_code",
    "label",
    "ehf_id",
    "fiscal_receipt_id",
    "invoice_id",
    "order_id",
    "status",
  ];
  for (const key of candidateKeys) {
    const v = props[key];
    if (typeof v === "string" && v.trim() !== "") {
      return v.trim();
    }
  }
  return obj.object_id;
}
