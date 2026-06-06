package com.pegasusx.supplier.ui.screens.dispatch

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierDispatchPreview
import com.pegasusx.supplier.data.model.SupplierTopologyWarehouse
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchPreviewScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var preview by remember { mutableStateOf<SupplierDispatchPreview?>(null) }
    var warehouses by remember { mutableStateOf<List<SupplierTopologyWarehouse>>(emptyList()) }
    var selectedWarehouseId by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val topologyResp = ops.getTopology()
                if (topologyResp.isSuccessful) {
                    warehouses = topologyResp.body()?.warehouses.orEmpty()
                }
                val resp = ops.getDispatchPreview(selectedWarehouseId)
                preview = if (resp.isSuccessful) resp.body() else null
                if (!resp.isSuccessful) error = "Preview failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(selectedWarehouseId) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Dispatch preview") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading && preview == null -> SupplierLoadingState("Loading dispatch preview…", "Auto-dispatch snapshot")
            error != null && preview == null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Preview unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                if (warehouses.isNotEmpty()) {
                    Text("Warehouse scope", style = MaterialTheme.typography.titleSmall)
                    warehouses.forEach { wh ->
                        FilterChip(
                            selected = selectedWarehouseId == wh.warehouseId,
                            onClick = {
                                selectedWarehouseId = if (selectedWarehouseId == wh.warehouseId) null else wh.warehouseId
                            },
                            label = { Text(wh.name.ifBlank { wh.warehouseId }) },
                        )
                    }
                }
                preview?.let { p ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text("Pending orders", style = MaterialTheme.typography.titleMedium)
                            Text("${p.pendingCount}", style = MaterialTheme.typography.headlineMedium)
                            Text("Available drivers: ${p.availableDriverCount}", style = MaterialTheme.typography.bodyMedium)
                            Text(
                                "Undispatched bucket: ${p.undispatchedOrders.size}",
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
                OutlinedButton(onClick = { load() }, enabled = !loading) { Text("Refresh preview") }
            }
        }
    }
}
