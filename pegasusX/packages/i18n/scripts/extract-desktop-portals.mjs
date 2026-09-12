/**
 * One-shot extractor for desktop portal i18n wiring.
 * Filters inventory to supplier/warehouse/factory/retailer desktop apps,
 * dedupes literals, reuses existing catalog keys when EN text matches,
 * and proposes new keys under supplier_portal / warehouse_portal /
 * factory_portal / retailer_desktop.
 *
 * Usage: node packages/i18n/scripts/extract-desktop-portals.mjs
 * Writes: packages/i18n/generated/desktop-extract.json
 */
import fs from "node:fs";
import path from "node:path";

import {
  flattenCatalog,
  generatedDir,
  readCatalog,
  repoRoot,
  writeFile,
} from "./shared.mjs";

const PORTALS = [
  {
    app: "supplier-portal",
    domain: "supplier_portal",
    roots: ["app", "components"],
  },
  {
    app: "warehouse-portal",
    domain: "warehouse_portal",
    roots: ["app", "components"],
  },
  {
    app: "factory-portal",
    domain: "factory_portal",
    roots: ["app", "components"],
  },
  {
    app: "retailer-app-desktop",
    domain: "retailer_desktop",
    roots: ["app", "components"],
  },
];

const SKIP_EXACT = new Set([
  "div",
  "span",
  "button",
  "input",
  "true",
  "false",
  "null",
  "undefined",
  "use client",
  "use server",
]);

/** Status / enum tokens product already shows as codes — leave unwired. */
const STATUS_CODE = /^[A-Z][A-Z0-9_]{1,40}$/;

const ISO_LIKE = /^\d{4}-\d{2}-\d{2}/;
const MOSTLY_NUMERIC = /^[\d\s.,:%+/−–—\-°CFkmx×]+$/i;
const CSS_OR_PATH = /^(?:[./#]|className|var\(--|https?:|@\/|[a-z]+\/[a-z])/;

function isFalsePositive(text) {
  const t = text.trim();
  if (t.length < 2 || t.length > 180) return true;
  if (SKIP_EXACT.has(t)) return true;
  if (STATUS_CODE.test(t) && !/[a-z]/.test(t)) return true;
  if (ISO_LIKE.test(t)) return true;
  if (MOSTLY_NUMERIC.test(t)) return true;
  if (CSS_OR_PATH.test(t)) return true;
  // Icon names / camelCase identifiers without spaces
  if (/^[a-z]+[A-Z][a-zA-Z0-9]*$/.test(t)) return true;
  // Pure snake_case / kebab technical ids
  if (/^[a-z0-9]+(?:[_-][a-z0-9]+)+$/.test(t) && !/\s/.test(t)) return true;
  return false;
}

function slugify(text) {
  return text
    .normalize("NFKD")
    .replace(/['']/g, "")
    .replace(/&/g, " and ")
    .replace(/[^a-zA-Z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .replace(/_+/g, "_")
    .toLowerCase()
    .slice(0, 64) || "text";
}

function pathSlug(relativeFile, app) {
  const prefix = `apps/${app}/`;
  let rest = relativeFile.startsWith(prefix)
    ? relativeFile.slice(prefix.length)
    : relativeFile;
  rest = rest
    .replace(/\([^)]+\)\//g, "") // drop route groups
    .replace(/\.(tsx|ts|jsx|js)$/, "")
    .replace(/\/page$/, "")
    .replace(/\/layout$/, "")
    .replace(/\/index$/, "")
    .replace(/^(app|components)\//, "");
  const parts = rest
    .split("/")
    .filter(Boolean)
    .map((p) =>
      p
        .replace(/([a-z])([A-Z])/g, "$1_$2")
        .replace(/[^a-zA-Z0-9]+/g, "_")
        .toLowerCase(),
    )
    .filter((p) => p && p !== "page");
  return parts.slice(0, 4).join(".") || "ui";
}

function roleForAttr(attr) {
  if (!attr) return "text";
  if (attr === "placeholder") return "placeholder";
  if (attr === "aria-label") return "aria";
  if (attr === "title") return "title";
  return "text";
}

function suggestKey(domain, pathKey, text, attr, usedKeys) {
  const base = `${domain}.${pathKey}.${roleForAttr(attr)}.${slugify(text)}`;
  let key = base;
  let n = 2;
  while (usedKeys.has(key)) {
    key = `${base}_${n}`;
    n += 1;
  }
  usedKeys.add(key);
  return key;
}

function setNested(tree, dottedKey, value) {
  const parts = dottedKey.split(".");
  let cursor = tree;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    if (!cursor[part] || typeof cursor[part] !== "object") {
      cursor[part] = {};
    }
    cursor = cursor[part];
  }
  cursor[parts[parts.length - 1]] = value;
}

const inventoryPath = path.resolve(generatedDir, "inventory.json");
if (!fs.existsSync(inventoryPath)) {
  console.error("Run inventory.mjs first.");
  process.exit(1);
}

const inventory = JSON.parse(fs.readFileSync(inventoryPath, "utf8"));
const enFlat = flattenCatalog(readCatalog("en"));

/** EN value → preferred existing key (prefer common/portal/domain keys). */
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
      if (k.startsWith("portal.")) return 1;
      if (k.startsWith("common.")) return 2;
      if (k.startsWith("auth.")) return 3;
      return 4;
    };
    return score(a) - score(b) || a.length - b.length || a.localeCompare(b);
  });
  return scored[0];
}

const usedNewKeys = new Set(Object.keys(enFlat));
const byDomain = {};
const proposals = [];
const reuseHits = [];
const skipped = [];

for (const portal of PORTALS) {
  byDomain[portal.domain] = {
    app: portal.app,
    hits: 0,
    reuse: 0,
    proposed: 0,
    skipped: 0,
  };
}

for (const entry of inventory.entries) {
  if (entry.kind !== "web-jsx-text") continue;

  const portal = PORTALS.find((p) =>
    entry.file.startsWith(`apps/${p.app}/`),
  );
  if (!portal) continue;

  const rest = entry.file.slice(`apps/${portal.app}/`.length);
  if (!portal.roots.some((r) => rest.startsWith(`${r}/`))) continue;

  const text = String(entry.text || "").trim();
  if (isFalsePositive(text)) {
    byDomain[portal.domain].skipped += 1;
    skipped.push({ ...entry, reason: "false_positive" });
    continue;
  }

  byDomain[portal.domain].hits += 1;

  const existing = pickExistingKey(text, portal.domain);
  if (existing) {
    byDomain[portal.domain].reuse += 1;
    reuseHits.push({
      file: entry.file,
      line: entry.line,
      text,
      key: existing,
    });
    continue;
  }

  // Infer attr from inventory text patterns is weak; inventory doesn't store attr.
  // Path-based key is enough for uniqueness.
  const pathKey = pathSlug(entry.file, portal.app);
  const key = suggestKey(portal.domain, pathKey, text, null, usedNewKeys);
  byDomain[portal.domain].proposed += 1;
  proposals.push({
    file: entry.file,
    line: entry.line,
    text,
    key,
    domain: portal.domain,
  });
}

/** Dedupe proposals by key — keep first EN text (stable). */
const uniqueByKey = new Map();
for (const p of proposals) {
  if (!uniqueByKey.has(p.key)) {
    uniqueByKey.set(p.key, p);
  }
}

/** Also dedupe by text within domain — reuse first key for same EN. */
const textToKey = new Map();
const dedupedProposals = [];
for (const p of uniqueByKey.values()) {
  const textKey = `${p.domain}::${p.text}`;
  if (textToKey.has(textKey)) {
    // Remap later via alias table
    continue;
  }
  textToKey.set(textKey, p.key);
  dedupedProposals.push(p);
}

/** Alias: same text in domain → canonical key (including cross-file). */
const aliases = [];
for (const p of proposals) {
  const textKey = `${p.domain}::${p.text}`;
  const canonical = textToKey.get(textKey);
  if (canonical && canonical !== p.key) {
    aliases.push({
      file: p.file,
      line: p.line,
      text: p.text,
      key: canonical,
      aliasOf: p.key,
    });
  }
}

/** Build nested EN additions only for new keys. */
const enAdditions = {};
for (const p of dedupedProposals) {
  setNested(enAdditions, p.key, p.text);
}

/** Occurrence map for wiring: file → [{text, key}] exact replacements. */
const wireMap = new Map(); // file -> Map(text -> key)

function addWire(file, text, key) {
  if (!wireMap.has(file)) wireMap.set(file, new Map());
  const m = wireMap.get(file);
  if (!m.has(text)) m.set(text, key);
}

for (const hit of reuseHits) {
  addWire(hit.file, hit.text, hit.key);
}
for (const p of proposals) {
  const textKey = `${p.domain}::${p.text}`;
  const key = textToKey.get(textKey) || p.key;
  addWire(p.file, p.text, key);
}

const wireFiles = [...wireMap.entries()].map(([file, map]) => ({
  file,
  replacements: [...map.entries()].map(([text, key]) => ({ text, key })),
}));

const output = {
  generated_at: new Date().toISOString(),
  summary: byDomain,
  new_key_count: dedupedProposals.length,
  reuse_count: reuseHits.length,
  skipped_count: skipped.length,
  proposals: dedupedProposals,
  reuse: reuseHits,
  aliases,
  en_additions: enAdditions,
  wire_files: wireFiles,
};

const outPath = path.resolve(generatedDir, "desktop-extract.json");
writeFile(outPath, `${JSON.stringify(output, null, 2)}\n`);

console.log("Desktop portal extract summary:");
for (const [domain, s] of Object.entries(byDomain)) {
  console.log(
    `  ${domain}: hits=${s.hits} reuse=${s.reuse} proposed_raw=${s.proposed} skipped=${s.skipped}`,
  );
}
console.log(`  unique new keys: ${dedupedProposals.length}`);
console.log(`  wire files: ${wireFiles.length}`);
console.log(`Wrote ${outPath}`);
