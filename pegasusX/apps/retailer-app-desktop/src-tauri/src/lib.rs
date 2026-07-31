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
        if let Some(window) = app.webview_windows().values().next() {
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
    .setup(|app| {
      if cfg!(debug_assertions) {
        app.handle().plugin(
          tauri_plugin_log::Builder::default()
            .level(log::LevelFilter::Info)
            .build(),
        )?;
      }
      Ok(())
    })
    .invoke_handler(tauri::generate_handler![
      commands::get_app_info,
      commands::security::store_token,
      commands::security::get_token,
      commands::security::clear_token,
    ])
    .run(tauri::generate_context!())
    .expect("error while running tauri application");
}
