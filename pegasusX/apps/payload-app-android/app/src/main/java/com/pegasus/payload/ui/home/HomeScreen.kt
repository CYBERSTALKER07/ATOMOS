package com.pegasus.payload.ui.home

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
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
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.MenuOpen
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.QrCodeScanner
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Undo
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material.icons.filled.Warning
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
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
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.barcode.EanBarcodeScannerPreview
import com.pegasus.payload.data.model.LiveOrder
import com.pegasus.payload.data.model.Manifest
import com.pegasus.payload.data.model.ManifestExceptionRow
import com.pegasus.payload.data.model.RecommendReassignResponse
import com.pegasus.payload.data.model.NotificationItem
import com.pegasus.payload.data.model.Truck
import com.pegasus.payload.data.model.TruckRecommendation
import com.pegasus.payload.ui.components.ExplainStatusBanner
import com.pegasus.payload.ui.components.HandoffInboxCard
import com.pegasus.payload.ui.components.ManifestKpiGrid
import com.pegasus.payload.ui.components.PayloadConnectionStatus
import com.pegasus.payload.ui.components.PulseStrip
import com.pegasus.payload.ui.components.PayloadSectionTitle
import com.pegasus.payload.ui.components.PayloadSpacing
import com.pegasus.payload.ui.components.PayloadStatusChip
import com.pegasus.design.PegasusStatePane
import com.pegasus.design.PegasusStateKind

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
    onInboundReturns: () -> Unit = {},
    viewModel: HomeViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val navigator = rememberListDetailPaneScaffoldNavigator<String>()
    val snackbarHostState = remember { SnackbarHostState() }
    val lifecycleOwner = LocalLifecycleOwner.current

    var showInjectDialog by remember { mutableStateOf(false) }
    var showChecklistScanner by remember { mutableStateOf(false) }
    var exceptionTargetOrderId by remember { mutableStateOf<String?>(null) }
    var isListExpanded by remember { mutableStateOf(true) }

    LaunchedEffect(state.trucks, state.selectedTruckId) {
        val truckId = state.selectedTruckId ?: state.trucks.firstOrNull()?.id ?: return@LaunchedEffect
        if (navigator.scaffoldValue[ListDetailPaneScaffoldRole.Detail] == null) {
            navigator.navigateTo(ListDetailPaneScaffoldRole.Detail, truckId)
        }
    }

    LaunchedEffect(state.missingItemsReportedMessage) {
        state.missingItemsReportedMessage?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearMissingItemsReportedMessage()
        }
    }
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
    LaunchedEffect(state.barcodeScanMessage) {
        if (state.barcodeScanMessage != null) {
            kotlinx.coroutines.delay(3000)
            viewModel.clearBarcodeScanMessage()
        }
    }
    LaunchedEffect(state.handoffNavigationMessage) {
        state.handoffNavigationMessage?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearHandoffNavigationMessage()
        }
    }
    LaunchedEffect(state.online) {
        if (!state.online) return@LaunchedEffect
        viewModel.refreshTrucks(silent = state.trucks.isNotEmpty())
        viewModel.refreshManifest(silent = state.manifest != null || state.orders.isNotEmpty())
        viewModel.refreshPulse()
    }
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                viewModel.refreshTrucks(silent = state.trucks.isNotEmpty())
                viewModel.refreshManifest(silent = state.manifest != null || state.orders.isNotEmpty())
                viewModel.refreshPulse()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text("Pegasus Payload Terminal")
                        PayloadConnectionStatus(online = state.online, queued = state.queuedActions)
                    }
                },
                actions = {
                    IconButton(onClick = onInboundReturns) {
                        Icon(Icons.Filled.Undo, contentDescription = "Inbound returns")
                    }
                    IconButton(onClick = { viewModel.toggleExceptionsPanel() }) {
                        Icon(Icons.Filled.Warning, contentDescription = "Manifest exceptions")
                    }
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
                        Icon(Icons.AutoMirrored.Filled.Logout, contentDescription = "Logout")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surfaceContainer,
                ),
            )
        },
    ) { padding ->
        Column(modifier = Modifier.padding(padding).fillMaxSize()) {
            PulseStrip(
                events = state.pulseEvents,
                loading = state.pulseLoading,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = PayloadSpacing.lg, vertical = PayloadSpacing.sm),
            )
            ListDetailPaneScaffold(
            modifier = Modifier.fillMaxSize(),
            directive = navigator.scaffoldDirective,
            value = navigator.scaffoldValue,
            listPane = {
                AnimatedPane {
                    TruckListPane(
                        trucks = state.trucks,
                        selectedTruckId = state.selectedTruckId,
                        loading = state.loadingTrucks,
                        error = state.error,
                        batchReadyCount = state.batchReadyManifestIds.size,
                        batchSealing = state.batchSealing,
                        batchSealFailures = state.batchSealFailures,
                        isExpanded = isListExpanded,
                        onToggleExpanded = { isListExpanded = !isListExpanded },
                        onFinalizeBatch = viewModel::finalizeBatchSeal,
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
                        onReportMissingItems = viewModel::reportMissingItems,
                        reportingMissingItems = state.reportingMissingItems,
                        canSealOrder = viewModel::canSealOrder,
                        allOrdersSealed = viewModel.allOrdersSealed,
                        onSealManifest = viewModel::sealManifest,
                        onStartNewManifest = viewModel::startNewManifest,
                        onShowInject = { showInjectDialog = true },
                        onShowException = { exceptionTargetOrderId = it },
                        onShowReDispatch = viewModel::openReDispatch,
                        onClearEscalated = viewModel::clearEscalatedMessage,
                        onScanProduct = { showChecklistScanner = true },
                    )
                }
            },
        )

        if (showChecklistScanner) {
            ModalBottomSheet(
                onDismissRequest = { showChecklistScanner = false },
                sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
            ) {
                Column(
                    Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Text("Scan product EAN", style = MaterialTheme.typography.titleMedium)
                    EanBarcodeScannerPreview(
                        enabled = true,
                        onBarcode = { code ->
                            viewModel.onBarcodeScanned(code)
                            showChecklistScanner = false
                        },
                    )
                    TextButton(onClick = { showChecklistScanner = false }) { Text("Close") }
                    Spacer(Modifier.height(24.dp))
                }
            }
        }

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
                onPick = { driverId, isPartial -> viewModel.reassignTo(driverId, isPartial) },
            )
        }
        if (state.showNotificationsPanel) {
            NotificationsSheet(
                items = state.notifications,
                unreadCount = state.unreadCount,
                onDismiss = viewModel::toggleNotificationsPanel,
                onMarkRead = viewModel::markNotificationRead,
                onMarkAllRead = viewModel::markAllNotificationsRead,
                onHandoffAction = viewModel::handleHandoffLink,
            )
        }
        if (state.showExceptionsPanel) {
            ManifestExceptionsSheet(
                items = state.manifestExceptions,
                loading = state.loadingExceptions,
                onDismiss = viewModel::toggleExceptionsPanel,
                onRefresh = viewModel::loadManifestExceptions,
            )
        }
        }
    }
}


// ── Sidebar (truck list) ─────────────────────────────────────────────────────

// ── Detail pane (Phase 4 loading workflow) ───────────────────────────────────

/*
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
    onReportMissingItems: (String) -> Unit,
    reportingMissingItems: Boolean,
    canSealOrder: (String) -> Boolean,
    allOrdersSealed: Boolean,
    onSealManifest: () -> Unit,
    onStartNewManifest: () -> Unit,
    onShowInject: () -> Unit,
    onShowException: (String) -> Unit,
    onShowReDispatch: (String) -> Unit,
    onClearEscalated: () -> Unit,
    onScanProduct: () -> Unit,
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
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "Select a vehicle",
                    body = "Pick a truck from the sidebar to load its manifest.",
                )
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

            state.barcodeScanMessage?.let { msg ->
                com.pegasus.design.PegasusRuntimeBanner(
                    tone = if (msg.contains("error", ignoreCase = true) || msg.contains("failed", ignoreCase = true) || msg.contains("not", ignoreCase = true)) com.pegasus.design.PegasusRuntimeTone.Warning else com.pegasus.design.PegasusRuntimeTone.Live,
                    message = msg,
                    onRetry = null,
                )
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
                state.loadingManifest -> com.pegasus.design.PegasusLoadingState(
                    title = "Loading manifest",
                    body = "Syncing orders and volume for this vehicle.",
                )
                state.manifest == null -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No open manifest",
                    body = "This vehicle doesn't have an active loading manifest.",
                )
                else -> {
                    ManifestKpiGrid(manifest = state.manifest)

                    if (state.error != null || state.errorExplain != null) {
                        ExplainStatusBanner(
                            explain = state.errorExplain,
                            fallbackTitle = state.error,
                            modifier = Modifier.fillMaxWidth(),
                        )
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
                                reportingMissingItems = reportingMissingItems,
                                onDismiss = onDismissCountdown,
                                onReportMissingItems = { onReportMissingItems(state.postSealOrderId) },
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
                            onScanProduct = onScanProduct,
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
        Column(Modifier.weight(1f)) {
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
        Spacer(Modifier.width(12.dp))
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
*/

// ── Per-order checklist ──────────────────────────────────────────────────────



// ── Post-seal 60s countdown card (Edge 33 placeholder for missing-items report) ─

@Composable
internal fun PostSealCountdownCard(
    orderId: String,
    dispatchCode: String,
    secondsLeft: Int,
    reportingMissingItems: Boolean,
    onDismiss: () -> Unit,
    onReportMissingItems: () -> Unit,
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
                OutlinedButton(
                    onClick = onReportMissingItems,
                    enabled = !reportingMissingItems,
                ) {
                    if (reportingMissingItems) {
                        CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Text("Report missing items")
                    }
                }
                TextButton(onClick = onDismiss) {
                    Text("Continue", style = MaterialTheme.typography.labelLarge)
                }
            }
        }
    }
}

// ── All Sealed success terminal screen ───────────────────────────────────────

@Composable
internal fun AllSealedSuccessCard(
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
                    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                        verticalArrangement = Arrangement.spacedBy(6.dp),
                        contentPadding = PaddingValues(12.dp),
                        modifier = Modifier.height(160.dp),
        horizontalArrangement = Arrangement.spacedBy(6.dp)
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

// ── Phase 5 dialogs & banners ────────────────────────────────────────────────────────

@Composable
internal fun EscalatedBanner(message: String, onDismiss: () -> Unit) {
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
    var showScanner by remember { mutableStateOf(false) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Inject Order") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    "Add an order mid-load. Scan an order label or enter the order ID.",
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
                OutlinedButton(
                    onClick = { showScanner = !showScanner },
                    enabled = !injecting,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Icon(Icons.Filled.QrCodeScanner, contentDescription = null, modifier = Modifier.size(18.dp))
                    Spacer(Modifier.size(8.dp))
                    Text(if (showScanner) "Hide scanner" else "Scan order label")
                }
                if (showScanner) {
                    EanBarcodeScannerPreview(
                        enabled = !injecting,
                        onBarcode = { scanned ->
                            orderId = scanned.trim()
                            showScanner = false
                        },
                    )
                }
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
    onPick: (String, Boolean) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Re-Dispatch ${orderId.take(8)}") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (loading) {
                    com.pegasus.design.PegasusLoadingState(
                        title = "Loading recommendations",
                        body = "Finding nearby drivers to re-dispatch this order.",
                        modifier = Modifier.fillMaxWidth().padding(32.dp)
                    )
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
                        LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                            verticalArrangement = Arrangement.spacedBy(6.dp),
                            modifier = Modifier.fillMaxWidth().height(280.dp),
        horizontalArrangement = Arrangement.spacedBy(6.dp)
    ) {
                            items(response.recommendations, key = { it.driverId }) { rec ->
                                RecommendationCard(
                                    rec = rec,
                                    enabled = !reassigning,
                                    onPickComplete = { onPick(rec.driverId, false) },
                                    onPickPartial = { onPick(rec.driverId, true) },
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


