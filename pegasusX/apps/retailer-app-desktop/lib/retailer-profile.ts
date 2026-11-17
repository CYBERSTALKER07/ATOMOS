import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";
import { isTauri } from "@pegasusx/desktop-bridge";
import type { RetailerProfile } from "./types";

export const RETAILER_PROFILE_CACHE_KEY = "retailer_profile";

let memoryProfile: RetailerProfile | null = null;

function readLegacyLocalProfile(): RetailerProfile | null {
  if (typeof localStorage === "undefined") return null;
  try {
    const raw = localStorage.getItem(RETAILER_PROFILE_CACHE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as RetailerProfile;
  } catch {
    return null;
  }
}

/** Load profile from SQLite (Tauri) or migrate legacy localStorage into cache. */
export async function initRetailerProfile(): Promise<void> {
  const cached = await cacheGet<RetailerProfile>(RETAILER_PROFILE_CACHE_KEY, {
    maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS,
  });
  if (cached?.id) {
    memoryProfile = cached;
    return;
  }
  const legacy = readLegacyLocalProfile();
  if (legacy?.id) {
    memoryProfile = legacy;
    await cacheSet(RETAILER_PROFILE_CACHE_KEY, legacy);
    if (isTauri() && typeof localStorage !== "undefined") {
      localStorage.removeItem(RETAILER_PROFILE_CACHE_KEY);
    }
  }
}

/** Synchronous read — call after `initRetailerProfile` on app boot. */
export function getRetailerProfile(): RetailerProfile | null {
  return memoryProfile ?? readLegacyLocalProfile();
}

export function getRetailerId(): string {
  return getRetailerProfile()?.id ?? "";
}

/** Persist profile to SQLite (and localStorage only on non-Tauri web dev). */
export async function setRetailerProfile(profile: RetailerProfile): Promise<void> {
  memoryProfile = profile;
  await cacheSet(RETAILER_PROFILE_CACHE_KEY, profile);
  if (!isTauri() && typeof localStorage !== "undefined") {
    try {
      localStorage.setItem(RETAILER_PROFILE_CACHE_KEY, JSON.stringify(profile));
    } catch {
      // ignore quota
    }
  } else if (isTauri() && typeof localStorage !== "undefined") {
    localStorage.removeItem(RETAILER_PROFILE_CACHE_KEY);
  }
}

export async function mergeRetailerProfile(
  patch: Partial<RetailerProfile> & { retailer_id?: string },
): Promise<void> {
  const current = getRetailerProfile();
  const id = patch.id ?? patch.retailer_id ?? current?.id ?? "";
  if (!id) return;
  const next: RetailerProfile = {
    id,
    name: patch.name ?? current?.name ?? "",
    company: patch.company ?? current?.company ?? "",
    email: patch.email ?? current?.email ?? "",
    avatar_url: patch.avatar_url ?? current?.avatar_url ?? null,
    receiving_window_open:
      patch.receiving_window_open ?? current?.receiving_window_open ?? null,
    receiving_window_close:
      patch.receiving_window_close ?? current?.receiving_window_close ?? null,
    country_code: patch.country_code ?? current?.country_code,
  };
  await setRetailerProfile(next);
}
