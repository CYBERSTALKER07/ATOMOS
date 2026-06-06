use keyring::Entry;
use serde::Serialize;

const SERVICE: &str = "com.pegasusx.supplier";
const ACCOUNT_TOKEN: &str = "supplier_jwt";

#[derive(Serialize)]
pub struct TokenResult {
    pub success: bool,
    pub token: Option<String>,
    pub error: Option<String>,
}

#[tauri::command]
pub fn store_token(token: String) -> TokenResult {
    let entry = match Entry::new(SERVICE, ACCOUNT_TOKEN) {
        Ok(e) => e,
        Err(e) => return TokenResult { success: false, token: None, error: Some(e.to_string()) },
    };
    if let Err(e) = entry.set_password(&token) {
        return TokenResult { success: false, token: None, error: Some(e.to_string()) };
    }
    TokenResult { success: true, token: Some(token), error: None }
}

#[tauri::command]
pub fn get_token() -> TokenResult {
    match Entry::new(SERVICE, ACCOUNT_TOKEN) {
        Ok(entry) => match entry.get_password() {
            Ok(token) => TokenResult { success: true, token: Some(token), error: None },
            Err(keyring::Error::NoEntry) => TokenResult { success: true, token: None, error: None },
            Err(e) => TokenResult { success: false, token: None, error: Some(e.to_string()) },
        },
        Err(e) => TokenResult { success: false, token: None, error: Some(e.to_string()) },
    }
}

#[tauri::command]
pub fn clear_token() -> TokenResult {
    match Entry::new(SERVICE, ACCOUNT_TOKEN) {
        Ok(entry) => match entry.delete_credential() {
            Ok(()) => TokenResult { success: true, token: None, error: None },
            Err(keyring::Error::NoEntry) => TokenResult { success: true, token: None, error: None },
            Err(e) => TokenResult { success: false, token: None, error: Some(e.to_string()) },
        },
        Err(e) => TokenResult { success: false, token: None, error: Some(e.to_string()) },
    }
}
