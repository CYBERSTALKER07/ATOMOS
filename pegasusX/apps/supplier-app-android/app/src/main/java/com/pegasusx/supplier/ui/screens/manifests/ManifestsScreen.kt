package com.pegasusx.supplier.ui.screens.manifests

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasus.design.RealtimeRefreshEffect
import com.pegasus.design.showFullScreenLoading
import com.pegasusx.supplier.data.model.SupplierManifestRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ManifestsScreen(
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onBack: () -> Unit,
    onOpenManifest: (String) -> Unit = {},
    onOpenGateExceptions: () -> Unit = {},
) {
    var rows by remember { mutableStateOf<List<SupplierManifestRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        scope.launch {
            if (!silent) {
                loading = true
                error = null
            }
            try {
                val resp = ops.getManifests()
                if (resp.isSuccessful) {
                    rows = resp.body()?.manifests.orEmpty()
                } else if (!silent) {
                    error = "Failed (${resp.code()})"
                    rows = emptyList()
                }
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    RealtimeRefreshEffect(
        refreshTick = realtimeSignals.refreshTick,
        reconnectTick = realtimeSignals.reconnectTick,
        onRefresh = { load(silent = it) },
    )

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Manifests") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    TextButton(onClick = onOpenGateExceptions) { Text("Gate") }
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        when {
            showFullScreenLoading(loading, rows.isNotEmpty()) -> SupplierLoadingState(
                title = "Loading manifests…",
                body = "Supplier manifest queue",
                modifier = Modifier.padding(padding),
            )
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Manifests unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No manifests",
                body = "Loading manifests will appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(rows, key = { it.manifestId }) { row ->
                    val state = row.state.ifBlank { row.status }
                    SupplierOpsListCard(
                        headline = row.manifestId.take(12),
                        supporting = buildString {
                            append("${row.ordersCount} orders")
                            if (row.stopCount > 0) append(" · ${row.stopCount} stops")
                            val driver = row.driverName.ifBlank { row.driverId.orEmpty() }
                            if (driver.isNotBlank()) append(" · $driver")
                            row.vehiclePlate?.takeIf { it.isNotBlank() }?.let { append(" · $it") }
                        },
                        status = state,
                        onClick = { onOpenManifest(row.manifestId) },
                    )
                }
            }
        }
    }
}
