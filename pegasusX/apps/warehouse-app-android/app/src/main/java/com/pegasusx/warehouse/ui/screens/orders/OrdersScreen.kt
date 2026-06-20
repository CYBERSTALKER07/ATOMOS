package com.pegasusx.warehouse.ui.screens.orders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.FilterList
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.Order
import com.pegasusx.warehouse.data.model.WarehouseOrderMutationRequest
import com.pegasusx.warehouse.data.model.WarehousePreorderRow
import com.pegasusx.warehouse.data.model.WarehouseProposeDeliveryRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.components.OrderOpsCard
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import com.pegasusx.warehouse.util.orderActionFlags
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.Locale

private val STATES = listOf("ALL", "PENDING", "LOADED", "IN_TRANSIT", "ARRIVED", "COMPLETED", "CANCELLED")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrdersScreen(
    api: WarehouseApi,
    opsRepository: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    onOrderClick: (String) -> Unit,
    onBack: (() -> Unit)? = null,
) {
    var hubTab by remember { mutableIntStateOf(0) }
    var orders by remember { mutableStateOf<List<Order>>(emptyList()) }
    var preorders by remember { mutableStateOf<List<WarehousePreorderRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selectedState by remember { mutableStateOf("ALL") }
    var filterExpanded by remember { mutableStateOf(false) }
    var actingId by remember { mutableStateOf<String?>(null) }
    var proposeActiveTarget by remember { mutableStateOf<String?>(null) }
    var rejectTarget by remember { mutableStateOf<String?>(null) }
    var rejectPreorderTarget by remember { mutableStateOf<WarehousePreorderRow?>(null) }
    var proposeTarget by remember { mutableStateOf<WarehousePreorderRow?>(null) }
    var reasonInput by remember { mutableStateOf("") }
    var actionMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun loadActive(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        scope.launch {
            try {
                val state = if (selectedState == "ALL") null else selectedState
                val resp = api.getOrders(state = state)
                if (resp.isSuccessful && resp.body() != null) {
                    orders = resp.body()!!.orders
                } else if (!silent) {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                if (!silent) error = e.message ?: "Network error"
            } finally {
                if (!silent) loading = false
            }
        }
    }

    fun loadPreorders(silent: Boolean = false) {
        if (!silent) loading = true
        scope.launch {
            try {
                val resp = api.getPreorders()
                preorders = if (resp.isSuccessful) {
                    resp.body()?.items.orEmpty().ifEmpty { resp.body()?.preorders.orEmpty() }
                } else {
                    if (!silent) error = "Failed (${resp.code()})"
                    emptyList()
                }
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    fun load(silent: Boolean = false) {
        if (hubTab == 0) loadActive(silent) else loadPreorders(silent)
    }

    LaunchedEffect(hubTab, selectedState) { load() }

    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    proposeActiveTarget?.let { orderId ->
        val datePickerState = rememberDatePickerState(initialSelectedDateMillis = System.currentTimeMillis())
        var showReasonDialog by remember(orderId) { mutableStateOf(false) }

        if (showReasonDialog) {
            AlertDialog(
                onDismissRequest = { showReasonDialog = false },
                title = { Text("Reason for new delivery date") },
                text = {
                    OutlinedTextField(
                        value = reasonInput,
                        onValueChange = { reasonInput = it },
                        label = { Text("Reason (required)") },
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
                            actingId = orderId
                            scope.launch {
                                try {
                                    val resp = opsRepository.proposeOrderDelivery(
                                        orderId,
                                        iso,
                                        reasonInput.trim(),
                                    )
                                    actionMessage = if (resp.isSuccessful) {
                                        "New delivery date proposed · retailer notified"
                                    } else {
                                        "Propose failed (${resp.code()})"
                                    }
                                    proposeActiveTarget = null
                                    reasonInput = ""
                                    load(silent = true)
                                } catch (e: Exception) {
                                    actionMessage = e.message
                                } finally {
                                    actingId = null
                                }
                            }
                        },
                        enabled = reasonInput.isNotBlank(),
                    ) { Text("Notify retailer") }
                },
                dismissButton = { TextButton(onClick = { showReasonDialog = false }) { Text("Back") } },
            )
        }

        DatePickerDialog(
            onDismissRequest = { proposeActiveTarget = null },
            confirmButton = { TextButton(onClick = { showReasonDialog = true }) { Text("Next") } },
            dismissButton = { TextButton(onClick = { proposeActiveTarget = null }) { Text("Cancel") } },
        ) {
            DatePicker(state = datePickerState)
        }
    }

    rejectTarget?.let { orderId ->
        AlertDialog(
            onDismissRequest = { rejectTarget = null; reasonInput = "" },
            title = { Text("Reject order") },
            text = {
                OutlinedTextField(
                    value = reasonInput,
                    onValueChange = { reasonInput = it },
                    label = { Text("Reason (required)") },
                    modifier = Modifier.fillMaxWidth(),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        if (reasonInput.isBlank()) return@TextButton
                        actingId = orderId
                        scope.launch {
                            try {
                                val resp = opsRepository.rejectOrder(orderId, reasonInput.trim())
                                actionMessage = if (resp.isSuccessful) {
                                    "Order cancelled · retailer notified"
                                } else {
                                    "Reject failed (${resp.code()})"
                                }
                                rejectTarget = null
                                reasonInput = ""
                                load(silent = true)
                            } catch (e: Exception) {
                                actionMessage = e.message
                            } finally {
                                actingId = null
                            }
                        }
                    },
                    enabled = reasonInput.isNotBlank(),
                ) { Text("Reject") }
            },
            dismissButton = { TextButton(onClick = { rejectTarget = null; reasonInput = "" }) { Text("Cancel") } },
        )
    }

    rejectPreorderTarget?.let { row ->
        AlertDialog(
            onDismissRequest = { rejectPreorderTarget = null; reasonInput = "" },
            title = { Text("Reject pre-order") },
            text = {
                OutlinedTextField(
                    value = reasonInput,
                    onValueChange = { reasonInput = it },
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
                                    WarehouseOrderMutationRequest(reasonInput.trim()),
                                    WarehouseIdempotencyKeys.orderReject(row.orderId, reasonInput.trim()),
                                )
                                actionMessage = if (resp.isSuccessful) "Pre-order rejected" else "Reject failed"
                                rejectPreorderTarget = null
                                reasonInput = ""
                                load(silent = true)
                            } catch (e: Exception) {
                                actionMessage = e.message
                            }
                        }
                    },
                    enabled = reasonInput.isNotBlank(),
                ) { Text("Reject") }
            },
            dismissButton = { TextButton(onClick = { rejectPreorderTarget = null; reasonInput = "" }) { Text("Cancel") } },
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
        var showReasonDialog by remember(row.orderId) { mutableStateOf(false) }

        if (showReasonDialog) {
            AlertDialog(
                onDismissRequest = { showReasonDialog = false },
                title = { Text("Reason for date change") },
                text = {
                    OutlinedTextField(
                        value = reasonInput,
                        onValueChange = { reasonInput = it },
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
                                        WarehouseProposeDeliveryRequest(proposedDeliveryDate = iso, reason = reasonInput.trim()),
                                        WarehouseIdempotencyKeys.orderProposeDelivery(row.orderId, iso, reasonInput.trim()),
                                    )
                                    actionMessage = if (resp.isSuccessful) "Delivery date proposed" else "Propose failed"
                                    proposeTarget = null
                                    reasonInput = ""
                                    load(silent = true)
                                } catch (e: Exception) {
                                    actionMessage = e.message
                                }
                            }
                        },
                        enabled = reasonInput.isNotBlank(),
                    ) { Text("Send proposal") }
                },
                dismissButton = { TextButton(onClick = { showReasonDialog = false }) { Text("Back") } },
            )
        }

        DatePickerDialog(
            onDismissRequest = { proposeTarget = null },
            confirmButton = { TextButton(onClick = { showReasonDialog = true }) { Text("Next") } },
            dismissButton = { TextButton(onClick = { proposeTarget = null }) { Text("Cancel") } },
        ) {
            DatePicker(state = datePickerState)
        }
    }

    Scaffold(
        topBar = {
            Column {
                TopAppBar(
                    title = { Text("Orders") },
                    navigationIcon = {
                        if (onBack != null) {
                            IconButton(onClick = onBack) {
                                Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                            }
                        }
                    },
                    actions = {
                        if (hubTab == 0) {
                            Box {
                                IconButton(onClick = { filterExpanded = true }) {
                                    Icon(Icons.Default.FilterList, "Filter")
                                }
                                DropdownMenu(expanded = filterExpanded, onDismissRequest = { filterExpanded = false }) {
                                    STATES.forEach { s ->
                                        DropdownMenuItem(
                                            text = { Text(s) },
                                            onClick = { selectedState = s; filterExpanded = false },
                                        )
                                    }
                                }
                            }
                        }
                        IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                    },
                )
                TabRow(selectedTabIndex = hubTab) {
                    Tab(selected = hubTab == 0, onClick = { hubTab = 0 }, text = { Text("Active orders") })
                    Tab(selected = hubTab == 1, onClick = { hubTab = 1 }, text = { Text("Pre-orders") })
                }
            }
        },
        snackbarHost = {
            actionMessage?.let { msg ->
                Snackbar { Text(msg) }
            }
        },
    ) { innerPadding ->
        when {
            loading && (if (hubTab == 0) orders.isEmpty() else preorders.isEmpty()) ->
                Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            error != null && (if (hubTab == 0) orders.isEmpty() else preorders.isEmpty()) ->
                Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(error!!, color = MaterialTheme.colorScheme.error)
                        Spacer(Modifier.height(PegasusSpacing.lg))
                        Button(onClick = { load() }) { Text("Retry") }
                    }
                }
            hubTab == 0 && orders.isEmpty() ->
                Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                    Text("No orders", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            hubTab == 1 && preorders.isEmpty() ->
                Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                    Text("No scheduled pre-orders", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            hubTab == 0 -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(orders, key = { it.orderId }) { order ->
                    val flags = orderActionFlags(order.state)
                    OrderOpsCard(
                        retailerName = order.retailerName,
                        orderId = order.orderId,
                        state = order.state,
                        amountLabel = "${fmt.format(order.totalUzs)} UZS",
                        enabled = actingId != order.orderId,
                        canDelay = flags.canDelay,
                        canReject = flags.canReject,
                        onOpenDetail = { onOrderClick(order.orderId) },
                        onDelay = { proposeActiveTarget = order.orderId; reasonInput = "" },
                        onReject = { rejectTarget = order.orderId; reasonInput = "" },
                    )
                }
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(preorders, key = { it.orderId }) { row ->
                    OrderOpsCard(
                        retailerName = row.orderId.take(12),
                        orderId = row.orderId,
                        state = row.status,
                        amountLabel = row.requestedDeliveryDate?.take(10) ?: "Pre-order",
                        meta = row.proposedDeliveryDate?.let { "Proposed: $it" },
                        badge = "Pre-order",
                        delayLabel = "Propose delivery",
                        rejectLabel = "Reject",
                        canDelay = true,
                        canReject = true,
                        onOpenDetail = { onOrderClick(row.orderId) },
                        onDelay = { proposeTarget = row; reasonInput = "" },
                        onReject = { rejectPreorderTarget = row; reasonInput = "" },
                    )
                }
            }
        }
    }
}
