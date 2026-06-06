package com.pegasusx.supplier.ui.screens.dashboard

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierDashboard
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    api: SupplierApi,
    ops: SupplierOperationsRepository,
    showBillingBanner: Boolean,
    onOpenBilling: () -> Unit,
) {
    var dashboard by remember { mutableStateOf<SupplierDashboard?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = api.getDashboard()
                if (resp.isSuccessful) {
                    dashboard = resp.body()
                    runCatching {
                        ops.getActivity()
                        ops.getExceptions()
                    }
                } else {
                    error = "Failed to load (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Dashboard") }) },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading dashboard…", "Fetching supplier KPIs")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Dashboard unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(padding),
            )
            dashboard != null -> {
                val d = dashboard!!
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding)
                        .padding(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.lg),
                ) {
                    if (showBillingBanner || !d.isConfigured) {
                        ElevatedCard(
                            modifier = Modifier.fillMaxWidth(),
                            onClick = onOpenBilling,
                        ) {
                            Column(Modifier.padding(PegasusSpacing.lg)) {
                                Text("Complete billing setup", style = MaterialTheme.typography.titleMedium)
                                Text(
                                    "Configure bank and payment gateway to finish onboarding.",
                                    style = MaterialTheme.typography.bodySmall,
                                )
                            }
                        }
                    }
                    LazyVerticalGrid(
                        columns = GridCells.Adaptive(minSize = 150.dp),
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    ) {
                        items(
                            listOf(
                                Triple("Pending orders", "${d.pendingOrders}", Icons.Default.LocalShipping),
                                Triple("Inventory SKUs", "${d.inventorySKUs}", Icons.Default.Archive),
                                Triple("Configured", if (d.isConfigured) "Yes" else "No", Icons.Default.CheckCircle),
                            ),
                        ) { (title, value, icon) ->
                            ElevatedCard(Modifier.fillMaxWidth()) {
                                Column(Modifier.padding(PegasusSpacing.lg)) {
                                    Icon(icon, contentDescription = null)
                                    Spacer(Modifier.height(PegasusSpacing.sm))
                                    Text(value, style = MaterialTheme.typography.headlineMedium)
                                    Text(title, style = MaterialTheme.typography.labelLarge)
                                }
                            }
                        }
                    }
                    Text(
                        "Updated ${d.updatedAt}",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}
