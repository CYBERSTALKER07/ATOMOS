package com.thelab.warehouse.ui.screens.analytics

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
import com.thelab.warehouse.data.model.AnalyticsData
import com.thelab.warehouse.data.remote.WarehouseApi
import com.thelab.warehouse.ui.theme.LabSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

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
                    FilterChip(selected = period == "7d", onClick = { period = "7d" }, label = { Text("7d") }, modifier = Modifier.padding(end = LabSpacing.xs))
                    FilterChip(selected = period == "30d", onClick = { period = "30d" }, label = { Text("30d") }, modifier = Modifier.padding(end = LabSpacing.sm))
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
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            data != null -> LazyColumn(
                contentPadding = PaddingValues(LabSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                // KPI row
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(LabSpacing.md), modifier = Modifier.fillMaxWidth()) {
                        KpiCard("Total Orders", data!!.totalOrders.toString(), Modifier.weight(1f))
                        KpiCard("Revenue", "${fmt.format(data!!.totalRevenue)} UZS", Modifier.weight(1f))
                    }
                }
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(LabSpacing.md), modifier = Modifier.fillMaxWidth()) {
                        KpiCard("Avg Delivery", "${data!!.avgDeliveryMinutes} min", Modifier.weight(1f))
                        KpiCard("Completion", "${data!!.completionRate}%", Modifier.weight(1f))
                    }
                }
                // Top products
                item {
                    Spacer(Modifier.height(LabSpacing.sm))
                    Text("Top Products", style = MaterialTheme.typography.titleMedium)
                }
                items(data!!.topProducts) { tp ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                            Text(tp.productName, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.weight(1f))
                            Text("${tp.unitsSold} units · ${fmt.format(tp.revenue)} UZS", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun KpiCard(label: String, value: String, modifier: Modifier = Modifier) {
    ElevatedCard(modifier = modifier) {
        Column(modifier = Modifier.padding(LabSpacing.md)) {
            Text(value, style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(2.dp))
            Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
