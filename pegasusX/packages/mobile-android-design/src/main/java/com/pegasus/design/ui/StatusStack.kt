package com.pegasus.design.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp

enum class StatusStackMode { Empty, Zero, Live, Unavailable }

data class StatusStackRow(val key: String, val count: Int?, val share: Float)

data class StatusStackModel(
    val mode: StatusStackMode,
    val rows: List<StatusStackRow>,
    val total: Int,
)

val ORDER_STATUS_FUNNEL = listOf(
    "PENDING", "SCHEDULED", "AUTO_ACCEPTED", "BACKORDERED",
    "LOADED", "IN_TRANSIT", "DELAYED",
    "ARRIVED", "ARRIVED_SHOP_CLOSED",
    "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION", "DELIVERED_ON_CREDIT",
    "FISCALIZING", "FISCAL_FAILED", "RECONCILIATION_REQUIRED",
    "COMPLETED", "CANCELLED",
)

val MANIFEST_STATES = listOf(
    "DRAFT", "LOADING", "SEALED", "DISPATCHED", "COMPLETED", "CANCELLED",
)

val TRUCK_DUTY_STATUSES = listOf(
    "AVAILABLE",
    "IN_TRANSIT",
    "RETURNING_TO_WAREHOUSE",
    "OFF_SHIFT",
    "UNASSIGNED",
    "VEHICLE_INACTIVE",
    "UNAVAILABLE",
    "INACTIVE",
)

val FACTORY_TRANSFER_STATES = listOf(
    "CREATED", "APPROVED", "PENDING", "ASSIGNED", "LOADING",
    "DISPATCHED", "IN_TRANSIT", "ARRIVED", "RECEIVED", "CANCELLED", "REASSIGNED",
)

val FACTORY_VEHICLE_STATES = listOf("READY", "AVAILABLE", "UNAVAILABLE")

val FACTORY_DRIVER_DUTY = listOf("ON_SHIFT", "OFF_SHIFT")

fun canonicalizeOrderStatus(status: String): String {
    return when (status.trim().uppercase()) {
        "DISPATCHED" -> "LOADED"
        "EN_ROUTE" -> "IN_TRANSIT"
        "ARRIVING" -> "ARRIVED"
        "SHOP_CLOSED_PENDING" -> "ARRIVED_SHOP_CLOSED"
        else -> status.trim().uppercase()
    }
}

fun incrementOrderStatusCount(counts: Map<String, Int>?, status: String): Map<String, Int> {
    val next = ORDER_STATUS_FUNNEL.associateWith { 0 }.toMutableMap()
    for ((key, value) in counts.orEmpty()) {
        val normalized = canonicalizeOrderStatus(key)
        if (normalized in next) {
            next[normalized] = value
        }
    }
    val key = canonicalizeOrderStatus(status)
    if (key in next) {
        next[key] = (next[key] ?: 0) + 1
    }
    return next
}

fun statusStackModel(
    dictionary: List<String> = ORDER_STATUS_FUNNEL,
    counts: Map<String, Int>?,
    available: Boolean = true,
): StatusStackModel {
    if (!available) {
        return StatusStackModel(
            mode = StatusStackMode.Unavailable,
            rows = dictionary.map { StatusStackRow(it, null, 0f) },
            total = 0,
        )
    }
    if (counts == null) {
        return StatusStackModel(StatusStackMode.Empty, emptyList(), 0)
    }
    val rows = dictionary.map { key -> StatusStackRow(key, counts[key] ?: 0, 0f) }.toMutableList()
    val total = rows.sumOf { it.count ?: 0 }
    if (total > 0) {
        for (i in rows.indices) {
            rows[i] = rows[i].copy(share = (rows[i].count ?: 0).toFloat() / total.toFloat())
        }
    }
    return StatusStackModel(
        mode = if (total == 0) StatusStackMode.Zero else StatusStackMode.Live,
        rows = rows,
        total = total,
    )
}

@Composable
fun SourceChip(source: String, modifier: Modifier = Modifier) {
    Text(
        text = source.uppercase(),
        style = MaterialTheme.typography.labelSmall,
        modifier = modifier
            .border(1.dp, MaterialTheme.colorScheme.outline, RoundedCornerShape(50))
            .padding(horizontal = 8.dp, vertical = 2.dp)
            .testTag("gs-u-source-chip"),
    )
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun StatusStack(
    counts: Map<String, Int>?,
    modifier: Modifier = Modifier,
    dictionary: List<String> = ORDER_STATUS_FUNNEL,
    available: Boolean = true,
    source: String? = null,
    onSelect: ((String) -> Unit)? = null,
) {
    val model = statusStackModel(dictionary, counts, available)
    val chipSource = source
        ?: when (model.mode) {
            StatusStackMode.Unavailable -> "unavailable"
            StatusStackMode.Empty -> "empty"
            else -> "live"
        }
    Column(modifier = modifier.testTag("gs-u-status-stack"), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
            SourceChip(chipSource)
        }
        when (model.mode) {
            StatusStackMode.Empty -> Text("No status counts", style = MaterialTheme.typography.bodySmall)
            StatusStackMode.Unavailable -> Text("Status counts unavailable", style = MaterialTheme.typography.bodySmall)
            else -> Unit
        }
        if (model.mode == StatusStackMode.Live) {
            Row(
                Modifier
                    .fillMaxWidth()
                    .height(10.dp)
                    .clip(RoundedCornerShape(50))
                    .background(MaterialTheme.colorScheme.surfaceVariant),
            ) {
                model.rows.filter { it.share > 0f }.forEach { row ->
                    Box(
                        Modifier
                            .fillMaxHeight()
                            .weight(row.share)
                            .background(MaterialTheme.colorScheme.onSurface.copy(alpha = 0.35f + row.share * 0.65f)),
                    )
                }
            }
        }
        if (model.mode != StatusStackMode.Empty) {
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                model.rows.forEach { row ->
                    Row(
                        modifier = Modifier
                            .heightIn(min = 48.dp)
                            .border(1.dp, MaterialTheme.colorScheme.outline, RoundedCornerShape(8.dp))
                            .clickable(
                                enabled = onSelect != null,
                                role = Role.Button,
                                onClick = { onSelect?.invoke(row.key) },
                            )
                            .padding(horizontal = 8.dp, vertical = 4.dp)
                            .testTag("gs-u-chip-${row.key}"),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        Text(row.key.replace('_', ' '), style = MaterialTheme.typography.labelSmall)
                        Text(row.count?.toString() ?: "—", style = MaterialTheme.typography.labelMedium)
                    }
                }
            }
        }
    }
}
