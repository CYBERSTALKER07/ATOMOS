use futures_util::StreamExt;
use serde::Serialize;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tauri::{AppHandle, Emitter};
use tokio::sync::Mutex;
use tokio_tungstenite::connect_async;

/// Shared state for the persistent telemetry WebSocket.
struct TelemetryState {
    connected: bool,
    reconnect_count: u32,
    last_message_epoch: u64,
    uptime_start: Option<Instant>,
    shutdown: bool,
}

static WS_STATE: std::sync::OnceLock<Arc<Mutex<TelemetryState>>> = std::sync::OnceLock::new();

fn ws_state() -> &'static Arc<Mutex<TelemetryState>> {
    WS_STATE.get_or_init(|| {
        Arc::new(Mutex::new(TelemetryState {
            connected: false,
            reconnect_count: 0,
            last_message_epoch: 0,
            uptime_start: None,
            shutdown: false,
        }))
    })
}

#[derive(Serialize)]
pub struct WsHealth {
    pub connected: bool,
    pub reconnect_count: u32,
    pub last_message_epoch: u64,
    pub uptime_seconds: u64,
}

#[derive(Serialize, Clone)]
struct TelemetryPing {
    driver_id: String,
    latitude: f64,
    longitude: f64,
    timestamp: Option<u64>,
}

/// Connect persistent Rust-backed WebSocket to /ws/telemetry.
/// Falls back with exponential backoff on disconnect. Emits
/// `telemetry:ping` events to the frontend for each GPS payload.
#[tauri::command]
pub async fn connect_telemetry(
    app: AppHandle,
    api_url: String,
    token: String,
) -> Result<String, String> {
    // Reset shutdown flag
    {
        let mut state = ws_state().lock().await;
        state.shutdown = false;
    }

    let ws_url = api_url
        .replace("https://", "wss://")
        .replace("http://", "ws://");

    let full_url = format!("{ws_url}/ws/telemetry?token={token}");

    let app_clone = app.clone();
    let url_clone = full_url.clone();

    tokio::spawn(async move {
        telemetry_loop(app_clone, url_clone).await;
    });

    Ok("Telemetry WebSocket connecting…".into())
}

/// Disconnect persistent telemetry WebSocket.
#[tauri::command]
pub async fn disconnect_telemetry() -> Result<String, String> {
    let mut state = ws_state().lock().await;
    state.shutdown = true;
    state.connected = false;
    Ok("Telemetry disconnected.".into())
}

/// Returns health metrics for the telemetry WebSocket.
#[tauri::command]
pub async fn get_ws_health() -> WsHealth {
    let state = ws_state().lock().await;
    let uptime = state
        .uptime_start
        .map(|s| s.elapsed().as_secs())
        .unwrap_or(0);

    WsHealth {
        connected: state.connected,
        reconnect_count: state.reconnect_count,
        last_message_epoch: state.last_message_epoch,
        uptime_seconds: uptime,
    }
}

async fn telemetry_loop(app: AppHandle, url: String) {
    let mut backoff = Duration::from_secs(1);
    let max_backoff = Duration::from_secs(30);

    loop {
        // Check shutdown
        {
            let state = ws_state().lock().await;
            if state.shutdown {
                log::info!("Telemetry loop shutdown requested.");
                return;
            }
        }

        let _ = app.emit("telemetry:status", "CONNECTING");

        match connect_async(&url).await {
            Ok((ws_stream, _response)) => {
                {
                    let mut state = ws_state().lock().await;
                    state.connected = true;
                    state.uptime_start = Some(Instant::now());
                }
                backoff = Duration::from_secs(1); // Reset backoff on success

                let _ = app.emit("telemetry:status", "LIVE");
                log::info!("Telemetry WebSocket connected.");

                let (_write, mut read) = ws_stream.split();

                while let Some(msg) = read.next().await {
                    // Check shutdown
                    {
                        let state = ws_state().lock().await;
                        if state.shutdown {
                            log::info!("Telemetry loop shutdown during read.");
                            return;
                        }
                    }

                    match msg {
                        Ok(tokio_tungstenite::tungstenite::Message::Text(text)) => {
                            // Parse GPS ping and emit to frontend
                            if let Ok(ping) =
                                serde_json::from_str::<serde_json::Value>(&text)
                            {
                                let now = std::time::SystemTime::now()
                                    .duration_since(std::time::UNIX_EPOCH)
                                    .unwrap_or_default()
                                    .as_millis() as u64;

                                {
                                    let mut state = ws_state().lock().await;
                                    state.last_message_epoch = now;
                                }

                                let _ = app.emit("telemetry:ping", ping);
                            }
                        }
                        Ok(tokio_tungstenite::tungstenite::Message::Close(_)) => {
                            log::info!("Telemetry WebSocket closed by server.");
                            break;
                        }
                        Err(e) => {
                            log::warn!("Telemetry WebSocket error: {e}");
                            break;
                        }
                        _ => {} // Ping/Pong/Binary — ignore
                    }
                }

                // Connection lost
                {
                    let mut state = ws_state().lock().await;
                    state.connected = false;
                    state.reconnect_count += 1;
                }
                let _ = app.emit("telemetry:status", "OFFLINE");
            }
            Err(e) => {
                log::warn!("Telemetry connection failed: {e}");
                {
                    let mut state = ws_state().lock().await;
                    state.connected = false;
                    state.reconnect_count += 1;
                }
                let _ = app.emit("telemetry:status", "OFFLINE");
            }
        }

        // Check shutdown before sleeping
        {
            let state = ws_state().lock().await;
            if state.shutdown {
                return;
            }
        }

        log::info!("Reconnecting in {}s…", backoff.as_secs());
        tokio::time::sleep(backoff).await;
        backoff = std::cmp::min(backoff * 2, max_backoff);
    }
}
