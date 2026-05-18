'use client';

import type {
  GraphAnalyticsQueryRequest,
  GraphAnalyticsQueryResponse,
  GraphAnalyticsQueryResult,
} from '@pegasus/types';

import { apiFetchNoQueue } from '@/lib/auth';

const GRAPH_ANALYTICS_QUERY_PATH = '/v1/supplier/analytics/graph/query';

export class GraphAnalyticsRequestError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    super(`Graph analytics request failed: ${status}`);
    this.name = 'GraphAnalyticsRequestError';
    this.status = status;
    this.body = body;
  }
}

export async function querySupplierGraphAnalytics(
  input: GraphAnalyticsQueryRequest,
): Promise<GraphAnalyticsQueryResult> {
  const response = await apiFetchNoQueue(GRAPH_ANALYTICS_QUERY_PATH, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  });

  const body = await readJsonBody(response);
  if (!response.ok) {
    throw new GraphAnalyticsRequestError(response.status, body);
  }

  return readGraphAnalyticsData(body);
}

async function readJsonBody(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return null;
  }
}

function readGraphAnalyticsData(body: unknown): GraphAnalyticsQueryResult {
  if (typeof body !== 'object' || body === null) {
    throw new Error('Graph analytics response body is not a JSON object');
  }

  const envelope = body as Partial<GraphAnalyticsQueryResponse>;
  if (typeof envelope.timestamp !== 'number' || envelope.data === undefined) {
    throw new Error('Graph analytics response payload is malformed');
  }

  return envelope.data;
}
