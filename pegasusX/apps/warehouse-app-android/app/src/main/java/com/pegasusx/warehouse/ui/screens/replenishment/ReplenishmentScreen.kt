package com.pegasusx.warehouse.ui.screens.replenishment

import androidx.compose.foundation.layout.*

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
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
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
            loading && insights.isEmpty() -> PegasusLoadingState(
                title = "Loading replenishment…",
                body = "Stock insights and reorder signals",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            error != null && insights.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Replenishment unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            insights.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No replenishment insights",
                body = "Open insights from the replenishment engine will appear here.",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            else -> ReplenishmentList(
                insights = insights,
                actingId = actingId,
                onApprove = { runAction(it, "approve") },
                onDismiss = { runAction(it, "dismiss") },
                modifier = Modifier.padding(padding)
            )
        }
    }
}

