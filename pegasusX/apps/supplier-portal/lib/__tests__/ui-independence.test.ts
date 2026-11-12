import { describe, it, expect } from "vitest";
import * as fs from "fs";
import * as path from "path";

describe("Supplier Portal UI Independence & Localization Gate", () => {
  const portalDir = path.resolve(__dirname, "../..");
  const pkgJsonPath = path.join(portalDir, "package.json");
  const globalsCssPath = path.join(portalDir, "app/globals.css");

  it("must not declare any shared UI package in package.json", () => {
    const pkg = JSON.parse(fs.readFileSync(pkgJsonPath, "utf-8"));
    const deps = { ...pkg.dependencies, ...pkg.devDependencies };
    const forbidden = [
      "@pegasusx/ui-kit",
      "@pegasus/ui-kit",
      "@pegasusx/pulse-ui",
      "@pegasusx/explain-ui",
      "@pegasusx/ui-maps",
      "@pegasusx/ui-charts",
      "@pegasusx/motion-tokens",
    ];
    for (const f of forbidden) {
      expect(deps[f], `Forbidden shared UI dependency found: ${f}`).toBeUndefined();
    }
  });

  it("must not import from shared UI packages in globals.css", () => {
    const css = fs.readFileSync(globalsCssPath, "utf-8");
    expect(css).not.toContain("@pegasusx/ui-kit");
    expect(css).not.toContain("@pegasus/ui-kit");
  });

  it("must have local styles, motion tokens, and native UI components", () => {
    const primitivesExist = fs.existsSync(path.join(portalDir, "components/ui/PortalPrimitives.tsx"));
    const motionExists = fs.existsSync(path.join(portalDir, "components/ui/motion.ts"));
    const explainExists = fs.existsSync(path.join(portalDir, "components/ui/ExplainStatusBanner.tsx"));
    const pulseExists = fs.existsSync(path.join(portalDir, "components/ui/PulseTimeline.tsx"));
    const fleetMapExists = fs.existsSync(path.join(portalDir, "components/ui/GenericFleetLiveMap.tsx"));
    const chartsExists = fs.existsSync(path.join(portalDir, "components/ui/LiveEKGNetworkGraph.tsx"));
    const foundationCssExists = fs.existsSync(path.join(portalDir, "app/styles/desktop-foundation.css"));

    expect(primitivesExist).toBe(true);
    expect(motionExists).toBe(true);
    expect(explainExists).toBe(true);
    expect(pulseExists).toBe(true);
    expect(fleetMapExists).toBe(true);
    expect(chartsExists).toBe(true);
    expect(foundationCssExists).toBe(true);
  });
});
