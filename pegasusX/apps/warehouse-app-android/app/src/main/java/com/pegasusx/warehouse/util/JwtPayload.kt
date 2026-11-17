package com.pegasusx.warehouse.util

import org.json.JSONObject
import java.util.Base64

object JwtPayload {
    fun isConfigured(token: String?): Boolean {
        return decode(token)?.optBoolean("is_configured", false) ?: false
    }

    fun homeNodeId(token: String?): String? = stringClaim(token, "home_node_id")

    fun homeCell(token: String?): String? = stringClaim(token, "home_cell")

    fun supplierId(token: String?): String? =
        stringClaim(token, "supplier_id") ?: stringClaim(token, "org_id")

    fun subject(token: String?): String? =
        stringClaim(token, "sub") ?: stringClaim(token, "email")

    private fun stringClaim(token: String?, key: String): String? {
        return runCatching {
            decode(token)?.optString(key)?.takeIf { it.isNotBlank() }
        }.getOrNull()
    }

    private fun decode(token: String?): JSONObject? {
        if (token.isNullOrBlank()) return null
        val parts = token.split('.')
        if (parts.size < 2) return null
        return runCatching {
            val decoded = Base64.getUrlDecoder().decode(parts[1])
            JSONObject(String(decoded, Charsets.UTF_8))
        }.getOrNull()
    }
}
