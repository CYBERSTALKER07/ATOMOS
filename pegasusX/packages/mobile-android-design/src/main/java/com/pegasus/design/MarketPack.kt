package com.pegasus.design

import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
import java.util.concurrent.atomic.AtomicReference

data class MarketPack(
    val code: String,
    val name: String,
    val timezone: String,
    val currencyCode: String,
    val fiscalAdapter: String,
    val mapsAdapter: String,
    val checkoutReadsThis: Boolean,
) {
    val receiptLabel: String get() = fiscalReceiptLabel(fiscalAdapter)
}

data class AuthSession(
    val marketCode: String,
    val homeCell: String,
    val apiUrl: String,
    val pack: MarketPack?,
    val checkoutReadsThis: Boolean,
)

fun fiscalReceiptLabel(adapter: String?): String {
    return when (adapter.orEmpty().trim().uppercase()) {
        "MY_SOLIQ" -> "Soliq"
        "COMMERCIAL", "PEGASUS", "FAKE" -> "commercial"
        "PEPPOL" -> "PEPPOL"
        "PLANNED", "" -> "planned"
        else -> adapter.orEmpty().trim().uppercase()
    }
}

fun packCurrency(pack: MarketPack?, fallback: String = ""): String {
    val code = pack?.currencyCode?.trim().orEmpty()
    return if (code.isNotEmpty()) code.uppercase() else fallback
}

object MarketPackStore {
    private val ref = AtomicReference<AuthSession?>(null)

    val current: AuthSession? get() = ref.get()
    val pack: MarketPack? get() = current?.pack
    val sessionApiUrl: String? get() = current?.apiUrl?.takeIf { it.isNotBlank() }

    fun set(session: AuthSession?) {
        ref.set(session)
    }

    fun clear() {
        ref.set(null)
    }
}

object MarketPackBinder {
    fun fetch(baseUrl: String, token: String): AuthSession? {
        if (token.isBlank()) return null
        val pinned = CellApi.pinApiBaseUrl(baseUrl, CellApi.homeCellFromJwt(token), MarketPackStore.sessionApiUrl)
        val url = URL("${pinned.trimEnd('/')}/v1/auth/session")
        val conn = (url.openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            setRequestProperty("Authorization", "Bearer $token")
            connectTimeout = 15_000
            readTimeout = 15_000
        }
        return try {
            if (conn.responseCode !in 200..299) return null
            val body = conn.inputStream.bufferedReader().use { it.readText() }
            parseAuthSession(body)?.also { MarketPackStore.set(it) }
        } catch (_: Exception) {
            null
        } finally {
            conn.disconnect()
        }
    }

    fun parseAuthSession(json: String): AuthSession? {
        return try {
            val root = JSONObject(json)
            val packObj = root.optJSONObject("pack")
            val pack = packObj?.let {
                MarketPack(
                    code = it.optString("code"),
                    name = it.optString("name"),
                    timezone = it.optString("timezone"),
                    currencyCode = it.optString("currency_code"),
                    fiscalAdapter = it.optString("fiscal_adapter"),
                    mapsAdapter = it.optString("maps_adapter"),
                    checkoutReadsThis = it.optBoolean("checkout_reads_this", false),
                )
            }
            AuthSession(
                marketCode = root.optString("market_code"),
                homeCell = root.optString("home_cell"),
                apiUrl = root.optString("api_url"),
                pack = pack,
                checkoutReadsThis = root.optBoolean("checkout_reads_this", false),
            )
        } catch (_: Exception) {
            null
        }
    }
}
