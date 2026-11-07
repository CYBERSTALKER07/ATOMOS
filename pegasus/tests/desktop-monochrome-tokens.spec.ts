import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

test.describe('Desktop Monochrome Design System Tokens', () => {
  test('desktop-foundation.css enforces strict monochrome black, white and zinc tokens', () => {
    const cssPath = path.resolve(__dirname, '../packages/ui-kit/styles/desktop-foundation.css');
    expect(fs.existsSync(cssPath)).toBe(true);

    const content = fs.readFileSync(cssPath, 'utf-8');

    // No legacy vibrant orange accent in root tokens
    expect(content).not.toContain('--desk-accent: #ff7a1a');
    expect(content).toContain('--desk-canvas: #f5f5f5');
    expect(content).toContain('--desk-surface: #ffffff');
    expect(content).toContain('--desk-surface-raised: #ffffff');
    expect(content).toContain('--desk-accent: #000000');

    // Dark mode monochrome tokens
    expect(content).toContain('--desk-canvas: #000000');
    expect(content).toContain('--desk-surface: #0a0a0a');
    expect(content).toContain('--desk-surface-raised: #18181b');
    expect(content).toContain('--desk-accent: #ffffff');
  });

  test('void-theme.css enforces monochrome tokens without neon violet/magenta', () => {
    const cssPath = path.resolve(__dirname, '../packages/ui-kit/styles/void-theme.css');
    expect(fs.existsSync(cssPath)).toBe(true);

    const content = fs.readFileSync(cssPath, 'utf-8');

    // No legacy neon void colors
    expect(content).not.toContain('#cc66cc');
    expect(content).not.toContain('#f92aad');
    expect(content).not.toContain('#36f9f6');

    // Linear monochrome
    expect(content).toContain('--void-canvas: #000000');
    expect(content).toContain('--void-surface: #09090b');
    expect(content).toContain('--void-accent: #ffffff');
  });
});
