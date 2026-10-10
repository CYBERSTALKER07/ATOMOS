package com.pegasusx.warehouse.ui.screens.returns

import androidx.compose.ui.res.stringResource

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
import com.pegasusx.warehouse.R

@Composable
fun ReturnsList(
    items: List<InboundReturnRow>,
    isQueueTab: Boolean,
    selected: Set<String>,
    onToggleSelect: (String) -> Unit,
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
                                r.productName.ifBlank { r.returnId },
                                style = MaterialTheme.typography.titleSmall,
                                modifier = Modifier.weight(1f),
                            )
                            AssistChip(onClick = {}, label = { Text(r.physicalStatus) })
                        }
                        if (r.isClaimTicket) {
                            Spacer(Modifier.height(PegasusSpacing.xs))
                            AssistChip(onClick = {}, label = { Text("Claim ticket") })
                        }
                        Spacer(Modifier.height(PegasusSpacing.xs))
                        Text(
                            "Qty: ${r.receivedQty}/${r.expectedQty} · ${r.reason} · ${
                                r.driverName.ifBlank { if (r.isClaimTicket) "store return" else "—" }
                            }",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        if (r.suggestedDisposition.isNotBlank()) {
                            Text(
                                stringResource(R.string.mobile_warehouse_ui_suggested_suggesteddisposition, r.suggestedDisposition),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        if (r.driverNotes.isNotBlank()) {
                            Text(
                                r.driverNotes,
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        if (r.barcode.isNotBlank()) {
                            Text(
                                stringResource(R.string.mobile_warehouse_ui_ean_barcode, r.barcode),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        if (isQueueTab && selected.contains(r.returnId)) {
                            Text(
                                "Selected",
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.primary,
                            )
                        }
                    }
                }
            }
        }
    }
}
