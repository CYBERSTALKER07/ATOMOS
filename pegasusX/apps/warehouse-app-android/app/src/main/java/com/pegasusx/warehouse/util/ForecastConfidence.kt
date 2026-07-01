package com.pegasusx.warehouse.util

import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.longOrNull

data class ForecastConfidenceData(
    val lowUnits: Long? = null,
    val highUnits: Long? = null,
    val confidencePct: Int? = null,
    val baselineSource: String? = null,
    val blockedReason: String? = null,
    val label: String? = null,
) {
    val blocked: Boolean
        get() = !blockedReason.isNullOrBlank() || label == "insufficient_history"

    val seasonalActive: Boolean
        get() = baselineSource == "seasonal_template"
}

fun parseForecastConfidence(breakdown: JsonObject?): ForecastConfidenceData? {
    if (breakdown == null || breakdown.isEmpty()) return null
    val low = jsonLong(breakdown["low_units"])
    val high = jsonLong(breakdown["high_units"])
    var confidencePct = jsonLong(breakdown["confidence_pct"])?.toInt()
    if (confidencePct == null) {
        val raw = jsonDouble(breakdown["confidence"])
        if (raw != null) {
            confidencePct = if (raw <= 1.0) (raw * 100).toInt() else raw.toInt()
        }
    }
    val baselineSource = jsonString(breakdown["baseline_source"])
    val blockedReason = jsonString(breakdown["blocked_reason"])
    val label = jsonString(breakdown["label"])
    val predicted = jsonLong(breakdown["predictedQty"]) ?: jsonLong(breakdown["predicted_qty"])
    val derivedLow = low ?: predicted?.let { (it * 0.9).toLong().coerceAtLeast(0) }
    val derivedHigh = high ?: predicted?.let { (it * 1.1).toLong().coerceAtLeast(it) }

    if (derivedLow == null && derivedHigh == null && confidencePct == null
        && blockedReason.isNullOrBlank() && baselineSource.isNullOrBlank() && label.isNullOrBlank()
    ) {
        return null
    }
    return ForecastConfidenceData(
        lowUnits = derivedLow,
        highUnits = derivedHigh ?: derivedLow,
        confidencePct = confidencePct,
        baselineSource = baselineSource,
        blockedReason = blockedReason,
        label = label,
    )
}

fun formatSourceBadge(source: String?): String = when (source) {
    "ml" -> "ML"
    "moving_average" -> "Baseline"
    "seasonal_template" -> "Seasonal"
    "mixed" -> "Mixed"
    null, "" -> ""
    else -> source.replace('_', ' ')
}

private fun jsonLong(element: JsonElement?): Long? {
    val primitive = element as? JsonPrimitive ?: return null
    return primitive.longOrNull ?: primitive.content.trim('"').toLongOrNull()
}

private fun jsonDouble(element: JsonElement?): Double? {
    val primitive = element as? JsonPrimitive ?: return null
    return primitive.doubleOrNull ?: primitive.content.trim('"').toDoubleOrNull()
}

private fun jsonString(element: JsonElement?): String? {
    val primitive = element as? JsonPrimitive ?: return null
    val raw = primitive.content.trim('"')
    return raw.ifBlank { null }
}
