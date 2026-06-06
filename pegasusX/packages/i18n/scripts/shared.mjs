import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const supportedLocales = ["en", "ru", "uz"];
export const defaultLocale = "en";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
export const repoRoot = path.resolve(scriptDir, "../../../");
export const packageRoot = path.resolve(repoRoot, "packages/i18n");
export const catalogsDir = path.resolve(packageRoot, "catalogs");
export const generatedDir = path.resolve(packageRoot, "generated");

export function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

export function writeFile(filePath, content) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, content);
}

export function flattenCatalog(tree, prefix = "", entries = {}) {
  for (const [key, value] of Object.entries(tree)) {
    const nextKey = prefix ? `${prefix}.${key}` : key;

    if (typeof value === "string") {
      entries[nextKey] = value;
      continue;
    }

    flattenCatalog(value, nextKey, entries);
  }

  return entries;
}

export function readCatalog(locale) {
  return readJSON(path.resolve(catalogsDir, `${locale}.json`));
}

export function readFlatCatalog(locale) {
  return flattenCatalog(readCatalog(locale));
}

export function readAllFlatCatalogs() {
  return Object.fromEntries(
    supportedLocales.map((locale) => [locale, readFlatCatalog(locale)]),
  );
}

export function collectPlaceholders(value) {
  return [...value.matchAll(/\{([a-zA-Z0-9_]+)\}/g)]
    .map((match) => match[1])
    .sort();
}

export function toAndroidName(key) {
  return key.replace(/[^a-zA-Z0-9]+/g, "_").replace(/^_+|_+$/g, "");
}

export function escapeXml(value) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, '\\"')
    .replace(/'/g, "\\'");
}

export function escapeIos(value) {
  return value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

export function validateCatalogs(flatCatalogs) {
  const baseCatalog = flatCatalogs[defaultLocale];
  const baseKeys = Object.keys(baseCatalog).sort();
  const androidNames = new Map();

  for (const [locale, catalog] of Object.entries(flatCatalogs)) {
    const localeKeys = Object.keys(catalog).sort();
    const missingKeys = baseKeys.filter((key) => !(key in catalog));
    const extraKeys = localeKeys.filter((key) => !(key in baseCatalog));

    if (missingKeys.length > 0 || extraKeys.length > 0) {
      throw new Error(
        `${locale} catalog shape mismatch. Missing: ${missingKeys.join(", ") || "none"}. Extra: ${extraKeys.join(", ") || "none"}.`,
      );
    }

    for (const key of baseKeys) {
      const basePlaceholders = collectPlaceholders(baseCatalog[key]);
      const localePlaceholders = collectPlaceholders(catalog[key]);

      if (basePlaceholders.join(",") !== localePlaceholders.join(",")) {
        throw new Error(
          `${locale} placeholders mismatch for ${key}. Expected ${basePlaceholders.join(", ") || "none"}, received ${localePlaceholders.join(", ") || "none"}.`,
        );
      }

      const androidName = toAndroidName(key);
      const previousKey = androidNames.get(androidName);

      if (previousKey && previousKey !== key) {
        throw new Error(
          `Android string key collision: ${previousKey} and ${key} both map to ${androidName}.`,
        );
      }

      androidNames.set(androidName, key);
    }
  }

  return {
    keyCount: baseKeys.length,
    keys: baseKeys,
  };
}
