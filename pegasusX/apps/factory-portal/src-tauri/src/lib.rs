mod commands;

use tauri::Manager;

pub fn run() {
    env_logger::init();

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
            .plugin(tauri_plugin_updater::Builder::new().build());
    }

    builder
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_process::init())
        .invoke_handler(tauri::generate_handler![
            commands::get_app_info,
            commands::security::store_token,
            commands::security::get_token,
            commands::security::clear_token,
        ])
        .setup(|_app| {
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running Pegasus Factory Desktop");
}
