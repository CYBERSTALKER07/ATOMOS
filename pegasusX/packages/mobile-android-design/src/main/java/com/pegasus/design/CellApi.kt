package com.pegasus.design

import android.util.Base64
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.Interceptor
import okhttp3.Response
import org.json.JSONObject

/**
 * GS-R / GS-C5 leftover: pin Retrofit/OkHttp to session home_cell.
 * Local / emulator bootstraps stay on BuildConfig.
 */
object CellApi {
    val CELL_API_URLS: Map<String, String> = mapOf(
        "cell-uz" to "https://api.pegasusx.app",
        "cell-eu" to "https://api-eu.pegasusx.app",
        "cell-us" to "https://api-us.pegasusx.app",
        "cell-kz" to "https://api-kz.pegasusx.app",
    )

    fun trimApiBase(url: String): String = url.trim().trimEnd('/')

    fun isDevApiBootstrap(url: String): Boolean {
        val raw = trimApiBase(url).lowercase()
        if (raw.isEmpty()) return true
        if (raw == "/api" || raw.endsWith("/api")) return true
        val host = try {
            val withScheme = if (raw.contains("://")) raw else "http://$raw"
            java.net.URI(withScheme).host?.lowercase().orEmpty()
        } catch (_: Exception) {
            return true
        }
        if (host == "localhost" || host == "127.0.0.1" || host == "10.0.2.2") return true
        if (host.startsWith("192.168.") || host.startsWith("10.")) return true
        if (host.startsWith("172.")) {
            val second = host.split(".").getOrNull(1)?.toIntOrNull() ?: 0
            if (second in 16..31) return true
        }
        return false
    }

    fun homeCellFromJwt(token: String?): String {
        val parts = token?.split(".").orEmpty()
        if (parts.size < 2) return ""
        return try {
            val decoded = Base64.decode(parts[1], Base64.URL_SAFE or Base64.NO_WRAP or Base64.NO_PADDING)
            val json = JSONObject(String(decoded, Charsets.UTF_8))
            json.optString("home_cell").lowercase().trim()
        } catch (_: Exception) {
            ""
        }
    }

    fun pinApiBaseUrl(bootstrap: String, homeCell: String? = null, sessionApiUrl: String? = null): String {
        val boot = trimApiBase(bootstrap)
        if (isDevApiBootstrap(boot)) return boot.ifBlank { "http://localhost:8180" }
        val fromSession = trimApiBase(sessionApiUrl.orEmpty())
        if (fromSession.isNotEmpty()) return fromSession
        val cell = homeCell.orEmpty().lowercase().trim()
        return CELL_API_URLS[cell] ?: boot
    }
}

/** Rewrites request host to the pinned cell URL. Dev bootstrap is a no-op. */
class CellPinInterceptor(
    private val bootstrap: String,
    private val token: () -> String?,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val pinned = CellApi.pinApiBaseUrl(
            bootstrap,
            CellApi.homeCellFromJwt(token()),
            MarketPackStore.sessionApiUrl,
        )
        val target = pinned.toHttpUrlOrNull() ?: return chain.proceed(chain.request())
        val req = chain.request()
        if (req.url.host == target.host && req.url.scheme == target.scheme && req.url.port == target.port) {
            return chain.proceed(req)
        }
        if (CellApi.isDevApiBootstrap(bootstrap)) {
            return chain.proceed(req)
        }
        val newUrl = req.url.newBuilder()
            .scheme(target.scheme)
            .host(target.host)
            .port(target.port)
            .build()
        return chain.proceed(req.newBuilder().url(newUrl).build())
    }
}
