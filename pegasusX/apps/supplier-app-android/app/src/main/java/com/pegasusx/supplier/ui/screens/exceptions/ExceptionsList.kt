package com.pegasusx.supplier.ui.screens.exceptions

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierExceptionRow
import com.pegasusx.supplier.ui.theme.PegasusSpacing

private val resolvableKinds = setOf("CASH_DISCREPANCY", "CREDIT_NOTE_DRAFT", "CREDIT_FREEZE")

@Composable
fun ExceptionsList(
    rows: List<SupplierExceptionRow>,
    busyKey: String? = null,
    onResolve: (SupplierExceptionRow) -> Unit = {},
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        items(rows, key = { it.orderId + it.kind }) { row ->
            ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                Column(
                    Modifier.padding(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Text(row.orderId, style = MaterialTheme.typography.titleMedium)
                    Text(stringResource(R.string.mobile_supplier_ui_kind_status, row.kind, row.status), style = MaterialTheme.typography.bodyMedium)
                    row.note?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
                    row.manifestId?.let {
                        Text(stringResource(R.string.mobile_supplier_ui_manifest_it, it), style = MaterialTheme.typography.bodySmall)
                    }
                    if (row.kind.uppercase() in resolvableKinds) {
                        val key = "${row.kind}:${row.orderId}"
                        Button(
                            onClick = { onResolve(row) },
                            enabled = busyKey != key,
                            modifier = Modifier.padding(top = PegasusSpacing.xs),
                        ) {
                            Text("Resolve")
                        }
                    }
                }
            }
        }
    }
}
