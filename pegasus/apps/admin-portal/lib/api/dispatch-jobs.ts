'use client';

import type { DispatchJobActiveListResponse, DispatchJobProjection } from '@pegasus/types';

import { apiFetchNoQueue } from '@/lib/auth';

const DISPATCH_JOBS_BASE = '/v1/supplier/dispatch/jobs';

export class DispatchJobsRequestError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    super(`Dispatch jobs request failed: ${status}`);
    this.name = 'DispatchJobsRequestError';
    this.status = status;
    this.body = body;
  }
}

export async function listActiveDispatchJobs(): Promise<DispatchJobActiveListResponse> {
  return getDispatchJobs<DispatchJobActiveListResponse>(`${DISPATCH_JOBS_BASE}/active`);
}

export async function getDispatchJobProjection(jobId: string): Promise<DispatchJobProjection> {
  return getDispatchJobs<DispatchJobProjection>(`${DISPATCH_JOBS_BASE}/${encodeURIComponent(jobId)}/projection`);
}

async function getDispatchJobs<T>(path: string): Promise<T> {
  const response = await apiFetchNoQueue(path, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  const body = await readJsonBody(response);
  if (!response.ok) {
    throw new DispatchJobsRequestError(response.status, body);
  }

  return body as T;
}

async function readJsonBody(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return null;
  }
}