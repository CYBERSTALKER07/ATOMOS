package com.pegasusx.warehouse.ui.screens.supply

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.WarehouseSupplyRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.components.WarehouseLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseOpsListCard
import com.pegasusx.warehouse.ui.components.WarehouseStateKind
import com.pegasusx.warehouse.ui.components.WarehouseStatePane
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

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

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Supply requests") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") } } },
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
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> WarehouseLoadingState(
                title = "Loading supply requests…",
                body = "Factory supply queue",
                modifier = Modifier.padding(padding),
            )
            error != null -> WarehouseStatePane(
                kind = WarehouseStateKind.Error,
                headline = "Supply requests unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(padding),
            )
            requests.isEmpty() -> WarehouseStatePane(
                kind = WarehouseStateKind.Empty,
                headline = "No supply requests",
                body = if (stateFilter == "ALL") {
                    "Submitted factory supply requests will appear here."
                } else {
                    "No requests match the $stateFilter filter."
                },
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
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
}
