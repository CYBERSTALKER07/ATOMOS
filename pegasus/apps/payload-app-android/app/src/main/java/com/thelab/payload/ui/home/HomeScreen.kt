package com.pegasus.payload.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.material3.adaptive.ExperimentalMaterial3AdaptiveApi
import androidx.compose.material3.adaptive.layout.AnimatedPane
import androidx.compose.material3.adaptive.layout.ListDetailPaneScaffold
import androidx.compose.material3.adaptive.layout.ListDetailPaneScaffoldRole
import androidx.compose.material3.adaptive.navigation.rememberListDetailPaneScaffoldNavigator
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Logout
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material.icons.filled.Warning
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.payload.data.model.LiveOrder
import com.pegasus.payload.data.model.Manifest
import com.pegasus.payload.data.model.RecommendReassignResponse
import com.pegasus.payload.data.model.NotificationItem
import com.pegasus.payload.data.model.Truck
import com.pegasus.payload.data.model.TruckRecommendation

/**
 * Master-detail home with Phase 4 loading workflow.
 * Sidebar = trucks. Detail = manifest summary, per-order checklist with seal,
 * 60s post-seal double-check countdown, manifest seal, All Sealed success.
 * Uses Material3 Adaptive ListDetailPaneScaffold so it adapts phone ↔ tablet.
 */
@OptIn(ExperimentalMaterial3AdaptiveApi::class, ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    onLogout: () -> Unit,
    viewModel: HomeViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val navigator = rememberListDetailPaneScaffoldNavigator<String>()
    val snackbarHostState = remember { SnackbarHostState() }

    var showInjectDialog by remember { mutableStateOf(false) }
    var exceptionTargetOrderId by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(state.syncCompleteMessage) {
        state.syncCompleteMessage?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearSyncCompleteMessage()
        }
    }
    LaunchedEffect(state.queuedNoticeMessage) {
        state.queuedNoticeMessage?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearQueuedNoticeMessage()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        topBar = {
            TopAppBar(
                title = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("Lab Payload Terminal")
                        Spacer(Modifier.size(12.dp))
                        OnlineDot(online = state.online, queued = state.queuedActions)
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.toggleNotificationsPanel() }) {
                        BadgedBox(badge = {
                            if (state.unreadCount > 0) Badge { Text(state.unreadCount.toString()) }
                        }) {
                            Icon(Icons.Filled.Notifications, contentDescription = "Notifications")
                        }
                    }
                    IconButton(onClick = { viewModel.refreshTrucks() }) {
                        Icon(Icons.Filled.Refresh, contentDescription = "Refresh trucks")
                    }
                    IconButton(onClick = onLogout) {
                        Icon(Icons.Filled.Logout, contentDescription = "Logout")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surfaceContainer,
                ),
            )
        },
    ) { padding ->
        ListDetailPaneScaffold(
            modifier = Modifier.padding(padding),
            directive = navigator.scaffoldDirective,
            value = navigator.scaffoldValue,
            listPane = {
                AnimatedPane {
                    TruckListPane(
                        trucks = state.trucks,
                        selectedTruckId = state.selectedTruckId,
                        loading = state.loadingTrucks,
                        error = state.error,
                        onSelect = { id ->
                            viewModel.selectTruck(id)
                            navigator.navigateTo(ListDetailPaneScaffoldRole.Detail, id)
                        },
                    )
                }
            },
            detailPane = {
                AnimatedPane {
                    ManifestDetailPane(
                        truck = state.trucks.firstOrNull { it.id == state.selectedTruckId },
                        state = state,
                        onRefresh = viewModel::refreshManifest,
                        onStartLoading = viewModel::startLoading,
                        onSelectOrder = viewModel::selectOrder,
                        onToggleItem = viewModel::toggleItem,
                        onSealOrder = viewModel::sealSelectedOrder,
                        onDismissCountdown = viewModel::dismissCountdown,
                        canSealOrder = viewModel::canSealOrder,
                        allOrdersSealed = viewModel.allOrdersSealed,
                        onSealManifest = viewModel::sealManifest,
                        onStartNewManifest = viewModel::startNewManifest,
                        onShowInject = { showInjectDialog = true },
                        onShowException = { exceptionTargetOrderId = it },
                        onShowReDispatch = viewModel::openReDispatch,
                        onClearEscalated = viewModel::clearEscalatedMessage,
                    )
                }
            },
        )

        // ── Phase 5 dialogs ──
        if (showInjectDialog && state.manifest != null) {
            InjectOrderDialog(
                injecting = state.injectingOrder,
                onDismiss = { showInjectDialog = false },
                onSubmit = { id ->
                    viewModel.injectOrder(id)
                    showInjectDialog = false
                },
            )
        }
        exceptionTargetOrderId?.let { orderId ->
            ExceptionReasonDialog(
                orderId = orderId,
                inFlight = state.exceptionLoadingOrderId == orderId,
                onDismiss = { exceptionTargetOrderId = null },
                onSelect = { reason ->
                    viewModel.reportException(orderId, reason)
                    exceptionTargetOrderId = null
                },
            )
        }
        if (state.reDispatchOrderId != null) {
            ReDispatchDialog(
                orderId = state.reDispatchOrderId!!,
                loading = state.loadingRecommendations,
                response = state.recommendations,
                reassigning = state.reassigning,
                onDismiss = viewModel::closeReDispatch,
                onPick = { driverId -> viewModel.reassignTo(driverId) },
            )
        }
        if (state.showNotificationsPanel) {
            NotificationsSheet(
                items = state.notifications,
                unreadCount = state.unreadCount,
                onDismiss = viewModel::toggleNotificationsPanel,
                onMarkRead = viewModel::markNotificationRead,
                onMarkAllRead = viewModel::markAllNotificationsRead,
            )
        }
    }
}

@Composable
private fun OnlineDot(online: Boolean, queued: Int) {
    val color = if (online) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier
                .size(10.dp)
                .clip(RoundedCornerShape(50))
                .background(color),
        )
        Spacer(Modifier.size(6.dp))
        Text(
            text = if (online) "Live" else if (queued > 0) "Offline · $queued queued" else "Offline",
            style = MaterialTheme.typography.labelMedium,
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun NotificationsSheet(
    items: List<NotificationItem>,
    unreadCount: Int,
    onDismiss: () -> Unit,
    onMarkRead: (String) -> Unit,
    onMarkAllRead: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text("Notifications", style = MaterialTheme.typography.titleLarge)
                if (unreadCount > 0) {
                    TextButton(onClick = onMarkAllRead) { Text("Mark all read") }
                }
            }
            HorizontalDivider()
            if (items.isEmpty()) {
                EmptyState(label = "No notifications", hint = "New events will appear here in real time.")
            } else {
                LazyColumn(Modifier.fillMaxWidth()) {
                    items(items, key = { it.notificationId }) { n ->
                        NotificationRow(n, onClick = { if (n.isUnread) onMarkRead(n.notificationId) })
                        HorizontalDivider()
                    }
                }
            }
            Spacer(Modifier.height(12.dp))
        }
    }
}

@Composable
private fun NotificationRow(item: NotificationItem, onClick: () -> Unit) {
    val bg = if (item.isUnread) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.35f)
             else MaterialTheme.colorScheme.surface
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(bg)
            .clickable(onClick = onClick)
            .padding(horizontal = 8.dp, vertical = 12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        if (item.isUnread) {
            Box(
                Modifier
                    .size(8.dp)
                    .clip(RoundedCornerShape(50))
                    .background(MaterialTheme.colorScheme.primary)
                    .padding(top = 6.dp),
            )
            Spacer(Modifier.size(10.dp))
        } else {
            Spacer(Modifier.size(18.dp))
        }
        Column(Modifier.fillMaxWidth()) {
            Text(item.title.ifEmpty { item.type }, style = MaterialTheme.typography.titleSmall)
            if (item.body.isNotEmpty()) {
                Text(item.body, style = MaterialTheme.typography.bodySmall)
            }
            if (item.createdAt.isNotEmpty()) {
                Text(item.createdAt, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

// ── Sidebar (truck list) ─────────────────────────────────────────────────────

@Composable
private fun TruckListPane(
    trucks: List<Truck>,
    selectedTruckId: String?,
    loading: Boolean,
    error: String?,
    onSelect: (String) -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        modifier = Modifier.fillMaxSize(),
    ) {
        Column(Modifier.fillMaxSize()) {
            Text(
                "Vehicles",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(horizontal = 20.dp, vertical = 16.dp),
            )
            if (loading) LinearProgressIndicator(Modifier.fillMaxWidth())
            if (error != null) {
                Text(
                    error,
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
                )
            }
            if (!loading && trucks.isEmpty() && error == null) {
                EmptyState(label = "No vehicles available", hint = "Pull to refresh once dispatch assigns trucks.")
            }
            LazyColumn(contentPadding = PaddingValues(horizontal = 12.dp, vertical = 4.dp)) {
                items(trucks, key = { it.id }) { truck ->
                    TruckRow(truck, selected = truck.id == selectedTruckId, onClick = { onSelect(truck.id) })
                    Spacer(Modifier.height(6.dp))
                }
            }
        }
    }
}

@Composable
private fun TruckRow(truck: Truck, selected: Boolean, onClick: () -> Unit) {
    val bg = if (selected) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceContainerHigh
    val fg = if (selected) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface
    Surface(
        color = bg,
        contentColor = fg,
        shape = RoundedCornerShape(14.dp),
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .clickable(onClick = onClick),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(14.dp),
        ) {
            Icon(Icons.Filled.LocalShipping, contentDescription = null)
            Spacer(Modifier.size(12.dp))
            Column(Modifier.fillMaxWidth()) {
                Text(
                    truck.label.ifBlank { truck.licensePlate.ifBlank { truck.id.take(8) } },
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Medium,
                )
                Text(
                    listOfNotNull(
                        truck.licensePlate.takeIf { it.isNotBlank() },
                        truck.vehicleClass.takeIf { it.isNotBlank() },
                    ).joinToString(" • "),
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
}

// ── Detail pane (Phase 4 loading workflow) ───────────────────────────────────

@Composable
private fun ManifestDetailPane(
    truck: Truck?,
    state: HomeUiState,
    onRefresh: () -> Unit,
    onStartLoading: () -> Unit,
    onSelectOrder: (String) -> Unit,
    onToggleItem: (String) -> Unit,
    onSealOrder: () -> Unit,
    onDismissCountdown: () -> Unit,
    canSealOrder: (String) -> Boolean,
    allOrdersSealed: Boolean,
    onSealManifest: () -> Unit,
    onStartNewManifest: () -> Unit,
    onShowInject: () -> Unit,
    onShowException: (String) -> Unit,
    onShowReDispatch: (String) -> Unit,
    onClearEscalated: () -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier.fillMaxSize(),
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            if (truck == null) {
                EmptyState(label = "Select a vehicle", hint = "Pick a truck from the sidebar to load its manifest.")
                return@Column
            }
            DetailHeader(
                truck = truck,
                onRefresh = onRefresh,
                showInject = state.manifest?.state == "LOADING",
                onShowInject = onShowInject,
            )

            state.escalatedMessage?.let { msg ->
                EscalatedBanner(message = msg, onDismiss = onClearEscalated)
            }

            // All Sealed success — terminal state, supersedes everything else.
            if (state.manifestSealed) {
                AllSealedSuccessCard(
                    dispatchCodes = state.dispatchCodes,
                    onStartNewManifest = onStartNewManifest,
                )
                return@Column
            }

            when {
                state.loadingManifest -> CenteredSpinner()
                state.manifest == null -> EmptyState(
                    label = "No open manifest",
                    hint = "This truck has no DRAFT or LOADING manifest. Wait for dispatch.",
                )
                else -> {
                    ManifestSummaryCard(state.manifest)

                    if (state.error != null) {
                        ErrorBanner(state.error)
                    }

                    val phase = state.manifest.state
                    if (phase == "DRAFT") {
                        StartLoadingButton(
                            loading = state.startingLoading,
                            onClick = onStartLoading,
                        )
                        Text(
                            "Tap Start Loading to open the manifest for tap-check and per-order seal.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    } else if (phase == "LOADING" || phase == "SEALED") {
                        // 60-second post-seal countdown takes the spotlight.
                        if (state.postSealOrderId != null) {
                            PostSealCountdownCard(
                                orderId = state.postSealOrderId,
                                dispatchCode = state.dispatchCodes[state.postSealOrderId].orEmpty(),
                                secondsLeft = state.postSealCountdown,
                                onDismiss = onDismissCountdown,
                            )
                        }

                        OrderChecklist(
                            orders = state.orders,
                            loading = state.loadingOrders,
                            selectedOrderId = state.selectedOrderId,
                            checkedItems = state.checkedItems,
                            sealedOrderIds = state.sealedOrderIds,
                            dispatchCodes = state.dispatchCodes,
                            sealingOrderId = state.sealingOrderId,
                            exceptionLoadingOrderId = state.exceptionLoadingOrderId,
                            onSelectOrder = onSelectOrder,
                            onToggleItem = onToggleItem,
                            onSealOrder = onSealOrder,
                            canSealSelected = state.selectedOrderId?.let { canSealOrder(it) } ?: false,
                            onShowException = onShowException,
                            onShowReDispatch = onShowReDispatch,
                        )

                        if (allOrdersSealed && phase != "SEALED") {
                            SealManifestButton(
                                loading = state.sealingManifest,
                                onClick = onSealManifest,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun DetailHeader(
    truck: Truck,
    onRefresh: () -> Unit,
    showInject: Boolean,
    onShowInject: () -> Unit,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.fillMaxWidth(0.7f)) {
            Text(
                truck.label.ifBlank { truck.licensePlate.ifBlank { truck.id.take(8) } },
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                listOfNotNull(
                    truck.licensePlate.takeIf { it.isNotBlank() },
                    truck.vehicleClass.takeIf { it.isNotBlank() },
                ).joinToString(" • "),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Spacer(Modifier.fillMaxWidth(0.5f))
        if (showInject) {
            IconButton(onClick = onShowInject) {
                Icon(Icons.Filled.Add, contentDescription = "Inject order")
            }
        }
        IconButton(onClick = onRefresh) {
            Icon(Icons.Filled.Refresh, contentDescription = "Refresh manifest")
        }
    }
}

@Composable
private fun ManifestSummaryCard(manifest: Manifest) {
    val total = manifest.totalVolumeVu
    val cap = manifest.maxVolumeVu.coerceAtLeast(0.001)
    val pct = (total / cap).coerceIn(0.0, 1.0).toFloat()
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(20.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                StatePill(state = manifest.state)
                Spacer(Modifier.size(10.dp))
                Text(
                    "Manifest ${manifest.manifestId.take(8)}",
                    style = MaterialTheme.typography.titleMedium,
                )
            }
            Text("Stops: ${manifest.stopCount}", style = MaterialTheme.typography.bodyMedium)
            Text(
                "Volume: %.1f / %.1f VU".format(total, manifest.maxVolumeVu),
                style = MaterialTheme.typography.bodyMedium,
            )
            LinearProgressIndicator(
                progress = { pct },
                modifier = Modifier.fillMaxWidth().height(6.dp),
            )
            if (manifest.regionCode.isNotBlank()) {
                Text("Region: ${manifest.regionCode}", style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}

@Composable
private fun StartLoadingButton(loading: Boolean, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = !loading,
        modifier = Modifier.fillMaxWidth().height(56.dp),
    ) {
        if (loading) {
            CircularProgressIndicator(
                modifier = Modifier.size(20.dp),
                strokeWidth = 2.dp,
                color = MaterialTheme.colorScheme.onPrimary,
            )
        } else {
            Text("Start Loading", style = MaterialTheme.typography.titleMedium)
        }
    }
}

@Composable
private fun SealManifestButton(loading: Boolean, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = !loading,
        modifier = Modifier.fillMaxWidth().height(56.dp),
        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.tertiary),
    ) {
        Icon(Icons.Filled.Lock, contentDescription = null)
        Spacer(Modifier.size(8.dp))
        if (loading) {
            CircularProgressIndicator(
                modifier = Modifier.size(20.dp),
                strokeWidth = 2.dp,
                color = MaterialTheme.colorScheme.onTertiary,
            )
        } else {
            Text("Seal Manifest", style = MaterialTheme.typography.titleMedium)
        }
    }
}

// ── Per-order checklist ──────────────────────────────────────────────────────

@Composable
private fun OrderChecklist(
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
) {
    if (loading) {
        CenteredSpinner()
        return
    }
    if (orders.isEmpty()) {
        EmptyState(
            label = "No live orders",
            hint = "No LOADED orders for this vehicle yet. They appear once dispatch assigns them.",
        )
        return
    }
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(20.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(
                "Orders (${sealedOrderIds.size}/${orders.size} sealed)",
                style = MaterialTheme.typography.titleMedium,
            )
            // Order chips
            LazyColumn(
                verticalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth().height(160.dp),
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
                Text(
                    "Items — ${selected.orderId.take(8)}",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                )
                if (selected.items.isEmpty()) {
                    Text(
                        "No line items on this order.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                } else {
                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(4.dp),
                        modifier = Modifier.fillMaxWidth().height(220.dp),
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
                        "Order sealed. Dispatch code ${dispatchCodes[selected.orderId].orEmpty()}.",
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
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                        OutlinedButton(
                            onClick = { onShowException(selected.orderId) },
                            enabled = !excLoading,
                            modifier = Modifier.fillMaxWidth(0.5f).height(44.dp),
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
                            modifier = Modifier.fillMaxWidth().height(44.dp),
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
                Icon(Icons.Filled.CheckCircle, contentDescription = "Sealed", modifier = Modifier.size(18.dp))
                Spacer(Modifier.size(8.dp))
            }
            Column(Modifier.fillMaxWidth(0.7f)) {
                Text(
                    "Order ${order.orderId.take(8)}",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                )
                Text(
                    "${order.items.size} item${if (order.items.size == 1) "" else "s"}",
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
            .clip(RoundedCornerShape(10.dp))
            .clickable(enabled = enabled, onClick = onToggle)
            .padding(horizontal = 4.dp, vertical = 4.dp),
    ) {
        Checkbox(checked = checked, onCheckedChange = { if (enabled) onToggle() }, enabled = enabled)
        Spacer(Modifier.size(4.dp))
        Text(
            label,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.fillMaxWidth(0.7f),
        )
        Text(
            "x$quantity",
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium,
        )
    }
}

// ── Post-seal 60s countdown card (Edge 33 placeholder for missing-items report) ─

@Composable
private fun PostSealCountdownCard(
    orderId: String,
    dispatchCode: String,
    secondsLeft: Int,
    onDismiss: () -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.tertiaryContainer,
        contentColor = MaterialTheme.colorScheme.onTertiaryContainer,
        shape = RoundedCornerShape(20.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                "Order ${orderId.take(8)} sealed",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                "Dispatch code",
                style = MaterialTheme.typography.bodySmall,
            )
            Text(
                dispatchCode,
                style = MaterialTheme.typography.headlineMedium,
                fontFamily = FontFamily.Monospace,
                fontWeight = FontWeight.Bold,
            )
            Text(
                "Double-check window: ${secondsLeft}s",
                style = MaterialTheme.typography.bodyMedium,
            )
            LinearProgressIndicator(
                progress = { (secondsLeft / 60f).coerceIn(0f, 1f) },
                modifier = Modifier.fillMaxWidth().height(4.dp),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                TextButton(onClick = onDismiss) { Text("Continue", fontSize = 14.sp) }
            }
        }
    }
}

// ── All Sealed success terminal screen ───────────────────────────────────────

@Composable
private fun AllSealedSuccessCard(
    dispatchCodes: Map<String, String>,
    onStartNewManifest: () -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.primaryContainer,
        contentColor = MaterialTheme.colorScheme.onPrimaryContainer,
        shape = RoundedCornerShape(24.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            Modifier.padding(28.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Icon(
                Icons.Filled.CheckCircle,
                contentDescription = null,
                modifier = Modifier.size(64.dp),
            )
            Text(
                "Manifest Sealed",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
            )
            Text(
                "${dispatchCodes.size} order${if (dispatchCodes.size == 1) "" else "s"} dispatched",
                style = MaterialTheme.typography.bodyMedium,
            )
            if (dispatchCodes.isNotEmpty()) {
                Surface(
                    color = MaterialTheme.colorScheme.surface,
                    contentColor = MaterialTheme.colorScheme.onSurface,
                    shape = RoundedCornerShape(14.dp),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(6.dp),
                        contentPadding = PaddingValues(12.dp),
                        modifier = Modifier.height(160.dp),
                    ) {
                        items(dispatchCodes.entries.toList(), key = { it.key }) { (orderId, code) ->
                            Row(
                                Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(orderId.take(8), style = MaterialTheme.typography.bodySmall)
                                Text(
                                    code,
                                    style = MaterialTheme.typography.titleSmall,
                                    fontFamily = FontFamily.Monospace,
                                    fontWeight = FontWeight.Bold,
                                )
                            }
                        }
                    }
                }
            }
            OutlinedButton(
                onClick = onStartNewManifest,
                modifier = Modifier.fillMaxWidth().height(52.dp),
            ) {
                Text("Start New Manifest", style = MaterialTheme.typography.titleMedium)
            }
        }
    }
}

// ── Helpers ──────────────────────────────────────────────────────────────────

@Composable
private fun StatePill(state: String) {
    val (bg, fg) = when (state) {
        "DRAFT" -> MaterialTheme.colorScheme.tertiaryContainer to MaterialTheme.colorScheme.onTertiaryContainer
        "LOADING" -> MaterialTheme.colorScheme.primaryContainer to MaterialTheme.colorScheme.onPrimaryContainer
        "SEALED", "DISPATCHED" -> MaterialTheme.colorScheme.secondaryContainer to MaterialTheme.colorScheme.onSecondaryContainer
        else -> MaterialTheme.colorScheme.surfaceVariant to MaterialTheme.colorScheme.onSurfaceVariant
    }
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(50))
            .background(bg)
            .padding(horizontal = 10.dp, vertical = 4.dp),
    ) {
        Text(state, style = MaterialTheme.typography.labelSmall, color = fg, fontWeight = FontWeight.SemiBold)
    }
}

@Composable
private fun CenteredSpinner() {
    Box(Modifier.fillMaxWidth().padding(24.dp), contentAlignment = Alignment.Center) {
        CircularProgressIndicator()
    }
}

@Composable
private fun ErrorBanner(message: String) {
    Surface(
        color = MaterialTheme.colorScheme.errorContainer,
        contentColor = MaterialTheme.colorScheme.onErrorContainer,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Text(
            message,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(12.dp),
        )
    }
}

@Composable
private fun EmptyState(label: String, hint: String) {
    Column(
        modifier = Modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(label, style = MaterialTheme.typography.titleMedium)
        Spacer(Modifier.height(6.dp))
        Text(
            hint,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

// ── Phase 5 dialogs & banners ────────────────────────────────────────────────────────

@Composable
private fun EscalatedBanner(message: String, onDismiss: () -> Unit) {
    Surface(
        color = MaterialTheme.colorScheme.errorContainer,
        contentColor = MaterialTheme.colorScheme.onErrorContainer,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(12.dp),
        ) {
            Icon(Icons.Filled.Warning, contentDescription = null)
            Spacer(Modifier.size(8.dp))
            Text(message, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.fillMaxWidth(0.85f))
            IconButton(onClick = onDismiss) {
                Icon(Icons.Filled.Close, contentDescription = "Dismiss")
            }
        }
    }
}

@Composable
private fun InjectOrderDialog(
    injecting: Boolean,
    onDismiss: () -> Unit,
    onSubmit: (String) -> Unit,
) {
    var orderId by remember { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Inject Order") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    "Add an order mid-load. Dispatch will recompute the manifest.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                OutlinedTextField(
                    value = orderId,
                    onValueChange = { orderId = it },
                    label = { Text("Order ID") },
                    singleLine = true,
                    enabled = !injecting,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        confirmButton = {
            Button(
                onClick = { onSubmit(orderId) },
                enabled = !injecting && orderId.isNotBlank(),
            ) {
                if (injecting) {
                    CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text("Inject")
                }
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !injecting) { Text("Cancel") }
        },
    )
}

@Composable
private fun ExceptionReasonDialog(
    orderId: String,
    inFlight: Boolean,
    onDismiss: () -> Unit,
    onSelect: (String) -> Unit,
) {
    val reasons = listOf(
        "OVERFLOW" to "Overflow — no space",
        "DAMAGED" to "Damaged goods",
        "MANUAL" to "Manual exception",
    )
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Remove order ${orderId.take(8)}") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(
                    "Pick a reason. 3+ overflow attempts on this manifest will escalate to admin DLQ.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                reasons.forEach { (code, label) ->
                    OutlinedButton(
                        onClick = { onSelect(code) },
                        enabled = !inFlight,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(label) }
                }
            }
        },
        confirmButton = {},
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !inFlight) { Text("Cancel") }
        },
    )
}

@Composable
private fun ReDispatchDialog(
    orderId: String,
    loading: Boolean,
    response: RecommendReassignResponse?,
    reassigning: Boolean,
    onDismiss: () -> Unit,
    onPick: (String) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Re-Dispatch ${orderId.take(8)}") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (loading) {
                    CenteredSpinner()
                } else if (response == null) {
                    Text("No recommendations available.", style = MaterialTheme.typography.bodySmall)
                } else {
                    Text(
                        "${response.retailerName.ifBlank { "Order" }} • %.1f VU".format(response.orderVolumeVu),
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.SemiBold,
                    )
                    if (response.recommendations.isEmpty()) {
                        Text(
                            "No suitable trucks. Try again later or remove the order.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    } else {
                        LazyColumn(
                            verticalArrangement = Arrangement.spacedBy(6.dp),
                            modifier = Modifier.fillMaxWidth().height(280.dp),
                        ) {
                            items(response.recommendations, key = { it.driverId }) { rec ->
                                RecommendationCard(
                                    rec = rec,
                                    enabled = !reassigning,
                                    onPick = { onPick(rec.driverId) },
                                )
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {
            if (reassigning) {
                CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                TextButton(onClick = onDismiss) { Text("Close") }
            }
        },
    )
}

@Composable
private fun RecommendationCard(
    rec: TruckRecommendation,
    enabled: Boolean,
    onPick: () -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = enabled, onClick = onPick),
    ) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                Text(
                    rec.driverName.ifBlank { rec.driverId.take(8) },
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.fillMaxWidth(0.7f),
                )
                Text(
                    "score %.2f".format(rec.score),
                    style = MaterialTheme.typography.labelMedium,
                    fontFamily = FontFamily.Monospace,
                )
            }
            Text(
                listOfNotNull(
                    rec.licensePlate.takeIf { it.isNotBlank() },
                    rec.vehicleClass.takeIf { it.isNotBlank() },
                    rec.truckStatus.takeIf { it.isNotBlank() },
                ).joinToString(" • "),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                "%.1f km • free %.1f VU • %d orders".format(rec.distanceKm, rec.freeVolumeVu, rec.orderCount),
                style = MaterialTheme.typography.bodySmall,
            )
            if (rec.recommendation.isNotBlank()) {
                Text(
                    rec.recommendation,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.primary,
                )
            }
        }
    }
}
