package com.pegasus.warehouse.ui.screens.dashboard

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pegasus.warehouse.data.model.DashboardData
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.ui.navigation.WarehouseRoutes
import com.pegasus.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private data class KpiCard(
    val label: String,
    val icon: ImageVector,
    val route: String,
    val value: (DashboardData) -> String,
)

private val kpiCards = listOf(
    KpiCard("Active Orders", Icons.Default.ShoppingCart, WarehouseRoutes.ORDERS) { it.activeOrders.toString() },
    KpiCard("Completed Today", Icons.Default.CheckCircle, WarehouseRoutes.ORDERS) { it.completedToday.toString() },
    KpiCard("Pending Dispatch", Icons.Default.LocalShipping, WarehouseRoutes.DISPATCH) { it.pendingDispatch.toString() },
    KpiCard("Today Revenue", Icons.Default.AttachMoney, WarehouseRoutes.TREASURY) { "${it.todayRevenue / 1000}K" },
    KpiCard("Drivers On Route", Icons.Default.DirectionsCar, WarehouseRoutes.DRIVERS) { it.driversOnRoute.toString() },
    KpiCard("Idle Drivers", Icons.Default.PersonOff, WarehouseRoutes.DRIVERS) { it.idleDrivers.toString() },
    KpiCard("Vehicles", Icons.Default.DirectionsCar, WarehouseRoutes.VEHICLES) { it.vehicles.toString() },
    KpiCard("Low Stock", Icons.Default.Warning, WarehouseRoutes.INVENTORY) { it.lowStockItems.toString() },
    KpiCard("Staff", Icons.Default.People, WarehouseRoutes.STAFF) { it.totalStaff.toString() },
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    api: WarehouseApi,
    onNavigate: (String) -> Unit,
    onSignOut: () -> Unit,
) {
    var data by remember { mutableStateOf(DashboardData()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getDashboard()
                if (resp.isSuccessful && resp.body() != null) {
                    data = resp.body()!!
                } else {
                    error = "Failed to load (${resp.code()})"
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
                title = { Text("Dashboard") },
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
                columns = GridCells.Adaptive(minSize = 160.dp),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(kpiCards.size) { index ->
                    val card = kpiCards[index]
                    ElevatedCard(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onNavigate(card.route) },
                    ) {
                        Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                            Icon(
                                imageVector = card.icon,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.size(24.dp),
                            )
                            Spacer(Modifier.height(PegasusSpacing.md))
                            Text(
                                text = card.value(data),
                                style = MaterialTheme.typography.headlineMedium,
                            )
                            Spacer(Modifier.height(PegasusSpacing.xs))
                            Text(
                                text = card.label,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }
        }
    }
}
