import { describe, it, expect } from 'vitest';
import { decodeJwtPayload, parseFactoryLiveEvent, FactoryLiveEvent } from '../auth';

describe('auth.ts utilities', () => {
  describe('decodeJwtPayload', () => {
    it('decodes a valid JWT payload', () => {
      // Mock JWT format: header.payload.signature
      const payload = { sub: 'user123', role: 'factory_admin' };
      const encodedPayload = btoa(JSON.stringify(payload));
      const fakeToken = `header.${encodedPayload}.signature`;

      const decoded = decodeJwtPayload(fakeToken);
      expect(decoded).toEqual(payload);
    });

    it('returns null for invalid JWT formats', () => {
      expect(decodeJwtPayload('invalid-token')).toBeNull();
      expect(decodeJwtPayload('header.invalid_payload_not_base64.signature')).toBeNull();
    });
  });

  describe('parseFactoryLiveEvent', () => {
    it('parses valid factory live events', () => {
      const validPayload = JSON.stringify({
        type: 'FACTORY_SUPPLY_REQUEST_UPDATE',
        requestId: '123'
      });

      const parsed = parseFactoryLiveEvent(validPayload);
      expect(parsed).not.toBeNull();
      expect(parsed?.type).toBe('FACTORY_SUPPLY_REQUEST_UPDATE');
      expect((parsed as any).requestId).toBe('123');
    });

    it('returns null for unknown event types', () => {
      const invalidPayload = JSON.stringify({
        type: 'UNKNOWN_EVENT_TYPE',
        requestId: '123'
      });

      expect(parseFactoryLiveEvent(invalidPayload)).toBeNull();
    });

    it('returns null for invalid JSON', () => {
      expect(parseFactoryLiveEvent('{ invalid: json }')).toBeNull();
    });
  });
});
