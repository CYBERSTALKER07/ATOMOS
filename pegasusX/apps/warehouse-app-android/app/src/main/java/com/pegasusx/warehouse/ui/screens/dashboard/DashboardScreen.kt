package com.pegasusx.warehouse.ui.screens.dashboard

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pegasus.design.ui.ORDER_STATUS_FUNNEL
import com.pegasus.design.ui.SourceChip
import com.pegasus.design.ui.StatusStack
import com.pegasus.design.ui.TRUCK_DUTY_STATUSES
import com.pegasusx.warehouse.data.model.DashboardData
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.components.FleetLiveMapSection
import com.pegasusx.warehouse.ui.components.WarehouseKpiBadge
import com.pegasusx.warehouse.ui.components.WarehouseKpiTile
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.warehouse.ui.navigation.WarehouseRoutes
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.R
import com.pegasus.design.network.MarketPack
import com.pegasus.design.network.MarketPackBinder
import com.pegasus.design.ui.PackBanner
import com.pegasusx.warehouse.BuildConfig
import com.pegasusx.warehouse.data.remote.TokenHolder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

private data class KpiCard(
    val label: String,
    val icon: ImageVector,
    val route: String,
    val value: (DashboardData) -> String,
    val danger: (DashboardData) -> Boolean = { false },
    val highlight: (DashboardData) -> Boolean = { false },
)

private val kpiCards = listOf(
    KpiCard("Pending Dispatch", Icons.Default.LocalShipping, WarehouseRoutes.DISPATCH, value = { it.pendingDispatch.toString() }, danger = { it.pendingDispatch > 5 }),
    KpiCard("Active Orders", Icons.Default.ShoppingCart, WarehouseRoutes.ORDERS, value = { it.activeOrders.toString() }),
    KpiCard("Vehicles", Icons.Default.DirectionsCar, WarehouseRoutes.VEHICLES, value = { it.totalVehicles.toString() }),
    KpiCard("Low Stock", Icons.Default.Warning, WarehouseRoutes.INVENTORY, value = { it.lowStockCount.toString() }, danger = { it.lowStockCount > 0 }),
    KpiCard("Drivers", Icons.Default.People, WarehouseRoutes.DRIVERS, value = { it.totalDrivers.toString() }),
    KpiCard("Completed Today", Icons.Default.CheckCircle, WarehouseRoutes.ORDERS, value = { if (it.completedTodayAvailable) it.completedToday.toString() else "unavailable" }),
    KpiCard("Today Revenue", Icons.Default.AttachMoney, WarehouseRoutes.TREASURY, value = { if (it.todayRevenueAvailable) "${it.todayRevenue / 1000}K" else "unavailable" }),
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
    var pack by remember { mutableStateOf<MarketPack?>(null) }
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
        pack = withContext(Dispatchers.IO) {
            MarketPackBinder.fetch(BuildConfig.API_BASE_URL, TokenHolder.token.orEmpty())?.pack
        }
    }

    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Dashboard")
                        PackBanner(pack)
                    }
                },
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
            loading && !hasData -> PegasusLoadingState(
                title = stringResource(R.string.mobile_warehouse_ui_loading_dashboard),
                body = "Warehouse KPIs and fleet snapshot",
                modifier = Modifier.padding(innerPadding),
            )
            error != null && !hasData -> PegasusStatePane(
                kind = PegasusStateKind.Error,
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
                StatusStack(
                    counts = data.ordersByStatus,
                    dictionary = ORDER_STATUS_FUNNEL,
                    source = "live",
                    onSelect = { key -> onNavigate(WarehouseRoutes.orders(key)) },
                )
                StatusStack(
                    counts = data.truckDuty,
                    dictionary = TRUCK_DUTY_STATUSES,
                    source = "live",
                )
                if (data.holdReasons.isNotEmpty()) {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        WarehouseSectionTitle("Hold reasons")
                        data.holdReasons.forEach { row ->
                            Text("${row.code} · ${row.count}", style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    SourceChip(if (data.demandSource.isBlank()) "empty" else data.demandSource)
                    Text(
                        if (data.demandSource == "empty") "Planner empty" else "Demand ${data.demandSource}",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                if (!data.historyAvailable) {
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        SourceChip("unavailable")
                        Text("History unavailable", style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
        }
    }
}
