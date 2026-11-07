mod commands;

use tauri::{Emitter, Manager};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let mut builder = tauri::Builder::default();

    #[cfg(desktop)]
    {
        builder = builder
            .plugin(tauri_plugin_single_instance::init(|app, argv, _cwd| {
                for arg in argv.iter().skip(1) {
                    if arg.contains("://") {
                        let _ = app.emit("pegasusx-deep-link", arg.clone());
                        return;
                    }
                }
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.unminimize();
                    let _ = window.set_focus();
                }
            }))
            .plugin(tauri_plugin_deep_link::init())
            .plugin(tauri_plugin_updater::Builder::new().build())
            .plugin(tauri_plugin_process::init());
    }

    builder
        .plugin(tauri_plugin_sql::Builder::default().build())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            commands::security::store_token,
            commands::security::get_token,
            commands::security::clear_token,
            commands::telemetry::get_desktop_system_info,
            commands::telemetry::ping_cell_gateway,
            commands::diagnostics::export_diagnostic_bundle_native,
        ])
        .run(tauri::generate_context!())
        .expect("error while running PegasusX Control Tower Admin Desktop");
}
