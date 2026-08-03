package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierSupplyLaneRow
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun GeoReportLanesList(lanes: List<SupplierSupplyLaneRow>, modifier: Modifier = Modifier) {
    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        item {
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(PegasusSpacing.lg)) {
                    Text(
                        "Estimated H3 cells in service",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.outline,
                    )
                    Text(
                        lanes.sumOf { it.h3Cells }.toString(),
                        style = MaterialTheme.typography.headlineMedium,
                    )
                }
            }
        }
        items(lanes, key = { it.laneId }) { lane ->
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(PegasusSpacing.lg)) {
                    Text(lane.name.ifEmpty { lane.warehouseId }, style = MaterialTheme.typography.titleMedium)
                    Text(
                        "%d cells · %.0f%% utilization today".format(lane.h3Cells, lane.utilizationPct),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.outline,
                    )
                }
            }
        }
    }
}
