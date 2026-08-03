package com.pegasusx.warehouse.ui.screens.preorders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.WarehousePreorderRow

@Composable
fun PreordersList(
    rows: List<WarehousePreorderRow>,
    modifier: Modifier = Modifier,
    onPropose: (WarehousePreorderRow) -> Unit,
    onReject: (WarehousePreorderRow) -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        modifier = modifier.padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        items(rows, key = { it.orderId }) { row ->
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Text(row.orderId, style = MaterialTheme.typography.titleSmall)
                    Text("Status: ${row.status}")
                    row.requestedDeliveryDate?.let { Text("Delivery: $it") }
                    row.proposedDeliveryDate?.let { Text("Proposed: $it", color = MaterialTheme.colorScheme.primary) }
                    row.deliveryProposalReason?.let { Text("Reason: $it", style = MaterialTheme.typography.bodySmall) }
                    if (row.confirmationStatus == "PENDING_WAREHOUSE" || row.preorderBadge == "REVIEW_DELIVERY") {
                        AssistChip(onClick = {}, label = { Text("Awaiting retailer review") })
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        TextButton(onClick = { onPropose(row) }) { Text("Propose date") }
                        TextButton(onClick = { onReject(row) }) { Text("Reject") }
                    }
                }
            }
        }
    }
}
