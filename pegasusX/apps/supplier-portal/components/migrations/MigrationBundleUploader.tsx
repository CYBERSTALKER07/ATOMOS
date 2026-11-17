"use client";

import { useCallback, useRef, useState } from "react";
import { usePortalT } from "@/lib/i18n";
import {
  fromBrowserFiles,
  isNativeImportPickerAvailable,
  MIGRATION_IMPORT_EXTENSIONS,
  pickNativeImportDirectory,
  pickNativeImportFiles,
  type PickedImportFile,
} from "@pegasusx/desktop-bridge";
import {
  migrationBundleCreateKey,
  migrationBundleUploadedKey,
  type ApiClient,
} from "@pegasusx/api-core";
import type { MigrationBundleUploadedResponse } from "@pegasusx/types";
import {
  prepareArtifactsForBundle,
  uploadArtifactToSignedTicket,
} from "@/lib/migration-upload";

export interface MigrationBundleUploaderProps {
  api: Pick<ApiClient, "createMigrationBundle" | "markMigrationBundleUploaded">;
  projectId: string;
  onBundleSorted: (response: MigrationBundleUploadedResponse) => void;
  onError: (msg: string) => void;
}

const ACCEPT_EXTENSIONS = MIGRATION_IMPORT_EXTENSIONS.map((e) => `.${e}`).join(",");

export function MigrationBundleUploader({
  api,
  projectId,
  onBundleSorted,
  onError,
}: MigrationBundleUploaderProps) {
  const t = usePortalT();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadedCount, setUploadedCount] = useState(0);
  const [totalCount, setTotalCount] = useState(0);
  const [lastCompleted, setLastCompleted] = useState(false);

  const handlePickedFiles = useCallback(
    async (picked: PickedImportFile[]) => {
      if (picked.length === 0 || uploading) return;
      const batch = picked.slice(0, 50);
      setUploading(true);
      setLastCompleted(false);
      setUploadedCount(0);
      setTotalCount(batch.length);

      try {
        const prepared = await prepareArtifactsForBundle(batch);
        const fingerprint = prepared.map((a) => `${a.file.name}:${a.sha256}`).join("|");
        const created = await api.createMigrationBundle(
          projectId,
          prepared.map((p) => ({
            file_name: p.file.name,
            byte_size: p.bytes.byteLength,
            content_sha256: p.sha256,
          })),
          migrationBundleCreateKey(projectId, fingerprint),
        );

        const uploadedMeta: Record<string, { byte_size: number; content_sha256: string }> = {};
        for (let i = 0; i < created.upload_tickets.length; i++) {
          const ticket = created.upload_tickets[i]!;
          const artifact = prepared[i];
          if (!artifact) continue;
          await uploadArtifactToSignedTicket(ticket, artifact.bytes);
          uploadedMeta[ticket.artifact_id] = {
            byte_size: artifact.bytes.byteLength,
            content_sha256: artifact.sha256,
          };
          setUploadedCount(i + 1);
        }

        const completed = await api.markMigrationBundleUploaded(
          projectId,
          created.bundle.bundle_id,
          uploadedMeta,
          migrationBundleUploadedKey(created.bundle.bundle_id, fingerprint),
          true,
        );
        setLastCompleted(true);
        onBundleSorted(completed);
      } catch (err) {
        onError(err instanceof Error ? err.message : String(err));
      } finally {
        setUploading(false);
      }
    },
    [api, onBundleSorted, onError, projectId, uploading],
  );

  async function onPickNativeFiles() {
    try {
      const files = await pickNativeImportFiles({ multiple: true });
      await handlePickedFiles(files);
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    }
  }

  async function onPickNativeFolder() {
    try {
      const files = await pickNativeImportDirectory(50);
      await handlePickedFiles(files);
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <section
      onDragOver={(e) => {
        e.preventDefault();
        setDragging(true);
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={(e) => {
        e.preventDefault();
        setDragging(false);
        if (e.dataTransfer.files.length > 0) {
          void handlePickedFiles(fromBrowserFiles(e.dataTransfer.files));
        }
      }}
      className={`rounded-xl border-2 border-dashed p-6 transition ${
        dragging
          ? "border-[var(--color-md-primary)] bg-[var(--color-md-surface-variant)]"
          : "border-[var(--border)] bg-[var(--surface)]"
      }`}
    >
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h3 className="md-typescale-title-medium font-semibold">
            {t("supplier_portal.migrations.dropzone_title")}
          </h3>
          <p className="text-sm text-[var(--muted)] mt-1">
            {t("supplier_portal.migrations.dropzone_subtitle")}
          </p>
          <p className="text-xs text-[var(--color-md-outline)] mt-1">
            {t("supplier_portal.migrations.resumable_hint")}
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          <input
            ref={fileInputRef}
            data-project-id={projectId}
            type="file"
            multiple
            accept={ACCEPT_EXTENSIONS}
            className="hidden"
            onChange={(e) => {
              if (e.target.files && e.target.files.length > 0) {
                void handlePickedFiles(fromBrowserFiles(e.target.files));
                e.target.value = "";
              }
            }}
          />
          <button
            type="button"
            disabled={uploading}
            onClick={() => fileInputRef.current?.click()}
            className="md-btn md-btn-filled px-4 py-2 text-sm"
          >
            {t("supplier_portal.migrations.select_files")}
          </button>

          {isNativeImportPickerAvailable() && (
            <>
              <button
                type="button"
                disabled={uploading}
                onClick={() => void onPickNativeFiles()}
                className="md-btn md-btn-outlined px-4 py-2 text-sm"
              >
                {t("supplier_portal.migrations.select_files")} (Native)
              </button>
              <button
                type="button"
                disabled={uploading}
                onClick={() => void onPickNativeFolder()}
                className="md-btn md-btn-outlined px-4 py-2 text-sm"
              >
                {t("supplier_portal.migrations.select_folder")}
              </button>
            </>
          )}
        </div>
      </div>

      {uploading && (
        <p className="mt-4 text-sm font-medium text-[var(--color-md-primary)]">
          {t("supplier_portal.migrations.uploading_progress", {
            uploaded: String(uploadedCount),
            total: String(totalCount),
          })}
        </p>
      )}
      {lastCompleted && !uploading && (
        <p className="mt-4 text-sm font-medium text-emerald-600">
          {t("supplier_portal.migrations.upload_complete")}
        </p>
      )}
    </section>
  );
}
