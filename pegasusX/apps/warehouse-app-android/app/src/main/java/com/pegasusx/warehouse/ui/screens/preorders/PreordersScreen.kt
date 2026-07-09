package com.pegasusx.warehouse.ui.screens.preorders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.WarehouseOrderMutationRequest
import com.pegasusx.warehouse.data.model.WarehousePreorderRow
import com.pegasusx.warehouse.data.model.WarehouseProposeDeliveryRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PreordersScreen(
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var rows by remember { mutableStateOf<List<WarehousePreorderRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var actionMessage by remember { mutableStateOf<String?>(null) }
    var rejectTarget by remember { mutableStateOf<WarehousePreorderRow?>(null) }
    var proposeTarget by remember { mutableStateOf<WarehousePreorderRow?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        scope.launch {
            if (!silent) loading = true
            error = null
            try {
                val resp = api.getPreorders()
                rows = if (resp.isSuccessful) resp.body()?.items.orEmpty().ifEmpty { resp.body()?.preorders.orEmpty() } else emptyList()
                if (!resp.isSuccessful && !silent) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }
    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

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

    proposeTarget?.let { row ->
        val initialMillis = remember(row.orderId) {
            row.requestedDeliveryDate?.take(10)?.let { date ->
                runCatching { LocalDate.parse(date).atStartOfDay(ZoneOffset.ofHours(5)).toInstant().toEpochMilli() }
                    .getOrDefault(System.currentTimeMillis())
            } ?: System.currentTimeMillis()
        }
        val datePickerState = rememberDatePickerState(initialSelectedDateMillis = initialMillis)
        var reason by remember(row.orderId) { mutableStateOf("") }
        var showReasonDialog by remember(row.orderId) { mutableStateOf(false) }

        if (showReasonDialog) {
            AlertDialog(
                onDismissRequest = { showReasonDialog = false },
                title = { Text("Reason for date change") },
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
                            val selectedMillis = datePickerState.selectedDateMillis ?: return@TextButton
                            val iso = Instant.ofEpochMilli(selectedMillis)
                                .atOffset(ZoneOffset.ofHours(5))
                                .withHour(12).withMinute(0).withSecond(0)
                                .format(DateTimeFormatter.ISO_OFFSET_DATE_TIME)
                            scope.launch {
                                try {
                                    val resp = api.proposePreorderDelivery(
                                        row.orderId,
                                        WarehouseProposeDeliveryRequest(proposedDeliveryDate = iso, reason = reason.trim()),
                                        WarehouseIdempotencyKeys.orderProposeDelivery(row.orderId, iso, reason.trim()),
                                    )
                                    if (resp.isSuccessful) {
                                        actionMessage = "Delivery date proposed"
                                        proposeTarget = null
                                        load()
                                    } else {
                                        actionMessage = "Propose failed (${resp.code()})"
                                    }
                                } catch (e: Exception) {
                                    actionMessage = e.message
                                }
                            }
                        },
                        enabled = reason.isNotBlank(),
                    ) { Text("Send proposal") }
                },
                dismissButton = { TextButton(onClick = { showReasonDialog = false }) { Text("Back") } },
            )
        }

        DatePickerDialog(
            onDismissRequest = { proposeTarget = null },
            confirmButton = {
                TextButton(onClick = { showReasonDialog = true }) { Text("Next") }
            },
            dismissButton = { TextButton(onClick = { proposeTarget = null }) { Text("Cancel") } },
        ) {
            DatePicker(state = datePickerState)
        }
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
            loading && rows.isEmpty() -> PegasusLoadingState(
                title = "Loading pre-orders…",
                body = "Fetching scheduled deliveries",
                modifier = Modifier.fillMaxSize().padding(padding)
            )
            error != null && rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Pre-orders unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(padding)
            )
            rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No scheduled pre-orders",
                body = "No future orders waiting for delivery scheduling.",
                modifier = Modifier.fillMaxSize().padding(padding)
            )
            else -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                modifier = Modifier.padding(padding).padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                items(rows, key = { it.orderId }) { row ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(row.orderId, style = MaterialTheme.typography.titleSmall)
                            Text("Status: ${row.status}")
                            row.requestedDeliveryDate?.let { Text("Delivery: $it") }
                            row.proposedDeliveryDate?.let { Text("Proposed: $it", color = MaterialTheme.colorScheme.primary) }
                            row.deliveryProposalReason?.let { Text("Reason: $it", style = MaterialTheme.typography.bodySmall) }
                            if (row.confirmationStatus == "PENDING_WAREHOUSE" || row.preorderBadge == "REVIEW_DELIVERY") {
                                AssistChip(onClick = {}, label = { Text("Awaiting retailer review") })
                            }
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                TextButton(onClick = { proposeTarget = row }) { Text("Propose date") }
                                TextButton(onClick = { rejectTarget = row }) { Text("Reject") }
                            }
                        }
                    }
                }
            }
        }
    }
}
