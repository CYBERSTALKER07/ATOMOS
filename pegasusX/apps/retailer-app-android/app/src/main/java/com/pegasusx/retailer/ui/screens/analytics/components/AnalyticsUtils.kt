package com.pegasusx.retailer.ui.screens.analytics.components

import androidx.compose.ui.graphics.Color
import java.text.NumberFormat
import java.util.Locale

internal val Purple600 = Color(0xFF6750A4)
internal val Purple200 = Color(0xFFD0BCFF)
internal val Green500 = Color(0xFF4CAF50)
internal val GoalRed = Color(0xFFE91E63)

internal val StateColors = mapOf(
    "COMPLETED" to Color(0xFF4CAF50),
    "ARRIVED" to Color(0xFF2196F3),
    "IN_TRANSIT" to Color(0xFFFF9800),
    "PENDING" to Color(0xFFFFC107),
    "LOADED" to Color(0xFF9C27B0),
    "CANCELLED" to Color(0xFFE91E63),
    "CANCELLED_BY_ADMIN" to Color(0xFFF44336),
)

fun formatAmount(value: Long): String {
    val formatter = NumberFormat.getNumberInstance(Locale.US)
    return "${formatter.format(value)}"
}

fun formatCompact(value: Long): String {
    return when {
        value >= 1_000_000 -> "${value / 1_000_000}.${(value % 1_000_000) / 100_000}M"
        value >= 1_000 -> "${value / 1_000},${(value % 1_000) / 100}00"
        else -> "$value"
    }
}
