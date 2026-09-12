package com.pegasus.payload.ui.home

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.QrCodeScanner
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasus.payload.data.model.LiveOrder
import com.pegasus.payload.ui.components.PayloadSectionTitle
import com.pegasus.payload.ui.components.PayloadSpacing
import com.pegasus.payload.R

@Composable
fun OrderChecklist(
    orders: List<LiveOrder>,
    loading: Boolean,
    selectedOrderId: String?,
    checkedItems: Set<String>,
    sealedOrderIds: Set<String>,
    dispatchCodes: Map<String, String>,
    sealingOrderId: String?,
    exceptionLoadingOrderId: String?,
    onSelectOrder: (String) -> Unit,
    onToggleItem: (String) -> Unit,
    onSealOrder: () -> Unit,
    canSealSelected: Boolean,
    onShowException: (String) -> Unit,
    onShowReDispatch: (String) -> Unit,
    onScanProduct: () -> Unit,
) {
    if (loading) {
        com.pegasus.design.PegasusLoadingState(
            title = stringResource(R.string.mobile_payload_ui_fetching_manifest),
            body = "Loading the checklist items assigned to this vehicle.",
            modifier = Modifier.fillMaxWidth().heightIn(min = 160.dp),
        )
        return
    }
    if (orders.isEmpty()) {
        PegasusStatePane(
            kind = PegasusStateKind.Empty,
            headline = "No live orders",
            body = "No LOADED orders for this vehicle yet. They appear once dispatch assigns them.",
            modifier = Modifier.fillMaxWidth().heightIn(min = 160.dp),
        )
        return
    }
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(20.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(PayloadSpacing.lg), verticalArrangement = Arrangement.spacedBy(PayloadSpacing.md)) {
            PayloadSectionTitle(
                title = stringResource(R.string.mobile_payload_ui_orders_size_size_2_sealed, sealedOrderIds.size, orders.size),
            )
            // Order chips
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth().height(160.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                items(orders, key = { it.orderId }) { order ->
                    OrderChip(
                        order = order,
                        selected = order.orderId == selectedOrderId,
                        sealed = order.orderId in sealedOrderIds,
                        dispatchCode = dispatchCodes[order.orderId],
                        onClick = { onSelectOrder(order.orderId) },
                    )
                }
            }
            val selected = orders.firstOrNull { it.orderId == selectedOrderId }
            if (selected != null) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        stringResource(R.string.mobile_payload_ui_items_take, selected.orderId.take(8)),
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold,
                    )
                    OutlinedButton(onClick = onScanProduct) {
                        Icon(Icons.Filled.QrCodeScanner, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.size(6.dp))
                        Text("Scan product")
                    }
                }
                if (selected.items.isEmpty()) {
                    Text(
                        "No line items on this order.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                } else {
                    LazyVerticalGrid(
                        columns = GridCells.Adaptive(minSize = 340.dp),
                        verticalArrangement = Arrangement.spacedBy(4.dp),
                        modifier = Modifier.fillMaxWidth().height(220.dp),
                        horizontalArrangement = Arrangement.spacedBy(4.dp)
                    ) {
                        items(selected.items, key = { it.lineItemId }) { item ->
                            ItemRow(
                                checked = item.lineItemId in checkedItems,
                                enabled = selected.orderId !in sealedOrderIds,
                                label = item.skuName.ifBlank { item.skuId },
                                quantity = item.quantity,
                                onToggle = { onToggleItem(item.lineItemId) },
                            )
                        }
                    }
                }
                if (selected.orderId in sealedOrderIds) {
                    Text(
                        stringResource(R.string.mobile_payload_ui_order_sealed_dispatch_code_orempty, dispatchCodes[selected.orderId].orEmpty()),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.tertiary,
                    )
                } else {
                    val sealing = sealingOrderId == selected.orderId
                    FilledTonalButton(
                        onClick = onSealOrder,
                        enabled = canSealSelected && !sealing,
                        modifier = Modifier.fillMaxWidth().height(48.dp),
                    ) {
                        if (sealing) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                        } else {
                            Icon(Icons.Filled.Lock, contentDescription = null)
                            Spacer(Modifier.size(8.dp))
                            Text("Seal Order", style = MaterialTheme.typography.labelLarge)
                        }
                    }
                    val excLoading = exceptionLoadingOrderId == selected.orderId
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        OutlinedButton(
                            onClick = { onShowException(selected.orderId) },
                            enabled = !excLoading,
                            modifier = Modifier.weight(1f).height(48.dp),
                            colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error),
                        ) {
                            if (excLoading) {
                                CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                            } else {
                                Icon(Icons.Filled.Warning, contentDescription = null, modifier = Modifier.size(18.dp))
                                Spacer(Modifier.size(6.dp))
                                Text("Remove", style = MaterialTheme.typography.labelLarge)
                            }
                        }
                        OutlinedButton(
                            onClick = { onShowReDispatch(selected.orderId) },
                            modifier = Modifier.weight(1f).height(48.dp),
                        ) {
                            Icon(Icons.Filled.SwapHoriz, contentDescription = null, modifier = Modifier.size(18.dp))
                            Spacer(Modifier.size(6.dp))
                            Text("Re-Dispatch", style = MaterialTheme.typography.labelLarge)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun OrderChip(
    order: LiveOrder,
    selected: Boolean,
    sealed: Boolean,
    dispatchCode: String?,
    onClick: () -> Unit,
) {
    val bg = when {
        sealed -> MaterialTheme.colorScheme.tertiaryContainer
        selected -> MaterialTheme.colorScheme.primaryContainer
        else -> MaterialTheme.colorScheme.surfaceContainerHigh
    }
    val fg = when {
        sealed -> MaterialTheme.colorScheme.onTertiaryContainer
        selected -> MaterialTheme.colorScheme.onPrimaryContainer
        else -> MaterialTheme.colorScheme.onSurface
    }
    Surface(
        color = bg,
        contentColor = fg,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier.fillMaxWidth().clickable(onClick = onClick),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
        ) {
            if (sealed) {
                Icon(Icons.Filled.CheckCircle, contentDescription = stringResource(R.string.mobile_payload_ui_sealed), modifier = Modifier.size(18.dp))
                Spacer(Modifier.size(8.dp))
            }
            Column(Modifier.weight(1f)) {
                Text(
                    stringResource(R.string.mobile_payload_ui_order_take, order.orderId.take(8)),
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                )
                Text(stringResource(R.string.mobile_payload_ui_size_itemif_else_s, order.items.size, if (order.items.size == 1) "" else "s"),
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            if (sealed && !dispatchCode.isNullOrBlank()) {
                Text(
                    dispatchCode,
                    style = MaterialTheme.typography.labelLarge,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                )
            }
        }
    }
}

@Composable
private fun ItemRow(
    checked: Boolean,
    enabled: Boolean,
    label: String,
    quantity: Int,
    onToggle: () -> Unit,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 48.dp)
            .clip(RoundedCornerShape(10.dp))
            .clickable(enabled = enabled, onClick = onToggle)
            .padding(horizontal = 8.dp, vertical = 8.dp),
    ) {
        Checkbox(checked = checked, onCheckedChange = { if (enabled) onToggle() }, enabled = enabled)
        Spacer(Modifier.size(8.dp))
        Text(
            label,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.weight(1f),
        )
        Text(
            stringResource(R.string.mobile_payload_ui_xquantity, quantity),
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium,
        )
    }
}
