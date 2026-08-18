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
    val mapCenterLat: Double = 0.0,
    val mapCenterLng: Double = 0.0,
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

/** Empty pack prints the number only. Never invents UZS. */
fun formatPackMoney(minor: Long, pack: MarketPack?, decimalPlaces: Int = 2): String {
    val places = decimalPlaces.coerceAtLeast(0)
    val denom = Math.pow(10.0, places.toDouble())
    val units = minor / denom
    val nf = java.text.NumberFormat.getNumberInstance(java.util.Locale.US)
    nf.minimumFractionDigits = 0
    nf.maximumFractionDigits = places
    val formatted = nf.format(units).replace(',', ' ')
    val ccy = packCurrency(pack)
    return if (ccy.isEmpty()) formatted else "$formatted $ccy"
}

fun sessionPackCurrency(): String = packCurrency(MarketPackStore.pack)

/** Stored/event currency, else session pack. Empty pack does not invent UZS. */
fun moneyCurrency(raw: String?): String {
    val fromEvent = raw?.trim().orEmpty()
    if (fromEvent.isNotEmpty()) return fromEvent.uppercase()
    return sessionPackCurrency()
}

data class PackMapCenter(val lat: Double, val lng: Double)

/** Shipped pack camera. Empty/planned pack does not invent Tashkent. */
fun packMapCenter(pack: MarketPack?): PackMapCenter? {
    val lat = pack?.mapCenterLat ?: 0.0
    val lng = pack?.mapCenterLng ?: 0.0
    if (lat == 0.0 && lng == 0.0) return null
    return PackMapCenter(lat, lng)
}

fun sessionMapCenter(): PackMapCenter? = packMapCenter(MarketPackStore.pack)

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
                    mapCenterLat = it.optDouble("map_center_lat", 0.0),
                    mapCenterLng = it.optDouble("map_center_lng", 0.0),
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
