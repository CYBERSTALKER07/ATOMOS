package com.pegasusx.supplier.ui.screens.manifests

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierManifestRow
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun ManifestsList(
    rows: List<SupplierManifestRow>,
    modifier: Modifier = Modifier,
    onOpenManifest: (String) -> Unit,
) {
    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(rows, key = { it.manifestId }) { row ->
            val state = row.state.ifBlank { row.status }
            SupplierOpsListCard(
                headline = row.manifestId.take(12),
                supporting = buildString {
                    append("${row.ordersCount} orders")
                    if (row.stopCount > 0) append(" · ${row.stopCount} stops")
                    val driver = row.driverName.ifBlank { row.driverId.orEmpty() }
                    if (driver.isNotBlank()) append(" · $driver")
                    row.vehiclePlate?.takeIf { it.isNotBlank() }?.let { append(" · $it") }
                },
                status = state,
                onClick = { onOpenManifest(row.manifestId) },
            )
        }
    }
}
