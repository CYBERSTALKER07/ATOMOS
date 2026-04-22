mod commands;
mod tray;

use tauri::Manager;

#[cfg(target_os = "macos")]
mod macos_vibrancy;

pub fn run() {
    env_logger::init();

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_process::init())
        .invoke_handler(tauri::generate_handler![
            // ── App info ──
            commands::get_app_info,
            // ── Security / Keyring ──
            commands::security::store_token,
            commands::security::get_token,
            commands::security::clear_token,
            // ── Evidence / Video ──
            commands::evidence::compress_and_upload,
            commands::evidence::get_compression_status,
            // ── WebSocket telemetry ──
            commands::websocket::connect_telemetry,
            commands::websocket::disconnect_telemetry,
            commands::websocket::get_ws_health,
            // ── Gateway connect ──
            commands::gateway::open_gateway_connect,
            commands::gateway::close_gateway_connect,
        ])
        .setup(|app| {
            // ── System tray ──
            tray::create_tray(app)?;

            // ── macOS vibrancy (disabled for now — transparent:false) ──
            // #[cfg(target_os = "macos")]
            // {
            //     if let Some(window) = app.get_webview_window("main") {
            //         macos_vibrancy::apply_vibrancy(&window);
            //     }
            // }

            // ── Mark desktop environment ──
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.eval("window.__TAURI_DESKTOP__ = true;");
            }

            log::info!("Lab Admin Desktop initialized.");
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running Lab Admin Desktop");
}
