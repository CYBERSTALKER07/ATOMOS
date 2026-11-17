import { withDatabase } from "./db";

export interface PendingCommand {
  commandId: string;
  commandType: string;
  entityId: string;
  knownVersion: number;
  payloadJson: string;
  createdAt: number;
  retryCount: number;
  status: "PENDING" | "FAILED" | "DISPUTED";
  lastError?: string;
}

export async function insertPendingCommand(
  command: Omit<PendingCommand, "createdAt" | "retryCount" | "status">
): Promise<void> {
  await withDatabase(async (db) => {
    await db.execute(
      `INSERT INTO pending_commands 
      (command_id, command_type, entity_id, known_version, payload_json, created_at, retry_count, status)
      VALUES (?, ?, ?, ?, ?, ?, 0, 'PENDING')`,
      [
        command.commandId,
        command.commandType,
        command.entityId,
        command.knownVersion,
        command.payloadJson,
        Date.now(),
      ]
    );
  });
}

interface PendingCommandRow {
  command_id: string;
  command_type: string;
  entity_id: string;
  known_version: number;
  payload_json: string;
  created_at: number;
  retry_count: number;
  status: "PENDING" | "FAILED" | "DISPUTED";
  last_error?: string;
}

export async function getPendingCommands(): Promise<PendingCommand[]> {
  const result = await withDatabase(async (db) => {
    return db.select<PendingCommandRow[]>(
      `SELECT * FROM pending_commands WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT 50`
    );
  });

  if (!result) return [];

  return result.map((row) => ({
    commandId: row.command_id,
    commandType: row.command_type,
    entityId: row.entity_id,
    knownVersion: row.known_version,
    payloadJson: row.payload_json,
    createdAt: row.created_at,
    retryCount: row.retry_count,
    status: row.status,
    lastError: row.last_error,
  }));
}

export async function markCommandFailed(
  commandId: string,
  error: string,
  status: "FAILED" | "DISPUTED" = "FAILED"
): Promise<void> {
  await withDatabase(async (db) => {
    await db.execute(
      `UPDATE pending_commands SET status = ?, last_error = ?, retry_count = retry_count + 1 WHERE command_id = ?`,
      [status, error, commandId]
    );
  });
}

export async function deleteCommand(commandId: string): Promise<void> {
  await withDatabase(async (db) => {
    await db.execute(`DELETE FROM pending_commands WHERE command_id = ?`, [
      commandId,
    ]);
  });
}
