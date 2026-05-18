'use client';

import type {
  EntityResolutionExplainRequest,
  EntityResolutionExplainResult,
  EntityResolutionResolveRequest,
  EntityResolutionResolveResult,
} from '@pegasus/types';

import { apiFetchNoQueue } from '@/lib/auth';

const ENTITY_RESOLUTION_BASE = '/v1/supplier/entity-resolution';

interface EntityResolutionEnvelope<T> {
  status: 'ok';
  result: T;
}

export class EntityResolutionRequestError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    super(`Entity resolution request failed: ${status}`);
    this.name = 'EntityResolutionRequestError';
    this.status = status;
    this.body = body;
  }
}

export async function resolveSupplierEntity(
  input: EntityResolutionResolveRequest,
): Promise<EntityResolutionResolveResult> {
  const payload: EntityResolutionResolveRequest = {
    entity_type: input.entity_type,
    ...(input.entity_id ? { entity_id: input.entity_id } : {}),
    ...(input.query ? { query: input.query } : {}),
    ...(typeof input.max_candidates === 'number' ? { max_candidates: input.max_candidates } : {}),
  };

  return postEntityResolution<EntityResolutionResolveResult>(
    `${ENTITY_RESOLUTION_BASE}/resolve`,
    payload,
  );
}

export async function explainSupplierEntity(
  input: EntityResolutionExplainRequest,
): Promise<EntityResolutionExplainResult> {
  return postEntityResolution<EntityResolutionExplainResult>(
    `${ENTITY_RESOLUTION_BASE}/explain`,
    {
      entity_type: input.entity_type,
      entity_id: input.entity_id,
    },
  );
}

async function postEntityResolution<T>(path: string, payload: unknown): Promise<T> {
  const response = await apiFetchNoQueue(path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const body = await readJsonBody(response);

  if (!response.ok) {
    throw new EntityResolutionRequestError(response.status, body);
  }

  return readEnvelopeResult<T>(body);
}

async function readJsonBody(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return null;
  }
}

function readEnvelopeResult<T>(body: unknown): T {
  if (typeof body !== 'object' || body === null) {
    throw new Error('Entity resolution response body is not a JSON object');
  }

  const envelope = body as Partial<EntityResolutionEnvelope<T>>;
  if (envelope.status !== 'ok' || envelope.result === undefined) {
    throw new Error('Entity resolution response payload is malformed');
  }

  return envelope.result;
}
