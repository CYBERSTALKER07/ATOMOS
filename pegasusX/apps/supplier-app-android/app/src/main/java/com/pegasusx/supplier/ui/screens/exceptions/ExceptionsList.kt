package com.pegasusx.supplier.ui.screens.exceptions

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierExceptionRow
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun ExceptionsList(rows: List<SupplierExceptionRow>) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        items(rows, key = { it.orderId + it.kind }) { row ->
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(PegasusSpacing.lg)) {
                    Text(row.orderId, style = MaterialTheme.typography.titleMedium)
                    Text("${row.kind} · ${row.status}", style = MaterialTheme.typography.bodyMedium)
                    row.note?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
                    row.manifestId?.let {
                        Text("Manifest $it", style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
        }
    }
}
