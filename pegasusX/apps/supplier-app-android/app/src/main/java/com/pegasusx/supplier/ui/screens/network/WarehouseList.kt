package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.runtime.Composable
import com.pegasusx.supplier.data.model.SupplierTopologyWarehouse
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun WarehouseList(warehouses: List<SupplierTopologyWarehouse>) {
    LazyColumn(
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(warehouses, key = { it.warehouseId }) { warehouse ->
            val locationLabel = warehouse.address.ifBlank { "Coordinates on file" }
            SupplierOpsListCard(
                headline = warehouse.name.ifBlank { warehouse.warehouseId },
                supporting = "Radius ${warehouse.coverageRadiusKm} km · $locationLabel",
                status = if (warehouse.isActive) "ACTIVE" else "INACTIVE",
            )
        }
    }
}
