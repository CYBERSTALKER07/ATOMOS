package com.pegasus.payload.ui.design

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

/** Pure 64-bit integer minor-unit formatter without IEEE-754 float division. */
fun formatMinorDecimal(minor: Long, decimalPlaces: Int = 2, groupSeparator: Char? = ' '): String {
    val places = decimalPlaces.coerceIn(0, 18)
    var scale = 1L
    repeat(places) { scale *= 10L }
    val sign = if (minor < 0L) "-" else ""
    val wholeStr = if (places == 0) {
        if (minor == Long.MIN_VALUE) "9223372036854775808" else kotlin.math.abs(minor).toString()
    } else {
        kotlin.math.abs(minor / scale).toString()
    }
    val groupedWhole = if (groupSeparator != null) {
        buildString(wholeStr.length + wholeStr.length / 3) {
            wholeStr.forEachIndexed { idx, ch ->
                if (idx > 0 && (wholeStr.length - idx) % 3 == 0) append(groupSeparator)
                append(ch)
            }
        }
    } else {
        wholeStr
    }
    if (places == 0) return "$sign$groupedWhole"
    val rem = kotlin.math.abs(minor % scale)
    if (rem == 0L) return "$sign$groupedWhole"
    val fracTrimmed = rem.toString().padStart(places, '0').trimEnd('0')
    return "$sign$groupedWhole.$fracTrimmed"
}

/** Fixed 2-decimal tiyin formatter (tiyin / 100L and abs(tiyin % 100L)). */
fun formatTiyinFixed(minor: Long, groupSeparator: Char? = null): String {
    val sign = if (minor < 0L) "-" else ""
    val wholeStr = kotlin.math.abs(minor / 100L).toString()
    val groupedWhole = if (groupSeparator != null) {
        buildString(wholeStr.length + wholeStr.length / 3) {
            wholeStr.forEachIndexed { idx, ch ->
                if (idx > 0 && (wholeStr.length - idx) % 3 == 0) append(groupSeparator)
                append(ch)
            }
        }
    } else {
        wholeStr
    }
    val rem = kotlin.math.abs(minor % 100L).toString().padStart(2, '0')
    return "$sign$groupedWhole.$rem"
}

/** Deterministic major-unit decimal string to 64-bit tiyin parser without float math. */
fun parseMajorToMinor(raw: String): Long? {
    val trimmed = raw.trim().replace(" ", "").replace("_", "")
    if (trimmed.isEmpty() || trimmed.startsWith('-')) return null
    if ('.' in trimmed && ',' in trimmed) return null
    val normalized = trimmed.replace(',', '.')
    if (normalized.count { it == '.' } > 1) return null
    val parts = normalized.split('.')
    val wholePart = parts[0]
    val fracPart = parts.getOrNull(1).orEmpty()
    if (wholePart.isEmpty() && fracPart.isEmpty()) return null
    if (!wholePart.all { it in '0'..'9' } || !fracPart.all { it in '0'..'9' }) return null
    if (fracPart.length > 2) return null
    val major = if (wholePart.isEmpty()) 0L else wholePart.toLongOrNull() ?: return null
    val minor = if (fracPart.isEmpty()) 0L else fracPart.padEnd(2, '0').toLongOrNull() ?: return null
    return runCatching { Math.addExact(Math.multiplyExact(major, 100L), minor) }.getOrNull()
}

/** Empty pack prints the number only. Never invents UZS. */
fun formatPackMoney(minor: Long, pack: MarketPack?, decimalPlaces: Int = 2): String {
    val formatted = formatMinorDecimal(minor, decimalPlaces, groupSeparator = ' ')
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
