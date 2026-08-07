package com.pegasusx.supplier.ui.screens.exceptions

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.ShopClosedAttemptRow
import com.pegasusx.supplier.data.model.ShopClosedResolveRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.realtime.SupplierReconnectRecoveryEffect
import com.pegasusx.supplier.util.SUPPLIER_RECONNECT_RECOVERY_HINT
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ShopClosedScreen(
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onBack: () -> Unit,
) {
    var rows by remember { mutableStateOf<List<ShopClosedAttemptRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var busyId by remember { mutableStateOf<String?>(null) }
    val snackbar = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getShopClosedActive()
                rows = if (resp.isSuccessful) resp.body()?.data.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun resolve(attemptId: String, action: String) {
        busyId = attemptId
        scope.launch {
            try {
                val resp = ops.resolveShopClosed(
                    ShopClosedResolveRequest(attemptId, action),
                    SupplierIdempotencyKeys.shopClosedResolve(attemptId, action),
                )
                if (resp.isSuccessful) {
                    snackbar.showSnackbar("Resolved · $action")
                    load()
                } else {
                    snackbar.showSnackbar("Failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbar.showSnackbar(e.message ?: "Network error")
            } finally {
                busyId = null
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    SupplierReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { busyId != null },
    ) { hadInFlight ->
        if (hadInFlight) {
            busyId = null
            scope.launch { snackbar.showSnackbar(SUPPLIER_RECONNECT_RECOVERY_HINT) }
        }
        load()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Shop closed") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading shop-closed queue…", "Active attempts")
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Queue unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No active attempts",
                body = "Driver-reported shop-closed cases appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(rows, key = { it.attemptId }) { row ->
                    val busy = busyId == row.attemptId
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text(row.orderId, style = MaterialTheme.typography.titleMedium)
                            Text(stringResource(R.string.mobile_supplier_ui_driver_driverid_retailer_retailerid, row.driverId, row.retailerId), style = MaterialTheme.typography.bodySmall)
                            row.shopClosedReason?.takeIf { it.isNotBlank() }?.let {
                                Text(stringResource(R.string.mobile_supplier_ui_reason_it, it), style = MaterialTheme.typography.labelSmall)
                            }
                            row.graceEndsAt?.takeIf { it.isNotBlank() }?.let {
                                Text(stringResource(R.string.mobile_supplier_ui_grace_ends_it, it), style = MaterialTheme.typography.labelSmall)
                            }
                            row.shopClosedResolution?.takeIf { it.isNotBlank() }?.let {
                                Text(stringResource(R.string.mobile_supplier_ui_resolution_it, it), style = MaterialTheme.typography.labelSmall)
                            }
                            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                TextButton(onClick = { resolve(row.attemptId, "WAIT") }, enabled = !busy) { Text("Wait") }
                                TextButton(onClick = { resolve(row.attemptId, "BYPASS") }, enabled = !busy) { Text("Bypass") }
                                TextButton(onClick = { resolve(row.attemptId, "RETURN_TO_DEPOT") }, enabled = !busy) { Text("Return") }
                            }
                        }
                    }
                }
            }
        }
    }
}
