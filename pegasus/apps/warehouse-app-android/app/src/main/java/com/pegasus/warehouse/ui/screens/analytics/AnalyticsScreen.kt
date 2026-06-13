package com.pegasus.warehouse.ui.screens.analytics

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.warehouse.data.model.AnalyticsData
import com.pegasus.warehouse.data.model.ImportAnomalyQueue
import com.pegasus.warehouse.data.model.ImportFreshness
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale
import kotlin.math.roundToLong

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AnalyticsScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var data by remember { mutableStateOf<AnalyticsData?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var period by remember { mutableStateOf("7d") }
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
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
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
            data != null -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                // KPI row
                item {
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
                        KpiCard("Anomaly Rows", data!!.importAnomalyQueue.openRows30d.toString(), Modifier.weight(1f))
                    }
                }
                item {
                    ImportFreshnessCard(data!!.importFreshness, fmt)
                }
                item {
                    ImportAnomalyCard(data!!.importAnomalyQueue, fmt)
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
                            Text("${tp.displayUnits} units · ${fmt.format(tp.revenue)} UZS", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ImportFreshnessCard(freshness: ImportFreshness, fmt: NumberFormat) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
            Text("Import Freshness", style = MaterialTheme.typography.titleSmall)
            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                DetailMetric("SKUs Updated (30d)", fmt.format(freshness.appliedSkus30d), Modifier.weight(1f))
                DetailMetric("Qty Delta (30d)", fmt.format(freshness.quantityDelta30d), Modifier.weight(1f))
            }
            val lastApplied = freshness.lastAppliedAt.ifBlank { "No imports applied yet" }
            val session = freshness.lastSessionId.ifBlank { "N/A" }
            Text("Last session: $session • $lastApplied", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun ImportAnomalyCard(queue: ImportAnomalyQueue, fmt: NumberFormat) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
            Text("Import Anomaly Queue", style = MaterialTheme.typography.titleSmall)
            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                DetailMetric("Open Rows (30d)", fmt.format(queue.openRows30d), Modifier.weight(1f))
                DetailMetric("Affected Sessions", fmt.format(queue.affectedSessions30d), Modifier.weight(1f))
            }
            val lastDetected = queue.lastDetectedAt.ifBlank { "No anomalies detected" }
            val session = queue.lastSessionId.ifBlank { "N/A" }
            Text("Last session: $session • $lastDetected", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (queue.lastDetail.isNotBlank()) {
                Text(queue.lastDetail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

@Composable
private fun DetailMetric(label: String, value: String, modifier: Modifier = Modifier) {
    Column(modifier = modifier) {
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.titleMedium)
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
