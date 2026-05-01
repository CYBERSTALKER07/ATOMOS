import { describe, it, expect, beforeEach, vi } from 'vitest';
import { readToken } from '../../lib/auth';

vi.mock('../../lib/bridge', () => ({
  isTauri: () => false,
  getStoredToken: vi.fn(),
  storeToken: vi.fn(),
  clearStoredToken: vi.fn(),
}));

describe('readToken', () => {
  beforeEach(() => {
    Object.defineProperty(document, 'cookie', { writable: true, value: '' });
    localStorage.clear();
  });

  it('returns empty string when no cookies or localStorage', () => {
    expect(readToken()).toBe('');
  });

  it('reads pegasus_retailer_jwt from cookie', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'pegasus_retailer_jwt=abc123',
    });
    expect(readToken()).toBe('abc123');
  });

  it('decodes URI-encoded cookie', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'pegasus_retailer_jwt=' + encodeURIComponent('tok/en+special'),
    });
    expect(readToken()).toBe('tok/en+special');
  });

  it('does not read from localStorage when no cookie', () => {
    localStorage.setItem('pegasus_retailer_jwt', 'stored-token');
    expect(readToken()).toBe('');
  });

  it('prefers cookie over localStorage', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'pegasus_retailer_jwt=cookie-val',
    });
    localStorage.setItem('pegasus_retailer_jwt', 'storage-val');
    expect(readToken()).toBe('cookie-val');
  });

  it('handles cookie among other cookies', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'theme=dark; pegasus_retailer_jwt=middle; lang=en',
    });
    expect(readToken()).toBe('middle');
  });

  it('returns empty for unrelated cookies', () => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'theme=dark; lang=en',
    });
    expect(readToken()).toBe('');
  });
});
