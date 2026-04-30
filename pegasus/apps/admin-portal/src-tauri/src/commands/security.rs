use keyring::Entry;
use serde::Serialize;

const SERVICE_NAME: &str = "com.pegasus.admin";
const LEGACY_SERVICE_NAME: &str = "uz.thelabindustries.admin";
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
    match read_token(SERVICE_NAME) {
        Ok(Some(token)) => {
            return TokenResult {
                success: true,
                token: Some(token),
                error: None,
            }
        }
        Ok(None) => {}
        Err(e) => {
            return TokenResult {
                success: false,
                token: None,
                error: Some(e),
            }
        }
    }

    match read_token(LEGACY_SERVICE_NAME) {
        Ok(Some(token)) => {
            if let Ok(primary) = Entry::new(SERVICE_NAME, TOKEN_KEY) {
                let _ = primary.set_password(&token);
            }
            TokenResult {
                success: true,
                token: Some(token),
                error: None,
            }
        }
        Ok(None) => TokenResult {
            success: true,
            token: None,
            error: None,
        },
        Err(e) => TokenResult {
            success: false,
            token: None,
            error: Some(e),
        },
    }
}

/// Clear all tokens from OS keychain (logout).
#[tauri::command]
pub fn clear_token() -> TokenResult {
    let ok_primary_token = delete_if_present(SERVICE_NAME, TOKEN_KEY);
    let ok_primary_refresh = delete_if_present(SERVICE_NAME, REFRESH_KEY);
    let ok_legacy_token = delete_if_present(LEGACY_SERVICE_NAME, TOKEN_KEY);
    let ok_legacy_refresh = delete_if_present(LEGACY_SERVICE_NAME, REFRESH_KEY);

    if ok_primary_token && ok_primary_refresh && ok_legacy_token && ok_legacy_refresh {
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

fn read_token(service: &str) -> Result<Option<String>, String> {
    let entry = Entry::new(service, TOKEN_KEY).map_err(|e| format!("Keyring init failed: {e}"))?;

    match entry.get_password() {
        Ok(token) => Ok(Some(token)),
        Err(keyring::Error::NoEntry) => Ok(None),
        Err(e) => Err(format!("Failed to retrieve token: {e}")),
    }
}

fn delete_if_present(service: &str, key: &str) -> bool {
    let result = Entry::new(service, key).and_then(|e| e.delete_credential());
    result.is_ok() || matches!(result.as_ref().err(), Some(keyring::Error::NoEntry))
}
