package com.pegasusx.factory.ui.screens.dashboard

import androidx.compose.ui.res.stringResource

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
import androidx.compose.ui.platform.LocalContext
import com.pegasusx.factory.BuildConfig
import com.pegasusx.factory.data.model.DashboardStats
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.service.AutoUpdater
import com.pegasusx.factory.service.EnterpriseUpdateConfig
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
import com.pegasusx.factory.ui.screens.dashboard.components.DashboardHeroCard
import com.pegasusx.factory.ui.screens.dashboard.components.WorkflowLaunchCard
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
    var clientPolicyForce by remember { mutableStateOf(false) }
    var pendingManifest by remember { mutableStateOf<AutoUpdater.Manifest?>(null) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val autoUpdater = remember { AutoUpdater(context.applicationContext) }

    DisposableEffect(autoUpdater) {
        autoUpdater.register()
        onDispose { autoUpdater.cleanup() }
    }

    fun loadClientPolicy() {
        scope.launch {
            try {
                val resp = api.getClientPolicy(
                    role = EnterpriseUpdateConfig.POLICY_ROLE,
                    platform = "android",
                    version = BuildConfig.VERSION_NAME,
                    channel = EnterpriseUpdateConfig.CHANNEL,
                )
                if (resp.isSuccessful && resp.body() != null) {
                    val state = autoUpdater.checkFromPolicy(resp.body()!!, autoDownload = false)
                    clientPolicyMessage = state.message
                    clientPolicyForce = state.force
                    pendingManifest = state.manifest
                }
            } catch (_: Exception) {
                // Policy fetch is optional on local/dev stacks.
            }
        }
    }

    fun onUpdateClick() {
        scope.launch {
            autoUpdater.startUpdate(pendingManifest)
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
                            text = stringResource(R.string.mobile_factory_ui_dispatch_loading_fleet_and_staffing_status),
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
                title = stringResource(R.string.mobile_factory_ui_loading_dashboard),
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
                    ClientPolicyBanner(
                        message = clientPolicyMessage,
                        force = clientPolicyForce,
                        onUpdate = if (clientPolicyMessage != null) {
                            { onUpdateClick() }
                        } else {
                            null
                        },
                        onDismiss = if (!clientPolicyForce) {
                            { clientPolicyMessage = null }
                        } else {
                            null
                        },
                    )
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
                    FactorySectionTitle(title = stringResource(R.string.mobile_factory_ui_operations_at_a_glance))
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


