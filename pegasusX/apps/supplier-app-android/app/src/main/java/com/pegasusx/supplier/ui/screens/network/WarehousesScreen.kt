package com.pegasusx.supplier.ui.screens.network

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WarehousesScreen(
    ops: SupplierOperationsRepository,
    geocodeApi: GeocodeApi,
    onBack: () -> Unit,
) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var warehouses by remember { mutableStateOf(emptyList<com.pegasusx.supplier.data.model.SupplierTopologyWarehouse>()) }
    var showAdd by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
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

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Warehouses") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    IconButton(onClick = { showAdd = true }) {
                        Icon(Icons.Default.Add, contentDescription = stringResource(R.string.supplier_portal_warehouses_components_warehouse_form_text_add_warehouse))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading warehouses…", "Topology nodes", Modifier.padding(padding))
            error != null && warehouses.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Warehouses unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            warehouses.isEmpty() -> Box(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize(),
                contentAlignment = Alignment.Center,
            ) {
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No warehouses",
                    body = "Add your first distribution node to start fulfilling orders.",
                    actionLabel = "Add first warehouse",
                    onAction = { showAdd = true },
                )
            }
            else -> Box(modifier = Modifier.padding(padding)) {
                WarehouseList(warehouses = warehouses)
            }
        }
    }

    if (showAdd) {
        AddWarehouseDialog(
            geocodeApi = geocodeApi,
            onDismiss = { showAdd = false },
            onSave = { name, location, radius ->
                scope.launch {
                    appendWarehouseNode(ops, name, location, radius)
                        .onSuccess {
                            showAdd = false
                            load()
                        }
                        .onFailure { error = it.message }
                }
            },
        )
    }
}
