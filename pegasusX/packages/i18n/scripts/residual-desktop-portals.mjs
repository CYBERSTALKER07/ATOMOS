/**
 * Second-pass residual extract + catalog merge + wire for desktop portals.
 * Consumes packages/i18n/generated/desktop-residual.json (unique leftover UI strings).
 *
 * Usage: node packages/i18n/scripts/residual-desktop-portals.mjs
 */
import fs from "node:fs";
import path from "node:path";

import {
  catalogsDir,
  flattenCatalog,
  generatedDir,
  readCatalog,
  supportedLocales,
  writeFile,
} from "./shared.mjs";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));

const APP_DOMAIN = {
  "supplier-portal": "supplier_portal",
  "warehouse-portal": "warehouse_portal",
  "factory-portal": "factory_portal",
  "retailer-app-desktop": "retailer_desktop",
};

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

function deepMerge(target, source) {
  for (const [key, value] of Object.entries(source)) {
    if (value && typeof value === "object" && !Array.isArray(value)) {
      if (!target[key] || typeof target[key] !== "object") target[key] = {};
      deepMerge(target[key], value);
    } else if (typeof target[key] === "undefined") {
      target[key] = value;
    }
  }
  return target;
}

const residualPath = path.resolve(generatedDir, "desktop-residual.json");
const residual = JSON.parse(fs.readFileSync(residualPath, "utf8"));

const catalogs = Object.fromEntries(
  supportedLocales.map((l) => [l, readCatalog(l)]),
);
const enFlat = flattenCatalog(catalogs.en);
const ruFlat = flattenCatalog(catalogs.ru);
const uzFlat = flattenCatalog(catalogs.uz);

const valueToKey = new Map();
for (const [key, value] of Object.entries(enFlat)) {
  if (!valueToKey.has(value)) valueToKey.set(value, key);
}

const usedKeys = new Set(Object.keys(enFlat));
const enAdditions = {};
const wireByFile = new Map();
let reuse = 0;
let proposed = 0;

function addWire(file, text, key) {
  if (!wireByFile.has(file)) wireByFile.set(file, new Map());
  wireByFile.get(file).set(text, key);
}

for (const [app, payload] of Object.entries(residual.by_app || {})) {
  const domain = APP_DOMAIN[app];
  if (!domain) continue;

  const textToKey = new Map();
  for (const text of payload.unique || []) {
    const existing = valueToKey.get(text);
    if (existing) {
      textToKey.set(text, existing);
      reuse += 1;
      continue;
    }
    let key = `${domain}.residual.text.${slugify(text)}`;
    let n = 2;
    while (usedKeys.has(key)) {
      key = `${domain}.residual.text.${slugify(text)}_${n}`;
      n += 1;
    }
    usedKeys.add(key);
    textToKey.set(text, key);
    setNested(enAdditions, key, text);
    // draft via existing locale maps or passthrough (patch later if needed)
    const ru = ruFlat[existing] || text; // existing unused
    void ru;
    proposed += 1;
  }

  for (const fileEntry of payload.files || []) {
    for (const text of fileEntry.texts || []) {
      const key = textToKey.get(text);
      if (!key) continue;
      addWire(fileEntry.file, text, key);
    }
  }
}

// Draft ru/uz for new keys using reverse phrase map + passthrough
const phraseRu = new Map();
const phraseUz = new Map();
for (const [key, en] of Object.entries(enFlat)) {
  if (ruFlat[key] && ruFlat[key] !== en) phraseRu.set(en, ruFlat[key]);
  if (uzFlat[key] && uzFlat[key] !== en) phraseUz.set(en, uzFlat[key]);
}

function draft(en, map) {
  return map.get(en) || en;
}

const flatNew = flattenCatalog(enAdditions);
for (const [key, en] of Object.entries(flatNew)) {
  setNested(catalogs.ru, key, draft(en, phraseRu));
  setNested(catalogs.uz, key, draft(en, phraseUz));
}

deepMerge(catalogs.en, enAdditions);

for (const locale of supportedLocales) {
  writeFile(
    path.resolve(catalogsDir, `${locale}.json`),
    `${JSON.stringify(catalogs[locale], null, 2)}\n`,
  );
}

const wireFiles = [...wireByFile.entries()].map(([file, map]) => ({
  file,
  replacements: [...map.entries()].map(([text, key]) => ({ text, key })),
}));

const residualExtract = {
  generated_at: new Date().toISOString(),
  new_key_count: Object.keys(flatNew).length,
  reuse_count: reuse,
  proposed_count: proposed,
  wire_files: wireFiles,
  en_additions: enAdditions,
};

writeFile(
  path.resolve(generatedDir, "desktop-residual-extract.json"),
  `${JSON.stringify(residualExtract, null, 2)}\n`,
);

console.log(
  `Residual merge: new keys=${residualExtract.new_key_count} reuse=${reuse} wire_files=${wireFiles.length}`,
);

// Invoke wire against residual extract by temporarily swapping desktop-extract wire_files
const mainExtractPath = path.resolve(generatedDir, "desktop-extract.json");
const mainExtract = JSON.parse(fs.readFileSync(mainExtractPath, "utf8"));
const backup = structuredClone(mainExtract);
mainExtract.wire_files = wireFiles;
fs.writeFileSync(mainExtractPath, `${JSON.stringify(mainExtract, null, 2)}\n`);

const wireResult = spawnSync(
  process.execPath,
  [path.resolve(scriptDir, "wire-desktop-portals.mjs")],
  { stdio: "inherit", cwd: path.resolve(scriptDir, "../../../") },
);

// restore proposals metadata but keep? Actually restore full backup then
// we don't need old wire_files for anything critical — leave residual wire_files
// merged into backup for audit trail
backup.residual_wire_files = wireFiles;
backup.residual_new_key_count = residualExtract.new_key_count;
fs.writeFileSync(mainExtractPath, `${JSON.stringify(backup, null, 2)}\n`);

if (wireResult.status !== 0) {
  process.exit(wireResult.status ?? 1);
}
