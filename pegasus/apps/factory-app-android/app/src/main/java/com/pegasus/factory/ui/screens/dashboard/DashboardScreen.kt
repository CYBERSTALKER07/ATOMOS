package com.pegasus.factory.ui.screens.dashboard

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pegasus.factory.data.model.DashboardStats
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.factory.ui.navigation.FactoryRoutes
import com.pegasus.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasus.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private data class KpiCard(
    val label: String,
    val icon: ImageVector,
    val route: String,
    val value: (DashboardStats) -> String,
    val supporting: (DashboardStats) -> String,
)

private val kpiCards = listOf(
    KpiCard("Pending Transfers", Icons.Default.MoveToInbox, FactoryRoutes.TRANSFERS, { it.pendingTransfers.toString() }, { "Awaiting release to loading" }),
    KpiCard("Now Loading", Icons.Default.LocalShipping, FactoryRoutes.LOADING_BAY, { it.loadingTransfers.toString() }, { "Transfers staged at the bay" }),
    KpiCard("Active Manifests", Icons.AutoMirrored.Filled.List, FactoryRoutes.LOADING_BAY, { it.activeManifests.toString() }, { "Live outbound manifest groups" }),
    KpiCard("Dispatched Today", Icons.Default.CheckCircle, FactoryRoutes.TRANSFERS, { it.dispatchedToday.toString() }, { "Completed releases this shift" }),
    KpiCard("Vehicles Total", Icons.Default.DirectionsCar, FactoryRoutes.FLEET, { it.vehiclesTotal.toString() }, { "Fleet capacity on record" }),
    KpiCard("Available", Icons.Default.DirectionsCar, FactoryRoutes.FLEET, { it.vehiclesAvailable.toString() }, { "Vehicles ready for assignment" }),
    KpiCard("Staff on Shift", Icons.Default.People, FactoryRoutes.STAFF, { it.staffOnShift.toString() }, { "Operators currently active" }),
    KpiCard("Critical Insights", Icons.Default.Warning, FactoryRoutes.INSIGHTS, { it.criticalInsights.toString() }, { "Restock and exception pressure" }),
)
private const val DASHBOARD_REFRESH_MS = 30_000L

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    api: FactoryApi,
    onNavigate: (String) -> Unit,
    onSignOut: () -> Unit,
) {
    var stats by remember { mutableStateOf(DashboardStats()) }
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
                val resp = api.getDashboard()
                if (resp.isSuccessful && resp.body() != null) {
                    stats = resp.body()!!
                } else {
                    error = "Failed to load (${resp.code()})"
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

    LaunchedEffect(Unit) {
        load()
        while (true) {
            delay(DASHBOARD_REFRESH_MS)
            load(silent = true)
        }
    }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.SupplyRequestUpdate,
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        load(silent = true)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Factory dashboard")
                        Text(
                            text = "Dispatch, loading, fleet, and staffing status",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, "Refresh")
                    }
                    IconButton(onClick = onSignOut) {
                        Icon(Icons.AutoMirrored.Filled.ExitToApp, "Sign out")
                    }
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
            else -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 168.dp),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item(span = { GridItemSpan(maxLineSpan) }) {
                    DashboardHeroCard(
                        stats = stats,
                        onNavigate = onNavigate,
                    )
                }
                item(span = { GridItemSpan(maxLineSpan) }) {
                    WorkflowLaunchCard(onNavigate = onNavigate)
                }
                item(span = { GridItemSpan(maxLineSpan) }) {
                    Text(
                        text = "Operations at a glance",
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.onSurface,
                    )
                }
                items(kpiCards, key = { it.label }) { card ->
                    KpiMetricCard(
                        card = card,
                        stats = stats,
                        onClick = { onNavigate(card.route) },
                    )
                }
            }
        }
    }
}

@Composable
private fun WorkflowLaunchCard(
    onNavigate: (String) -> Unit,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Surface(
                    shape = MaterialTheme.shapes.small,
                    color = MaterialTheme.colorScheme.tertiaryContainer,
                ) {
                    Icon(
                        imageVector = Icons.Default.Computer,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onTertiaryContainer,
                        modifier = Modifier
                            .padding(PegasusSpacing.sm)
                            .size(20.dp),
                    )
                }
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                    Text(
                        text = "Operator workflows",
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = "Warehouse demand acknowledgements and live manifest overrides are available on mobile in streamlined native flows.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            WorkflowLaunchRow(
                title = "Supply requests",
                supporting = "Review warehouse demand and advance requests through production states.",
                actionLabel = "Open requests",
                onClick = { onNavigate(FactoryRoutes.SUPPLY_REQUESTS) },
            )
            WorkflowLaunchRow(
                title = "Payload override",
                supporting = "Move transfers between loading manifests or release them back to approved stock.",
                actionLabel = "Open override",
                onClick = { onNavigate(FactoryRoutes.PAYLOAD_OVERRIDE) },
            )
        }
    }
}

@Composable
private fun WorkflowLaunchRow(
    title: String,
    supporting: String,
    actionLabel: String,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
    ) {
        Row(
            modifier = Modifier.padding(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleSmall,
                )
                Text(
                    text = supporting,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            FilledTonalButton(onClick = onClick) {
                Text(actionLabel)
            }
        }
    }
}

@Composable
private fun DashboardHeroCard(
    stats: DashboardStats,
    onNavigate: (String) -> Unit,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.lg),
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                Text(
                    text = "Outbound floor status",
                    style = MaterialTheme.typography.titleLarge,
                )
                Text(
                    text = "${stats.pendingTransfers + stats.loadingTransfers} transfers are active across release and bay lanes.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                OverviewMetric(
                    label = "Queued",
                    value = stats.pendingTransfers.toString(),
                    modifier = Modifier.weight(1f),
                )
                OverviewMetric(
                    label = "Loading",
                    value = stats.loadingTransfers.toString(),
                    modifier = Modifier.weight(1f),
                )
                OverviewMetric(
                    label = "Critical",
                    value = stats.criticalInsights.toString(),
                    modifier = Modifier.weight(1f),
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FilledTonalButton(
                    onClick = { onNavigate(FactoryRoutes.LOADING_BAY) },
                    modifier = Modifier.weight(1f),
                ) {
                    Icon(Icons.Default.LocalShipping, contentDescription = null)
                    Spacer(Modifier.width(PegasusSpacing.sm))
                    Text("Open bay")
                }
                OutlinedButton(
                    onClick = { onNavigate(FactoryRoutes.TRANSFERS) },
                    modifier = Modifier.weight(1f),
                ) {
                    Icon(Icons.AutoMirrored.Filled.List, contentDescription = null)
                    Spacer(Modifier.width(PegasusSpacing.sm))
                    Text("View transfers")
                }
            }
        }
    }
}

@Composable
private fun OverviewMetric(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier,
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainer,
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            Text(
                text = value,
                style = MaterialTheme.typography.headlineSmall,
            )
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun KpiMetricCard(
    card: KpiCard,
    stats: DashboardStats,
    onClick: () -> Unit,
) {
    ElevatedCard(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Surface(
                shape = MaterialTheme.shapes.small,
                color = MaterialTheme.colorScheme.secondaryContainer,
            ) {
                Icon(
                    imageVector = card.icon,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSecondaryContainer,
                    modifier = Modifier
                        .padding(PegasusSpacing.sm)
                        .size(24.dp),
                )
            }
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                Text(
                    text = card.value(stats),
                    style = MaterialTheme.typography.headlineSmall,
                )
                Text(
                    text = card.label,
                    style = MaterialTheme.typography.titleSmall,
                )
                Text(
                    text = card.supporting(stats),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
