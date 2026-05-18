'use client';

import type {
  ForecastTournamentQueryRequest,
  ForecastTournamentQueryResponse,
  ForecastTournamentResult,
} from '@pegasus/types';

import { apiFetchNoQueue } from '@/lib/auth';

const FORECAST_TOURNAMENT_PATH = '/v1/supplier/analytics/forecast/tournament';

export class ForecastTournamentRequestError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    super(`Forecast tournament request failed: ${status}`);
    this.name = 'ForecastTournamentRequestError';
    this.status = status;
    this.body = body;
  }
}

export async function querySupplierForecastTournament(
  input: ForecastTournamentQueryRequest,
): Promise<ForecastTournamentResult> {
  const response = await apiFetchNoQueue(FORECAST_TOURNAMENT_PATH, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  });

  const body = await readJsonBody(response);
  if (!response.ok) {
    throw new ForecastTournamentRequestError(response.status, body);
  }

  return readForecastTournamentData(body);
}

async function readJsonBody(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return null;
  }
}

function readForecastTournamentData(body: unknown): ForecastTournamentResult {
  if (typeof body !== 'object' || body === null) {
    throw new Error('Forecast tournament response body is not a JSON object');
  }

  const envelope = body as Partial<ForecastTournamentQueryResponse>;
  if (typeof envelope.timestamp !== 'number' || envelope.data === undefined) {
    throw new Error('Forecast tournament response payload is malformed');
  }

  return envelope.data;
}
