import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('react-native', () => ({}));
vi.mock('expo-secure-store', () => ({
  setItemAsync: vi.fn(),
  getItemAsync: vi.fn(),
}));
vi.mock('../authSession', () => ({
  authFetch: vi.fn(),
}));

import {
  isIdempotent409,
  processQueueItem,
  type QueuedAction,
} from '../hooks/useOfflineQueue';

describe('useOfflineQueue 409 conflict handling (S1-9)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('isIdempotent409 classifier', () => {
    it('returns true for 409 with idempotent markers', () => {
      expect(isIdempotent409(409, '{"error":"duplicate_request"}')).toBe(true);
      expect(isIdempotent409(409, '{"code":"already_exists"}')).toBe(true);
      expect(isIdempotent409(409, 'idempotent operation acknowledged')).toBe(true);
      expect(isIdempotent409(409, '{"status":"already_processed"}')).toBe(true);
    });

    it('returns false for 409 with business conflicts', () => {
      expect(isIdempotent409(409, '{"error":"manifest_already_sealed"}')).toBe(false);
      expect(isIdempotent409(409, '{"error":"invalid_state_transition"}')).toBe(false);
      expect(isIdempotent409(409, '{"error":"pallet_reassigned"}')).toBe(false);
      expect(isIdempotent409(409, '')).toBe(false);
    });

    it('returns false for non-409 status even if text contains duplicate', () => {
      expect(isIdempotent409(200, 'duplicate')).toBe(false);
      expect(isIdempotent409(422, 'duplicate')).toBe(false);
      expect(isIdempotent409(500, 'duplicate')).toBe(false);
    });
  });

  describe('processQueueItem', () => {
    const action: QueuedAction = {
      id: 'act-1',
      endpoint: '/v1/payload/seal',
      method: 'POST',
      body: '{"manifest_id":"m-123"}',
      createdAt: 1700000000000,
    };

    it('marks succeeded on HTTP 200', async () => {
      const mockFetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ok: true }), { status: 200 })
      );

      const result = await processQueueItem(action, mockFetch as any);
      expect(result.outcome).toBe('succeeded');
    });

    it('marks succeeded on HTTP 409 when idempotent replay', async () => {
      const mockFetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 'already_exists' }), { status: 409 })
      );

      const result = await processQueueItem(action, mockFetch as any);
      expect(result.outcome).toBe('succeeded');
    });

    it('marks conflict and preserves action when HTTP 409 is a business state conflict', async () => {
      const mockFetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'manifest_already_sealed_by_supervisor' }), {
          status: 409,
        })
      );

      const result = await processQueueItem(action, mockFetch as any);
      expect(result.outcome).toBe('conflict');
      if (result.outcome === 'conflict') {
        expect(result.action.status).toBe('conflict');
        expect(result.action.error).toContain('manifest_already_sealed');
        expect(result.action.id).toBe('act-1');
      }
    });

    it('marks retry on 503 or network failure', async () => {
      const mockFetch503 = vi.fn().mockResolvedValue(
        new Response('Service Unavailable', { status: 503 })
      );

      const result = await processQueueItem(action, mockFetch503 as any);
      expect(result.outcome).toBe('retry');

      const mockFetchThrow = vi.fn().mockRejectedValue(new Error('Network offline'));
      const resultThrow = await processQueueItem(action, mockFetchThrow as any);
      expect(resultThrow.outcome).toBe('retry');
    });

    it('marks dropped on non-retryable 4xx like 422', async () => {
      const mockFetch422 = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'invalid_format' }), { status: 422 })
      );

      const result = await processQueueItem(action, mockFetch422 as any);
      expect(result.outcome).toBe('dropped');
    });
  });
});
