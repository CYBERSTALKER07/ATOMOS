package com.pegasusx.factory.ui.screens.analytics

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import com.pegasusx.factory.data.model.FactoryAnalyticsOverview
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.ui.components.FactoryLoadingState
import com.pegasusx.factory.ui.components.FactoryStateKind
import com.pegasusx.factory.ui.components.FactoryStatePane
import com.pegasusx.factory.ui.theme.Destructive
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

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
                title = { Text("Analytics Overview") },
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
            Text(
                text = "Factory throughput, manifest pressure, and exception queue.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        item {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                AnalyticsKpiCard("Transfers Total", overview.transfersTotal.toString())
                AnalyticsKpiCard("Active Manifests", overview.manifestsActive.toString())
                AnalyticsKpiCard(
                    label = "Exception Queue",
                    value = overview.exceptionQueue.toString(),
                    alert = overview.exceptionQueue > 0,
                )
                AnalyticsKpiCard(
                    label = "Avg Lead Time (min)",
                    value = String.format("%.1f", overview.avgLeadTimeMins),
                )
            }
        }
        if (overview.dailyActivity.isNotEmpty()) {
            item {
                Text(
                    text = "7-day transfer activity",
                    style = MaterialTheme.typography.titleSmall,
                )
            }
            items(overview.dailyActivity, key = { it.date }) { day ->
                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(PegasusSpacing.md),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(day.date, style = MaterialTheme.typography.bodyMedium)
                        Text(
                            text = "${day.transfers} transfers",
                            style = MaterialTheme.typography.bodyMedium,
                            fontFamily = FontFamily.Monospace,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun AnalyticsKpiCard(
    label: String,
    value: String,
    alert: Boolean = false,
) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.lg),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column {
                Text(
                    text = label,
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text(
                    text = value,
                    style = MaterialTheme.typography.headlineSmall,
                    fontFamily = FontFamily.Monospace,
                )
            }
            if (alert) {
                Icon(
                    imageVector = Icons.Default.Warning,
                    contentDescription = "Alert",
                    tint = Destructive,
                )
            }
        }
    }
}
