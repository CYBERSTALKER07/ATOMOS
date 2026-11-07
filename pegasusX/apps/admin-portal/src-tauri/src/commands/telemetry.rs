use serde::Serialize;
use std::time::Instant;

#[derive(Serialize)]
pub struct DesktopSystemInfo {
    pub os_name: String,
    pub arch: String,
    pub app_version: String,
    pub is_desktop: bool,
    pub default_cell: String,
}

#[derive(Serialize)]
pub struct GatewayPingResult {
    pub target_cell: String,
    pub reachable: bool,
    pub latency_ms: u64,
    pub timestamp_utc: String,
    pub error: Option<String>,
}

#[tauri::command]
pub fn get_desktop_system_info() -> DesktopSystemInfo {
    DesktopSystemInfo {
        os_name: std::env::consts::OS.to_string(),
        arch: std::env::consts::ARCH.to_string(),
        app_version: env!("CARGO_PKG_VERSION").to_string(),
        is_desktop: true,
        default_cell: "cell-uz-tashkent".to_string(),
    }
}

#[tauri::command]
pub async fn ping_cell_gateway(cell_id: String) -> GatewayPingResult {
    let start = Instant::now();
    // Simulate real local socket probe or gateway lookup
    let target = if cell_id.is_empty() { "cell-uz-tashkent".to_string() } else { cell_id };
    
    // Simulating deterministic low-latency preflight
    let elapsed = start.elapsed().as_millis() as u64;

    GatewayPingResult {
        target_cell: target,
        reachable: true,
        latency_ms: if elapsed == 0 { 4 } else { elapsed },
        timestamp_utc: chrono_lite_iso(),
        error: None,
    }
}

fn chrono_lite_iso() -> String {
    use std::time::SystemTime;
    let now = SystemTime::now();
    let dur = now.duration_since(SystemTime::UNIX_EPOCH).unwrap_or_default();
    format!("{}.{:03}Z", dur.as_secs(), dur.subsec_millis())
}
