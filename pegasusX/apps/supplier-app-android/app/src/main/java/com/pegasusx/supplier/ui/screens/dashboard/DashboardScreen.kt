package com.pegasusx.supplier.ui.screens.dashboard

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.BuildConfig
import com.pegasusx.supplier.data.model.SupplierDashboard
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.ClientPolicyBanner
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private data class DashboardKpi(
    val label: String,
    val value: (SupplierDashboard) -> String,
    val icon: ImageVector,
)

private val dashboardKpis = listOf(
    DashboardKpi("Pending orders", { "${it.pendingOrders}" }, Icons.Default.LocalShipping),
    DashboardKpi("Inventory SKUs", { "${it.inventorySKUs}" }, Icons.Default.Archive),
    DashboardKpi("Configured", { if (it.isConfigured) "Yes" else "No" }, Icons.Default.CheckCircle),
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    api: SupplierApi,
    ops: SupplierOperationsRepository,
    showBillingBanner: Boolean,
    onOpenBilling: () -> Unit,
    onOpenNotifications: () -> Unit = {},
) {
    var dashboard by remember { mutableStateOf<SupplierDashboard?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var clientPolicyMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun loadClientPolicy() {
        scope.launch {
            try {
                val resp = api.getClientPolicy(
                    platform = "android",
                    version = BuildConfig.VERSION_NAME,
                )
                if (resp.isSuccessful && resp.body() != null) {
                    val policy = resp.body()!!
                    if (policy.outdated || policy.forceUpdate) {
                        clientPolicyMessage = buildString {
                            append(if (policy.forceUpdate) "Update required" else "Update available")
                            if (policy.minimumVersion.isNotBlank()) {
                                append(" — minimum version ${policy.minimumVersion}")
                            }
                            policy.deferReason?.takeIf { it.isNotBlank() }?.let { append(". $it") }
                        }
                    }
                }
            } catch (_: Exception) {
                // Policy fetch is optional on local/dev stacks.
            }
        }
    }

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

    LaunchedEffect(Unit) {
        load()
        loadClientPolicy()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Dashboard", fontWeight = FontWeight.Bold) },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                    IconButton(onClick = onOpenNotifications) {
                        Icon(Icons.Default.Notifications, contentDescription = "Notifications")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState(
                title = "Loading dashboard…",
                body = "Fetching supplier KPIs",
                modifier = Modifier.padding(padding),
            )
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
                        .padding(horizontal = PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    ClientPolicyBanner(clientPolicyMessage)
                    if (showBillingBanner || !d.isConfigured) {
                        ElevatedCard(
                            modifier = Modifier.fillMaxWidth(),
                            onClick = onOpenBilling,
                        ) {
                            Row(
                                modifier = Modifier.padding(PegasusSpacing.lg),
                                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                            ) {
                                Surface(
                                    shape = MaterialTheme.shapes.small,
                                    color = MaterialTheme.colorScheme.tertiaryContainer,
                                ) {
                                    Icon(
                                        Icons.Default.CreditCard,
                                        contentDescription = null,
                                        tint = MaterialTheme.colorScheme.onTertiaryContainer,
                                        modifier = Modifier
                                            .padding(PegasusSpacing.sm)
                                            .size(24.dp),
                                    )
                                }
                                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                    Text("Complete billing setup", style = MaterialTheme.typography.titleMedium)
                                    Text(
                                        "Configure bank and payment gateway to finish onboarding.",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                            }
                        }
                    }
                    LazyVerticalGrid(
                        columns = GridCells.Adaptive(minSize = 160.dp),
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        modifier = Modifier.weight(1f, fill = false),
                    ) {
                        items(dashboardKpis, key = { it.label }) { kpi ->
                            SupplierKpiTile(
                                label = kpi.label,
                                value = kpi.value(d),
                                icon = kpi.icon,
                            )
                        }
                    }
                    Text(
                        "Updated ${d.updatedAt}",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(bottom = PegasusSpacing.lg),
                    )
                }
            }
        }
    }
}
