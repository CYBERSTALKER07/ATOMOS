package com.pegasusx.warehouse.ui.screens.returns

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.AssistChip
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.InboundReturnRow
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

@Composable
fun ReturnsList(
    items: List<InboundReturnRow>,
    isQueueTab: Boolean,
    selected: Set<String>,
    onToggleSelect: (String) -> Unit
) {
    if (items.isEmpty()) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            Text(
                if (isQueueTab) "No returns at gate" else "No completed receives yet",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    } else {
        LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            modifier = Modifier.fillMaxSize(),
        ) {
            items(items, key = { it.returnId }) { r ->
                val checked = selected.contains(r.returnId)
                ElevatedCard(
                    modifier = Modifier.fillMaxWidth(),
                    onClick = {
                        if (isQueueTab) {
                            onToggleSelect(r.returnId)
                        }
                    },
                ) {
                    Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                r.productName,
                                style = MaterialTheme.typography.titleSmall,
                                modifier = Modifier.weight(1f),
                            )
                            AssistChip(onClick = {}, label = { Text(r.physicalStatus) })
                        }
                        Spacer(Modifier.height(PegasusSpacing.xs))
                        Text(
                            "Qty: ${r.receivedQty}/${r.expectedQty} · ${r.reason} · ${r.driverName}",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        if (r.barcode.isNotBlank()) {
                            Text(
                                "EAN ${r.barcode}",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }
        }
    }
}
