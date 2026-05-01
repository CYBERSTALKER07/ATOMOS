import { describe, it, expect, beforeEach, vi } from 'vitest';
import { readTokenFromCookie } from '../auth';

// bridge is mocked so isTauri() returns false
vi.mock('../bridge', () => ({
  isTauri: () => false,
  getStoredToken: vi.fn(),
  storeToken: vi.fn(),
  clearStoredToken: vi.fn(),
}));

describe('readTokenFromCookie', () => {
  beforeEach(() => {
    // Reset all cookies
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: '',
    });
  });

  it('returns empty string when no cookies', () => {
    expect(readTokenFromCookie()).toBe('');
  });

  it('reads pegasus_admin_jwt cookie', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'pegasus_admin_jwt=abc123',
    });
    expect(readTokenFromCookie()).toBe('abc123');
  });

  it('reads pegasus_supplier_jwt when pegasus_admin_jwt absent', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'pegasus_supplier_jwt=sup456',
    });
    expect(readTokenFromCookie()).toBe('sup456');
  });

  it('prefers pegasus_admin_jwt over pegasus_supplier_jwt', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'pegasus_admin_jwt=admin1; pegasus_supplier_jwt=sup2',
    });
    expect(readTokenFromCookie()).toBe('admin1');
  });

  it('decodes URI-encoded token', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'pegasus_admin_jwt=' + encodeURIComponent('tok/en+val=ue'),
    });
    expect(readTokenFromCookie()).toBe('tok/en+val=ue');
  });

  it('handles cookie among other cookies', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'theme=dark; pegasus_admin_jwt=middle; lang=en',
    });
    expect(readTokenFromCookie()).toBe('middle');
  });

  it('returns empty for unrelated cookies', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'theme=dark; lang=en',
    });
    expect(readTokenFromCookie()).toBe('');
  });
});
