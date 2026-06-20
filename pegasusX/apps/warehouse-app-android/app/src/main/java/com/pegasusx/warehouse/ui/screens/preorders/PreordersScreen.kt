package com.pegasusx.warehouse.ui.screens.preorders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.WarehouseOrderMutationRequest
import com.pegasusx.warehouse.data.model.WarehousePreorderEditRequest
import com.pegasusx.warehouse.data.model.WarehousePreorderRow
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PreordersScreen(api: WarehouseApi, onBack: (() -> Unit)? = null) {
    var rows by remember { mutableStateOf<List<WarehousePreorderRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var actionMessage by remember { mutableStateOf<String?>(null) }
    var rejectTarget by remember { mutableStateOf<WarehousePreorderRow?>(null) }
    var editTarget by remember { mutableStateOf<WarehousePreorderRow?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = api.getPreorders()
                rows = if (resp.isSuccessful) resp.body()?.items.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    rejectTarget?.let { row ->
        var reason by remember(row.orderId) { mutableStateOf("") }
        AlertDialog(
            onDismissRequest = { rejectTarget = null },
            title = { Text("Reject pre-order") },
            text = {
                OutlinedTextField(
                    value = reason,
                    onValueChange = { reason = it },
                    label = { Text("Reason") },
                    modifier = Modifier.fillMaxWidth(),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        scope.launch {
                            try {
                                val resp = api.rejectPreorder(
                                    row.orderId,
                                    WarehouseOrderMutationRequest(reason.trim()),
                                    WarehouseIdempotencyKeys.orderReject(row.orderId, reason.trim()),
                                )
                                if (resp.isSuccessful) {
                                    actionMessage = "Pre-order rejected"
                                    rejectTarget = null
                                    load()
                                } else {
                                    actionMessage = "Reject failed (${resp.code()})"
                                }
                            } catch (e: Exception) {
                                actionMessage = e.message
                            }
                        }
                    },
                    enabled = reason.isNotBlank(),
                ) { Text("Reject") }
            },
            dismissButton = { TextButton(onClick = { rejectTarget = null }) { Text("Cancel") } },
        )
    }

    editTarget?.let { row ->
        var deliveryDate by remember(row.orderId) { mutableStateOf(row.requestedDeliveryDate?.take(10).orEmpty()) }
        var reason by remember(row.orderId) { mutableStateOf("") }
        AlertDialog(
            onDismissRequest = { editTarget = null },
            title = { Text("Edit delivery date") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = deliveryDate,
                        onValueChange = { deliveryDate = it },
                        label = { Text("Delivery date (YYYY-MM-DD)") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = reason,
                        onValueChange = { reason = it },
                        label = { Text("Reason") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val iso = "${deliveryDate}T12:00:00+05:00"
                        scope.launch {
                            try {
                                val resp = api.editPreorder(
                                    row.orderId,
                                    WarehousePreorderEditRequest(requestedDeliveryDate = iso, reason = reason.trim()),
                                    WarehouseIdempotencyKeys.orderDelay(row.orderId),
                                )
                                if (resp.isSuccessful) {
                                    actionMessage = "Pre-order updated"
                                    editTarget = null
                                    load()
                                } else {
                                    actionMessage = "Edit failed (${resp.code()})"
                                }
                            } catch (e: Exception) {
                                actionMessage = e.message
                            }
                        }
                    },
                    enabled = deliveryDate.isNotBlank() && reason.isNotBlank(),
                ) { Text("Save") }
            },
            dismissButton = { TextButton(onClick = { editTarget = null }) { Text("Cancel") } },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Pre-orders") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                        }
                    }
                },
            )
        },
        snackbarHost = {
            actionMessage?.let { msg ->
                Snackbar { Text(msg) }
            }
        },
    ) { padding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(padding), contentAlignment = androidx.compose.ui.Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Text(error!!, modifier = Modifier.padding(padding).padding(16.dp))
            rows.isEmpty() -> Text("No scheduled pre-orders", modifier = Modifier.padding(padding).padding(16.dp))
            else -> LazyColumn(Modifier.padding(padding).padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                items(rows, key = { it.orderId }) { row ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(row.orderId, style = MaterialTheme.typography.titleSmall)
                            Text("Status: ${row.status}")
                            row.requestedDeliveryDate?.let { Text("Delivery: $it") }
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                TextButton(onClick = { editTarget = row }) { Text("Edit date") }
                                TextButton(onClick = { rejectTarget = row }) { Text("Reject") }
                            }
                        }
                    }
                }
            }
        }
    }
}
