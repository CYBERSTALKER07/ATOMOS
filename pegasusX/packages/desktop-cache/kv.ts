import { withDatabase } from "./db";

const WEB_PREFIX = "pegasus_kv_cache:";

/** Default max age for live API cache hydration (15 minutes). */
export const DEFAULT_CACHE_MAX_AGE_MS = 15 * 60 * 1000;

export type CacheGetOptions = {
  /** When set, values older than this many ms are treated as a miss (and deleted). */
  maxAgeMs?: number;
};

type StoredEnvelope = { v: unknown; t: number };

function webGet(key: string): string | null {
  if (typeof localStorage === "undefined") return null;
  try {
    return localStorage.getItem(WEB_PREFIX + key);
  } catch {
    return null;
  }
}

function webSet(key: string, value: string): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(WEB_PREFIX + key, value);
  } catch {
    // quota
  }
}

function webDelete(key: string): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.removeItem(WEB_PREFIX + key);
  } catch {
    // ignore
  }
}

function webClearAll(): void {
  if (typeof localStorage === "undefined") return;
  try {
    const doomed: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (k?.startsWith(WEB_PREFIX)) doomed.push(k);
    }
    for (const k of doomed) localStorage.removeItem(k);
  } catch {
    // ignore
  }
}

function webClearPrefix(prefix: string): void {
  if (typeof localStorage === "undefined") return;
  try {
    const full = WEB_PREFIX + prefix;
    const doomed: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (k?.startsWith(full)) doomed.push(k);
    }
    for (const k of doomed) localStorage.removeItem(k);
  } catch {
    // ignore
  }
}

function parseStored(raw: string): { value: unknown; updatedAt: number } | null {
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (
      parsed &&
      typeof parsed === "object" &&
      "v" in parsed &&
      "t" in parsed &&
      typeof (parsed as StoredEnvelope).t === "number"
    ) {
      const env = parsed as StoredEnvelope;
      return { value: env.v, updatedAt: env.t };
    }
    // Legacy bare JSON (no timestamp) — unknown age; treat as updatedAt 0.
    return { value: parsed, updatedAt: 0 };
  } catch {
    return null;
  }
}

function isStale(updatedAt: number, maxAgeMs: number | undefined): boolean {
  if (maxAgeMs == null || maxAgeMs < 0) return false;
  if (updatedAt <= 0) return true;
  return Date.now() - updatedAt > maxAgeMs;
}

/** Build a tenant-/user-scoped cache key (never cache by raw URL alone). */
export function scopedCacheKey(scope: string, key: string): string {
  const s = scope.trim() || "anon";
  return `${s}::${key}`;
}

/** Read a JSON blob by cache key. Uses SQLite on Tauri, localStorage on web dev. */
export async function cacheGet<T>(
  key: string,
  opts: CacheGetOptions = {},
): Promise<T | null> {
  const maxAgeMs = opts.maxAgeMs;

  const fromDb = await withDatabase(async (db) => {
    const rows = await db.select<{ value: string; updated_at: number }[]>(
      "SELECT value, updated_at FROM kv_cache WHERE cache_key = ?1",
      [key],
    );
    const row = rows[0];
    if (!row) return null;
    return { raw: row.value, updatedAt: row.updated_at };
  });

  let value: unknown = null;
  let updatedAt = 0;
  let found = false;

  if (fromDb) {
    try {
      value = JSON.parse(fromDb.raw);
      updatedAt = fromDb.updatedAt;
      found = true;
    } catch {
      await cacheDelete(key);
      return null;
    }
  } else {
    const webRaw = webGet(key);
    if (webRaw) {
      const stored = parseStored(webRaw);
      if (!stored) {
        await cacheDelete(key);
        return null;
      }
      value = stored.value;
      updatedAt = stored.updatedAt;
      found = true;
    }
  }

  if (!found) return null;

  if (isStale(updatedAt, maxAgeMs)) {
    await cacheDelete(key);
    return null;
  }

  return value as T;
}

/** Persist a JSON blob. */
export async function cacheSet<T>(key: string, value: T): Promise<void> {
  const serialized = JSON.stringify(value);
  const now = Date.now();
  const wrote = await withDatabase(async (db) => {
    await db.execute(
      `INSERT INTO kv_cache (cache_key, value, updated_at)
       VALUES (?1, ?2, ?3)
       ON CONFLICT(cache_key) DO UPDATE SET value = ?2, updated_at = ?3`,
      [key, serialized, now],
    );
    return true;
  });
  if (!wrote) {
    webSet(key, JSON.stringify({ v: value, t: now } satisfies StoredEnvelope));
  }
}

export async function cacheDelete(key: string): Promise<void> {
  await withDatabase(async (db) => {
    await db.execute("DELETE FROM kv_cache WHERE cache_key = ?1", [key]);
  });
  webDelete(key);
}

/** Wipe all KV cache rows (call on logout / org switch). */
export async function cacheClearAll(): Promise<void> {
  await withDatabase(async (db) => {
    await db.execute("DELETE FROM kv_cache");
  });
  webClearAll();
}

/** Wipe KV rows whose key starts with prefix (include trailing `::` for scopes). */
export async function cacheClearPrefix(prefix: string): Promise<void> {
  await withDatabase(async (db) => {
    await db.execute("DELETE FROM kv_cache WHERE cache_key LIKE ?1", [`${prefix}%`]);
  });
  webClearPrefix(prefix);
}
