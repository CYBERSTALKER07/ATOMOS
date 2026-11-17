import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const supportedLocales = ["en", "ru", "uz"];
export const defaultLocale = "uz";

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

const ALLOWED_MULTIWORD_TECH_LITERALS = new Set([
  "+998 XX XXX XX XX",
  "Aziz Karimov",
  "EHF ID",
  "Empathy Engine",
  "Global Pay",
  "HTTP {status}",
  "POST /partner/v1/oauth/token",
  "Pegasus Beverages Ltd.",
  "Pegasus Hub",
  "Pegasus · v1.0.0",
  "SFTP + EDI",
  "SWIFT / BIC",
  "EAN / GTIN",
  "ETA {formatetatime}",
  "EAN {barcode}",
  "EAN {code}",
  "SKU: {productid}",
  "SKU: {adjustingsku}",
]);

const ENGLISH_SENTENCE_STOPWORDS =
  /\b(the|and|with|for|from|not|are|this|that|your|you|click|enter|select|loading|failed|error|available|cannot|without|before|after|between|during|through|unable|invalid|missing|loaded|selected|created|updated|deleted|configured|required|optional|please)\b/i;

const ALLOWED_TECHNICAL_MIX_KEYS = new Set([
  // Contains payment provider brand "Click Up"
  "supplier_portal.configuration.system.gateway.click_up_merchant_id",
  // Shows literal target expression examples `retailer:123` / `city:Tashkent`
  "supplier_portal.demand.signals.text.e_g_retailer_123_or_city_tashkent",
  // Keyboard key `Enter`
  "warehouse_portal.returns.text.scan_or_type_wedge_scanner_enter",
]);

function stripTemplateTokens(value) {
  return value.replace(/\{[a-zA-Z0-9_]+\}/g, "").replace(/\$[a-zA-Z0-9_]+/g, "");
}

function hasUntranslatedMultiWordEnglish(value) {
  const cleaned = stripTemplateTokens(value).trim();
  if (!cleaned || ALLOWED_MULTIWORD_TECH_LITERALS.has(value)) {
    return false;
  }
  const hasMultipleAsciiWords = /\b[a-zA-Z]{2,}\s+[a-zA-Z]{2,}\b/.test(cleaned);
  return hasMultipleAsciiWords && ENGLISH_SENTENCE_STOPWORDS.test(cleaned);
}

export function validateCatalogs(flatCatalogs) {
  if (defaultLocale !== "uz") {
    throw new Error(`Expected canonical defaultLocale to be "uz", received "${defaultLocale}".`);
  }

  const baseCatalog = flatCatalogs[defaultLocale];
  const baseKeys = Object.keys(baseCatalog).sort();
  const androidNames = new Map();
  const enCatalog = flatCatalogs.en;

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
      const value = catalog[key];
      if (typeof value !== "string" || value.trim().length === 0) {
        throw new Error(`${locale} catalog contains an empty translation for ${key}.`);
      }

      if (locale === "uz" && /[а-яА-ЯёЁ]/.test(value)) {
        throw new Error(`uz catalog contains Cyrillic characters for ${key}: "${value}".`);
      }

      if ((locale === "uz" || locale === "ru") && !ALLOWED_TECHNICAL_MIX_KEYS.has(key)) {
        const enValue = enCatalog?.[key] ?? "";
        const strippedEn = stripTemplateTokens(enValue).trim();
        if (
          value === enValue &&
          !ALLOWED_MULTIWORD_TECH_LITERALS.has(enValue) &&
          /\b[a-zA-Z]{2,}\s+[a-zA-Z]{2,}\b/.test(strippedEn)
        ) {
          throw new Error(
            `${locale} catalog has untranslated multi-word English fallback for ${key}: "${value}".`,
          );
        }
        if (hasUntranslatedMultiWordEnglish(value)) {
          throw new Error(
            `${locale} catalog has partial English sentence leakage for ${key}: "${value}".`,
          );
        }
      }

      const basePlaceholders = collectPlaceholders(baseCatalog[key]);
      const localePlaceholders = collectPlaceholders(value);

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
