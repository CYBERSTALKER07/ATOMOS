package com.pegasusx.warehouse.ui.screens.orders

import androidx.compose.ui.res.stringResource

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.Order
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale
import com.pegasusx.warehouse.util.orderActionFlags
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import com.pegasusx.warehouse.ui.components.orders.orderOpsActions
import com.pegasusx.warehouse.ui.components.orders.orderLineItems
import com.pegasusx.warehouse.R

private enum class OrderMutationAction {
    ProposeDelivery,
    Reject,
    Overflow,
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrderDetailScreen(
    api: WarehouseApi,
    opsRepository: WarehouseOperationsRepository,
    orderId: String,
    onBack: (() -> Unit)? = null,
) {
    var order by remember { mutableStateOf<Order?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var mutating by remember { mutableStateOf(false) }
    var pendingAction by remember { mutableStateOf<OrderMutationAction?>(null) }
    var showProposeDatePicker by remember { mutableStateOf(false) }
    var proposeDateMillis by remember { mutableStateOf<Long?>(null) }
    var reasonInput by remember { mutableStateOf("") }
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun openReceipt() {
        scope.launch {
            try {
                val resp = api.getOrderReceipt(orderId)
                val meta = resp.body()
                val url = meta?.htmlUrl?.ifBlank { null }
                    ?: meta?.qrUrl?.ifBlank { null }
                    ?: meta?.pdfUrl?.ifBlank { null }
                if (resp.isSuccessful && !url.isNullOrBlank()) {
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                } else {
                    snackbarHostState.showSnackbar("Receipt unavailable (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Receipt unavailable")
            }
        }
    }

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getOrder(orderId)
                if (resp.isSuccessful && resp.body() != null) {
                    order = resp.body()!!
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    fun runMutation(action: OrderMutationAction, reason: String?, proposedIso: String? = null) {
        mutating = true
        scope.launch {
            try {
                val resp = when (action) {
                    OrderMutationAction.ProposeDelivery -> opsRepository.proposeOrderDelivery(
                        orderId,
                        proposedIso.orEmpty(),
                        reason.orEmpty(),
                    )
                    OrderMutationAction.Reject -> opsRepository.rejectOrder(orderId, reason.orEmpty())
                    OrderMutationAction.Overflow -> opsRepository.overflowOrder(orderId, reason)
                }
                if (resp.isSuccessful && resp.body() != null) {
                    val msg = when (action) {
                        OrderMutationAction.ProposeDelivery -> "New delivery date proposed · retailer notified"
                        OrderMutationAction.Reject -> "Order cancelled · retailer notified"
                        OrderMutationAction.Overflow -> "Order updated · ${resp.body()!!.status}"
                    }
                    snackbarHostState.showSnackbar(msg)
                    load()
                } else {
                    snackbarHostState.showSnackbar("Action failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Network error")
            } finally {
                mutating = false
                pendingAction = null
                reasonInput = ""
                showProposeDatePicker = false
                proposeDateMillis = null
            }
        }
    }

    LaunchedEffect(orderId) { load() }

    if (showProposeDatePicker) {
        val datePickerState = rememberDatePickerState(initialSelectedDateMillis = System.currentTimeMillis())
        DatePickerDialog(
            onDismissRequest = { if (!mutating) showProposeDatePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    proposeDateMillis = datePickerState.selectedDateMillis
                    showProposeDatePicker = false
                    pendingAction = OrderMutationAction.ProposeDelivery
                }) { Text("Next") }
            },
            dismissButton = { TextButton(onClick = { showProposeDatePicker = false }) { Text("Cancel") } },
        ) {
            DatePicker(state = datePickerState)
        }
    }

    pendingAction?.let { action ->
        val requiresReason = action != OrderMutationAction.Overflow
        val title = when (action) {
            OrderMutationAction.ProposeDelivery -> "Reason for new delivery date"
            OrderMutationAction.Reject -> "Cancel order?"
            OrderMutationAction.Overflow -> "Return to dispatch pool?"
        }
        val confirmLabel = when (action) {
            OrderMutationAction.ProposeDelivery -> "Notify retailer"
            OrderMutationAction.Reject -> "Cancel order"
            OrderMutationAction.Overflow -> "Overflow"
        }
        AlertDialog(
            onDismissRequest = {
                if (!mutating) {
                    pendingAction = null
                    reasonInput = ""
                }
            },
            title = { Text(title) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    Text(
                        stringResource(R.string.mobile_warehouse_ui_order_orderid, orderId),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    OutlinedTextField(
                        value = reasonInput,
                        onValueChange = { reasonInput = it },
                        label = {
                            Text(if (requiresReason) "Reason (required)" else "Reason (optional)")
                        },
                        modifier = Modifier.fillMaxWidth(),
                        enabled = !mutating,
                        singleLine = false,
                        minLines = 2,
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val trimmed = reasonInput.trim()
                        if (requiresReason && trimmed.isEmpty()) {
                            scope.launch { snackbarHostState.showSnackbar("Reason is required") }
                            return@TextButton
                        }
                        val proposedIso = proposeDateMillis?.let { millis ->
                            Instant.ofEpochMilli(millis)
                                .atOffset(ZoneOffset.ofHours(5))
                                .withHour(12).withMinute(0).withSecond(0)
                                .format(DateTimeFormatter.ISO_OFFSET_DATE_TIME)
                        }
                        if (action == OrderMutationAction.ProposeDelivery && proposedIso.isNullOrBlank()) {
                            scope.launch { snackbarHostState.showSnackbar("Select a delivery date") }
                            return@TextButton
                        }
                        runMutation(action, trimmed.ifBlank { null }, proposedIso)
                    },
                    enabled = !mutating && (!requiresReason || reasonInput.trim().isNotEmpty()),
                ) {
                    if (mutating) {
                        CircularProgressIndicator(modifier = Modifier.height(16.dp))
                    } else {
                        Text(confirmLabel)
                    }
                }
            },
            dismissButton = {
                TextButton(
                    onClick = {
                        pendingAction = null
                        reasonInput = ""
                    },
                    enabled = !mutating,
                ) { Text("Cancel") }
            },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Order Detail") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } } },
                actions = { IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") } },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            order != null -> {
                val current = order!!
                val state = current.state.uppercase(Locale.US)
                val flags = orderActionFlags(current.state)
                val canDelay = flags.canDelay
                val canReject = flags.canReject
                val canOverflow = flags.canOverflow
                val showOps = canDelay || canReject || canOverflow

                LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 340.dp),
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    modifier = Modifier.fillMaxSize().padding(innerPadding),
                ) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            SummaryCard("State", current.state, Modifier.weight(1f))
                            SummaryCard("Total", "${fmt.format(current.totalUzs)} UZS", Modifier.weight(1f))
                        }
                    }
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Text(
                            text = stringResource(R.string.mobile_warehouse_ui_retailer_ifblank, current.retailerName.ifBlank { "—" }),
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    if (state == "COMPLETED" || state == "FISCALIZING" || state == "FISCAL_FAILED") {
                        item(span = { GridItemSpan(maxLineSpan) }) {
                            OutlinedButton(
                                onClick = { openReceipt() },
                                enabled = !mutating,
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Text("View Pegasus receipt")
                            }
                        }
                    }
                    orderOpsActions(
                        canDelay = canDelay,
                        canOverflow = canOverflow,
                        canReject = canReject,
                        mutating = mutating,
                        onProposeNewDate = { showProposeDatePicker = true },
                        onOverflow = { pendingAction = OrderMutationAction.Overflow },
                        onReject = { pendingAction = OrderMutationAction.Reject }
                    )
                    orderLineItems(current, fmt)
                }
            }
        }
    }
}

@Composable
private fun SummaryCard(label: String, value: String, modifier: Modifier = Modifier) {
    ElevatedCard(modifier = modifier) {
        Column(modifier = Modifier.padding(PegasusSpacing.md)) {
            Text(value, style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(2.dp))
            Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
