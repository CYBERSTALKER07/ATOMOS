package com.pegasusx.driver.offline

/**
 * Shared HTTP flush semantics for durable offline queues (§8.8).
 * 2xx / 409 → ACK (purge); 408/429/5xx → retry; other 4xx → DEAD/discard.
 */
object OfflineHttpSemantics {
    const val STATUS_PENDING = "PENDING"
    const val STATUS_DEAD = "DEAD"
    const val MAX_ATTEMPTS_DEFAULT = 8

    fun normalizeEndpoint(endpoint: String): String =
        endpoint.trim().removePrefix("/").removePrefix("api/")

    fun isRetryableHttp(code: Int): Boolean =
        code == 408 || code == 429 || code in 500..599

    fun isSuccessHttp(code: Int): Boolean =
        code in 200..299

    fun isConflictHttp(code: Int): Boolean =
        code == 409

    fun isIdempotentReplay409(statusCode: Int, responseBody: String?): Boolean {
        if (statusCode != 409) return false
        val body = responseBody?.lowercase() ?: return false
        val markers = listOf("idempotent", "duplicate", "already_exists", "already_processed")
        return markers.any { body.contains(it) }
    }

    fun isDeadHttp(code: Int): Boolean =
        code in 400..499 && !isRetryableHttp(code)

    fun isNetworkEnqueueable(error: Throwable): Boolean = when (error) {
        is java.io.IOException -> true
        else -> {
            val msg = error.message.orEmpty().lowercase()
            "unable to resolve host" in msg ||
                "failed to connect" in msg ||
                "timeout" in msg ||
                "connection reset" in msg
        }
    }

    enum class FlushOutcome {
        ACK,
        RETRY,
        DEAD,
    }

    fun outcomeForHttp(code: Int, body: String? = null): FlushOutcome = when {
        isSuccessHttp(code) || isIdempotentReplay409(code, body) -> FlushOutcome.ACK
        isRetryableHttp(code) -> FlushOutcome.RETRY
        else -> FlushOutcome.DEAD
    }
}
