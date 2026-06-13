package com.pegasusx.warehouse.ui.screens.replenishment

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.ReplenishmentInsight
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReplenishmentScreen(
    opsRepository: WarehouseOperationsRepository,
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
                    statusMessage = if (action == "approve") "Insight approved" else "Insight dismissed"
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

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Replenishment") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") } } },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
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
            loading -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { CircularProgressIndicator() }

            error != null -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.md))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }

            insights.isEmpty() -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { Text("No replenishment insights", color = MaterialTheme.colorScheme.onSurfaceVariant) }

            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
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
            Text(insight.productName, style = MaterialTheme.typography.titleMedium)
            Text("${insight.urgency} · ${insight.status}", style = MaterialTheme.typography.bodySmall)
            Text("Stock: ${insight.currentStock} · Reorder: ${insight.reorderQuantity}")
            Text("Days until stockout: ${insight.daysUntilStockout}")
            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                Button(onClick = onApprove, enabled = !busy) { Text("Approve") }
                OutlinedButton(onClick = onDismiss, enabled = !busy) { Text("Dismiss") }
            }
        }
    }
}
