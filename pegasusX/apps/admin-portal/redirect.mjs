#!/usr/bin/env node
/**
 * Enterprise discoverability stub for retired admin-portal.
 * Canonical surface: apps/supplier-portal (web + Tauri).
 */
const checkOnly = process.argv.includes("--check");
const target = new URL("../supplier-portal", import.meta.url).pathname;

const message = [
  "",
  "admin-portal is retired.",
  "Use supplier-portal for supplier + admin order/finance ops:",
  `  cd ${target}`,
  "  pnpm run tauri:dev   # or pnpm dev",
  "",
  "Admin-capable routes require ADMIN (or WAREHOUSE_ADMIN/FACTORY_ADMIN) JWT.",
  "",
].join("\n");

console.error(message);
process.exit(checkOnly ? 0 : 1);
