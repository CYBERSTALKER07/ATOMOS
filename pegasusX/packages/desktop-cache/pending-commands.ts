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

export async function getPendingCommands(): Promise<PendingCommand[]> {
  const result = await withDatabase(async (db) => {
    return db.select<any[]>(
      `SELECT * FROM pending_commands WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT 50`
    );
  });

  if (!result) return [];

  return result.map((row) => ({
    commandId: row.command_id as string,
    commandType: row.command_type as string,
    entityId: row.entity_id as string,
    knownVersion: row.known_version as number,
    payloadJson: row.payload_json as string,
    createdAt: row.created_at as number,
    retryCount: row.retry_count as number,
    status: row.status as "PENDING" | "FAILED" | "DISPUTED",
    lastError: row.last_error as string | undefined,
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
