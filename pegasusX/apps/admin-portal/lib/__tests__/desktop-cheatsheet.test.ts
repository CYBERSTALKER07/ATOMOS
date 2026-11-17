import { describe, it, expect } from "vitest";
import * as fs from "fs";
import * as path from "path";

describe("Admin Portal Desktop Keyboard Cheatsheet & HUD Invariants", () => {
  const portalDir = path.resolve(__dirname, "../..");
  const cheatsheetPath = path.join(portalDir, "components/ui/DesktopKeyboardCheatsheet.tsx");
  const headerPath = path.join(portalDir, "components/layout/AdminHeader.tsx");
  const shellPath = path.join(portalDir, "components/layout/AdminAppShell.tsx");

  it("must provide local DesktopKeyboardCheatsheet component with NOC shortcut groups", () => {
    expect(fs.existsSync(cheatsheetPath)).toBe(true);
    const content = fs.readFileSync(cheatsheetPath, "utf-8");
    expect(content).toContain("Desktop Keyboard Shortcuts");
    expect(content).toContain("Global Navigation & Search");
    expect(content).toContain("Triad Consensus & Governance");
    expect(content).toContain("Operations, Fleet & Dead-Letter Queue");
    expect(content).toContain("System & NOC Telemetry");
    expect(content).toContain("DesktopKeyboardCheatsheetProps");
  });

  it("must configure window drag regions and telemetry pill in AdminHeader", () => {
    const headerContent = fs.readFileSync(headerPath, "utf-8");
    expect(headerContent).toContain("data-tauri-drag-region");
    expect(headerContent).toContain("data-tauri-no-drag");
    expect(headerContent).toContain("SPK-UZ: 8ms");
    expect(headerContent).toContain("SQLITE OK");
    expect(headerContent).toContain("onOpenCheatsheet");
  });

  it("must mount DesktopKeyboardCheatsheet and keyboard shortcut listeners in AdminAppShell", () => {
    const shellContent = fs.readFileSync(shellPath, "utf-8");
    expect(shellContent).toContain("DesktopKeyboardCheatsheet");
    expect(shellContent).toContain("cheatsheetOpen");
    expect(shellContent).toContain("keydown");
  });
});
