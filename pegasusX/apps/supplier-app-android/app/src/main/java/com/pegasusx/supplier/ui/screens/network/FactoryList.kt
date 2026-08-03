package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.runtime.Composable
import com.pegasusx.supplier.data.model.SupplierTopologyFactory
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun FactoryList(factories: List<SupplierTopologyFactory>) {
    LazyColumn(
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(factories, key = { it.factoryId }) { factory ->
            SupplierOpsListCard(
                headline = factory.name.ifBlank { factory.factoryId },
                supporting = factory.address.ifBlank { "Coordinates on file" },
                status = if (factory.isActive) "ACTIVE" else "INACTIVE",
            )
        }
    }
}
