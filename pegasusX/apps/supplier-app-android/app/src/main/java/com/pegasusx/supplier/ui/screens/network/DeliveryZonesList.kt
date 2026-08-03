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
import com.pegasusx.supplier.data.model.SupplierTopologyWarehouse
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun DeliveryZonesList(warehouses: List<SupplierTopologyWarehouse>, modifier: Modifier = Modifier) {
    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(warehouses, key = { it.warehouseId }) { node ->
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(PegasusSpacing.lg)) {
                    Text(node.name.ifEmpty { "Unnamed warehouse" }, style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Radius ${node.coverageRadiusKm} km · %.4f, %.4f".format(node.lat, node.lng),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.outline,
                    )
                }
            }
        }
        item {
            Text(
                "H3 perimeter and warehouse coverage are configured via topology.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.outline,
            )
        }
    }
}
