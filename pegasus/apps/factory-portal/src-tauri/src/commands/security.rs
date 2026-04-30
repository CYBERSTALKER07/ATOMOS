use keyring::Entry;
use serde::Serialize;

const SERVICE: &str = "com.pegasus.factory";
const ACCOUNT_TOKEN: &str = "factory_jwt";
const ACCOUNT_REFRESH: &str = "factory_refresh_token";

#[derive(Serialize)]
pub struct TokenResult {
    pub success: bool,
    pub token: Option<String>,
    pub error: Option<String>,
}

#[tauri::command]
pub fn store_token(token: String, refresh_token: Option<String>) -> TokenResult {
    let entry = match Entry::new(SERVICE, ACCOUNT_TOKEN) {
        Ok(e) => e,
        Err(e) => return TokenResult { success: false, token: None, error: Some(e.to_string()) },
    };
    if let Err(e) = entry.set_password(&token) {
        return TokenResult { success: false, token: None, error: Some(e.to_string()) };
    }
    if let Some(rt) = refresh_token {
        if let Ok(re) = Entry::new(SERVICE, ACCOUNT_REFRESH) {
            let _ = re.set_password(&rt);
        }
    }
    TokenResult { success: true, token: Some(token), error: None }
}

#[tauri::command]
pub fn get_token() -> TokenResult {
    match read_token(SERVICE) {
        Ok(Some(token)) => TokenResult { success: true, token: Some(token), error: None },
        Ok(None) => TokenResult { success: true, token: None, error: None },
        Err(e) => return TokenResult { success: false, token: None, error: Some(e) },
    }
}

#[tauri::command]
pub fn clear_token() -> TokenResult {
    let ok_primary_token = delete_if_present(SERVICE, ACCOUNT_TOKEN);
    let ok_primary_refresh = delete_if_present(SERVICE, ACCOUNT_REFRESH);

    if ok_primary_token && ok_primary_refresh {
        TokenResult { success: true, token: None, error: None }
    } else {
        TokenResult {
            success: false,
            token: None,
            error: Some("Failed to clear one or more tokens.".into()),
        }
    }
}

fn read_token(service: &str) -> Result<Option<String>, String> {
    let entry = Entry::new(service, ACCOUNT_TOKEN).map_err(|e| format!("Keyring init failed: {e}"))?;
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
