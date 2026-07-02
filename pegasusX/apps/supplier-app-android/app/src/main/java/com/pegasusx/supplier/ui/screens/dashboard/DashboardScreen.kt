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
import com.pegasus.design.RealtimeRefreshEffect
import com.pegasus.design.showFullScreenLoading
import com.pegasusx.supplier.data.model.SupplierDashboard
import com.pegasusx.supplier.data.model.SupplierMEIONetworkSummary
import com.pegasusx.supplier.data.model.ForecastConfidence
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierPulseStrip
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.screens.planning.ForecastConfidenceView
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.forecastConfidenceFromDemand
import com.pegasusx.supplier.util.formatForecastUpdatedAt
import com.pegasusx.supplier.util.isForecastStale
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
    realtimeSignals: SupplierRealtimeSignals,
    showBillingBanner: Boolean,
    onOpenBilling: () -> Unit,
    onOpenNotifications: () -> Unit = {},
) {
    var dashboard by remember { mutableStateOf<SupplierDashboard?>(null) }
    var meiSummary by remember { mutableStateOf<SupplierMEIONetworkSummary?>(null) }
    var pulseEvents by remember { mutableStateOf<List<com.pegasusx.supplier.data.model.PulseEvent>>(emptyList()) }
    var pulseLoading by remember { mutableStateOf(true) }
    var demandConfidence by remember { mutableStateOf<ForecastConfidence?>(null) }
    var demandGeneratedAt by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        scope.launch {
            if (!silent) {
                loading = true
                error = null
            }
            try {
                val resp = api.getDashboard()
                if (resp.isSuccessful) {
                    dashboard = resp.body()
                    runCatching {
                        ops.getActivity()
                        ops.getExceptions()
                        ops.getMEIONetworkSummary().body()?.let { meiSummary = it }
                        ops.getDemandToday().body()?.let { demand ->
                            demandGeneratedAt = demand.generatedAt
                            demandConfidence = forecastConfidenceFromDemand(demand)
                        }
                    }
                } else if (!silent) {
                    error = "Failed to load (${resp.code()})"
                }
            } catch (e: Exception) {
                if (!silent) error = e.message ?: "Network error"
            } finally {
                if (!silent) loading = false
            }
            pulseLoading = true
            runCatching {
                ops.getPulse().body()?.let { pulseEvents = it.events }
            }.onFailure {
                pulseEvents = emptyList()
            }
            pulseLoading = false
        }
    }

    LaunchedEffect(Unit) {
        load()
    }

    RealtimeRefreshEffect(
        refreshTick = realtimeSignals.refreshTick,
        reconnectTick = realtimeSignals.reconnectTick,
        onRefresh = { load(silent = it) },
    )

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
            showFullScreenLoading(loading, dashboard != null) -> SupplierLoadingState(
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
                    meiSummary?.let { mei ->
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                Text("MEIO network", style = MaterialTheme.typography.titleMedium)
                                Text(
                                    "${mei.warehousesScanned} warehouses · ${mei.transferRecommendations} transfer recs · ${mei.insightsGenerated} insights",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                    }
                    demandConfidence?.let { confidence ->
                        ForecastConfidenceView(
                            confidence = confidence,
                            updatedAt = formatForecastUpdatedAt(demandGeneratedAt),
                            stale = isForecastStale(demandGeneratedAt),
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                    SupplierPulseStrip(
                        events = pulseEvents,
                        loading = pulseLoading,
                        modifier = Modifier.fillMaxWidth(),
                    )
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
