package com.pegasusx.factory.ui.screens.transfer

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.Transfer
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.components.FactoryStatusChip
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.screens.transfer.components.TransferFilters
import com.pegasusx.factory.ui.screens.transfer.components.TransferList
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.factory.R

private val STATE_FILTERS = listOf("ALL", "DRAFT", "APPROVED", "LOADING", "DISPATCHED", "IN_TRANSIT", "ARRIVED", "RECEIVED", "CANCELLED")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransferListScreen(
    api: FactoryApi,
    onTransferClick: (String) -> Unit,
    onCreateTransfer: () -> Unit,
    onBack: () -> Unit,
) {
    var transfers by remember { mutableStateOf<List<Transfer>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selectedFilter by remember { mutableStateOf("ALL") }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
        error = null
        scope.launch {
            try {
                val state = if (selectedFilter == "ALL") null else selectedFilter
                val resp = api.getTransfers(state = state)
                if (resp.isSuccessful && resp.body() != null) {
                    transfers = resp.body()!!.transfers
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                if (!silent) {
                    loading = false
                }
            }
        }
    }

    LaunchedEffect(selectedFilter) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        load(silent = true)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Transfers")
                        Text(
                            text = stringResource(R.string.mobile_factory_ui_factory_to_warehouse_movement_pipeline),
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = onCreateTransfer) { Icon(Icons.Default.Add, "Create transfer") }
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            TransferFilters(
                filters = STATE_FILTERS,
                selectedFilter = selectedFilter,
                onFilterSelected = { selectedFilter = it }
            )

            when {
                loading && transfers.isEmpty() -> PegasusLoadingState(
                    title = stringResource(R.string.mobile_factory_ui_loading_transfers),
                    body = "Fetching the current transfer pipeline for this factory.",
                    modifier = Modifier.fillMaxSize(),
                )
                error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Unable to load transfers",
                    body = error!!,
                    actionLabel = "Retry",
                    onAction = { load() },
                    modifier = Modifier.fillMaxSize(),
                )
                transfers.isEmpty() -> PegasusStatePane(
                    kind = if (selectedFilter == "ALL") PegasusStateKind.Empty else PegasusStateKind.NoResults,
                    headline = if (selectedFilter == "ALL") "No transfers available" else "No ${selectedFilter.replace('_', ' ')} transfers",
                    body = if (selectedFilter == "ALL") {
                        "Transfers will appear here as soon as warehouse demand enters the factory pipeline."
                    } else {
                        "Adjust the active state filter or wait for the next transfer update."
                    },
                    actionLabel = if (selectedFilter == "ALL") null else "Clear Filter",
                    onAction = if (selectedFilter == "ALL") null else ({ selectedFilter = "ALL" }),
                    modifier = Modifier.fillMaxSize(),
                )
                else -> TransferList(
                    transfers = transfers,
                    selectedFilter = selectedFilter,
                    onTransferClick = onTransferClick
                )
            }
        }
    }
}

