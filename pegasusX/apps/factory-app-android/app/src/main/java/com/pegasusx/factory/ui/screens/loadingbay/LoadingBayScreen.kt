package com.pegasusx.factory.ui.screens.loadingbay

import androidx.compose.ui.res.stringResource

import androidx.compose.ui.unit.dp

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.factory.ui.screens.loadingbay.components.LoadingBayGrid
import com.pegasusx.factory.ui.screens.loadingbay.components.LoadingBayControls
import com.pegasusx.factory.data.model.DispatchRequest
import com.pegasusx.factory.data.model.PulseEvent
import com.pegasusx.factory.data.model.Transfer
import com.pegasusx.factory.util.filterHandoffPulseEvents
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasus.design.PulseHonesty
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.util.FactoryIdempotencyKeys
import kotlinx.coroutines.launch
import com.pegasusx.factory.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoadingBayScreen(
    api: FactoryApi,
    onTransferClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var transfers by remember { mutableStateOf<List<Transfer>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var dispatching by remember { mutableStateOf(false) }
    var handoffEvents by remember { mutableStateOf<List<PulseEvent>>(emptyList()) }
    var handoffLoading by remember { mutableStateOf(true) }
    var handoffError by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
        error = null
        scope.launch {
            try {
                val resp = api.getLoadingBayTransfers()
                if (resp.isSuccessful && resp.body() != null) {
                    transfers = resp.body()!!.transfers
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            }
            handoffLoading = true
            handoffError = null
            try {
                val pulseResp = api.getPulse()
                val result = PulseHonesty.applyHttp(
                    pulseResp.isSuccessful,
                    pulseResp.body()?.events?.let { filterHandoffPulseEvents(it) },
                    handoffEvents,
                )
                handoffEvents = result.events
                handoffError = result.error
            } catch (_: Exception) {
                handoffError = PulseHonesty.FAILED
            }
            handoffLoading = false
            if (!silent) {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        if (!dispatching) {
            load(silent = transfers.isNotEmpty())
        }
    }

    val approved = transfers.filter { it.state == "APPROVED" }
    val loadingState = transfers.filter { it.state == "LOADING" }
    val dispatched = transfers.filter { it.state == "DISPATCHED" }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Loading Bay")
                        Text(
                            text = stringResource(R.string.mobile_factory_ui_approved_loading_and_dispatched_queues),
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
        floatingActionButton = {
            if (loadingState.isNotEmpty()) {
                LoadingBayControls(
                    dispatching = dispatching,
                    onClick = {
                        if (!dispatching) {
                            dispatching = true
                            scope.launch {
                                try {
                                    val ids = loadingState.map { it.id }
                                    val resp = api.dispatch(
                                        DispatchRequest(mode = "AUTO", transferIds = ids, reason = "factory-loading-bay"),
                                        FactoryIdempotencyKeys.batchDispatch(ids),
                                    )
                                    if (resp.isSuccessful) {
                                        val body = resp.body()
                                        val count = body?.createdManifestCount ?: body?.manifestsCreated ?: ids.size
                                        val algo = body?.dispatchAlgo?.ifBlank { body.optimizerClass } ?: ""
                                        snackbarHostState.showSnackbar(
                                            if (count == 0) "No transfers to dispatch"
                                            else "Dispatched $count · $algo"
                                        )
                                        load()
                                    } else {
                                        snackbarHostState.showSnackbar("Dispatch failed (${resp.code()})")
                                    }
                                } catch (e: Exception) {
                                    snackbarHostState.showSnackbar(e.message ?: "Error")
                                } finally {
                                    dispatching = false
                                }
                            }
                        }
                    },
                )
            }
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading && transfers.isEmpty() -> PegasusLoadingState(
                title = stringResource(R.string.mobile_factory_ui_loading_bay_status),
                body = "Fetching approved, loading, and dispatched transfer groups for the bay.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load loading bay",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LoadingBayGrid(
                approved = approved,
                loadingState = loadingState,
                dispatched = dispatched,
                handoffEvents = handoffEvents,
                handoffLoading = handoffLoading,
                handoffError = handoffError,
                onTransferClick = onTransferClick,
                innerPadding = innerPadding
            )
        }
    }
}
