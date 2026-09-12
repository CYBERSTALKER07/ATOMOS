package com.pegasusx.factory.ui.screens.supply.components

import com.pegasusx.factory.data.model.SupplyRequest
import java.text.DateFormat
import java.util.Date

data class RequestActionSpec(
    val label: String,
    val action: String,
    val emphasized: Boolean,
)

val requestFilters = listOf("ALL", "SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY", "FULFILLED", "CANCELLED")
val boardLanes = listOf("SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY")

fun actionsForState(state: String): List<RequestActionSpec> = when (state) {
    "SUBMITTED" -> listOf(
        RequestActionSpec("Acknowledge", "ACKNOWLEDGE", true),
        RequestActionSpec("Cancel", "CANCEL", false),
    )
    "ACKNOWLEDGED" -> listOf(
        RequestActionSpec("Start production", "START_PRODUCTION", true),
        RequestActionSpec("Cancel", "CANCEL", false),
    )
    "IN_PRODUCTION" -> listOf(
        RequestActionSpec("Mark ready", "MARK_READY", true),
    )
    "READY" -> listOf(
        RequestActionSpec("Fulfill", "FULFILL", true),
    )
    else -> emptyList()
}

fun requestLabel(request: SupplyRequest): String =
    request.warehouseId.takeIf { it.isNotBlank() }?.take(8)?.let { "Warehouse $it" } ?: "Warehouse"

fun formatDate(value: String?): String {
    if (value.isNullOrBlank()) return "Unscheduled"
    return value.substringBefore('T')
}

fun trimDecimal(value: Double): String =
    if (value % 1.0 == 0.0) value.toInt().toString() else String.format("%.1f", value)

fun slaBadgeVisible(status: String?): Boolean {
    val normalized = status.orEmpty().uppercase()
    return normalized.isNotBlank() && normalized != "N/A" && normalized != "MET"
}

fun slaHoursLabel(hours: Double?): String? {
    if (hours == null) return null
    return if (hours > 0) "${hours.toInt()}h left" else "${kotlin.math.abs(hours).toInt()}h overdue"
}

fun formatSyncTime(value: Long?): String {
    if (value == null) return "waiting"
    return DateFormat.getTimeInstance(DateFormat.SHORT).format(Date(value))
}
