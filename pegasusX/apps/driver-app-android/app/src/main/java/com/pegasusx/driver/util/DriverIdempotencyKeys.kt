package com.pegasusx.driver.util

import com.pegasusx.driver.data.remote.TokenHolder

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object DriverIdempotencyKeys {
    private fun driverId(): String = TokenHolder.userId?.takeIf { it.isNotBlank() } ?: "driver"

    fun deliver(orderId: String): String = "driver-deliver:${driverId()}:$orderId"

    fun offload(orderId: String): String = "driver-offload:${driverId()}:$orderId"

    fun complete(orderId: String): String = "driver-complete:${driverId()}:$orderId"

    fun collectCash(orderId: String): String = "driver-collect-cash:${driverId()}:$orderId"

    fun confirmPaymentBypass(orderId: String): String = "driver-confirm-payment-bypass:${driverId()}:$orderId"
}
