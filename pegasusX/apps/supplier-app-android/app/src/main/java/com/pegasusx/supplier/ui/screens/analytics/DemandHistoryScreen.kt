package com.pegasusx.supplier.ui.screens.analytics

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.DemandHistoryPoint
import com.pegasusx.supplier.data.model.DemandUpcomingRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DemandHistoryScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var timeSeries by remember { mutableStateOf<List<DemandHistoryPoint>>(emptyList()) }
    var upcoming by remember { mutableStateOf<List<DemandUpcomingRow>>(emptyList()) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getDemandHistory()
                if (resp.isSuccessful) {
                    val body = resp.body()
                    timeSeries = body?.timeSeries.orEmpty()
                    upcoming = body?.upcoming.orEmpty()
                } else {
                    error = "Failed (${resp.code()})"
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
                title = { Text("Demand history") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading demand history…", "14-day series")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Demand history unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            timeSeries.isEmpty() && upcoming.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No demand data",
                body = "Predictions and actuals will appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    Text("Time series", style = MaterialTheme.typography.titleSmall)
                }
                items(timeSeries, key = { it.date }) { point ->
                    SupplierOpsListCard(
                        headline = point.date,
                        supporting = "Predicted ${point.predictedQty} · Actual ${point.actualQty}",
                        status = "FORECAST",
                    )
                }
                if (upcoming.isNotEmpty()) {
                    item {
                        Text(
                            "Upcoming",
                            style = MaterialTheme.typography.titleSmall,
                            modifier = Modifier.padding(top = PegasusSpacing.md),
                        )
                    }
                    items(upcoming, key = { "${it.date}-${it.skuId}" }) { row ->
                        SupplierOpsListCard(
                            headline = row.productName.ifBlank { row.skuId },
                            supporting = "${row.retailerName} · qty ${row.predictedQty} · ${row.date}",
                        )
                    }
                }
            }
        }
    }
}
