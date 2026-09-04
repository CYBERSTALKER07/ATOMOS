import { isTauri } from "@pegasusx/desktop-bridge";

const DB_URI = "sqlite:pegasus_desktop_cache.db";

type SqlDatabase = {
  execute: (query: string, bindValues?: unknown[]) => Promise<unknown>;
  select: <T>(query: string, bindValues?: unknown[]) => Promise<T>;
};

let dbPromise: Promise<SqlDatabase | null> | null = null;
let migratePromise: Promise<void> | null = null;

async function loadDatabase(): Promise<SqlDatabase | null> {
  if (!isTauri()) return null;
  if (!dbPromise) {
    dbPromise = (async () => {
      const mod = await import("@tauri-apps/plugin-sql");
      const Database = mod.default;
      return Database.load(DB_URI);
    })();
  }
  return dbPromise;
}

async function ensureMigrated(): Promise<SqlDatabase | null> {
  if (!migratePromise) {
    migratePromise = (async () => {
      const db = await loadDatabase();
      if (!db) return;
      await db.execute(`
        CREATE TABLE IF NOT EXISTS kv_cache (
          cache_key TEXT PRIMARY KEY NOT NULL,
          value TEXT NOT NULL,
          updated_at INTEGER NOT NULL
        )
      `);
      await db.execute(`
        CREATE TABLE IF NOT EXISTS pending_checkouts (
          id TEXT PRIMARY KEY NOT NULL,
          payload_json TEXT NOT NULL,
          idempotency_key TEXT NOT NULL,
          created_at INTEGER NOT NULL,
          retry_count INTEGER NOT NULL DEFAULT 0,
          last_error TEXT
        )
      `);
      await db.execute(`
        CREATE TABLE IF NOT EXISTS pending_pos_sales (
          id TEXT PRIMARY KEY NOT NULL,
          client_sale_id TEXT NOT NULL,
          client_receipt TEXT NOT NULL,
          session_id TEXT NOT NULL,
          payload_json TEXT NOT NULL,
          idempotency_key TEXT NOT NULL,
          created_at INTEGER NOT NULL,
          retry_count INTEGER NOT NULL DEFAULT 0,
          status TEXT NOT NULL DEFAULT 'PENDING',
          last_error TEXT,
          server_sale_id TEXT,
          server_receipt_number TEXT
        )
      `);
      await db.execute(`
        CREATE TABLE IF NOT EXISTS pending_commands (
          command_id TEXT PRIMARY KEY NOT NULL,
          command_type TEXT NOT NULL,
          entity_id TEXT NOT NULL,
          known_version INTEGER NOT NULL,
          payload_json TEXT NOT NULL,
          created_at INTEGER NOT NULL,
          retry_count INTEGER NOT NULL DEFAULT 0,
          status TEXT NOT NULL DEFAULT 'PENDING',
          last_error TEXT
        )
      `);
    })();
  }
  await migratePromise;
  return loadDatabase();
}

/** Initialize SQLite schema (no-op on web). Safe to call multiple times. */
export async function initDesktopCache(): Promise<boolean> {
  const db = await ensureMigrated();
  return db !== null;
}

export async function withDatabase<T>(
  fn: (db: SqlDatabase) => Promise<T>,
): Promise<T | null> {
  const db = await ensureMigrated();
  if (!db) return null;
  return fn(db);
}

export function isDesktopCacheAvailable(): boolean {
  return isTauri();
}
