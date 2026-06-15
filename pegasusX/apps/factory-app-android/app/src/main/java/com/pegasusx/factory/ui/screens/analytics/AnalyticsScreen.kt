package com.pegasusx.factory.ui.screens.analytics

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Analytics
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pegasusx.factory.data.model.FactoryAnalyticsOverview
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.ui.components.FactoryKpiBadge
import com.pegasusx.factory.ui.components.FactoryKpiTile
import com.pegasusx.factory.ui.components.FactoryLoadingState
import com.pegasusx.factory.ui.components.FactoryOpsListCard
import com.pegasusx.factory.ui.components.FactorySectionTitle
import com.pegasusx.factory.ui.components.FactoryStateKind
import com.pegasusx.factory.ui.components.FactoryStatePane
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private data class AnalyticsKpi(
    val label: String,
    val value: (FactoryAnalyticsOverview) -> String,
    val icon: ImageVector,
    val alert: (FactoryAnalyticsOverview) -> Boolean = { false },
)

private val analyticsKpis = listOf(
    AnalyticsKpi("Transfers Total", { it.transfersTotal.toString() }, Icons.Default.LocalShipping),
    AnalyticsKpi("Active Manifests", { it.manifestsActive.toString() }, Icons.Default.Analytics),
    AnalyticsKpi(
        label = "Exception Queue",
        value = { it.exceptionQueue.toString() },
        icon = Icons.Default.Warning,
        alert = { it.exceptionQueue > 0 },
    ),
    AnalyticsKpi(
        label = "Avg Lead Time (min)",
        value = { String.format("%.1f", it.avgLeadTimeMins) },
        icon = Icons.Default.Schedule,
    ),
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AnalyticsScreen(
    api: FactoryApi,
    onBack: () -> Unit,
) {
    var overview by remember { mutableStateOf<FactoryAnalyticsOverview?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getFactoryAnalyticsOverview()
                if (resp.isSuccessful && resp.body() != null) {
                    overview = resp.body()!!
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

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Analytics Overview")
                        Text(
                            text = "Throughput, manifest pressure, and exceptions",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, "Refresh")
                    }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading -> FactoryLoadingState(
                title = "Loading analytics",
                body = "Fetching factory throughput, manifest pressure, and exception queue.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> FactoryStatePane(
                kind = FactoryStateKind.Error,
                headline = "Unable to load analytics overview",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            overview != null -> AnalyticsContent(
                overview = overview!!,
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
        }
    }
}

@Composable
private fun AnalyticsContent(
    overview: FactoryAnalyticsOverview,
    modifier: Modifier = Modifier,
) {
    LazyColumn(
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        modifier = modifier,
    ) {
        item {
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 160.dp),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(max = 520.dp),
            ) {
                items(analyticsKpis, key = { it.label }) { kpi ->
                    FactoryKpiTile(
                        label = kpi.label,
                        value = kpi.value(overview),
                        icon = kpi.icon,
                        badge = if (kpi.alert(overview)) FactoryKpiBadge.Alert else null,
                    )
                }
            }
        }
        if (overview.dailyActivity.isNotEmpty()) {
            item {
                FactorySectionTitle(title = "7-day transfer activity")
            }
            items(overview.dailyActivity, key = { it.date }) { day ->
                FactoryOpsListCard(
                    headline = day.date,
                    supporting = "${day.transfers} transfers recorded",
                )
            }
        }
    }
}
