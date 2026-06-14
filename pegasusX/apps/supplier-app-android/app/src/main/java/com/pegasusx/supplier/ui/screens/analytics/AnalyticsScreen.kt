package com.pegasusx.supplier.ui.screens.analytics

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.async
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AnalyticsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var pendingOrders by remember { mutableIntStateOf(0) }
    var inventorySKUs by remember { mutableIntStateOf(0) }
    var revenueTotal by remember { mutableStateOf<String?>(null) }
    var predictionCount by remember { mutableIntStateOf(0) }
    var forecastUnits by remember { mutableIntStateOf(0) }
    var velocityCreated by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale.getDefault()) }

    fun formatMinor(minor: Long, currency: String): String {
        val major = minor / 100.0
        return "${fmt.format(major)} $currency"
    }

    fun load() {
        scope.launch {
            loading = true
            error = null
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
                    error = "Failed to load analytics authority"
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
                }
                velocityResp.body()?.let { velocity ->
                    velocityCreated = velocity.points.sumOf { point -> point.ordersCreated }
                }
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
                title = { Text("Analytics") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading analytics…", "Velocity, revenue, and demand")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Analytics unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> Column(
                modifier = Modifier.padding(padding).padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text("Intelligence", style = MaterialTheme.typography.titleMedium)
                AnalyticsKpi("30-day revenue", revenueTotal ?: "—")
                AnalyticsKpi("Demand predictions", predictionCount.toString())
                AnalyticsKpi("Forecast units (24h)", forecastUnits.toString())
                AnalyticsKpi("Orders created (velocity window)", velocityCreated.toString())
                HorizontalDivider()
                Text("Operational snapshot", style = MaterialTheme.typography.titleMedium)
                AnalyticsKpi("Pending orders", pendingOrders.toString())
                AnalyticsKpi("Inventory SKUs", inventorySKUs.toString())
            }
        }
    }
}

@Composable
private fun AnalyticsKpi(label: String, value: String) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg)) {
            Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.outline)
            Text(value, style = MaterialTheme.typography.headlineMedium)
        }
    }
}
