import { isTauri } from "./tauri-runtime";
import { saveTextFile } from "./file-export";

export type DesktopPrintOptions = {
  /** Optional document title for print / save-as-html flows. */
  title?: string;
};

/**
 * Opens the system print dialog (Save as PDF on Windows/macOS).
 * On Tauri, sets `document.title` briefly when `title` is provided.
 */
export function desktopPrint(options: DesktopPrintOptions = {}): void {
  if (typeof window === "undefined" || typeof window.print !== "function") {
    return;
  }
  const previousTitle = document.title;
  if (options.title) {
    document.title = options.title;
  }
  window.print();
  if (options.title) {
    window.setTimeout(() => {
      document.title = previousTitle;
    }, 0);
  }
}

/**
 * Optional: save a minimal printable HTML snapshot via native file dialog (Tauri)
 * or download on web. Use `desktopPrint` for PDF via the OS print dialog.
 */
export async function savePrintableHtml(
  filename: string,
  htmlBody: string,
  options: DesktopPrintOptions = {},
): Promise<{ saved: boolean }> {
  const title = options.title ?? "PegasusX Export";
  const html = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>${title}</title></head><body>${htmlBody}</body></html>`;
  const result = await saveTextFile({
    defaultFilename: filename.endsWith(".html") ? filename : `${filename}.html`,
    content: html,
    filters: [{ name: "HTML", extensions: ["html"] }],
  });
  return { saved: result.saved };
}

export function isDesktopPrintAvailable(): boolean {
  return typeof window !== "undefined" && typeof window.print === "function";
}

export function isNativeFileExportAvailable(): boolean {
  return isTauri();
}
