package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierOrder
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun OrdersList(
    orders: List<SupplierOrder>,
    onReassign: ((SupplierOrder) -> Unit)?
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(orders, key = { it.orderId }) { order ->
            val amount = formatMinorAmount(order.totalMinor, order.currency)
            SupplierOpsListCard(
                headline = order.orderId.take(12),
                supporting = buildString {
                    append(amount)
                    append(" · Retailer ")
                    append(order.retailerId.take(8))
                    order.updatedAt.takeIf { it.isNotBlank() }?.let { append(" · $it") }
                },
                status = order.status.ifBlank { order.decision },
                onReassign = onReassign?.let { { it(order) } },
            )
        }
    }
}
