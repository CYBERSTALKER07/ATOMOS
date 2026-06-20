package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierOrder
import com.pegasusx.supplier.data.model.WarehouseOrderDetail
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierStatusChip
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrderDetailScreen(
    orderId: String,
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var listOrder by remember { mutableStateOf<SupplierOrder?>(null) }
    var detail by remember { mutableStateOf<WarehouseOrderDetail?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var acting by remember { mutableStateOf(false) }
    var reason by remember { mutableStateOf("") }
    var delayDialog by remember { mutableStateOf(false) }
    var rejectDialog by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val warehouseId = listOrder?.warehouseId

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                if (listOrder == null) {
                    val active = ops.getOrders(filter = "ACTIVE", limit = 100)
                    listOrder = active.body()?.orders?.find { it.orderId == orderId }
                    if (listOrder == null) {
                        val scheduled = ops.getOrders(status = "SCHEDULED", limit = 100)
                        listOrder = scheduled.body()?.orders?.find { it.orderId == orderId }
                    }
                    if (listOrder == null) {
                        val review = ops.getOrders(status = "AWAITING_REVIEW", limit = 100)
                        listOrder = review.body()?.orders?.find { it.orderId == orderId }
                    }
                }
                val warehouseId = listOrder?.warehouseId
                if (!warehouseId.isNullOrBlank()) {
                    val resp = ops.getWarehouseOrder(orderId, warehouseId)
                    detail = if (resp.isSuccessful) resp.body() else null
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(orderId) { load() }

    if (delayDialog) {
        AlertDialog(
            onDismissRequest = { delayDialog = false; reason = "" },
            title = { Text("Delay delivery") },
            text = {
                OutlinedTextField(
                    value = reason,
                    onValueChange = { reason = it },
                    label = { Text("Reason (optional)") },
                    modifier = Modifier.fillMaxWidth(),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val wh = warehouseId ?: return@TextButton
                        acting = true
                        scope.launch {
                            try {
                                ops.delayWarehouseOrder(
                                    orderId,
                                    wh,
                                    reason.trim().ifBlank { null },
                                    SupplierIdempotencyKeys.warehouseOrderDelay(orderId),
                                )
                                delayDialog = false
                                reason = ""
                                load()
                            } finally {
                                acting = false
                            }
                        }
                    },
                ) { Text("Delay") }
            },
            dismissButton = { TextButton(onClick = { delayDialog = false; reason = "" }) { Text("Cancel") } },
        )
    }

    if (rejectDialog) {
        AlertDialog(
            onDismissRequest = { rejectDialog = false; reason = "" },
            title = { Text("Reject order") },
            text = {
                OutlinedTextField(
                    value = reason,
                    onValueChange = { reason = it },
                    label = { Text("Reason (required)") },
                    modifier = Modifier.fillMaxWidth(),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val wh = warehouseId ?: return@TextButton
                        if (reason.isBlank()) return@TextButton
                        acting = true
                        scope.launch {
                            try {
                                ops.rejectWarehouseOrder(
                                    orderId,
                                    wh,
                                    reason.trim(),
                                    SupplierIdempotencyKeys.warehouseOrderReject(orderId, reason.trim()),
                                )
                                rejectDialog = false
                                reason = ""
                                load()
                            } finally {
                                acting = false
                            }
                        }
                    },
                    enabled = reason.isNotBlank(),
                ) { Text("Reject") }
            },
            dismissButton = { TextButton(onClick = { rejectDialog = false; reason = "" }) { Text("Cancel") } },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Order ${orderId.take(8)}…") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(padding), contentAlignment = androidx.compose.ui.Alignment.Center) {
                CircularProgressIndicator()
            }
            listOrder == null && detail == null -> Text("Order not found", modifier = Modifier.padding(padding).padding(16.dp))
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text(
                                detail?.retailerName ?: listOrder?.retailerId ?: orderId,
                                style = MaterialTheme.typography.titleMedium,
                            )
                            SupplierStatusChip(status = detail?.state ?: detail?.status ?: listOrder?.status.orEmpty())
                            listOrder?.let {
                                Text(formatMinorAmount(it.totalMinor, it.currency), style = MaterialTheme.typography.bodyMedium)
                            }
                            Text(orderId, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
                if (!warehouseId.isNullOrBlank()) {
                    item {
                        Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            OutlinedButton(onClick = { delayDialog = true }, enabled = !acting) { Text("Delay") }
                            OutlinedButton(onClick = { rejectDialog = true }, enabled = !acting) { Text("Reject") }
                        }
                    }
                }
                val lineItems = detail?.lineItems.orEmpty()
                if (lineItems.isNotEmpty()) {
                    item { Text("Line items", style = MaterialTheme.typography.titleSmall) }
                    items(lineItems, key = { it.productId ?: it.productName ?: it.hashCode().toString() }) { item ->
                        ListItem(
                            headlineContent = { Text(item.productName ?: item.productId ?: "—") },
                            supportingContent = {
                                Text("Qty ${item.quantity ?: 0} · Unit ${item.unitPrice ?: 0}")
                            },
                        )
                    }
                }
                error?.let { msg ->
                    item { Text(msg, color = MaterialTheme.colorScheme.error) }
                }
            }
        }
    }
}
