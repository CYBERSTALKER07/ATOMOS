/**
 * Extract mobile hardcoded UI strings; reuse existing catalog keys or propose
 * mobile_<role>.* / grow common.*.
 *
 * Usage: node packages/i18n/scripts/extract-mobile.mjs
 * Writes: packages/i18n/generated/mobile-extract.json
 */
import fs from "node:fs";
import path from "node:path";

import {
  flattenCatalog,
  generatedDir,
  readCatalog,
  writeFile,
} from "./shared.mjs";

const APP_DOMAIN = {
  "driver-app-android": "mobile_driver",
  "driver-app-ios": "mobile_driver",
  "factory-app-android": "mobile_factory",
  "factory-app-ios": "mobile_factory",
  "payload-app-android": "mobile_payload",
  "payload-app-ios": "mobile_payload",
  "retailer-app-android": "mobile_retailer",
  "retailer-app-ios": "mobile_retailer",
  "supplier-app-android": "mobile_supplier",
  "supplier-app-ios": "mobile_supplier",
  "warehouse-app-android": "mobile_warehouse",
  "warehouse-app-ios": "mobile_warehouse",
};

const STATUS_CODE = /^[A-Z][A-Z0-9_]{1,40}$/;
const ISO_LIKE = /^\d{4}-\d{2}-\d{2}/;
const MOSTLY_NUMERIC = /^[\d\s.,:%+/−–—\-°CFkmx×]+$/i;

function isFalsePositive(text) {
  const t = text.trim();
  if (t.length < 2 || t.length > 180) return true;
  if (STATUS_CODE.test(t) && !/[a-z]/.test(t)) return true;
  if (ISO_LIKE.test(t)) return true;
  if (MOSTLY_NUMERIC.test(t)) return true;
  if (t.startsWith("http") || t.startsWith("/") || t.startsWith("com.")) return true;
  if (/^[a-z]+[A-Z][a-zA-Z0-9]*$/.test(t)) return true; // camelCase id
  if (/^[a-z0-9]+(?:[_-][a-z0-9]+)+$/.test(t) && !/\s/.test(t)) return true;
  if (t.includes("\\(") || t.includes("${") || t.includes("$(")) return true; // interpolations
  return false;
}

function slugify(text) {
  return (
    text
      .normalize("NFKD")
      .replace(/['']/g, "")
      .replace(/&/g, " and ")
      .replace(/[^a-zA-Z0-9]+/g, "_")
      .replace(/^_+|_+$/g, "")
      .replace(/_+/g, "_")
      .toLowerCase()
      .slice(0, 64) || "text"
  );
}

function setNested(tree, dottedKey, value) {
  const parts = dottedKey.split(".");
  let cursor = tree;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    if (!cursor[part] || typeof cursor[part] === "string") cursor[part] = {};
    cursor = cursor[part];
  }
  if (typeof cursor[parts[parts.length - 1]] === "undefined") {
    cursor[parts[parts.length - 1]] = value;
  }
}

const inventoryPath = path.resolve(generatedDir, "inventory.json");
const inventory = JSON.parse(fs.readFileSync(inventoryPath, "utf8"));
const enFlat = flattenCatalog(readCatalog("en"));

const valueToKeys = new Map();
for (const [key, value] of Object.entries(enFlat)) {
  if (!valueToKeys.has(value)) valueToKeys.set(value, []);
  valueToKeys.get(value).push(key);
}

function pickExistingKey(text, domain) {
  const keys = valueToKeys.get(text);
  if (!keys?.length) return null;
  const scored = [...keys].sort((a, b) => {
    const score = (k) => {
      if (k.startsWith(`${domain}.`)) return 0;
      if (k.startsWith("common.")) return 1;
      if (k.startsWith("portal.")) return 2;
      if (k.startsWith("auth.")) return 3;
      return 4;
    };
    return score(a) - score(b) || a.length - b.length || a.localeCompare(b);
  });
  return scored[0];
}

const usedKeys = new Set(Object.keys(enFlat));
const proposals = [];
const reuse = [];
const byDomain = {};

for (const domain of Object.values(APP_DOMAIN)) {
  byDomain[domain] = { hits: 0, reuse: 0, proposed: 0, skipped: 0 };
}

for (const entry of inventory.entries) {
  if (
    entry.kind !== "swiftui-inline-text" &&
    entry.kind !== "compose-inline-text" &&
    entry.kind !== "xml-inline-text"
  ) {
    continue;
  }
  if (!entry.file.startsWith("apps/")) continue;
  const app = entry.file.split("/")[1];
  const domain = APP_DOMAIN[app];
  if (!domain) continue;

  // Skip app xml strings.xml that are only app_name etc. — still process compose/swift
  if (entry.kind === "xml-inline-text" && entry.file.includes("/res/")) {
    // keep user-visible xml text
  }

  const text = String(entry.text || "").trim();
  if (isFalsePositive(text)) {
    byDomain[domain].skipped += 1;
    continue;
  }

  byDomain[domain].hits += 1;
  const existing = pickExistingKey(text, domain);
  if (existing) {
    byDomain[domain].reuse += 1;
    reuse.push({ file: entry.file, line: entry.line, text, key: existing, kind: entry.kind });
    continue;
  }

  let key = `${domain}.ui.${slugify(text)}`;
  let n = 2;
  while (usedKeys.has(key)) {
    key = `${domain}.ui.${slugify(text)}_${n}`;
    n += 1;
  }
  usedKeys.add(key);
  byDomain[domain].proposed += 1;
  proposals.push({
    file: entry.file,
    line: entry.line,
    text,
    key,
    domain,
    kind: entry.kind,
  });
}

/** Dedupe by domain+text */
const textToKey = new Map();
const deduped = [];
for (const p of proposals) {
  const tk = `${p.domain}::${p.text}`;
  if (textToKey.has(tk)) continue;
  textToKey.set(tk, p.key);
  deduped.push(p);
}

const enAdditions = {};
for (const p of deduped) {
  setNested(enAdditions, p.key, p.text);
}

const wireMap = new Map();
function addWire(file, text, key, kind) {
  if (!wireMap.has(file)) wireMap.set(file, new Map());
  const m = wireMap.get(file);
  if (!m.has(text)) m.set(text, { key, kind });
}
for (const hit of reuse) addWire(hit.file, hit.text, hit.key, hit.kind);
for (const p of proposals) {
  const key = textToKey.get(`${p.domain}::${p.text}`) || p.key;
  addWire(p.file, p.text, key, p.kind);
}

const wireFiles = [...wireMap.entries()].map(([file, map]) => ({
  file,
  replacements: [...map.entries()].map(([text, meta]) => ({
    text,
    key: meta.key,
    kind: meta.kind,
  })),
}));

const output = {
  generated_at: new Date().toISOString(),
  summary: byDomain,
  new_key_count: deduped.length,
  reuse_count: reuse.length,
  proposals: deduped,
  reuse,
  en_additions: enAdditions,
  wire_files: wireFiles,
};

writeFile(
  path.resolve(generatedDir, "mobile-extract.json"),
  `${JSON.stringify(output, null, 2)}\n`,
);

console.log("Mobile extract summary:");
for (const [domain, s] of Object.entries(byDomain)) {
  console.log(
    `  ${domain}: hits=${s.hits} reuse=${s.reuse} proposed=${s.proposed} skipped=${s.skipped}`,
  );
}
console.log(`  unique new keys: ${deduped.length}`);
console.log(`  wire files: ${wireFiles.length}`);
