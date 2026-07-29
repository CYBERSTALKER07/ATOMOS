package com.pegasusx.retailer.ui.screens.tracking.components

import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.data.model.TrackingOrder
import com.pegasusx.retailer.ui.components.RetailerListCard

@Composable
fun OrderInfoCard(order: TrackingOrder) {
    val statusLabel = buildString {
        append(order.state.replace('_', ' '))
        if (order.liveLocationAvailable) append(" • Live GPS")
    }
    RetailerListCard(
        headline = order.supplierName.ifEmpty { "Unknown Supplier" },
        supporting = buildString {
            append(order.items.joinToString { "${it.productName} ×${it.quantity}" }.ifEmpty { "No items" })
            append(" · ")
            append(String.format("%,d", order.totalAmount))
        },
        status = statusLabel,
        modifier = Modifier.padding(16.dp),
    )
}
