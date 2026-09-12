package com.pegasusx.warehouse.ui.screens.replenishment

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.warehouse.data.model.ReplenishmentInsight
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.R

@Composable
fun ReplenishmentList(
    insights: List<ReplenishmentInsight>,
    actingId: String?,
    onApprove: (String) -> Unit,
    onDismiss: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        item(span = { GridItemSpan(maxLineSpan) }) {
            WarehouseSectionTitle("Open insights (${insights.size})")
        }
        items(insights, key = { it.id }) { insight ->
            InsightCard(
                insight = insight,
                busy = actingId == insight.id,
                onApprove = { onApprove(insight.id) },
                onDismiss = { onDismiss(insight.id) },
            )
        }
    }
}

@Composable
internal fun InsightCard(
    insight: ReplenishmentInsight,
    busy: Boolean,
    onApprove: () -> Unit,
    onDismiss: () -> Unit,
) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Row(modifier = Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
                    Text(insight.productName, style = MaterialTheme.typography.titleMedium)
                    if (insight.reasonCode == "PREDICTIVE_PUSH") {
                        Spacer(Modifier.width(PegasusSpacing.xs))
                        Surface(
                            color = MaterialTheme.colorScheme.onSurface,
                            shape = MaterialTheme.shapes.small
                        ) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            ) {
                                Text(
                                    "AI PUSH",
                                    style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp),
                                    color = MaterialTheme.colorScheme.surface
                                )
                            }
                        }
                    }
                }
                WarehouseStatusChip(status = insight.urgency)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                WarehouseStatusChip(status = insight.status)
            }
            Text(
                stringResource(R.string.mobile_warehouse_ui_stock_currentstock_reorder_reorderquantity, insight.currentStock, insight.reorderQuantity),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            insight.demandBreakdown?.let { breakdown ->
                Text(formatDemandWhy(breakdown, insight.reasonCode), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(
                stringResource(R.string.mobile_warehouse_ui_days_until_stockout_daysuntilstockout, insight.daysUntilStockout),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (insight.status.equals("OPEN", ignoreCase = true)) {
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    Button(onClick = onApprove, enabled = !busy) { Text("Approve") }
                    OutlinedButton(onClick = onDismiss, enabled = !busy) { Text("Dismiss") }
                }
            }
        }
    }
}

internal fun formatDemandWhy(breakdown: kotlinx.serialization.json.JsonObject?, reasonCode: String?): String {
    if (breakdown == null || breakdown.isEmpty()) {
        return reasonCode?.replace('_', ' ') ?: "Threshold breach"
    }
    breakdown["blocked_reason"]?.toString()?.trim('"')?.takeIf { it.isNotBlank() }?.let { blocked ->
        return if (blocked == "insufficient_history") {
            "Insufficient history — forecast blocked"
        } else {
            blocked.replace('_', ' ')
        }
    }
    val parts = mutableListOf<String>()
    breakdown["burn_rate_7d"]?.toString()?.trim('"')?.toDoubleOrNull()?.let { parts.add("Burn ${"%.1f".format(it)}/d") }
    breakdown["days_cover"]?.toString()?.trim('"')?.toDoubleOrNull()?.let { parts.add("${"%.1f".format(it)}d cover") }
    if (breakdown.containsKey("mei_network")) parts.add("MEIO network transfer")
    return parts.joinToString(" · ").ifBlank { reasonCode?.replace('_', ' ') ?: "Demand signal" }
}
