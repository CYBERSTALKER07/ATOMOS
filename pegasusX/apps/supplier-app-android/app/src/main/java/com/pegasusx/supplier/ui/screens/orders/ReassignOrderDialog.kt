package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.RecommendReassignResponse

@Composable
fun ReassignOrderDialog(
    orderId: String,
    recs: RecommendReassignResponse?,
    isReassigning: Boolean,
    onDismiss: () -> Unit,
    onApplyReassign: (String, String, Boolean) -> Unit
) {
    AlertDialog(
        onDismissRequest = { if (!isReassigning) onDismiss() },
        title = { Text(stringResource(R.string.mobile_supplier_ui_reassign_order_take, orderId.take(8))) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (recs == null) {
                    CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally))
                } else if (recs.recommendations.isEmpty()) {
                    Text("No suitable trucks available.")
                } else {
                    Text(
                        stringResource(R.string.mobile_supplier_ui_retailername_1f_vu, recs.retailerName).format(recs.orderVolumeVu),
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(6.dp),
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(280.dp),
                    ) {
                        items(recs.recommendations, key = { it.driverId }) { rec ->
                            ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                    Row(verticalAlignment = Alignment.CenterVertically) {
                                        Text(
                                            rec.driverName.ifBlank { rec.driverId.take(8) },
                                            style = MaterialTheme.typography.titleSmall,
                                            modifier = Modifier.weight(1f)
                                        )
                                        Text("score %.2f".format(rec.score), style = MaterialTheme.typography.labelMedium)
                                    }
                                    Text(
                                        listOfNotNull(
                                            rec.licensePlate.takeIf { it.isNotBlank() },
                                            rec.vehicleClass.takeIf { it.isNotBlank() }
                                        ).joinToString(" • "),
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.spacedBy(8.dp, Alignment.End)
                                    ) {
                                        OutlinedButton(
                                            onClick = {
                                                onApplyReassign(orderId, rec.driverId, true)
                                            },
                                            enabled = !isReassigning,
                                            contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp),
                                        ) { Text("Partial") }
                                        Button(
                                            onClick = {
                                                onApplyReassign(orderId, rec.driverId, false)
                                            },
                                            enabled = !isReassigning,
                                            contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp),
                                        ) { Text("Complete") }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {},
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !isReassigning) {
                Text("Close")
            }
        }
    )
}
