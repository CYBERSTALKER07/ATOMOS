package com.pegasusx.warehouse.util

import android.util.Base64
import org.json.JSONObject

object JwtPayload {
    fun isConfigured(token: String?): Boolean {
        if (token.isNullOrBlank()) return false
        val parts = token.split('.')
        if (parts.size < 2) return false
        return runCatching {
            val decoded = Base64.decode(parts[1], Base64.URL_SAFE or Base64.NO_WRAP or Base64.NO_PADDING)
            JSONObject(String(decoded)).optBoolean("is_configured", false)
        }.getOrDefault(false)
    }

    fun homeNodeId(token: String?): String? {
        if (token.isNullOrBlank()) return null
        val parts = token.split('.')
        if (parts.size < 2) return null
        return runCatching {
            val decoded = Base64.decode(parts[1], Base64.URL_SAFE or Base64.NO_WRAP or Base64.NO_PADDING)
            JSONObject(String(decoded)).optString("home_node_id").takeIf { it.isNotBlank() }
        }.getOrNull()
    }
}
