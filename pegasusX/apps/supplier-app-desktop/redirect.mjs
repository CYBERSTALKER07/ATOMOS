#!/usr/bin/env node
/**
 * Enterprise discoverability stub for supplier-app-desktop.
 * Canonical desktop: apps/supplier-portal (Next.js + Tauri 2).
 */
const checkOnly = process.argv.includes("--check");
const target = new URL("../supplier-portal", import.meta.url).pathname;

const message = [
  "",
  "supplier-app-desktop has no separate codebase.",
  "Canonical desktop surface:",
  `  cd ${target}`,
  "  pnpm install",
  "  pnpm run tauri:dev      # desktop",
  "  pnpm run tauri:build    # release",
  "",
  "API default: http://localhost:8180",
  "Native row clients: supplier-app-ios, supplier-app-android",
  "",
].join("\n");

console.error(message);
process.exit(checkOnly ? 0 : 1);
