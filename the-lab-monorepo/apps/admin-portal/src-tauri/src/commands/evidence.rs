use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use std::sync::atomic::{AtomicU8, Ordering};
use tauri::{AppHandle, Emitter};

/// Global compression progress (0–100). Polled by get_compression_status.
static COMPRESSION_PROGRESS: AtomicU8 = AtomicU8::new(0);

#[derive(Serialize)]
pub struct EvidenceResult {
    pub success: bool,
    pub output_path: Option<String>,
    pub output_size_bytes: Option<u64>,
    pub upload_url: Option<String>,
    pub error: Option<String>,
}

#[derive(Serialize, Clone)]
pub struct CompressionProgress {
    pub percent: u8,
    pub stage: String,
}

#[derive(Deserialize)]
pub struct UploadConfig {
    pub api_url: String,
    pub token: String,
    pub order_id: String,
}

/// Compress a video file using FFmpeg CLI sidecar, then upload to backend.
///
/// Desktop-only: calls system FFmpeg for H.265/HEVC compression with hardware
/// acceleration (VideoToolbox on macOS, NVENC on Windows).
/// Emits `evidence:progress` events to the frontend.
#[tauri::command]
pub async fn compress_and_upload(
    app: AppHandle,
    file_path: String,
    upload_config: UploadConfig,
) -> EvidenceResult {
    COMPRESSION_PROGRESS.store(0, Ordering::SeqCst);

    let input = PathBuf::from(&file_path);
    if !input.exists() {
        return EvidenceResult {
            success: false,
            output_path: None,
            output_size_bytes: None,
            upload_url: None,
            error: Some(format!("File not found: {file_path}")),
        };
    }

    // Build output path in temp dir
    let output = std::env::temp_dir().join(format!(
        "lab_evidence_{}.mp4",
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_millis()
    ));

    // ── Stage 1: Compress with FFmpeg ──────────────────────────────────
    let _ = app.emit("evidence:progress", CompressionProgress {
        percent: 5,
        stage: "Compressing video…".into(),
    });
    COMPRESSION_PROGRESS.store(5, Ordering::SeqCst);

    let encoder = if cfg!(target_os = "macos") {
        "hevc_videotoolbox"
    } else if cfg!(target_os = "windows") {
        "hevc_nvenc"
    } else {
        "libx265"
    };

    let ffmpeg_result = tokio::process::Command::new("ffmpeg")
        .args([
            "-y",
            "-i", input.to_str().unwrap_or_default(),
            "-c:v", encoder,
            "-b:v", "2M",
            "-maxrate", "4M",
            "-bufsize", "8M",
            "-c:a", "aac",
            "-b:a", "128k",
            "-movflags", "+faststart",
            output.to_str().unwrap_or_default(),
        ])
        .output()
        .await;

    match ffmpeg_result {
        Ok(out) if out.status.success() => {}
        Ok(out) => {
            let stderr = String::from_utf8_lossy(&out.stderr);
            // Retry with software encoder if hardware fails
            log::warn!("Hardware encoder failed, falling back to libx265: {stderr}");
            let fallback = tokio::process::Command::new("ffmpeg")
                .args([
                    "-y",
                    "-i", input.to_str().unwrap_or_default(),
                    "-c:v", "libx265",
                    "-crf", "28",
                    "-preset", "fast",
                    "-c:a", "aac",
                    "-b:a", "128k",
                    "-movflags", "+faststart",
                    output.to_str().unwrap_or_default(),
                ])
                .output()
                .await;

            match fallback {
                Ok(fb) if fb.status.success() => {}
                _ => {
                    return EvidenceResult {
                        success: false,
                        output_path: None,
                        output_size_bytes: None,
                        upload_url: None,
                        error: Some("FFmpeg compression failed (both HW and SW)".into()),
                    };
                }
            }
        }
        Err(e) => {
            return EvidenceResult {
                success: false,
                output_path: None,
                output_size_bytes: None,
                upload_url: None,
                error: Some(format!("FFmpeg not found or failed to execute: {e}")),
            };
        }
    }

    let _ = app.emit("evidence:progress", CompressionProgress {
        percent: 60,
        stage: "Compression complete. Uploading…".into(),
    });
    COMPRESSION_PROGRESS.store(60, Ordering::SeqCst);

    // Get output file size
    let output_size = std::fs::metadata(&output)
        .map(|m| m.len())
        .unwrap_or(0);

    // ── Stage 2: Upload to backend ─────────────────────────────────────
    let upload_result = upload_evidence(
        &output,
        &upload_config.api_url,
        &upload_config.token,
        &upload_config.order_id,
    )
    .await;

    let _ = app.emit("evidence:progress", CompressionProgress {
        percent: 100,
        stage: "Done".into(),
    });
    COMPRESSION_PROGRESS.store(100, Ordering::SeqCst);

    // Cleanup temp file
    let _ = std::fs::remove_file(&output);

    match upload_result {
        Ok(url) => EvidenceResult {
            success: true,
            output_path: Some(output.to_string_lossy().into()),
            output_size_bytes: Some(output_size),
            upload_url: Some(url),
            error: None,
        },
        Err(e) => EvidenceResult {
            success: true, // Compression succeeded even if upload failed
            output_path: Some(output.to_string_lossy().into()),
            output_size_bytes: Some(output_size),
            upload_url: None,
            error: Some(format!("Upload failed: {e}")),
        },
    }
}

/// Returns current compression progress (0–100).
#[tauri::command]
pub fn get_compression_status() -> u8 {
    COMPRESSION_PROGRESS.load(Ordering::SeqCst)
}

async fn upload_evidence(
    file_path: &PathBuf,
    api_url: &str,
    token: &str,
    order_id: &str,
) -> Result<String, String> {
    let file_bytes = tokio::fs::read(file_path)
        .await
        .map_err(|e| format!("Failed to read compressed file: {e}"))?;

    let file_name = file_path
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("evidence.mp4")
        .to_string();

    let part = reqwest::multipart::Part::bytes(file_bytes)
        .file_name(file_name)
        .mime_str("video/mp4")
        .map_err(|e| format!("MIME error: {e}"))?;

    let form = reqwest::multipart::Form::new()
        .text("order_id", order_id.to_string())
        .part("file", part);

    let client = reqwest::Client::new();
    let url = format!("{api_url}/v1/evidence/upload");

    let resp = client
        .post(&url)
        .bearer_auth(token)
        .multipart(form)
        .send()
        .await
        .map_err(|e| format!("Request failed: {e}"))?;

    if !resp.status().is_success() {
        let status = resp.status();
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("Upload returned {status}: {body}"));
    }

    #[derive(Deserialize)]
    struct UploadResponse {
        url: Option<String>,
    }

    let body: UploadResponse = resp
        .json()
        .await
        .map_err(|e| format!("Failed to parse upload response: {e}"))?;

    Ok(body.url.unwrap_or_default())
}
