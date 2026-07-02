import { isTauri } from "./tauri-runtime";
import { formatCsv } from "./format-csv";

export type SaveTextFileResult =
  | { saved: true; path: string }
  | { saved: false; cancelled?: boolean; reason?: string };

export type SaveTextFileOptions = {
  defaultFilename: string;
  content: string;
  /** Dialog filter label → extensions (Tauri save picker). */
  filters?: Array<{ name: string; extensions: string[] }>;
};

/** Save text via native dialog on Tauri; browser download fallback on web. */
export async function saveTextFile(
  options: SaveTextFileOptions,
): Promise<SaveTextFileResult> {
  const { defaultFilename, content, filters } = options;

  if (isTauri()) {
    try {
      const { save } = await import("@tauri-apps/plugin-dialog");
      const { writeTextFile } = await import("@tauri-apps/plugin-fs");
      const path = await save({
        defaultPath: defaultFilename,
        filters: filters ?? [{ name: "Text", extensions: ["txt"] }],
      });
      if (!path) {
        return { saved: false, cancelled: true };
      }
      await writeTextFile(path, content);
      return { saved: true, path };
    } catch (err) {
      return {
        saved: false,
        reason: err instanceof Error ? err.message : String(err),
      };
    }
  }

  if (typeof document === "undefined") {
    return { saved: false, reason: "document_unavailable" };
  }

  const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = defaultFilename;
  anchor.click();
  URL.revokeObjectURL(url);
  return { saved: true, path: defaultFilename };
}

/** Format and export CSV — native save dialog on Tauri desktop. */
export async function exportCsv(
  filename: string,
  headers: string[],
  rows: string[][],
): Promise<SaveTextFileResult> {
  const content = formatCsv(headers, rows);
  const normalized = filename.endsWith(".csv") ? filename : `${filename}.csv`;
  return saveTextFile({
    defaultFilename: normalized,
    content,
    filters: [{ name: "CSV", extensions: ["csv"] }],
  });
}

/** @deprecated Alias for web callers; prefer `exportCsv` (async). */
export function downloadCsv(
  filename: string,
  headers: string[],
  rows: string[][],
): void {
  void exportCsv(filename, headers, rows);
}
