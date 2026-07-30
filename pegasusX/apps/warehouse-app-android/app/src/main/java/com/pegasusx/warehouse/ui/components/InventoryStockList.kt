package com.pegasusx.warehouse.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.InventoryItem
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.screens.inventory.InventoryPolicyPicker

/**
 * Inventory stock list grid.
 *
 * Renders product cards with quantity display, low-stock badges,
 * OOS policy pickers, and adjust buttons.
 */
@Composable
fun InventoryStockList(
    items: List<InventoryItem>,
    policySavingId: String?,
    modifier: Modifier = Modifier,
    onAdjust: (InventoryItem) -> Unit,
    onPolicyChange: (item: InventoryItem, policy: String) -> Unit,
) {
    if (items.isEmpty()) {
        PegasusStatePane(
            kind = PegasusStateKind.Empty,
            headline = "No Inventory Items",
            body = "There are no matching items.",
            modifier = modifier,
        )
    } else {
        LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            modifier = modifier,
        ) {
            items(items, key = { it.productId }) { item ->
                val isLow = item.quantity <= item.reorderThreshold
                val currentPolicy = item.outOfStockPolicy?.takeIf { it.isNotBlank() } ?: "INHERIT"
                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(item.productName, style = MaterialTheme.typography.titleSmall)
                                Text(
                                    "Qty: ${item.quantity} · Reorder at: ${item.reorderThreshold}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            if (isLow) {
                                AssistChip(
                                    onClick = {},
                                    label = { Text("LOW", style = MaterialTheme.typography.labelSmall) },
                                    colors = AssistChipDefaults.assistChipColors(containerColor = MaterialTheme.colorScheme.errorContainer),
                                )
                            }
                            Spacer(Modifier.width(PegasusSpacing.sm))
                            TextButton(onClick = { onAdjust(item) }) { Text("Adjust") }
                        }
                        Spacer(Modifier.height(PegasusSpacing.sm))
                        InventoryPolicyPicker(
                            currentPolicy = currentPolicy,
                            saving = policySavingId == item.productId,
                            onSelect = { policy -> onPolicyChange(item, policy) },
                        )
                    }
                }
            }
        }
    }
}
