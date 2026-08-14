package com.pegasusx.factory.ui.screens.supply

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.Badge
import androidx.compose.material3.Button
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.pegasusx.factory.data.remote.FactoryRealtimeStatus
import com.pegasusx.factory.data.model.SupplyRequest
import com.pegasusx.factory.data.model.SupplyFulfillOptions
import com.pegasusx.factory.data.model.SupplyRequestQCRequest
import com.pegasusx.factory.data.model.SupplyRequestTransitionRequest
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.util.FactoryIdempotencyKeys
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.ui.screens.supply.components.*
import java.text.DateFormat
import java.util.Date
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import com.pegasusx.factory.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplyRequestsScreen(
    api: FactoryApi,
    onBack: () -> Unit,
) {
    var requests by remember { mutableStateOf<List<SupplyRequest>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var filter by remember { mutableStateOf("ALL") }
    var viewMode by remember { mutableStateOf("TABLE") }
    var fulfillModal by remember { mutableStateOf<Pair<SupplyRequest, SupplyFulfillOptions>?>(null) }
    var transitioningId by remember { mutableStateOf<String?>(null) }
    var qcById by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var refreshing by remember { mutableStateOf(false) }
    var staleMessage by remember { mutableStateOf<String?>(null) }
    var lastSyncedAt by remember { mutableStateOf<Long?>(null) }
    var realtimeStatus by remember { mutableStateOf(FactoryRealtimeStatus.IDLE) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }
    val lifecycleOwner = LocalLifecycleOwner.current

    fun load(background: Boolean = false) {
        if (background) {
            refreshing = true
        } else if (requests.isEmpty()) {
            loading = true
            error = null
        }

        scope.launch {
            try {
                val resp = api.getSupplyRequests()
                if (resp.isSuccessful && resp.body() != null) {
                    requests = resp.body()!!.requests
                    lastSyncedAt = System.currentTimeMillis()
                    staleMessage = null
                    error = null
                    val ids = requests.map { it.id }
                    qcById = try {
                        coroutineScope {
                            ids.map { id ->
                                async {
                                    val qc = api.getSupplyRequestQC(id)
                                    id to (qc.body()?.result.orEmpty())
                                }
                            }.awaitAll().toMap()
                        }
                    } catch (_: Exception) {
                        emptyMap()
                    }
                } else {
                    val message = "Failed (${resp.code()})"
                    if (requests.isEmpty()) {
                        error = message
                    } else {
                        staleMessage = "Showing last synced queue. $message"
                    }
                }
            } catch (e: Exception) {
                val message = e.message ?: "Network error"
                if (requests.isEmpty()) {
                    error = message
                } else {
                    staleMessage = "Showing last synced queue. $message"
                }
            } finally {
                loading = false
                refreshing = false
            }
        }
    }

    fun runTransition(request: SupplyRequest, action: String) {
        transitioningId = request.id
        scope.launch {
            try {
                val resp = api.transitionSupplyRequest(
                    request.id,
                    FactoryIdempotencyKeys.supplyRequestTransition(request.id, action),
                    SupplyRequestTransitionRequest(action = action),
                )
                if (resp.isSuccessful) {
                    snackbarHostState.showSnackbar("${requestLabel(request)} moved to ${resp.body()?.state ?: action}")
                    load(background = true)
                } else {
                    snackbarHostState.showSnackbar("Transition failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Transition failed")
            } finally {
                transitioningId = null
            }
        }
    }

    fun onAction(request: SupplyRequest, action: String) {
        if (action == "FULFILL") {
            scope.launch {
                try {
                    val resp = api.getSupplyFulfillOptions(request.id)
                    val options = if (resp.isSuccessful) resp.body() else null
                    if (options != null) {
                        fulfillModal = request to options
                    } else {
                        snackbarHostState.showSnackbar("Could not load fulfill options")
                    }
                } catch (e: Exception) {
                    snackbarHostState.showSnackbar(e.message ?: "Could not load fulfill options")
                }
            }
            return
        }
        runTransition(request, action)
    }

    fun onQC(request: SupplyRequest, result: String) {
        transitioningId = request.id
        scope.launch {
            try {
                val resp = api.postSupplyRequestQC(
                    request.id,
                    FactoryIdempotencyKeys.supplyRequestQC(request.id, result),
                    SupplyRequestQCRequest(result = result),
                )
                if (resp.isSuccessful) {
                    qcById = qcById + (request.id to result)
                    snackbarHostState.showSnackbar("QC $result")
                } else {
                    snackbarHostState.showSnackbar("QC failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "QC failed")
            } finally {
                transitioningId = null
            }
        }
    }

    LaunchedEffect(Unit) {
        load()
        while (isActive) {
            delay(30_000)
            if (transitioningId == null) {
                load(background = true)
            }
        }
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                load(background = requests.isNotEmpty())
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(FactoryRealtimeEventType.SupplyRequestUpdate),
        onStatusChange = { status ->
            realtimeStatus = status
        },
    ) {
        if (transitioningId == null) {
            load(background = requests.isNotEmpty())
        }
    }

    val filteredRequests = if (filter == "ALL") requests else requests.filter { it.state == filter }
    val runtimeStatus = when {
        staleMessage != null -> staleMessage!!
        realtimeStatus == FactoryRealtimeStatus.OFFLINE -> "Offline — showing last sync ${formatSyncTime(lastSyncedAt)}"
        realtimeStatus == FactoryRealtimeStatus.RECONNECTING -> "Reconnecting live queue — last sync ${formatSyncTime(lastSyncedAt)}"
        realtimeStatus == FactoryRealtimeStatus.CONNECTING -> "Connecting to the live supply queue…"
        refreshing -> "Refreshing live queue — last sync ${formatSyncTime(lastSyncedAt)}"
        lastSyncedAt != null -> "Live sync active — last sync ${formatSyncTime(lastSyncedAt)}"
        else -> "Waiting for first sync"
    }
    val runtimeTone = when {
        staleMessage != null && realtimeStatus == FactoryRealtimeStatus.OFFLINE -> PegasusRuntimeTone.Offline
        staleMessage != null -> PegasusRuntimeTone.Warning
        realtimeStatus == FactoryRealtimeStatus.OFFLINE -> PegasusRuntimeTone.Offline
        realtimeStatus == FactoryRealtimeStatus.RECONNECTING || realtimeStatus == FactoryRealtimeStatus.CONNECTING -> PegasusRuntimeTone.Refreshing
        refreshing -> PegasusRuntimeTone.Refreshing
        else -> PegasusRuntimeTone.Live
    }

    fulfillModal?.let { (request, options) ->
        ModalBottomSheet(onDismissRequest = { fulfillModal = null }) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text("Confirm fulfill", style = MaterialTheme.typography.titleLarge)
                Text(
                    stringResource(R.string.mobile_factory_ui_warehouse_warehousename_mode_transfermode, options.warehouseName, options.transferMode) +
                        if (options.coLocated) " · Co-located site" else "",
                    style = MaterialTheme.typography.bodyMedium,
                )
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = MaterialTheme.shapes.medium,
                    color = MaterialTheme.colorScheme.surfaceContainerLowest,
                ) {
                    Column(Modifier.padding(PegasusSpacing.md), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text(stringResource(R.string.mobile_factory_ui_internal_outcomeinternal, options.outcomeInternal), style = MaterialTheme.typography.bodySmall)
                        Text(stringResource(R.string.mobile_factory_ui_truck_outcometruck, options.outcomeTruck), style = MaterialTheme.typography.bodySmall)
                        options.linkedDriverEta?.let {
                            Text(stringResource(R.string.mobile_factory_ui_linked_transfer_updated_it, it), style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    OutlinedButton(
                        onClick = { fulfillModal = null },
                        modifier = Modifier.weight(1f),
                    ) { Text("Cancel") }
                    FilledTonalButton(
                        onClick = {
                            fulfillModal = null
                            runTransition(request, "FULFILL")
                        },
                        enabled = transitioningId == null,
                        modifier = Modifier.weight(1f),
                    ) { Text("Confirm fulfill") }
                }
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Supply Requests") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load(background = requests.isNotEmpty()) }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading -> PegasusLoadingState(
                title = stringResource(R.string.mobile_factory_ui_loading_supply_requests),
                body = "Fetching the current warehouse demand queue for this factory.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = if (realtimeStatus == FactoryRealtimeStatus.OFFLINE) PegasusStateKind.Offline else PegasusStateKind.Error,
                headline = if (realtimeStatus == FactoryRealtimeStatus.OFFLINE) "Supply queue unavailable offline" else "Unable to load supply requests",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            filteredRequests.isEmpty() -> PegasusStatePane(
                kind = if (filter == "ALL") PegasusStateKind.Empty else PegasusStateKind.NoResults,
                headline = if (filter == "ALL") "No supply requests in queue" else "No ${filter.replace('_', ' ')} requests right now",
                body = if (filter == "ALL") {
                    "Warehouse demand will appear here as soon as requests reach this factory queue."
                } else {
                    "Adjust the active filter or wait for the next queue refresh."
                },
                actionLabel = if (filter == "ALL") null else "Clear Filter",
                onAction = if (filter == "ALL") null else ({ filter = "ALL" }),
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
                item {
                    FilterRow(
                        selected = filter,
                        onSelect = { filter = it },
                    )
                }
                item {
                    SupplySummaryCard(
                        total = requests.size,
                        visible = filteredRequests.size,
                        runtimeStatus = runtimeStatus,
                        runtimeTone = runtimeTone,
                    )
                }
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        FilterChip(selected = viewMode == "TABLE", onClick = { viewMode = "TABLE" }, label = { Text("Table") })
                        FilterChip(selected = viewMode == "BOARD", onClick = { viewMode = "BOARD" }, label = { Text("Board") })
                    }
                }
                if (viewMode == "BOARD") {
                    item {
                        SupplyBoard(
                            requests = filteredRequests,
                            transitioningId = transitioningId,
                            qcById = qcById,
                            onAction = { request, action -> onAction(request, action) },
                            onQC = { request, result -> onQC(request, result) },
                        )
                    }
                } else {
                items(filteredRequests, key = { it.id }) { request ->
                    SupplyRequestCard(
                        request = request,
                        transitioning = transitioningId == request.id,
                        qcResult = qcById[request.id].orEmpty(),
                        onAction = { action -> onAction(request, action) },
                        onQC = { result -> onQC(request, result) },
                    )
                }
                }
            }
        }
    }
}

