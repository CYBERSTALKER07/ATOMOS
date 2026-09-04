# LB-3 — Soliq + PlayMobile GSM shells (names only).
# Versions are created only when the matching var is non-empty.
# Do not terraform apply from the Layer B readiness plan. PKCS#12 bytes are a volume, not a GSM string.

locals {
  fiscal_sms_secret_ids = {
    fiscal_my_soliq_base_url        = local.secret_fiscal_my_soliq_base_url
    fiscal_my_soliq_api_key         = local.secret_fiscal_my_soliq_api_key
    fiscal_my_soliq_tin             = local.secret_fiscal_my_soliq_tin
    fiscal_my_soliq_signer          = local.secret_fiscal_my_soliq_signer
    fiscal_my_soliq_pkcs12_password = local.secret_fiscal_my_soliq_pkcs12_password
    playmobile_login                = local.secret_playmobile_login
    playmobile_password             = local.secret_playmobile_password
  }
  fiscal_sms_secret_data = {
    fiscal_my_soliq_base_url        = var.fiscal_my_soliq_base_url
    fiscal_my_soliq_api_key         = var.fiscal_my_soliq_api_key
    fiscal_my_soliq_tin             = var.fiscal_my_soliq_tin
    fiscal_my_soliq_signer          = var.fiscal_my_soliq_signer
    fiscal_my_soliq_pkcs12_password = var.fiscal_my_soliq_pkcs12_password
    playmobile_login                = var.playmobile_login
    playmobile_password             = var.playmobile_password
  }
}

resource "google_secret_manager_secret" "fiscal_sms" {
  for_each  = local.fiscal_sms_secret_ids
  secret_id = each.value
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "fiscal_sms" {
  for_each    = { for k, v in local.fiscal_sms_secret_data : k => v if trimspace(v) != "" }
  secret      = google_secret_manager_secret.fiscal_sms[each.key].id
  secret_data = each.value
}
