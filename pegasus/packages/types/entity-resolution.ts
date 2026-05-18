/**
 * @file packages/types/entity-resolution.ts
 * @description Shared contracts for supplier entity-resolution read APIs.
 */

export type EntityResolutionEntityType =
  | 'ANY'
  | 'SUPPLIER'
  | 'WAREHOUSE'
  | 'FACTORY'
  | 'DRIVER'
  | 'VEHICLE'
  | 'RETAILER'
  | 'ORDER'
  | 'INVOICE'
  | 'ROUTE';

export interface EntityResolutionResolveRequest {
  entity_type: EntityResolutionEntityType;
  entity_id?: string;
  query?: string;
  max_candidates?: number;
}

export interface EntityResolutionExplainRequest {
  entity_type: EntityResolutionEntityType;
  entity_id: string;
}

export interface EntityResolutionCandidate {
  node_id: string;
  entity_type: EntityResolutionEntityType;
  entity_id: string;
  label: string;
  score: number;
  confidence_class: string;
  deterministic: boolean;
  reasons?: string[];
  metadata?: Record<string, string>;
}

export interface EntityResolutionResolveResult {
  scope_supplier_id: string;
  requested_type: EntityResolutionEntityType;
  query?: string;
  resolved?: EntityResolutionCandidate;
  candidates: EntityResolutionCandidate[];
}

export interface EntityResolutionGraphNode {
  node_id: string;
  entity_type: EntityResolutionEntityType;
  entity_id: string;
  label: string;
}

export interface EntityResolutionGraphEdge {
  from: string;
  to: string;
  relation: string;
  confidence: number;
}

export interface EntityResolutionGraphProjection {
  nodes: EntityResolutionGraphNode[];
  edges: EntityResolutionGraphEdge[];
}

export interface EntityResolutionExplainResult {
  scope_supplier_id: string;
  source: EntityResolutionCandidate;
  projection: EntityResolutionGraphProjection;
}

export interface EntityResolutionResolveResponse {
  status: 'ok';
  result: EntityResolutionResolveResult;
}

export interface EntityResolutionExplainResponse {
  status: 'ok';
  result: EntityResolutionExplainResult;
}
