package com.pegasusx.factory.ui.screens.insights

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.factory.data.model.Insight
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.*
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

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
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
                if (!silent) {
                    loading = false
                }
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
        load(silent = true)
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
            loading && insights.isEmpty() -> PegasusLoadingState(
                title = "Loading insights",
                body = "Fetching replenishment pressure and restock signals for this factory.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load insights",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            insights.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No replenishment insights",
                body = "Insights will appear here when stock velocity produces factory-level alerts.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)
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

            insight.demandBreakdown?.let { breakdown ->
                Text(
                    text = formatDemandWhy(breakdown, insight.reasonCode),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

private fun formatDemandWhy(breakdown: kotlinx.serialization.json.JsonObject?, reasonCode: String?): String {
    if (breakdown == null || breakdown.isEmpty()) {
        return reasonCode?.replace('_', ' ') ?: "Threshold breach"
    }
    breakdown["blocked_reason"]?.toString()?.trim('"')?.takeIf { it.isNotBlank() }?.let { blocked ->
        return if (blocked == "insufficient_history") {
            "Insufficient history — forecast blocked"
        } else {
            blocked.replace('_', ' ')
        }
    }
    val parts = mutableListOf<String>()
    breakdown["burn_rate_7d"]?.toString()?.trim('"')?.toDoubleOrNull()?.let { parts.add("Burn ${"%.1f".format(it)}/d") }
    breakdown["days_cover"]?.toString()?.trim('"')?.toDoubleOrNull()?.let { parts.add("${"%.1f".format(it)}d cover") }
    breakdown["confidence"]?.toString()?.trim('"')?.toDoubleOrNull()?.let {
        parts.add("${(it * 100).toInt()}% conf")
    }
    if (breakdown.containsKey("mei_network")) parts.add("MEIO network transfer")
    return parts.joinToString(" · ").ifBlank { reasonCode?.replace('_', ' ') ?: "Demand signal" }
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
