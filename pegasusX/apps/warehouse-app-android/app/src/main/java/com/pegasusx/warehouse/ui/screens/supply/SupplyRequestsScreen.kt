package com.pegasusx.warehouse.ui.screens.supply

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.CreateWarehouseSupplyRequestRequest
import com.pegasusx.warehouse.data.model.WarehouseSupplyRequest
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseOpsListCard
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplyRequestsScreen(
    api: WarehouseApi,
    onRequestClick: (String) -> Unit,
    onBack: (() -> Unit)? = null,
) {
    var requests by remember { mutableStateOf<List<WarehouseSupplyRequest>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var stateFilter by remember { mutableStateOf("ALL") }
    var showCreate by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val filters = listOf("ALL", "OPEN", "CANCELLED")

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val state = if (stateFilter == "ALL") null else stateFilter
                val resp = api.getSupplyRequests(state)
                requests = if (resp.isSuccessful) resp.body()?.resolved().orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(stateFilter) { load() }

    fun createSupplyRequest(form: SupplyRequestFormResult) {
        scope.launch {
            try {
                val mode = if (form.useDemandForecast) "FORECAST" else "MANUAL"
                val key = WarehouseIdempotencyKeys.createSupplyRequest(form.factoryId, mode, form.notes)
                val resp = api.createSupplyRequest(
                    key,
                    CreateWarehouseSupplyRequestRequest(
                        factoryId = form.factoryId,
                        priority = form.priority,
                        notes = form.notes,
                        items = form.items,
                        useDemandForecast = form.useDemandForecast,
                        requestedDeliveryDate = form.requestedDeliveryDate,
                    ),
                )
                if (resp.isSuccessful) {
                    showCreate = false
                    load()
                }
            } catch (_: Exception) {
                // Errors surface on next refresh; dialog stays open for retry.
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Supply requests") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back)) } } },
                actions = {
                    var expanded by remember { mutableStateOf(false) }
                    Box {
                        TextButton(onClick = { expanded = true }) { Text(stateFilter) }
                        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                            filters.forEach { filter ->
                                DropdownMenuItem(
                                    text = { Text(filter) },
                                    onClick = {
                                        stateFilter = filter
                                        expanded = false
                                    },
                                )
                            }
                        }
                    }
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = stringResource(R.string.portal_page_orders_action_refresh))
                    }
                    IconButton(onClick = { showCreate = true }) {
                        Icon(Icons.Default.Add, contentDescription = stringResource(R.string.mobile_warehouse_ui_new_request))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState(
                title = stringResource(R.string.mobile_warehouse_ui_loading_supply_requests),
                body = "Factory supply queue",
                modifier = Modifier.padding(padding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Supply requests unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(padding),
            )
            requests.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No supply requests",
                body = if (stateFilter == "ALL") {
                    "Submitted factory supply requests will appear here."
                } else {
                    "No requests match the $stateFilter filter."
                },
                modifier = Modifier.padding(padding),
            )
            else -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(requests, key = { it.requestId }) { request ->
                    WarehouseOpsListCard(
                        headline = request.requestId.take(12),
                        supporting = "${request.priority} · ${request.totalVolumeVu} VU",
                        status = request.state,
                        onClick = { onRequestClick(request.requestId) },
                    )
                }
            }
        }
    }

    if (showCreate) {
        CreateSupplyRequestDialog(
            api = api,
            onDismiss = { showCreate = false },
            onCreate = { form -> createSupplyRequest(form) },
        )
    }
}
