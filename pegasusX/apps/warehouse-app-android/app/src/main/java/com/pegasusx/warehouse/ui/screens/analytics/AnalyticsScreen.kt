package com.pegasusx.warehouse.ui.screens.analytics

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.AnalyticsData
import com.pegasusx.warehouse.data.model.DailyMetric
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale
import kotlin.math.roundToLong
import com.pegasusx.warehouse.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AnalyticsScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var data by remember { mutableStateOf<AnalyticsData?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var period by remember { mutableStateOf("30d") }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val resp = api.getAnalytics(period = period)
                if (resp.isSuccessful && resp.body() != null) data = resp.body()!!
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    LaunchedEffect(period) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Analytics") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } } },
                actions = {
                    FilterChip(selected = period == "7d", onClick = { period = "7d" }, label = { Text("7d") }, modifier = Modifier.padding(end = PegasusSpacing.xs))
                    FilterChip(selected = period == "30d", onClick = { period = "30d" }, label = { Text("30d") }, modifier = Modifier.padding(end = PegasusSpacing.sm))
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            data != null -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                // KPI row
                item(span = { GridItemSpan(maxLineSpan) }) {
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                        KpiCard("Total Orders", data!!.totalOrders.toString(), Modifier.weight(1f))
                        KpiCard("Revenue", "${fmt.format(data!!.totalRevenue)} UZS", Modifier.weight(1f))
                    }
                }
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                        KpiCard("Avg Order", "${fmt.format(data!!.avgOrderValue.roundToLong())} UZS", Modifier.weight(1f))
                        KpiCard("Utilization", "${data!!.fleetUtilizationPct.roundToLong()}%", Modifier.weight(1f))
                    }
                }
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                        KpiCard("Imported Rows", data!!.importFreshness.appliedRows30d.toString(), Modifier.weight(1f))
                        KpiCard("Imported SKUs", data!!.importFreshness.appliedSkus30d.toString(), Modifier.weight(1f))
                    }
                }
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                        KpiCard("Qty Delta", data!!.importFreshness.quantityDelta30d.toString(), Modifier.weight(1f))
                        KpiCard("Anomaly Rows", data!!.importAnomalyQueue.openRows30d.toString(), Modifier.weight(1f))
                    }
                }
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                        KpiCard("Anomaly Sessions", data!!.importAnomalyQueue.affectedSessions30d.toString(), Modifier.weight(1f))
                        KpiCard("Completed", data!!.completedOrders.toString(), Modifier.weight(1f))
                    }
                }
                val freshness = data!!.importFreshness
                if (freshness.lastSessionId.isNotBlank() || freshness.lastAppliedAt.isNotBlank()) {
                    item {
                        ImportMetaCard(
                            title = stringResource(R.string.mobile_warehouse_ui_last_import),
                            sessionId = freshness.lastSessionId,
                            timestamp = freshness.lastAppliedAt,
                        )
                    }
                }
                val anomaly = data!!.importAnomalyQueue
                if (anomaly.lastSessionId.isNotBlank() || anomaly.lastDetail.isNotBlank()) {
                    item {
                        ImportMetaCard(
                            title = stringResource(R.string.mobile_warehouse_ui_latest_anomaly),
                            sessionId = anomaly.lastSessionId,
                            timestamp = anomaly.lastDetectedAt,
                            detail = anomaly.lastDetail,
                        )
                    }
                }
                if (data!!.chartDaily.isNotEmpty()) {
                    item {
                        AnalyticsChartGrid(
                            daily = data!!.chartDaily,
                            formatter = fmt,
                        )
                    }
                }
                // Top products
                item {
                    Spacer(Modifier.height(PegasusSpacing.sm))
                    Text("Top Products", style = MaterialTheme.typography.titleMedium)
                }
                items(data!!.topProducts) { tp ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Row(modifier = Modifier.padding(PegasusSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                            Text(tp.productName, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.weight(1f))
                            Text(stringResource(R.string.mobile_warehouse_ui_displayunits_units_format_uzs, tp.displayUnits, fmt.format(tp.revenue)), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
            }
        }
    }
}


@Composable
private fun ImportMetaCard(
    title: String,
    sessionId: String,
    timestamp: String,
    detail: String = "",
) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(PegasusSpacing.md), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(title, style = MaterialTheme.typography.labelLarge)
            if (sessionId.isNotBlank()) {
                Text(stringResource(R.string.mobile_warehouse_ui_session_sessionid, sessionId), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (timestamp.isNotBlank()) {
                Text(timestamp, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (detail.isNotBlank()) {
                Text(detail, style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}

@Composable
private fun KpiCard(label: String, value: String, modifier: Modifier = Modifier) {
    ElevatedCard(modifier = modifier) {
        Column(modifier = Modifier.padding(PegasusSpacing.md)) {
            Text(value, style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(2.dp))
            Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
