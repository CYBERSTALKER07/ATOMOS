package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.ui.res.stringResource

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
import com.pegasusx.supplier.R

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
        val datePickerState = rememberDatePickerState(initialSelectedDateMillis = System.currentTimeMillis())
        var showReasonStep by remember { mutableStateOf(false) }
        if (showReasonStep) {
            AlertDialog(
                onDismissRequest = { showReasonStep = false },
                title = { Text("Reason for new delivery date") },
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
                            val millis = datePickerState.selectedDateMillis ?: return@TextButton
                            val iso = java.time.Instant.ofEpochMilli(millis)
                                .atOffset(java.time.ZoneOffset.ofHours(5))
                                .withHour(12).withMinute(0).withSecond(0)
                                .format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME)
                            acting = true
                            scope.launch {
                                try {
                                    ops.proposeWarehouseOrder(
                                        orderId,
                                        wh,
                                        iso,
                                        reason.trim(),
                                        SupplierIdempotencyKeys.warehouseOrderPropose(orderId, iso, reason.trim()),
                                    )
                                    delayDialog = false
                                    showReasonStep = false
                                    reason = ""
                                    load()
                                } finally {
                                    acting = false
                                }
                            }
                        },
                        enabled = reason.isNotBlank(),
                    ) { Text("Notify retailer") }
                },
                dismissButton = { TextButton(onClick = { showReasonStep = false }) { Text(stringResource(R.string.common_action_back)) } },
            )
        } else {
            DatePickerDialog(
                onDismissRequest = { delayDialog = false },
                confirmButton = { TextButton(onClick = { showReasonStep = true }) { Text("Next") } },
                dismissButton = { TextButton(onClick = { delayDialog = false }) { Text("Cancel") } },
            ) {
                DatePicker(state = datePickerState)
            }
        }
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
                title = { Text(stringResource(R.string.mobile_supplier_ui_order_take, orderId.take(8))) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
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
                                Text(stringResource(R.string.mobile_supplier_ui_qty_quantity_0_unit_unitprice_0, item.quantity ?: 0, item.unitPrice ?: 0))
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
