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
import androidx.compose.ui.unit.sp
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
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
            Row(
                modifier = Modifier.fillMaxWidth().padding(horizontal = 4.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    "ORDER_CHECKLIST",
                    style = MaterialTheme.typography.labelLarge,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Black,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    letterSpacing = 1.sp,
                )
                Text(
                    "${sealedOrderIds.size} / ${orders.size} SEALED",
                    style = MaterialTheme.typography.labelLarge,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Black,
                    color = if (orders.isNotEmpty() && sealedOrderIds.size == orders.size) MaterialTheme.colorScheme.tertiary else MaterialTheme.colorScheme.primary,
                    letterSpacing = 0.5.sp,
                )
            }
            // Order chips
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 160.dp),
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
                        "LINE_ITEMS · ORD-${selected.orderId.takeLast(6).uppercase()}",
                        style = MaterialTheme.typography.labelMedium,
                        fontFamily = FontFamily.Monospace,
                        fontWeight = FontWeight.Black,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        letterSpacing = 0.8.sp,
                    )
                    OutlinedButton(onClick = onScanProduct) {
                        Icon(Icons.Filled.QrCodeScanner, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.size(6.dp))
                        Text("SCAN", fontFamily = FontFamily.Monospace, fontWeight = FontWeight.Bold, fontSize = 11.sp)
                    }
                }
                if (selected.items.isEmpty()) {
                    Text(
                        "NO_ITEMS_IN_ORDER",
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                } else {
                    LazyVerticalGrid(
                        columns = GridCells.Adaptive(minSize = 220.dp),
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
                    Surface(
                        color = MaterialTheme.colorScheme.tertiary.copy(alpha = 0.12f),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier.padding(14.dp),
                        ) {
                            Icon(Icons.Filled.Lock, contentDescription = null, tint = MaterialTheme.colorScheme.tertiary)
                            Spacer(Modifier.width(8.dp))
                            Text(
                                "ORDER SEALED · ${dispatchCodes[selected.orderId].orEmpty()}",
                                style = MaterialTheme.typography.labelLarge,
                                fontFamily = FontFamily.Monospace,
                                fontWeight = FontWeight.Black,
                                color = MaterialTheme.colorScheme.tertiary,
                            )
                        }
                    }
                } else {
                    val sealing = sealingOrderId == selected.orderId
                    Button(
                        onClick = onSealOrder,
                        enabled = canSealSelected && !sealing,
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.fillMaxWidth().height(52.dp),
                    ) {
                        if (sealing) {
                            CircularProgressIndicator(modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
                        } else {
                            Icon(Icons.Filled.Lock, contentDescription = null, modifier = Modifier.size(20.dp))
                            Spacer(Modifier.width(8.dp))
                            Text(
                                "SEAL ORDER",
                                fontFamily = FontFamily.Monospace,
                                fontWeight = FontWeight.Black,
                                fontSize = 15.sp,
                                letterSpacing = 0.5.sp,
                            )
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
                            shape = RoundedCornerShape(12.dp),
                            modifier = Modifier.weight(1f).height(48.dp),
                            colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error),
                        ) {
                            if (excLoading) {
                                CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                            } else {
                                Icon(Icons.Filled.Warning, contentDescription = null, modifier = Modifier.size(18.dp))
                                Spacer(Modifier.size(6.dp))
                                Text("REPORT ISSUE", fontFamily = FontFamily.Monospace, fontWeight = FontWeight.Bold, fontSize = 12.sp)
                            }
                        }
                        OutlinedButton(
                            onClick = { onShowReDispatch(selected.orderId) },
                            shape = RoundedCornerShape(12.dp),
                            modifier = Modifier.weight(1f).height(48.dp),
                        ) {
                            Icon(Icons.Filled.SwapHoriz, contentDescription = null, modifier = Modifier.size(18.dp))
                            Spacer(Modifier.size(6.dp))
                            Text("RE-DISPATCH", fontFamily = FontFamily.Monospace, fontWeight = FontWeight.Bold, fontSize = 12.sp)
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
        sealed -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.25f)
        selected -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.45f)
        else -> MaterialTheme.colorScheme.surface
    }
    Surface(
        color = bg,
        shape = RoundedCornerShape(16.dp),
        border = BorderStroke(
            width = if (selected) 2.dp else 1.dp,
            color = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f),
        ),
        modifier = Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).clickable(onClick = onClick),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 14.dp, vertical = 12.dp),
        ) {
            if (sealed) {
                Icon(
                    Icons.Filled.CheckCircle,
                    contentDescription = stringResource(R.string.mobile_payload_ui_sealed),
                    tint = MaterialTheme.colorScheme.tertiary,
                    modifier = Modifier.size(20.dp),
                )
            } else {
                Box(
                    modifier = Modifier
                        .size(20.dp)
                        .clip(RoundedCornerShape(50))
                        .background(
                            if (selected) MaterialTheme.colorScheme.primary.copy(alpha = 0.2f)
                            else MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.1f)
                        ),
                    contentAlignment = Alignment.Center,
                ) {
                    Box(
                        modifier = Modifier
                            .size(8.dp)
                            .clip(RoundedCornerShape(50))
                            .background(
                                if (selected) MaterialTheme.colorScheme.primary
                                else MaterialTheme.colorScheme.onSurfaceVariant
                            )
                    )
                }
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    "ORD-${order.orderId.takeLast(6).uppercase()}",
                    style = MaterialTheme.typography.titleSmall,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Black,
                )
                Spacer(Modifier.height(2.dp))
                Text(
                    "${order.items.size} UNITS",
                    style = MaterialTheme.typography.labelSmall,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (sealed && !dispatchCode.isNullOrBlank()) {
                Surface(
                    color = MaterialTheme.colorScheme.tertiary.copy(alpha = 0.15f),
                    shape = RoundedCornerShape(50),
                ) {
                    Text(
                        dispatchCode,
                        style = MaterialTheme.typography.labelLarge,
                        fontFamily = FontFamily.Monospace,
                        fontWeight = FontWeight.Black,
                        color = MaterialTheme.colorScheme.tertiary,
                        modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                    )
                }
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
    Surface(
        color = if (checked) MaterialTheme.colorScheme.primary.copy(alpha = 0.04f) else MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .clickable(enabled = enabled, onClick = onToggle),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp),
        ) {
            Box(
                modifier = Modifier
                    .size(32.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .background(
                        if (checked) MaterialTheme.colorScheme.primary
                        else MaterialTheme.colorScheme.onSurface.copy(alpha = 0.06f)
                    ),
                contentAlignment = Alignment.Center,
            ) {
                if (checked) {
                    Icon(
                        Icons.Filled.CheckCircle,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.surface,
                        modifier = Modifier.size(18.dp),
                    )
                }
            }
            Spacer(Modifier.width(14.dp))
            Text(
                label.uppercase(),
                style = MaterialTheme.typography.bodyMedium.copy(
                    textDecoration = if (checked) androidx.compose.ui.text.style.TextDecoration.LineThrough else null,
                ),
                fontFamily = FontFamily.Monospace,
                fontWeight = FontWeight.Bold,
                color = if (checked) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.onSurface,
                modifier = Modifier.weight(1f),
            )
            Surface(
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.06f),
                shape = RoundedCornerShape(6.dp),
            ) {
                Text(
                    "QTY ×$quantity",
                    style = MaterialTheme.typography.labelSmall,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Black,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                )
            }
        }
    }
}
