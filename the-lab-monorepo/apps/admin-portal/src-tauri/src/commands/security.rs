use keyring::Entry;
use serde::Serialize;

const SERVICE_NAME: &str = "uz.thelabindustries.admin";
const TOKEN_KEY: &str = "jwt_token";
const REFRESH_KEY: &str = "refresh_token";

#[derive(Serialize)]
pub struct TokenResult {
    pub success: bool,
    pub token: Option<String>,
    pub error: Option<String>,
}

/// Store JWT in OS keychain (macOS Keychain / Windows Credential Manager).
#[tauri::command]
pub fn store_token(token: String, refresh_token: Option<String>) -> TokenResult {
    let entry = match Entry::new(SERVICE_NAME, TOKEN_KEY) {
        Ok(e) => e,
        Err(e) => {
            return TokenResult {
                success: false,
                token: None,
                error: Some(format!("Keyring init failed: {e}")),
            }
        }
    };

    if let Err(e) = entry.set_password(&token) {
        return TokenResult {
            success: false,
            token: None,
            error: Some(format!("Failed to store token: {e}")),
        };
    }

    // Store refresh token if provided
    if let Some(rt) = refresh_token {
        if let Ok(refresh_entry) = Entry::new(SERVICE_NAME, REFRESH_KEY) {
            let _ = refresh_entry.set_password(&rt);
        }
    }

    TokenResult {
        success: true,
        token: Some(token),
        error: None,
    }
}

/// Retrieve JWT from OS keychain.
#[tauri::command]
pub fn get_token() -> TokenResult {
    let entry = match Entry::new(SERVICE_NAME, TOKEN_KEY) {
        Ok(e) => e,
        Err(e) => {
            return TokenResult {
                success: false,
                token: None,
                error: Some(format!("Keyring init failed: {e}")),
            }
        }
    };

    match entry.get_password() {
        Ok(token) => TokenResult {
            success: true,
            token: Some(token),
            error: None,
        },
        Err(keyring::Error::NoEntry) => TokenResult {
            success: true,
            token: None,
            error: None,
        },
        Err(e) => TokenResult {
            success: false,
            token: None,
            error: Some(format!("Failed to retrieve token: {e}")),
        },
    }
}

/// Clear all tokens from OS keychain (logout).
#[tauri::command]
pub fn clear_token() -> TokenResult {
    let cleared_jwt = Entry::new(SERVICE_NAME, TOKEN_KEY)
        .and_then(|e| e.delete_credential());
    let cleared_refresh = Entry::new(SERVICE_NAME, REFRESH_KEY)
        .and_then(|e| e.delete_credential());

    // Treat NoEntry as success (already cleared)
    let jwt_ok = cleared_jwt.is_ok()
        || matches!(cleared_jwt.as_ref().err(), Some(keyring::Error::NoEntry));
    let refresh_ok = cleared_refresh.is_ok()
        || matches!(cleared_refresh.as_ref().err(), Some(keyring::Error::NoEntry));

    if jwt_ok && refresh_ok {
        TokenResult {
            success: true,
            token: None,
            error: None,
        }
    } else {
        TokenResult {
            success: false,
            token: None,
            error: Some("Failed to clear one or more tokens.".into()),
        }
    }
}
