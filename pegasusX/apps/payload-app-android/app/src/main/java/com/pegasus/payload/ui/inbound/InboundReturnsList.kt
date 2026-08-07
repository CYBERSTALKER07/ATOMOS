package com.pegasus.payload.ui.inbound

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
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

@Composable
fun InboundReturnsList(
    rows: List<InboundRow>,
    selectable: Boolean,
    selected: Set<String>,
    onToggleSelection: (String) -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        modifier = Modifier.fillMaxSize()
    ) {
        items(rows, key = { it.returnId }) { row ->
            val checked = selected.contains(row.returnId)
            ElevatedCard(
                modifier = Modifier.fillMaxWidth(),
                onClick = {
                    if (selectable) {
                        onToggleSelection(row.returnId)
                    }
                },
            ) {
                Column(Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(row.productName, style = MaterialTheme.typography.titleSmall, modifier = Modifier.weight(1f))
                        AssistChip(onClick = {}, label = { Text(row.physicalStatus) })
                    }
                    Spacer(Modifier.height(4.dp))
                    Text(
                        stringResource(R.string.mobile_payload_ui_qty_receivedqty_expectedqty_reason, row.receivedQty, row.expectedQty, row.reason),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    if (row.barcode.isNotBlank()) {
                        Text(
                            stringResource(R.string.mobile_payload_ui_ean_barcode_3, row.barcode),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }
        }
    }
}
