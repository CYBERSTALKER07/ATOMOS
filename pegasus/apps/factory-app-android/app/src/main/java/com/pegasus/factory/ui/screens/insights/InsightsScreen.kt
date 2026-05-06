package com.pegasus.factory.ui.screens.insights

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
import com.pegasus.factory.data.model.Insight
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasus.factory.ui.theme.*
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InsightsScreen(
    api: FactoryApi,
    onBack: () -> Unit,
) {
    var insights by remember { mutableStateOf<List<Insight>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getInsights()
                if (resp.isSuccessful && resp.body() != null) {
                    insights = resp.body()!!.insights
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.SupplyRequestUpdate,
            FactoryRealtimeEventType.TransferUpdate,
        ),
    ) {
        load()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Replenishment Insights") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            insights.isEmpty() -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Text("No insights", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(insights, key = { it.id }) { insight ->
                    InsightCard(insight)
                }
            }
        }
    }
}

@Composable
private fun InsightCard(insight: Insight) {
    val urgencyColor = when (insight.urgency.uppercase()) {
        "CRITICAL" -> Destructive
        "HIGH" -> Warning
        "MEDIUM" -> Neutral50
        else -> Success
    }

    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = insight.productName.ifBlank { insight.productId.take(8) },
                        style = MaterialTheme.typography.titleSmall,
                    )
                    Text(
                        text = insight.warehouseName.ifBlank { insight.warehouseId.take(8) },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                SuggestionChip(
                    onClick = {},
                    label = {
                        Text(
                            text = insight.urgency,
                            style = MaterialTheme.typography.labelSmall,
                            color = urgencyColor,
                        )
                    },
                )
            }

            Spacer(Modifier.height(PegasusSpacing.md))

            Row(
                horizontalArrangement = Arrangement.SpaceBetween,
                modifier = Modifier.fillMaxWidth(),
            ) {
                MetricItem("Stock", "${insight.currentStock}")
                MetricItem("Velocity/day", String.format("%.1f", insight.avgDailyVelocity))
                MetricItem("Days left", "${insight.daysUntilStockout}")
                MetricItem("Reorder", "${insight.reorderQuantity}")
            }
        }
    }
}

@Composable
private fun MetricItem(label: String, value: String) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(value, style = MaterialTheme.typography.titleSmall)
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
