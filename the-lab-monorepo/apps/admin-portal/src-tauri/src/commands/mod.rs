pub mod evidence;
pub mod gateway;
pub mod security;
pub mod websocket;

use serde::Serialize;

#[derive(Serialize)]
pub struct AppInfo {
    pub version: String,
    pub platform: String,
    pub arch: String,
    pub debug: bool,
}

/// Returns desktop app metadata — version, platform, arch, debug mode.
#[tauri::command]
pub fn get_app_info() -> AppInfo {
    AppInfo {
        version: env!("CARGO_PKG_VERSION").to_string(),
        platform: std::env::consts::OS.to_string(),
        arch: std::env::consts::ARCH.to_string(),
        debug: cfg!(debug_assertions),
    }
}
