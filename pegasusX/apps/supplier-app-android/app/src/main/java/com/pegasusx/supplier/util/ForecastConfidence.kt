package com.pegasusx.supplier.util

import com.pegasusx.supplier.data.model.ForecastConfidence
import com.pegasusx.supplier.data.model.SupplierDemandSummaryResponse
import java.time.Instant
import java.time.temporal.ChronoUnit
import kotlin.math.max
import kotlin.math.roundToInt

private const val STALE_MINUTES = 30L

fun isForecastStale(generatedAt: String?): Boolean {
    if (generatedAt.isNullOrBlank()) return false
    return runCatching {
        val instant = Instant.parse(generatedAt)
        ChronoUnit.MINUTES.between(instant, Instant.now()) > STALE_MINUTES
    }.getOrDefault(false)
}

fun formatForecastUpdatedAt(generatedAt: String?): String? {
    if (generatedAt.isNullOrBlank()) return null
    return runCatching {
        val instant = Instant.parse(generatedAt)
        val mins = ChronoUnit.MINUTES.between(instant, Instant.now())
        when {
            mins < 1 -> "just now"
            mins < 60 -> "${mins}m ago"
            else -> generatedAt
        }
    }.getOrElse { generatedAt }
}

private fun mapBaselineSource(src: String?): String? = when (src) {
    "demand_forecast_baseline" -> "moving_average"
    "ai_recommendations" -> "ml"
    else -> src
}

fun forecastConfidenceFromDemand(summary: SupplierDemandSummaryResponse): ForecastConfidence {
    if (summary.predictionCount == 0) {
        return ForecastConfidence(
            baselineSource = mapBaselineSource(summary.baselineSource),
            blockedReason = "no_predictions",
            label = "insufficient_history",
        )
    }
    val mid = summary.totalPallets
    val spread = max(1, (mid * 0.1).roundToInt())
    val src = mapBaselineSource(summary.baselineSource)
    var confidence = 65
    when (src) {
        "seasonal_template" -> confidence = 75
        "ml" -> confidence = 85
        else -> if (summary.predictionCount >= 5) confidence = 72
    }
    return ForecastConfidence(
        lowUnits = max(0, mid - spread).toLong(),
        highUnits = (mid + spread).toLong(),
        confidencePct = confidence,
        baselineSource = src,
        label = if (summary.predictionCount < 3) "early_signal" else "standard",
    )
}
