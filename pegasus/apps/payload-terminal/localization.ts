import * as SecureStore from "expo-secure-store";

import { createTranslator, translateProblemDetail } from "../../packages/i18n/messages";
import { defaultLocale, resolveLocale, type Locale } from "../../packages/i18n/locales";

const localeStorageKey = "payload_terminal_locale";

interface ProblemLike {
  title: string;
  detail?: string;
  message_key?: string;
}

function isProblemLike(value: unknown): value is ProblemLike {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const problem = value as Record<string, unknown>;

  return (
    typeof problem.title === "string" &&
    (problem.detail === undefined || typeof problem.detail === "string") &&
    (problem.message_key === undefined || typeof problem.message_key === "string")
  );
}

function detectDeviceLocale(): Locale {
  try {
    return resolveLocale(Intl.DateTimeFormat().resolvedOptions().locale);
  } catch {
    return defaultLocale;
  }
}

export async function resolvePayloadLocale(): Promise<Locale> {
  const storedLocale = await SecureStore.getItemAsync(localeStorageKey);
  const locale = storedLocale ? resolveLocale(storedLocale) : detectDeviceLocale();

  if (!storedLocale) {
    await SecureStore.setItemAsync(localeStorageKey, locale);
  }

  return locale;
}

export function getPayloadTranslator(locale: Locale) {
  return createTranslator(locale);
}

export async function extractProblemMessage(
  response: Response,
  locale: Locale,
): Promise<string> {
  const contentType = response.headers.get("Content-Type") || "";

  if (contentType.includes("application/problem+json")) {
    try {
      const problem = await response.clone().json();
      if (isProblemLike(problem)) {
        return translateProblemDetail(problem, locale);
      }
    } catch {
      // Fall through to plain-text fallback.
    }
  }

  try {
    const text = (await response.clone().text()).trim();
    if (text.length > 0) {
      return text;
    }
  } catch {
    // Fall back to the HTTP status below.
  }

  return `HTTP ${response.status}`;
}
