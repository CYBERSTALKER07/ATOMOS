package com.pegasusx.warehouse.ui.screens.orders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.Order
import com.pegasusx.warehouse.data.model.WarehousePreorderRow
import com.pegasusx.warehouse.ui.components.OrderOpsCard
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.orderActionFlags
import java.text.NumberFormat
import java.util.Locale

@Composable
fun OrdersList(
    hubTab: Int,
    loading: Boolean,
    error: String?,
    orders: List<Order>,
    preorders: List<WarehousePreorderRow>,
    actingId: String?,
    onRetry: () -> Unit,
    onOrderClick: (String) -> Unit,
    onProposeActive: (String) -> Unit,
    onRejectActive: (String) -> Unit,
    onReassignActive: (String) -> Unit,
    onProposePreorder: (WarehousePreorderRow) -> Unit,
    onRejectPreorder: (WarehousePreorderRow) -> Unit,
    modifier: Modifier = Modifier,
) {
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    when {
        loading && (if (hubTab == 0) orders.isEmpty() else preorders.isEmpty()) ->
            Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
        error != null && (if (hubTab == 0) orders.isEmpty() else preorders.isEmpty()) ->
            Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = onRetry) { Text("Retry") }
                }
            }
        hubTab == 0 && orders.isEmpty() ->
            Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text(
                    "No orders",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        hubTab == 1 && preorders.isEmpty() ->
            Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text(
                    "No scheduled pre-orders",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        hubTab == 0 -> LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            modifier = modifier.fillMaxSize(),
        ) {
            items(orders, key = { it.orderId }) { order ->
                val flags = orderActionFlags(order.state)
                OrderOpsCard(
                    retailerName = order.retailerName,
                    orderId = order.orderId,
                    state = order.state,
                    amountLabel = "${fmt.format(order.totalUzs)} ${com.pegasus.design.sessionPackCurrency()}",
                    enabled = actingId != order.orderId,
                    canDelay = flags.canDelay,
                    canReject = flags.canReject,
                    canReassign = flags.canReassign,
                    onOpenDetail = { onOrderClick(order.orderId) },
                    onDelay = { onProposeActive(order.orderId) },
                    onReject = { onRejectActive(order.orderId) },
                    onReassign = { onReassignActive(order.orderId) },
                )
            }
        }
        else -> LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            modifier = modifier.fillMaxSize(),
        ) {
            items(preorders, key = { it.orderId }) { row ->
                OrderOpsCard(
                    retailerName = row.orderId.take(12),
                    orderId = row.orderId,
                    state = row.status,
                    amountLabel = row.requestedDeliveryDate?.take(10) ?: "Pre-order",
                    meta = row.proposedDeliveryDate?.let { "Proposed: $it" },
                    badge = "Pre-order",
                    delayLabel = "Propose delivery",
                    rejectLabel = "Reject",
                    canDelay = true,
                    canReject = true,
                    onOpenDetail = { onOrderClick(row.orderId) },
                    onDelay = { onProposePreorder(row) },
                    onReject = { onRejectPreorder(row) },
                )
            }
        }
    }
}
