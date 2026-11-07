package com.pegasus.design.network

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

/**
 * Enterprise HTTP resilience interceptor:
 * 1. Coalesces identical concurrent GET requests (single-flight) and caches 2xx GET responses
 *    for a short 2.0s micro-cache window (invalidated immediately on any POST/PUT/PATCH/DELETE).
 * 2. Retries transient network errors and 502/503/504 gateway errors with exponential backoff
 *    and jitter on idempotent requests (GET or requests carrying Idempotency-Key).
 */
class PegasusHttpResilienceInterceptor(
    private val microCacheTtlMs: Long = 2_000L,
) : Interceptor {

    private data class CachedPayload(
        val code: Int,
        val message: String,
        val headers: okhttp3.Headers,
        val mediaType: okhttp3.MediaType?,
        val bodyBytes: ByteArray,
        val timestampMs: Long,
    )

    private val microCache = java.util.concurrent.ConcurrentHashMap<String, CachedPayload>()
    private val inFlightGets = java.util.concurrent.ConcurrentHashMap<String, java.util.concurrent.CompletableFuture<CachedPayload>>()

    fun invalidateCache() {
        microCache.clear()
    }

    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val method = request.method.uppercase()

        if (method != "GET") {
            microCache.clear()
            return proceedWithRetry(chain, request)
        }

        val authKey = request.header("Authorization").orEmpty().hashCode()
        val cacheKey = "${request.url}|$authKey"
        val now = System.currentTimeMillis()

        microCache[cacheKey]?.let { cached ->
            if (now - cached.timestampMs < microCacheTtlMs) {
                return buildResponse(request, cached)
            } else {
                microCache.remove(cacheKey, cached)
            }
        }

        val newFuture = java.util.concurrent.CompletableFuture<CachedPayload>()
        val existingFuture = inFlightGets.putIfAbsent(cacheKey, newFuture)
        if (existingFuture != null) {
            return try {
                val shared = existingFuture.get(15, java.util.concurrent.TimeUnit.SECONDS)
                buildResponse(request, shared)
            } catch (_: Exception) {
                proceedWithRetry(chain, request)
            }
        }

        try {
            val response = proceedWithRetry(chain, request)
            val body = response.body
            val contentLength = body?.contentLength() ?: -1L
            if (response.isSuccessful && body != null && contentLength <= 262_144L) {
                val bytes = body.bytes()
                val mediaType = body.contentType()
                val payload = CachedPayload(
                    code = response.code,
                    message = response.message,
                    headers = response.headers,
                    mediaType = mediaType,
                    bodyBytes = bytes,
                    timestampMs = System.currentTimeMillis(),
                )
                microCache[cacheKey] = payload
                newFuture.complete(payload)
                @Suppress("DEPRECATION")
                return response.newBuilder()
                    .body(okhttp3.ResponseBody.create(mediaType, bytes))
                    .build()
            }
            newFuture.completeExceptionally(IllegalStateException("non_cacheable"))
            return response
        } catch (t: Throwable) {
            newFuture.completeExceptionally(t)
            throw t
        } finally {
            inFlightGets.remove(cacheKey, newFuture)
        }
    }

    private fun proceedWithRetry(chain: Interceptor.Chain, request: okhttp3.Request): Response {
        val method = request.method.uppercase()
        val isIdempotent = method == "GET" ||
            request.header("Idempotency-Key") != null ||
            request.header("X-Idempotency-Key") != null
        val maxAttempts = if (isIdempotent) 3 else 1
        var lastException: java.io.IOException? = null

        for (attempt in 0 until maxAttempts) {
            try {
                val response = chain.proceed(request)
                if ((response.code == 502 || response.code == 503 || response.code == 504) && attempt + 1 < maxAttempts) {
                    response.close()
                    sleepBackoff(attempt)
                    continue
                }
                return response
            } catch (cancelled: java.io.InterruptedIOException) {
                throw cancelled
            } catch (ioe: java.io.IOException) {
                lastException = ioe
                if (attempt + 1 < maxAttempts) {
                    sleepBackoff(attempt)
                    continue
                }
            }
        }
        throw lastException ?: java.io.IOException("Request failed")
    }

    private fun sleepBackoff(attempt: Int) {
        val jitter = java.util.concurrent.ThreadLocalRandom.current().nextLong(0, 120)
        val delayMs = (300L shl attempt) + jitter
        try {
            Thread.sleep(delayMs)
        } catch (_: InterruptedException) {
            Thread.currentThread().interrupt()
        }
    }

    @Suppress("DEPRECATION")
    private fun buildResponse(request: okhttp3.Request, cached: CachedPayload): Response {
        return Response.Builder()
            .request(request)
            .protocol(okhttp3.Protocol.HTTP_1_1)
            .code(cached.code)
            .message(cached.message)
            .headers(cached.headers)
            .body(okhttp3.ResponseBody.create(cached.mediaType, cached.bodyBytes))
            .build()
    }
}
