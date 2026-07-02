import { withDatabase } from "./db";

const WEB_PREFIX = "pegasus_kv_cache:";

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

/** Read a JSON blob by cache key. Uses SQLite on Tauri, localStorage on web dev. */
export async function cacheGet<T>(key: string): Promise<T | null> {
  const fromDb = await withDatabase(async (db) => {
    const rows = await db.select<{ value: string }[]>(
      "SELECT value FROM kv_cache WHERE cache_key = ?1",
      [key],
    );
    return rows[0]?.value ?? null;
  });

  const raw = fromDb ?? webGet(key);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

/** Persist a JSON blob. */
export async function cacheSet<T>(key: string, value: T): Promise<void> {
  const serialized = JSON.stringify(value);
  const wrote = await withDatabase(async (db) => {
    await db.execute(
      `INSERT INTO kv_cache (cache_key, value, updated_at)
       VALUES (?1, ?2, ?3)
       ON CONFLICT(cache_key) DO UPDATE SET value = ?2, updated_at = ?3`,
      [key, serialized, Date.now()],
    );
    return true;
  });
  if (!wrote) {
    webSet(key, serialized);
  }
}

export async function cacheDelete(key: string): Promise<void> {
  await withDatabase(async (db) => {
    await db.execute("DELETE FROM kv_cache WHERE cache_key = ?1", [key]);
  });
  webDelete(key);
}
