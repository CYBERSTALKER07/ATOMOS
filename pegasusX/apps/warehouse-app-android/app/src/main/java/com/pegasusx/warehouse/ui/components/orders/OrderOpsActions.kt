package com.pegasusx.warehouse.ui.components.orders

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridScope
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

fun LazyGridScope.orderOpsActions(
    canDelay: Boolean,
    canOverflow: Boolean,
    canReject: Boolean,
    mutating: Boolean,
    onProposeNewDate: () -> Unit,
    onOverflow: () -> Unit,
    onReject: () -> Unit
) {
    val showOps = canDelay || canReject || canOverflow
    if (!showOps) return

    item(span = { GridItemSpan(maxLineSpan) }) {
        HorizontalDivider()
        Spacer(Modifier.height(PegasusSpacing.xs))
        Text("Warehouse actions", style = MaterialTheme.typography.titleMedium)
        Spacer(Modifier.height(PegasusSpacing.xs))
        Row(
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (canDelay) {
                OutlinedButton(
                    onClick = onProposeNewDate,
                    enabled = !mutating,
                    modifier = Modifier.weight(1f),
                ) { Text("Propose new date") }
            }
            if (canOverflow) {
                OutlinedButton(
                    onClick = onOverflow,
                    enabled = !mutating,
                    modifier = Modifier.weight(1f),
                ) { Text("Overflow") }
            }
            if (canReject) {
                OutlinedButton(
                    onClick = onReject,
                    enabled = !mutating,
                    modifier = Modifier.weight(1f),
                ) { Text("Reject") }
            }
        }
    }
}
