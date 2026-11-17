import type { MigrationUploadTicket } from "@pegasusx/types";
import type { PickedImportFile } from "@pegasusx/desktop-bridge";

const CHUNK_SIZE_BYTES = 8 * 1024 * 1024; // 8 MiB GCS resumable chunk multiple of 256 KiB

export interface PreparedArtifact {
  file: PickedImportFile;
  bytes: Uint8Array;
  sha256: string;
}

function toHex(buf: ArrayBuffer): string {
  const arr = new Uint8Array(buf);
  let out = "";
  for (let i = 0; i < arr.length; i++) {
    out += arr[i]!.toString(16).padStart(2, "0");
  }
  return out;
}

/**
 * Compute lowercase hex SHA-256 digest of file bytes using WebCrypto SubtleCrypto.
 */
export async function computeSha256Hex(bytes: Uint8Array): Promise<string> {
  if (typeof globalThis.crypto !== "undefined" && globalThis.crypto.subtle) {
    const copy = new Uint8Array(bytes.byteLength);
    copy.set(bytes);
    const digest = await globalThis.crypto.subtle.digest("SHA-256", copy.buffer);
    return toHex(digest);
  }
  throw new Error("webcrypto_sha256_unavailable");
}

/**
 * Read picked files and compute their SHA-256 hashes before declaring the bundle.
 */
export async function prepareArtifactsForBundle(
  files: PickedImportFile[],
): Promise<PreparedArtifact[]> {
  const prepared: PreparedArtifact[] = [];
  for (const file of files) {
    const bytes = await file.readBytes();
    const sha256 = await computeSha256Hex(bytes);
    prepared.push({ file, bytes, sha256 });
  }
  return prepared;
}

function buildUploadHeaders(ticket: MigrationUploadTicket): Record<string, string> {
  const headers: Record<string, string> = {};
  if (ticket.required_headers) {
    for (const [k, v] of Object.entries(ticket.required_headers)) {
      if (k.toLowerCase() === "x-goog-content-length-range") {
        // Browsers automatically set Content-Length; sending unsigned/restricted headers verbatim when allowed.
        headers[k] = v;
      } else {
        headers[k] = v;
      }
    }
  }
  if (!headers["Content-Type"] && ticket.content_type) {
    headers["Content-Type"] = ticket.content_type;
  }
  return headers;
}

/**
 * Upload one artifact to a signed GCS V4 URL (direct PUT <= 50 MiB or chunked resumable > 50 MiB).
 * No-ops gracefully in local dev environments that issue a `local.invalid` stub ticket URL.
 */
export async function uploadArtifactToSignedTicket(
  ticket: MigrationUploadTicket,
  bytes: Uint8Array,
  onProgress?: (uploadedBytes: number, totalBytes: number) => void,
): Promise<void> {
  const total = bytes.byteLength;
  if (ticket.upload_url.startsWith("https://local.invalid/")) {
    onProgress?.(total, total);
    return;
  }

  const headers = buildUploadHeaders(ticket);

  if (!ticket.resumable || total <= CHUNK_SIZE_BYTES) {
    const copy = new Uint8Array(total);
    copy.set(bytes);
    const res = await fetch(ticket.upload_url, {
      method: ticket.method === "POST" ? "PUT" : ticket.method || "PUT",
      headers,
      body: copy.buffer,
    });
    if (!res.ok) {
      throw new Error(`GCS upload failed (${res.status}) for ${ticket.file_name}`);
    }
    onProgress?.(total, total);
    return;
  }

  // Initiate GCS resumable upload session
  const startRes = await fetch(ticket.upload_url, {
    method: "POST",
    headers: {
      ...headers,
      "x-goog-resumable": "start",
    },
  });
  if (!startRes.ok) {
    throw new Error(`Resumable upload initiation failed (${startRes.status}) for ${ticket.file_name}`);
  }
  const sessionUrl = startRes.headers.get("Location") || ticket.upload_url;

  let offset = 0;
  while (offset < total) {
    const end = Math.min(offset + CHUNK_SIZE_BYTES, total);
    const slice = bytes.slice(offset, end);
    const chunkRes = await fetch(sessionUrl, {
      method: "PUT",
      headers: {
        "Content-Range": `bytes ${offset}-${end - 1}/${total}`,
      },
      body: slice.buffer,
    });
    // GCS returns 308 Resume Incomplete for intermediate chunks and 200/201 for final chunk.
    if (!chunkRes.ok && chunkRes.status !== 308) {
      throw new Error(`Resumable chunk upload failed (${chunkRes.status}) for ${ticket.file_name}`);
    }
    offset = end;
    onProgress?.(offset, total);
  }
}
