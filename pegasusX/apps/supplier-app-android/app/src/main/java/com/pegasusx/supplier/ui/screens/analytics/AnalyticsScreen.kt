package com.pegasusx.supplier.ui.screens.analytics

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.AttachMoney
import androidx.compose.material.icons.filled.Inventory2
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.ShowChart
import androidx.compose.material.icons.filled.TrendingUp
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pegasus.design.RealtimeRefreshEffect
import com.pegasus.design.showFullScreenLoading
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.screens.planning.ForecastConfidenceView
import com.pegasusx.supplier.util.formatForecastUpdatedAt
import com.pegasusx.supplier.util.forecastConfidenceFromDemand
import com.pegasusx.supplier.util.isForecastStale
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.async
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

private data class AnalyticsKpi(
    val label: String,
    val icon: ImageVector,
    val value: () -> String,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AnalyticsScreen(
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onBack: () -> Unit,
    onOpenPlanningBrain: () -> Unit = {},
    onOpenKnowledgeGraph: () -> Unit = {},
    onOpenPlanningSettings: () -> Unit = {},
) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var hasSnapshot by remember { mutableStateOf(false) }
    var pendingOrders by remember { mutableIntStateOf(0) }
    var inventorySKUs by remember { mutableIntStateOf(0) }
    var revenueTotal by remember { mutableStateOf<String?>(null) }
    var predictionCount by remember { mutableIntStateOf(0) }
    var forecastUnits by remember { mutableIntStateOf(0) }
    var velocityCreated by remember { mutableIntStateOf(0) }
    var demandGeneratedAt by remember { mutableStateOf<String?>(null) }
    var demandConfidence by remember { mutableStateOf<com.pegasusx.supplier.data.model.ForecastConfidence?>(null) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale.getDefault()) }

    fun formatMinor(minor: Long, currency: String): String {
        val major = minor / 100.0
        return "${fmt.format(major)} $currency"
    }

    fun load(silent: Boolean = false) {
        scope.launch {
            if (!silent) {
                loading = true
                error = null
            }
            try {
                val dashDeferred = async { ops.getDashboard() }
                val revenueDeferred = async { ops.getAnalyticsRevenue() }
                val demandDeferred = async { ops.getDemandToday() }
                val velocityDeferred = async { ops.getAnalyticsVelocity() }

                val dashResp = dashDeferred.await()
                val revenueResp = revenueDeferred.await()
                val demandResp = demandDeferred.await()
                val velocityResp = velocityDeferred.await()

                if (!dashResp.isSuccessful || !revenueResp.isSuccessful || !demandResp.isSuccessful || !velocityResp.isSuccessful) {
                    if (!silent) error = "Failed to load analytics authority"
                    return@launch
                }

                dashResp.body()?.let {
                    pendingOrders = it.pendingOrders
                    inventorySKUs = it.inventorySKUs
                }
                revenueResp.body()?.let {
                    revenueTotal = formatMinor(it.totalMinor, it.currency)
                }
                demandResp.body()?.let {
                    predictionCount = it.predictionCount
                    forecastUnits = it.totalPallets
                    demandGeneratedAt = it.generatedAt
                    demandConfidence = forecastConfidenceFromDemand(it)
                }
                velocityResp.body()?.let { velocity ->
                    velocityCreated = velocity.points.sumOf { point -> point.ordersCreated }
                }
                hasSnapshot = true
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    RealtimeRefreshEffect(
        refreshTick = realtimeSignals.refreshTick,
        reconnectTick = realtimeSignals.reconnectTick,
        onRefresh = { load(silent = it) },
    )

    val intelligenceKpis = remember(revenueTotal, predictionCount, forecastUnits, velocityCreated) {
        listOf(
            AnalyticsKpi("30-day revenue", Icons.Default.AttachMoney) { revenueTotal ?: "—" },
            AnalyticsKpi("Demand predictions", Icons.Default.TrendingUp) { predictionCount.toString() },
            AnalyticsKpi("Forecast units (24h)", Icons.Default.ShowChart) { forecastUnits.toString() },
            AnalyticsKpi("Orders created (velocity)", Icons.Default.LocalShipping) { velocityCreated.toString() },
        )
    }
    val operationalKpis = remember(pendingOrders, inventorySKUs) {
        listOf(
            AnalyticsKpi("Pending orders", Icons.Default.LocalShipping) { pendingOrders.toString() },
            AnalyticsKpi("Inventory SKUs", Icons.Default.Inventory2) { inventorySKUs.toString() },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Analytics") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        when {
            showFullScreenLoading(loading, hasSnapshot) -> SupplierLoadingState(
                title = "Loading analytics…",
                body = "Velocity, revenue, and demand",
                modifier = Modifier.padding(padding),
            )
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Analytics unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 160.dp),
                modifier = Modifier
                    .padding(padding)
                    .padding(horizontal = PegasusSpacing.lg)
                    .fillMaxSize(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                contentPadding = PaddingValues(bottom = PegasusSpacing.lg),
            ) {
                item(span = { GridItemSpan(maxLineSpan) }) {
                    SupplierSectionTitle("Intelligence")
                }
                items(intelligenceKpis, key = { it.label }) { kpi ->
                    SupplierKpiTile(label = kpi.label, value = kpi.value(), icon = kpi.icon)
                }
                item(span = { GridItemSpan(maxLineSpan) }) {
                    Spacer(Modifier.height(PegasusSpacing.xs))
                    SupplierSectionTitle("Operational snapshot")
                }
                items(operationalKpis, key = { it.label }) { kpi ->
                    SupplierKpiTile(label = kpi.label, value = kpi.value(), icon = kpi.icon)
                }
                item(span = { GridItemSpan(maxLineSpan) }) {
                    PlanningBrainSection(ops)
                }
                demandConfidence?.let { confidence ->
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        ForecastConfidenceView(
                            confidence = confidence,
                            updatedAt = formatForecastUpdatedAt(demandGeneratedAt),
                            stale = isForecastStale(demandGeneratedAt),
                        )
                    }
                }
                item(span = { GridItemSpan(maxLineSpan) }) {
                    SupplierSectionTitle("Planning tools")
                }
                item(span = { GridItemSpan(maxLineSpan) }) {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        OutlinedButton(onClick = onOpenPlanningBrain, modifier = Modifier.fillMaxWidth()) {
                            Text("Planning sandbox")
                        }
                        OutlinedButton(onClick = onOpenKnowledgeGraph, modifier = Modifier.fillMaxWidth()) {
                            Text("Knowledge graph")
                        }
                        OutlinedButton(onClick = onOpenPlanningSettings, modifier = Modifier.fillMaxWidth()) {
                            Text("Planning settings")
                        }
                    }
                }
            }
        }
    }
}
