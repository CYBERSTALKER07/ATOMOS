import { describe, it, expect, beforeEach } from 'vitest';
import { isTauri } from '../bridge';

describe('isTauri', () => {
  beforeEach(() => {
    // Clean up TAURI flag
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    delete (window as any).__TAURI_INTERNALS__;
  });

  it('returns false in jsdom (no Tauri runtime)', () => {
    expect(isTauri()).toBe(false);
  });

  it('returns true when __TAURI_INTERNALS__ is present', () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).__TAURI_INTERNALS__ = {};
    expect(isTauri()).toBe(true);
  });

  it('returns false when __TAURI_INTERNALS__ is falsy', () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).__TAURI_INTERNALS__ = undefined;
    expect(isTauri()).toBe(false);
  });
});
