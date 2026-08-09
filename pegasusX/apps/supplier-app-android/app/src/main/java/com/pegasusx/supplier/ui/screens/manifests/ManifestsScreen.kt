package com.pegasusx.supplier.ui.screens.manifests

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.layout.*
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
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

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
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    TextButton(onClick = onOpenGateExceptions) { Text("Gate") }
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = stringResource(R.string.portal_page_orders_action_refresh))
                    }
                },
            )
        },
    ) { padding ->
        when {
            showFullScreenLoading(loading, rows.isNotEmpty()) -> PegasusLoadingState(
                title = stringResource(R.string.mobile_supplier_ui_loading_manifests),
                body = "Supplier manifest queue",
                modifier = Modifier.padding(padding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Manifests unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No manifests",
                body = "Loading manifests will appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> ManifestsList(
                rows = rows,
                modifier = Modifier.padding(padding),
                onOpenManifest = onOpenManifest
            )
        }
    }
}
