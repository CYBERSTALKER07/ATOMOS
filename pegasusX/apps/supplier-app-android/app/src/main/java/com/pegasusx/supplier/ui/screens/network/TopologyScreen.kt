package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TopologyScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var topology by remember { mutableStateOf<SupplierTopologyResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getTopology()
                if (resp.isSuccessful) topology = resp.body()
                else error = "Failed (${resp.code()})"
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
                title = { Text("Factories & warehouses") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        val data = topology
        when {
            loading -> SupplierLoadingState("Loading topology…", "Node topology")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Topology unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            data == null || (data.warehouses.isEmpty() && data.factories.isEmpty()) -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No nodes",
                body = "No warehouses or factories configured.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item { SectionLabel("Warehouses (${data.warehouses.size})") }
                items(data.warehouses, key = { it.warehouseId }) { node ->
                    NodeCard(node.name, node.lat, node.lng)
                }
                item { SectionLabel("Factories (${data.factories.size})") }
                items(data.factories, key = { it.factoryId }) { node ->
                    NodeCard(node.name, node.lat, node.lng)
                }
            }
        }
    }
}

@Composable
private fun SectionLabel(title: String) {
    Text(title, style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.primary)
}

@Composable
private fun NodeCard(name: String, lat: Double, lng: Double) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg)) {
            Text(name.ifEmpty { "Unnamed node" }, style = MaterialTheme.typography.titleMedium)
            Text(
                "%.4f, %.4f".format(lat, lng),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline,
            )
        }
    }
}
