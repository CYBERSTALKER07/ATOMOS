use tauri::{
    menu::{MenuBuilder, MenuItemBuilder},
    tray::TrayIconBuilder,
    App, Manager,
};

/// Create the system tray with fleet health indicator.
///
/// Icon color semantics:
/// - Default icon (blue) = app running, no telemetry data yet
/// - Updated via frontend → Rust events to green/amber/red based on fleet health
pub fn create_tray(app: &App) -> Result<(), Box<dyn std::error::Error>> {
    let dashboard = MenuItemBuilder::with_id("dashboard", "Open Dashboard")
        .build(app)?;
    let fleet = MenuItemBuilder::with_id("fleet", "Fleet Status")
        .build(app)?;
    let orders = MenuItemBuilder::with_id("orders", "Orders")
        .build(app)?;
    let separator = tauri::menu::PredefinedMenuItem::separator(app)?;
    let quit = MenuItemBuilder::with_id("quit", "Quit Pegasus Admin")
        .accelerator("CmdOrCtrl+Q")
        .build(app)?;

    let menu = MenuBuilder::new(app)
        .item(&dashboard)
        .item(&fleet)
        .item(&orders)
        .item(&separator)
        .item(&quit)
        .build()?;

    let _tray = TrayIconBuilder::new()
        .icon(tauri::include_image!("icons/icon.png"))
        .tooltip("Pegasus Admin — Desktop")
        .menu(&menu)
        .on_menu_event(move |app, event| {
            let id = event.id().as_ref();
            match id {
                "dashboard" => {
                    if let Some(w) = app.get_webview_window("main") {
                        let _ = w.show();
                        let _ = w.set_focus();
                        let _ = w.eval("window.location.hash = ''; window.location.pathname = '/';");
                    }
                }
                "fleet" => {
                    if let Some(w) = app.get_webview_window("main") {
                        let _ = w.show();
                        let _ = w.set_focus();
                        let _ = w.eval("window.location.pathname = '/fleet';");
                    }
                }
                "orders" => {
                    if let Some(w) = app.get_webview_window("main") {
                        let _ = w.show();
                        let _ = w.set_focus();
                        let _ = w.eval("window.location.pathname = '/supplier/orders';");
                    }
                }
                "quit" => {
                    app.exit(0);
                }
                _ => {}
            }
        })
        .build(app)?;

    Ok(())
}
