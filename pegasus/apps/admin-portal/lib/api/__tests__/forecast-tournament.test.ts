import { beforeEach, describe, expect, it, vi } from 'vitest';

const { apiFetchNoQueueMock } = vi.hoisted(() => ({
  apiFetchNoQueueMock: vi.fn(),
}));

vi.mock('@/lib/auth', () => ({
  apiFetchNoQueue: apiFetchNoQueueMock,
}));

import {
  ForecastTournamentRequestError,
  querySupplierForecastTournament,
} from '../forecast-tournament';

describe('forecast tournament api helper', () => {
  beforeEach(() => {
    apiFetchNoQueueMock.mockReset();
  });

  it('returns tournament data from analytics envelope', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          timestamp: 1716076800,
          data: {
            window_from: '2026-04-01T00:00:00Z',
            window_to: '2026-05-01T00:00:00Z',
            scope_supplier_id: 'sup-1',
            min_sample_size: 10,
            champion_variant: 'SKU_MEDIAN_V3',
            champion_score: 0.875,
            total_predictions: 120,
            variants: [
              {
                variant_key: 'SKU_MEDIAN_V3',
                label: 'SKU Median v3',
                predictions: 64,
                fired: 35,
                rejected: 5,
                waiting: 20,
                dormant: 4,
                conversion_rate: 0.875,
                score: 0.875,
                champion_eligible: true,
                avg_predicted_amount: 1200000,
                total_predicted_amount: 76800000,
              },
            ],
            data_sources: ['AIPredictions'],
            generated_at: '2026-05-19T13:00:00Z',
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );

    const result = await querySupplierForecastTournament({
      min_sample_size: 10,
    });

    expect(result.champion_variant).toBe('SKU_MEDIAN_V3');
    expect(apiFetchNoQueueMock).toHaveBeenCalledWith(
      '/v1/supplier/analytics/forecast/tournament',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('throws structured request error for non-2xx responses', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(JSON.stringify({ error: 'invalid_request' }), {
        status: 422,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    await expect(
      querySupplierForecastTournament({ min_sample_size: 999 }),
    ).rejects.toBeInstanceOf(ForecastTournamentRequestError);

    await expect(
      querySupplierForecastTournament({ min_sample_size: 999 }),
    ).rejects.toMatchObject({ status: 422 });
  });

  it('throws on malformed analytics envelope', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(JSON.stringify({ data: {} }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    await expect(querySupplierForecastTournament({})).rejects.toThrow(
      'Forecast tournament response payload is malformed',
    );
  });
});
