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
});
