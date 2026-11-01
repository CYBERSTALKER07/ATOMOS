use serde::Serialize;
use std::fs;

#[derive(Serialize)]
pub struct DiagnosticExportResult {
    pub success: bool,
    pub path: Option<String>,
    pub bytes_written: usize,
    pub error: Option<String>,
}

#[tauri::command]
pub fn export_diagnostic_bundle_native(file_path: String, payload: String) -> DiagnosticExportResult {
    if file_path.is_empty() {
        return DiagnosticExportResult {
            success: false,
            path: None,
            bytes_written: 0,
            error: Some("Destination file path cannot be empty".to_string()),
        };
    }

    match fs::write(&file_path, payload.as_bytes()) {
        Ok(()) => DiagnosticExportResult {
            success: true,
            path: Some(file_path),
            bytes_written: payload.len(),
            error: None,
        },
        Err(e) => DiagnosticExportResult {
            success: false,
            path: Some(file_path),
            bytes_written: 0,
            error: Some(e.to_string()),
        },
    }
}
