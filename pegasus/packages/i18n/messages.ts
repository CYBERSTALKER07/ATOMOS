import enCatalog from "./catalogs/en.json";
import ruCatalog from "./catalogs/ru.json";
import uzCatalog from "./catalogs/uz.json";
import { flattenCatalog, formatTranslation, getTranslation, type CatalogTree, type FlatCatalog } from "./catalog";
import { defaultLocale, resolveLocale, type Locale } from "./locales";

interface ProblemLike {
  title: string;
  detail?: string;
  message_key?: string;
}

export const messageCatalogs: Record<Locale, FlatCatalog> = {
  en: flattenCatalog(enCatalog as CatalogTree),
  ru: flattenCatalog(ruCatalog as CatalogTree),
  uz: flattenCatalog(uzCatalog as CatalogTree),
};

export function translate(
  locale: Locale,
  key: string,
  values: Record<string, string | number> = {},
): string {
  return formatTranslation(getTranslation(messageCatalogs, locale, key), values);
}

export function createTranslator(locale: string | null | undefined) {
  const resolvedLocale = resolveLocale(locale, defaultLocale);

  return (key: string, values: Record<string, string | number> = {}) =>
    translate(resolvedLocale, key, values);
}

export function translateProblemDetail(
  problem: ProblemLike,
  locale: string | null | undefined,
): string {
  if (problem.message_key) {
    const translated = translate(resolveLocale(locale, defaultLocale), problem.message_key);
    if (translated !== problem.message_key) {
      return translated;
    }
  }

  return problem.detail || problem.title;
}
