mod commands;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            commands::security::store_token,
            commands::security::get_token,
            commands::security::clear_token,
        ])
        .run(tauri::generate_context!())
        .expect("error while running PegasusX Supplier Portal");
}
