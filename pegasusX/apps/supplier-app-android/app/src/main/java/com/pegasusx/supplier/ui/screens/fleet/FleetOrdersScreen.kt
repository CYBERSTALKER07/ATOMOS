package com.pegasusx.supplier.ui.screens.fleet

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierFleetOrderRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FleetOrdersScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var rows by remember { mutableStateOf<List<SupplierFleetOrderRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getFleetOrders()
                rows = if (resp.isSuccessful) resp.body().orEmpty() else emptyList()
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
                title = { Text("Fleet orders") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState(
                title = "Loading fleet orders…",
                body = "In-flight assignments",
                modifier = Modifier.padding(padding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Fleet orders unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No fleet orders",
                body = "Active route assignments appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(rows, key = { it.orderId }) { row ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg)) {
                            Text(row.orderId, style = MaterialTheme.typography.titleMedium)
                            Text("${row.status} · Driver ${row.driverId ?: "—"}", style = MaterialTheme.typography.bodyMedium)
                            row.routeId?.let { Text("Route $it", style = MaterialTheme.typography.bodySmall) }
                        }
                    }
                }
            }
        }
    }
}
