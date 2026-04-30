use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, Manager, WebviewUrl, WebviewWindowBuilder};

/// Payload sent back to the React frontend for gateway connect lifecycle events.
#[derive(Clone, Serialize)]
pub struct GatewayConnectEvent {
    pub session_id: String,
    pub gateway: String,
    pub status: String, // "opened", "closed", "failed"
    pub error: Option<String>,
}

/// Input for opening a gateway connect window.
#[derive(Deserialize)]
pub struct ConnectWindowRequest {
    pub session_id: String,
    pub gateway: String,
    pub redirect_url: String,
}

/// Opens a child Tauri webview window for the gateway OAuth/redirect flow.
/// Emits `gateway:connect` events back to the main window on open/close/fail.
#[tauri::command]
pub async fn open_gateway_connect(
    app: AppHandle,
    request: ConnectWindowRequest,
) -> Result<String, String> {
    let label = format!("gateway-connect-{}", request.session_id);

    // Emit opened event immediately
    let _ = app.emit(
        "gateway:connect",
        GatewayConnectEvent {
            session_id: request.session_id.clone(),
            gateway: request.gateway.clone(),
            status: "opened".into(),
            error: None,
        },
    );

    let window = WebviewWindowBuilder::new(
        &app,
        &label,
        WebviewUrl::External(
            request
                .redirect_url
                .parse()
                .map_err(|e| format!("Invalid redirect URL: {e}"))?,
        ),
    )
    .title(format!("Connect {} — Pegasus", request.gateway))
    .inner_size(600.0, 700.0)
    .center()
    .resizable(false)
    .build()
    .map_err(|e| format!("Failed to open connect window: {e}"))?;

    // Monitor window close to emit lifecycle event
    let app_handle = app.clone();
    let session_id = request.session_id.clone();
    let gateway = request.gateway.clone();
    window.on_window_event(move |event| {
        if let tauri::WindowEvent::Destroyed = event {
            let _ = app_handle.emit(
                "gateway:connect",
                GatewayConnectEvent {
                    session_id: session_id.clone(),
                    gateway: gateway.clone(),
                    status: "closed".into(),
                    error: None,
                },
            );
        }
    });

    Ok(label)
}

/// Closes a gateway connect window by its label.
#[tauri::command]
pub async fn close_gateway_connect(
    app: AppHandle,
    label: String,
) -> Result<(), String> {
    if let Some(window) = app.get_webview_window(&label) {
        window
            .close()
            .map_err(|e| format!("Failed to close window: {e}"))?;
    }
    Ok(())
}
