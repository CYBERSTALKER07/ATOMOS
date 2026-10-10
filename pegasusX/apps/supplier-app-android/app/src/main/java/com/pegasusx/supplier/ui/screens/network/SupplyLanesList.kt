package com.pegasusx.supplier.ui.screens.network

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierSupplyLaneRow
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlin.math.min
import com.pegasusx.supplier.R

@Composable
fun SupplyLanesList(lanes: List<SupplierSupplyLaneRow>, modifier: Modifier = Modifier) {
    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(lanes, key = { it.laneId }) { lane ->
            LaneCard(lane)
        }
    }
}

@Composable
private fun LaneCard(lane: SupplierSupplyLaneRow) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(
            Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(lane.name.ifEmpty { lane.warehouseId }, style = MaterialTheme.typography.titleMedium)
                Text(stringResource(R.string.mobile_supplier_ui_h3cells_cells, lane.h3Cells), style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.primary)
            }
            LaneMetric("Active drivers", lane.drivers.toString())
            LaneMetric("Orders today", lane.ordersToday.toString())
            LaneMetric("Capacity limit", lane.capacity.toString())
            val tint = when {
                lane.utilizationPct > 85 -> MaterialTheme.colorScheme.error
                lane.utilizationPct > 75 -> MaterialTheme.colorScheme.tertiary
                else -> MaterialTheme.colorScheme.primary
            }
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text("Lane utilization", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
                Text("%.0f%%".format(lane.utilizationPct), style = MaterialTheme.typography.bodySmall, color = tint)
            }
            LinearProgressIndicator(
                progress = { (min(100.0, maxOf(0.0, lane.utilizationPct)) / 100.0).toFloat() },
                modifier = Modifier.fillMaxWidth(),
                color = tint,
            )
        }
    }
}

@Composable
private fun LaneMetric(label: String, value: String) {
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(label, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.outline)
        Text(value, style = MaterialTheme.typography.bodyMedium)
    }
}
