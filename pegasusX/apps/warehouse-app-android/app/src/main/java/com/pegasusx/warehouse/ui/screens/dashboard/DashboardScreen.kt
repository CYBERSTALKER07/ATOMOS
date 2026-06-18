package com.pegasusx.warehouse.ui.screens.dashboard

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.DashboardData
import com.pegasusx.warehouse.data.model.FleetStatusEntry
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.components.FleetLiveMapSection
import com.pegasusx.warehouse.ui.components.WarehouseKpiBadge
import com.pegasusx.warehouse.ui.components.WarehouseKpiTile
import com.pegasusx.warehouse.ui.components.WarehouseLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.components.WarehouseStateKind
import com.pegasusx.warehouse.ui.components.WarehouseStatePane
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.ui.navigation.WarehouseRoutes
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private data class KpiCard(
    val label: String,
    val icon: ImageVector,
    val route: String,
    val value: (DashboardData) -> String,
    val danger: (DashboardData) -> Boolean = { false },
    val highlight: (DashboardData) -> Boolean = { false },
)

private val kpiCards = listOf(
    KpiCard("Active Orders", Icons.Default.ShoppingCart, WarehouseRoutes.ORDERS, value = { it.activeOrders.toString() }),
    KpiCard("Completed Today", Icons.Default.CheckCircle, WarehouseRoutes.ORDERS, value = { it.completedToday.toString() }, highlight = { it.completedToday > 0 }),
    KpiCard("Pending Dispatch", Icons.Default.LocalShipping, WarehouseRoutes.DISPATCH, value = { it.pendingDispatch.toString() }, danger = { it.pendingDispatch > 5 }),
    KpiCard("Today Revenue", Icons.Default.AttachMoney, WarehouseRoutes.TREASURY, value = { "${it.todayRevenue / 1000}K" }),
    KpiCard("Drivers On Route", Icons.Default.DirectionsCar, WarehouseRoutes.DRIVERS, value = { it.driversOnRoute.toString() }),
    KpiCard("Idle Drivers", Icons.Default.PersonOff, WarehouseRoutes.DRIVERS, value = { it.driversIdle.toString() }),
    KpiCard("Vehicles", Icons.Default.DirectionsCar, WarehouseRoutes.VEHICLES, value = { it.totalVehicles.toString() }),
    KpiCard("Low Stock", Icons.Default.Warning, WarehouseRoutes.INVENTORY, value = { it.lowStockCount.toString() }, danger = { it.lowStockCount > 0 }),
    KpiCard("Staff", Icons.Default.People, WarehouseRoutes.STAFF, value = { it.totalStaff.toString() }),
    KpiCard("More", Icons.Default.Apps, WarehouseRoutes.MORE, value = { "…" }),
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    api: WarehouseApi,
    opsRepository: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    onNavigate: (String) -> Unit,
    onSignOut: () -> Unit,
) {
    var data by remember { mutableStateOf(DashboardData()) }
    var loading by remember { mutableStateOf(true) }
    var hasData by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getDashboard()
                if (resp.isSuccessful && resp.body() != null) {
                    data = resp.body()!!
                    hasData = true
                } else if (!silent) {
                    error = "Failed to load (${resp.code()})"
                }
            } catch (e: Exception) {
                if (!silent) error = e.message ?: "Network error"
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        load()
    }

    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Dashboard") },
                actions = {
                    IconButton(onClick = { onNavigate(WarehouseRoutes.MORE) }) {
                        Icon(Icons.Default.Apps, "More")
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
            loading && !hasData -> WarehouseLoadingState(
                title = "Loading dashboard…",
                body = "Warehouse KPIs and fleet snapshot",
                modifier = Modifier.padding(innerPadding),
            )
            error != null && !hasData -> WarehouseStatePane(
                kind = WarehouseStateKind.Error,
                headline = "Dashboard unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(innerPadding),
            )
            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding)
                    .padding(horizontal = PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                FleetLiveMapSection(
                    ops = opsRepository,
                    realtimeSignals = realtimeSignals,
                    onOpenFullMap = { onNavigate(WarehouseRoutes.FLEET_LIVE_MAP) },
                )
                LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 160.dp),
                    contentPadding = PaddingValues(bottom = PegasusSpacing.lg),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    modifier = Modifier.weight(1f, fill = false),
                ) {
                    items(kpiCards.size) { index ->
                        val card = kpiCards[index]
                        val badge = when {
                            card.danger(data) -> WarehouseKpiBadge.Alert
                            card.highlight(data) -> WarehouseKpiBadge.Done
                            else -> null
                        }
                        WarehouseKpiTile(
                            label = card.label,
                            value = card.value(data),
                            icon = card.icon,
                            badge = badge,
                            onClick = { onNavigate(card.route) },
                        )
                    }
                }
                if (data.fleetStatus.isNotEmpty()) {
                    FleetStatusBreakdown(data.fleetStatus)
                }
            }
        }
    }
}

@Composable
private fun FleetStatusBreakdown(entries: List<FleetStatusEntry>) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            WarehouseSectionTitle("Fleet status tracking")
            LazyRow(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                items(entries) { entry ->
                    WarehouseStatusChip(status = "${entry.status}: ${entry.count}")
                }
            }
        }
    }
}
