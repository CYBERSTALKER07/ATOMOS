import { describe, it, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';

describe('UI Independence & Zero Shared UI Packages Contract (payload-terminal)', () => {
  it('package.json must not depend on shared UI packages', () => {
    const pkgJsonPath = path.resolve(__dirname, '../package.json');
    const pkg = JSON.parse(fs.readFileSync(pkgJsonPath, 'utf8'));
    const allDeps = {
      ...pkg.dependencies,
      ...pkg.devDependencies,
    };

    const forbiddenPackages = [
      '@pegasusx/ui-kit',
      '@pegasusx/explain-ui',
      '@pegasusx/pulse-ui',
      '@pegasusx/ui-charts',
      '@pegasusx/ui-maps',
      '@pegasusx/motion-tokens',
      'mobile-android-design',
      'mobile-android-kit',
      'mobile-ios-design',
      'mobile-ios-kit',
    ];

    for (const forbidden of forbiddenPackages) {
      expect(allDeps[forbidden], `Found forbidden dependency ${forbidden}`).toBeUndefined();
    }
  });

  it('source files must not import from shared UI packages', () => {
    const rootDir = path.resolve(__dirname, '..');
    const checkDir = (dir: string) => {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        if (entry.name === 'node_modules' || entry.name === '.git' || entry.name === '__tests__') {
          continue;
        }
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
          checkDir(fullPath);
        } else if (/\.(tsx?|jsx?)$/.test(entry.name)) {
          const content = fs.readFileSync(fullPath, 'utf8');
          expect(content).not.toMatch(/@pegasusx\/(ui-kit|explain-ui|pulse-ui|ui-charts|ui-maps|motion-tokens)/);
          expect(content).not.toMatch(/mobile-android-(design|kit)/);
          expect(content).not.toMatch(/mobile-ios-(design|kit|core)/);
        }
      }
    };
    checkDir(rootDir);
  });

  it('local native UI components and theme tokens must exist', () => {
    const rootDir = path.resolve(__dirname, '..');
    expect(fs.existsSync(path.join(rootDir, 'theme.ts'))).toBe(true);
    expect(fs.existsSync(path.join(rootDir, 'global.css'))).toBe(true);
    expect(fs.existsSync(path.join(rootDir, 'components/PayloadStatePanel.tsx'))).toBe(true);
    expect(fs.existsSync(path.join(rootDir, 'components/StatusBadge.tsx'))).toBe(true);
  });
});
