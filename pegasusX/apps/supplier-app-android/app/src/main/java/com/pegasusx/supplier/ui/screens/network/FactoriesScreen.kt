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
fun FactoriesScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var factories by remember { mutableStateOf(emptyList<com.pegasusx.supplier.data.model.SupplierTopologyFactory>()) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                val resp = ops.getTopology()
                factories = if (resp.isSuccessful) resp.body()?.factories.orEmpty() else emptyList()
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
                title = { Text("Factories") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading factories…", "Topology nodes")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Factories unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
            )
            factories.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No factories",
                body = "Factory nodes from topology will appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(factories, key = { it.factoryId }) { factory ->
                    SupplierOpsListCard(
                        headline = factory.name.ifBlank { factory.factoryId },
                        supporting = "${factory.lat}, ${factory.lng}",
                        status = if (factory.isActive) "ACTIVE" else "INACTIVE",
                    )
                }
            }
        }
    }
}
