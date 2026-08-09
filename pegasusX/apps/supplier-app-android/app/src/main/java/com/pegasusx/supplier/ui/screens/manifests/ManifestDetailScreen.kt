package com.pegasusx.supplier.ui.screens.manifests

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.ShoppingCart
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierManifestDetail
import com.pegasusx.supplier.data.model.SupplierManifestInjectOrderRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.util.SUPPLIER_RECONNECT_RECOVERY_HINT
import com.pegasusx.supplier.ui.realtime.SupplierReconnectRecoveryEffect
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.components.SupplierStatusChip
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.util.UUID
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ManifestDetailScreen(
    manifestId: String,
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onBack: () -> Unit,
) {
    var detail by remember { mutableStateOf<SupplierManifestDetail?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var injectOrderId by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getManifestDetail(manifestId)
                if (resp.isSuccessful) detail = resp.body()
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(manifestId) { load() }

    SupplierReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { busy },
    ) { hadInFlight ->
        if (hadInFlight) {
            busy = false
            actionError = SUPPLIER_RECONNECT_RECOVERY_HINT
        }
    }

    fun idempotency(prefix: String, extra: String = ""): String =
        "$prefix:$manifestId:${extra.ifBlank { UUID.randomUUID().toString() }}"

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Manifest") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = stringResource(R.string.portal_page_orders_action_refresh))
                    }
                },
            )
        },
    ) { padding ->
        val data = detail
        when {
            loading -> PegasusLoadingState(
                title = stringResource(R.string.mobile_supplier_ui_loading_manifest),
                body = manifestId,
                modifier = Modifier.padding(padding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Manifest unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            data == null -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "Not found",
                body = "Manifest could not be loaded.",
                modifier = Modifier.padding(padding),
            )
            else -> {
                val state = (data.state.ifBlank { data.status }).uppercase()
                LazyColumn(
                    modifier = Modifier.padding(padding).fillMaxSize(),
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    item {
                        ElevatedCard(Modifier.fillMaxWidth()) {
                            Column(
                                Modifier.padding(PegasusSpacing.lg),
                                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                            ) {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Text(data.manifestId.take(12), style = MaterialTheme.typography.titleMedium)
                                    SupplierStatusChip(status = state)
                                }
                                Text(
                                    data.driverName.ifBlank { data.driverId ?: "No driver assigned" },
                                    style = MaterialTheme.typography.bodyMedium,
                                )
                                data.vehiclePlate?.let {
                                    Text(stringResource(R.string.mobile_supplier_ui_vehicle_it, it), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                }
                            }
                        }
                    }
                    item {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        ) {
                            SupplierKpiTile(
                                label = stringResource(R.string.portal_nav_orders),
                                value = data.ordersCount.toString(),
                                icon = Icons.Default.ShoppingCart,
                                modifier = Modifier.weight(1f),
                            )
                            val volume = data.totalVolumeVu.takeIf { it > 0.0 } ?: data.totalVu.toDouble()
                            if (volume > 0.0 || data.maxVolumeVu > 0.0) {
                                SupplierKpiTile(
                                    label = stringResource(R.string.supplier_portal_promotions_text_volume),
                                    value = if (data.maxVolumeVu > 0.0) {
                                        "%.1f / %.1f VU".format(volume, data.maxVolumeVu)
                                    } else {
                                        "%.1f VU".format(volume)
                                    },
                                    icon = Icons.Default.LocalShipping,
                                    modifier = Modifier.weight(1f),
                                )
                            }
                        }
                    }
                    if (actionError != null) {
                        item { Text(actionError!!, color = MaterialTheme.colorScheme.error) }
                    }
                    if (state == "DRAFT") {
                        item {
                            Button(
                                enabled = !busy,
                                onClick = {
                                    scope.launch {
                                        busy = true
                                        actionError = null
                                        try {
                                            val resp = ops.startManifestLoading(manifestId, idempotency("start-loading"))
                                            if (!resp.isSuccessful) actionError = "Start failed (${resp.code()})"
                                            else load()
                                        } catch (e: Exception) {
                                            actionError = e.message
                                        } finally {
                                            busy = false
                                        }
                                    }
                                },
                            ) { Text(if (busy) "Starting…" else "Start loading") }
                        }
                    }
                    if (state == "LOADING") {
                        item {
                            OutlinedTextField(
                                value = injectOrderId,
                                onValueChange = { injectOrderId = it },
                                label = { Text("Order ID to inject") },
                                modifier = Modifier.fillMaxWidth(),
                            )
                        }
                        item {
                            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                                Button(
                                    enabled = !busy && injectOrderId.isNotBlank(),
                                    onClick = {
                                        scope.launch {
                                            busy = true
                                            actionError = null
                                            try {
                                                val orderId = injectOrderId.trim()
                                                val resp = ops.injectManifestOrder(
                                                    manifestId,
                                                    SupplierManifestInjectOrderRequest(orderId = orderId),
                                                    idempotency("inject-order", orderId),
                                                )
                                                if (!resp.isSuccessful) actionError = "Inject failed (${resp.code()})"
                                                else {
                                                    injectOrderId = ""
                                                    load()
                                                }
                                            } catch (e: Exception) {
                                                actionError = e.message
                                            } finally {
                                                busy = false
                                            }
                                        }
                                    },
                                ) { Text("Inject order") }
                                Button(
                                    enabled = !busy,
                                    onClick = {
                                        scope.launch {
                                            busy = true
                                            actionError = null
                                            try {
                                                val resp = ops.sealManifest(manifestId, idempotency("seal"))
                                                if (!resp.isSuccessful) actionError = "Seal failed (${resp.code()})"
                                                else load()
                                            } catch (e: Exception) {
                                                actionError = e.message
                                            } finally {
                                                busy = false
                                            }
                                        }
                                    },
                                ) { Text(if (busy) "Sealing…" else "Seal manifest") }
                            }
                        }
                    }
                    if (data.orders.isNotEmpty()) {
                        item { SupplierSectionTitle("Orders") }
                        items(data.orders, key = { it.orderId }) { order ->
                            SupplierOpsListCard(
                                headline = order.orderId.take(12),
                                supporting = order.retailerId?.let { "Retailer $it" } ?: "Manifest order",
                                status = order.status.ifBlank { order.state },
                            )
                        }
                    }
                }
            }
        }
    }
}
