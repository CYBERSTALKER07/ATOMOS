use keyring::Entry;
use serde::Serialize;

const SERVICE: &str = "uz.thelabindustries.warehouse";
const ACCOUNT_TOKEN: &str = "warehouse_jwt";
const ACCOUNT_REFRESH: &str = "warehouse_refresh_token";

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
    let entry = match Entry::new(SERVICE, ACCOUNT_TOKEN) {
        Ok(e) => e,
        Err(e) => return TokenResult { success: false, token: None, error: Some(e.to_string()) },
    };
    match entry.get_password() {
        Ok(t) => TokenResult { success: true, token: Some(t), error: None },
        Err(e) => TokenResult { success: false, token: None, error: Some(e.to_string()) },
    }
}

#[tauri::command]
pub fn clear_token() -> TokenResult {
    let entry = match Entry::new(SERVICE, ACCOUNT_TOKEN) {
        Ok(e) => e,
        Err(e) => return TokenResult { success: false, token: None, error: Some(e.to_string()) },
    };
    let _ = entry.delete_credential();
    if let Ok(re) = Entry::new(SERVICE, ACCOUNT_REFRESH) {
        let _ = re.delete_credential();
    }
    TokenResult { success: true, token: None, error: None }
}
