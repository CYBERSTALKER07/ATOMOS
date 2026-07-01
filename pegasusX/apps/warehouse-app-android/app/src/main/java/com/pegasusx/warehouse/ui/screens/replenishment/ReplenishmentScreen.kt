package com.pegasusx.warehouse.ui.screens.replenishment

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.warehouse.data.model.ReplenishmentInsight
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.components.WarehouseLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.components.WarehouseStateKind
import com.pegasusx.warehouse.ui.components.WarehouseStatePane
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.ui.realtime.WAREHOUSE_RECONNECT_RECOVERY_HINT
import com.pegasusx.warehouse.ui.realtime.WarehouseReconnectRecoveryEffect
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReplenishmentScreen(
    opsRepository: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var insights by remember { mutableStateOf<List<ReplenishmentInsight>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var actingId by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = opsRepository.getReplenishmentInsights()
                insights = if (resp.isSuccessful) resp.body()?.resolved().orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun runAction(insightId: String, action: String) {
        actingId = insightId
        scope.launch {
            try {
                val resp = opsRepository.replenishmentInsightAction(insightId, action)
                if (resp.isSuccessful) {
                    val transferID = resp.body()?.transferId
                    statusMessage = when {
                        action == "approve" && !transferID.isNullOrBlank() ->
                            "Approved — transfer ${transferID.take(8)}"
                        action == "approve" -> "Insight approved"
                        else -> "Insight dismissed"
                    }
                    load()
                } else {
                    statusMessage = "Action failed (${resp.code()})"
                }
            } catch (e: Exception) {
                statusMessage = e.message
            } finally {
                actingId = null
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    WarehouseReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { actingId != null },
    ) { hadInFlight ->
        if (hadInFlight) {
            actingId = null
            statusMessage = WAREHOUSE_RECONNECT_RECOVERY_HINT
        }
        load()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Replenishment") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") } } },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
        snackbarHost = {
            statusMessage?.let { msg ->
                LaunchedEffect(msg) {
                    kotlinx.coroutines.delay(2500)
                    statusMessage = null
                }
            }
        },
    ) { padding ->
        when {
            loading -> WarehouseLoadingState(
                title = "Loading replenishment…",
                body = "Stock insights and reorder signals",
                modifier = Modifier.padding(padding),
            )
            error != null -> WarehouseStatePane(
                kind = WarehouseStateKind.Error,
                headline = "Replenishment unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(padding),
            )
            insights.isEmpty() -> WarehouseStatePane(
                kind = WarehouseStateKind.Empty,
                headline = "No replenishment insights",
                body = "Open insights from the replenishment engine will appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    WarehouseSectionTitle("Open insights (${insights.size})")
                }
                items(insights, key = { it.id }) { insight ->
                    InsightCard(
                        insight = insight,
                        busy = actingId == insight.id,
                        onApprove = { runAction(insight.id, "approve") },
                        onDismiss = { runAction(insight.id, "dismiss") },
                    )
                }
            }
        }
    }
}

@Composable
private fun InsightCard(
    insight: ReplenishmentInsight,
    busy: Boolean,
    onApprove: () -> Unit,
    onDismiss: () -> Unit,
) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Row(modifier = Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
                    Text(insight.productName, style = MaterialTheme.typography.titleMedium)
                    if (insight.reasonCode == "PREDICTIVE_PUSH") {
                        Spacer(Modifier.width(PegasusSpacing.xs))
                        Surface(
                            color = MaterialTheme.colorScheme.onSurface,
                            shape = MaterialTheme.shapes.small
                        ) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            ) {
                                Text(
                                    "AI PUSH",
                                    style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp),
                                    color = MaterialTheme.colorScheme.surface
                                )
                            }
                        }
                    }
                }
                WarehouseStatusChip(status = insight.urgency)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                WarehouseStatusChip(status = insight.status)
            }
            Text(
                "Stock: ${insight.currentStock} · Reorder: ${insight.reorderQuantity}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            insight.demandBreakdown?.let { breakdown ->
                Text(formatDemandWhy(breakdown, insight.reasonCode), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(
                "Days until stockout: ${insight.daysUntilStockout}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (insight.status.equals("OPEN", ignoreCase = true)) {
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    Button(onClick = onApprove, enabled = !busy) { Text("Approve") }
                    OutlinedButton(onClick = onDismiss, enabled = !busy) { Text("Dismiss") }
                }
            }
        }
    }
}

private fun formatDemandWhy(breakdown: kotlinx.serialization.json.JsonObject?, reasonCode: String?): String {
    if (breakdown == null || breakdown.isEmpty()) {
        return reasonCode?.replace('_', ' ') ?: "Threshold breach"
    }
    val parts = mutableListOf<String>()
    breakdown["burn_rate_7d"]?.toString()?.trim('"')?.toDoubleOrNull()?.let { parts.add("Burn ${"%.1f".format(it)}/d") }
    breakdown["days_cover"]?.toString()?.trim('"')?.toDoubleOrNull()?.let { parts.add("${"%.1f".format(it)}d cover") }
    if (breakdown.containsKey("mei_network")) parts.add("MEIO network transfer")
    return parts.joinToString(" · ").ifBlank { reasonCode?.replace('_', ' ') ?: "Demand signal" }
}
