package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WarehousesScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var warehouses by remember { mutableStateOf(emptyList<com.pegasusx.supplier.data.model.SupplierTopologyWarehouse>()) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                val resp = ops.getTopology()
                warehouses = if (resp.isSuccessful) resp.body()?.warehouses.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Warehouses") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading warehouses…", "Topology nodes")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Warehouses unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
            )
            warehouses.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No warehouses",
                body = "Warehouse nodes from topology will appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(warehouses, key = { it.warehouseId }) { warehouse ->
                    SupplierOpsListCard(
                        headline = warehouse.name.ifBlank { warehouse.warehouseId },
                        supporting = "Radius ${warehouse.coverageRadiusKm} km · ${warehouse.lat}, ${warehouse.lng}",
                        status = if (warehouse.isActive) "ACTIVE" else "INACTIVE",
                    )
                }
            }
        }
    }
}
