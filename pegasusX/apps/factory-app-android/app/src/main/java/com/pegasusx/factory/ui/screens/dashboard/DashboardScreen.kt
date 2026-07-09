package com.pegasusx.factory.ui.screens.dashboard

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
import com.pegasusx.factory.BuildConfig
import com.pegasusx.factory.data.model.DashboardStats
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.ui.components.ClientPolicyBanner
import com.pegasusx.factory.ui.components.FactoryKpiBadge
import com.pegasusx.factory.ui.components.FactoryKpiTile
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.components.FactorySectionTitle
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.navigation.FactoryRoutes
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
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
    KpiCard("Gate Exceptions", Icons.Default.Warning, FactoryRoutes.MANIFEST_EXCEPTIONS, { it.criticalInsights.toString() }, { "Transfers removed during loading" }),
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
    var clientPolicyMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun loadClientPolicy() {
        scope.launch {
            try {
                val resp = api.getClientPolicy(
                    platform = "android",
                    version = BuildConfig.VERSION_NAME,
                )
                if (resp.isSuccessful && resp.body() != null) {
                    val policy = resp.body()!!
                    if (policy.outdated || policy.forceUpdate) {
                        clientPolicyMessage = buildString {
                            append(if (policy.forceUpdate) "Update required" else "Update available")
                            if (policy.minimumVersion.isNotBlank()) {
                                append(" — minimum version ${policy.minimumVersion}")
                            }
                            policy.deferReason?.takeIf { it.isNotBlank() }?.let { append(". $it") }
                        }
                    }
                }
            } catch (_: Exception) {
                // Policy fetch is optional on local/dev stacks.
            }
        }
    }

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
        loadClientPolicy()
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
                    IconButton(onClick = { onNavigate(FactoryRoutes.NOTIFICATIONS) }) {
                        Icon(Icons.Default.Notifications, "Notifications")
                    }
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
            loading -> PegasusLoadingState(
                title = "Loading dashboard",
                body = "Fetching live factory metrics for loading, fleet, and staffing.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load dashboard",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 160.dp),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item(span = { GridItemSpan(maxLineSpan) }) {
                    ClientPolicyBanner(clientPolicyMessage)
                }
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
                    FactorySectionTitle(title = "Operations at a glance")
                }
                items(kpiCards, key = { it.label }) { card ->
                    val badge = when (card.label) {
                        "Gate Exceptions" -> if (stats.criticalInsights > 0) FactoryKpiBadge.Alert else null
                        "Dispatched Today" -> if (stats.dispatchedToday > 0) FactoryKpiBadge.Done else null
                        else -> null
                    }
                    FactoryKpiTile(
                        label = card.label,
                        value = card.value(stats),
                        icon = card.icon,
                        supporting = card.supporting(stats),
                        badge = badge,
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
            WorkflowLaunchRow(
                title = "Manifest lifecycle",
                supporting = "Advance manifests through draft, loading, sealed, dispatched, and completed.",
                actionLabel = "Open manifests",
                onClick = { onNavigate(FactoryRoutes.MANIFESTS) },
            )
            WorkflowLaunchRow(
                title = "Gate exceptions",
                supporting = "Review transfers removed from manifests and DLQ escalations.",
                actionLabel = "Open exceptions",
                onClick = { onNavigate(FactoryRoutes.MANIFEST_EXCEPTIONS) },
            )
            WorkflowLaunchRow(
                title = "Create transfer",
                supporting = "Stage a new factory-to-warehouse movement with volume and optional fleet assignment.",
                actionLabel = "Create transfer",
                onClick = { onNavigate(FactoryRoutes.TRANSFER_CREATE) },
            )
            WorkflowLaunchRow(
                title = "Replenishment insights",
                supporting = "Warehouse stock velocity and reorder pressure linked to this factory.",
                actionLabel = "Open insights",
                onClick = { onNavigate(FactoryRoutes.INSIGHTS) },
            )
            WorkflowLaunchRow(
                title = "Analytics overview",
                supporting = "Factory throughput, active manifests, exception queue, and lead time.",
                actionLabel = "Open analytics",
                onClick = { onNavigate(FactoryRoutes.ANALYTICS) },
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
                FactoryMetricTile(
                    label = "Queued",
                    value = stats.pendingTransfers.toString(),
                    modifier = Modifier.weight(1f),
                )
                FactoryMetricTile(
                    label = "Loading",
                    value = stats.loadingTransfers.toString(),
                    modifier = Modifier.weight(1f),
                )
                FactoryMetricTile(
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

