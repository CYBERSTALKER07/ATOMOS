package com.pegasusx.supplier.ui.screens.manifests

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierManifestDetail
import com.pegasusx.supplier.data.model.SupplierManifestInjectOrderRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.util.UUID

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ManifestDetailScreen(
    manifestId: String,
    ops: SupplierOperationsRepository,
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

    fun idempotency(prefix: String, extra: String = ""): String =
        "$prefix:$manifestId:${extra.ifBlank { UUID.randomUUID().toString() }}"

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Manifest") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        val data = detail
        when {
            loading -> SupplierLoadingState("Loading manifest…", manifestId)
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Manifest unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            data == null -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
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
                            Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                                Text(data.manifestId, style = MaterialTheme.typography.titleMedium)
                                Text("${data.status} · ${data.ordersCount} orders", style = MaterialTheme.typography.bodyMedium)
                                Text(data.driverName.ifBlank { data.driverId ?: "—" }, style = MaterialTheme.typography.bodySmall)
                                data.vehiclePlate?.let { Text("Vehicle $it", style = MaterialTheme.typography.bodySmall) }
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
                        item { Text("Orders", style = MaterialTheme.typography.titleSmall) }
                        items(data.orders, key = { it.orderId }) { order ->
                            ElevatedCard(Modifier.fillMaxWidth()) {
                                Column(Modifier.padding(PegasusSpacing.md)) {
                                    Text(order.orderId, style = MaterialTheme.typography.bodyMedium)
                                    Text(order.status.ifBlank { order.state }, style = MaterialTheme.typography.bodySmall)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
