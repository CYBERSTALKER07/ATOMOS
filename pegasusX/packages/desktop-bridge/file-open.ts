import { isTauri } from "./tauri-runtime";

export const MIGRATION_IMPORT_EXTENSIONS = [
  "csv",
  "tsv",
  "xlsx",
  "xls",
  "xml",
  "json",
  "dbf",
  "zip",
  "pdf",
  "txt",
] as const;

export interface PickedImportFile {
  name: string;
  size: number;
  mimeType: string;
  path?: string;
  readBytes: () => Promise<Uint8Array>;
}

const MIME_BY_EXT: Record<string, string> = {
  csv: "text/csv",
  tsv: "text/tab-separated-values",
  xlsx: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  xls: "application/vnd.ms-excel",
  xml: "application/xml",
  json: "application/json",
  dbf: "application/x-dbf",
  zip: "application/zip",
  pdf: "application/pdf",
  txt: "text/plain",
};

function mimeFromFileName(name: string): string {
  const idx = name.lastIndexOf(".");
  if (idx === -1) return "application/octet-stream";
  const ext = name.slice(idx + 1).toLowerCase();
  return MIME_BY_EXT[ext] ?? "application/octet-stream";
}

function isSupportedImportFile(name: string): boolean {
  const idx = name.lastIndexOf(".");
  if (idx === -1) return false;
  const ext = name.slice(idx + 1).toLowerCase();
  return (MIGRATION_IMPORT_EXTENSIONS as readonly string[]).includes(ext);
}

function basename(filePath: string): string {
  const normalized = filePath.replace(/\\/g, "/");
  const idx = normalized.lastIndexOf("/");
  return idx === -1 ? normalized : normalized.slice(idx + 1);
}

function joinPath(dir: string, entryName: string): string {
  const sep = dir.includes("\\") ? "\\" : "/";
  return dir.endsWith(sep) ? `${dir}${entryName}` : `${dir}${sep}${entryName}`;
}

/** True when Tauri native file and directory pickers can be invoked. */
export function isNativeImportPickerAvailable(): boolean {
  return isTauri();
}

/** Adapt browser File / FileList items into the unified PickedImportFile shape. */
export function fromBrowserFiles(files: FileList | File[]): PickedImportFile[] {
  return Array.from(files).map((file) => ({
    name: file.name,
    size: file.size,
    mimeType: file.type || mimeFromFileName(file.name),
    readBytes: async () => new Uint8Array(await file.arrayBuffer()),
  }));
}

/**
 * Open native OS multi-file picker inside Tauri desktop shells.
 * Returns [] on web or if the user cancels.
 */
export async function pickNativeImportFiles(
  options: { multiple?: boolean } = {},
): Promise<PickedImportFile[]> {
  if (!isTauri()) return [];
  const { open } = await import("@tauri-apps/plugin-dialog");
  const { readFile, stat } = await import("@tauri-apps/plugin-fs");
  const selected = await open({
    multiple: options.multiple ?? true,
    directory: false,
    filters: [
      {
        name: "ERP Exports",
        extensions: [...MIGRATION_IMPORT_EXTENSIONS],
      },
    ],
  });
  if (!selected) return [];
  const paths = Array.isArray(selected) ? selected : [selected];
  const results: PickedImportFile[] = [];
  for (const filePath of paths) {
    const info = await stat(filePath).catch(() => null);
    const name = basename(filePath);
    results.push({
      name,
      size: info?.size ?? 0,
      mimeType: mimeFromFileName(name),
      path: filePath,
      readBytes: async () => readFile(filePath),
    });
  }
  return results;
}

/**
 * Open native OS directory picker inside Tauri desktop shells and collect supported ERP files.
 * Returns [] on web or if cancelled.
 */
export async function pickNativeImportDirectory(
  maxFiles = 200,
): Promise<PickedImportFile[]> {
  if (!isTauri()) return [];
  const { open } = await import("@tauri-apps/plugin-dialog");
  const { readDir, readFile, stat } = await import("@tauri-apps/plugin-fs");
  const dir = await open({ directory: true, multiple: false });
  if (!dir || Array.isArray(dir)) return [];

  const picked: PickedImportFile[] = [];
  const walk = async (currentDir: string) => {
    if (picked.length >= maxFiles) return;
    const entries = await readDir(currentDir);
    for (const entry of entries) {
      if (picked.length >= maxFiles) break;
      const fullPath = joinPath(currentDir, entry.name);
      if (entry.isDirectory) {
        await walk(fullPath);
        continue;
      }
      if (!entry.isFile || !isSupportedImportFile(entry.name)) {
        continue;
      }
      const info = await stat(fullPath).catch(() => null);
      picked.push({
        name: entry.name,
        size: info?.size ?? 0,
        mimeType: mimeFromFileName(entry.name),
        path: fullPath,
        readBytes: async () => readFile(fullPath),
      });
    }
  };
  await walk(dir);
  return picked;
}
