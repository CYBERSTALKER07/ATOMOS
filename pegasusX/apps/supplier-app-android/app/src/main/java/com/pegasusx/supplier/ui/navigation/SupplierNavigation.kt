package com.pegasusx.supplier.ui.navigation

import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.Dashboard
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.MoreHoriz
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.data.remote.TokenHolder
import kotlinx.coroutines.flow.collectLatest
import com.pegasusx.supplier.ui.screens.auth.LoginScreen
import com.pegasusx.supplier.ui.screens.billing.BillingScreen
import com.pegasusx.supplier.ui.screens.dashboard.DashboardScreen
import com.pegasusx.supplier.ui.screens.earnings.EarningsScreen
import com.pegasusx.supplier.ui.screens.fleet.FleetScreen
import com.pegasusx.supplier.ui.screens.inventory.InventoryScreen
import com.pegasusx.supplier.ui.screens.activity.ActivityScreen
import com.pegasusx.supplier.ui.screens.dispatch.DispatchPreviewScreen
import com.pegasusx.supplier.ui.screens.exceptions.ExceptionsScreen
import com.pegasusx.supplier.ui.screens.exceptions.NegotiationsScreen
import com.pegasusx.supplier.ui.screens.exceptions.ShopClosedScreen
import com.pegasusx.supplier.ui.screens.fleet.FleetOrdersScreen
import com.pegasusx.supplier.ui.screens.manifests.ManifestsScreen
import com.pegasusx.supplier.ui.screens.more.MoreScreen
import com.pegasusx.supplier.ui.screens.operations.OperationsScreen
import com.pegasusx.supplier.ui.screens.orders.OrdersScreen
import com.pegasusx.supplier.ui.screens.profile.ProfileScreen
import com.pegasusx.supplier.ui.screens.treasury.LedgerScreen

object SupplierRoutes {
    const val LOGIN = "login"
    const val BILLING = "billing"
    const val DASHBOARD = "dashboard"
    const val ORDERS = "orders"
    const val FLEET = "fleet"
    const val MORE = "more"
    const val INVENTORY = "inventory"
    const val EARNINGS = "earnings"
    const val PROFILE = "profile"
    const val EXCEPTIONS = "exceptions"
    const val SHOP_CLOSED = "shop_closed"
    const val NEGOTIATIONS = "negotiations"
    const val MANIFESTS = "manifests"
    const val DISPATCH_PREVIEW = "dispatch_preview"
    const val ACTIVITY = "activity"
    const val FLEET_ORDERS = "fleet_orders"
    const val LEDGER = "ledger"
    const val OPERATIONS = "operations"
}

private data class SupplierTab(val route: String, val label: String, val icon: ImageVector)

@Composable
fun SupplierNavigation(
    api: SupplierApi,
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
) {
    var sessionEpoch by remember { mutableIntStateOf(0) }
    var refreshEpoch by remember { mutableIntStateOf(0) }
    LaunchedEffect(realtimeSignals) {
        realtimeSignals.refreshTick.collectLatest { refreshEpoch++ }
    }
    val loggedIn = remember(sessionEpoch) { TokenHolder.isLoggedIn }
    val navController = rememberNavController()
    val start = when {
        !loggedIn -> SupplierRoutes.LOGIN
        !TokenHolder.isConfigured -> SupplierRoutes.BILLING
        else -> SupplierRoutes.DASHBOARD
    }

    if (!loggedIn) {
        NavHost(navController = navController, startDestination = SupplierRoutes.LOGIN) {
            composable(SupplierRoutes.LOGIN) {
                LoginScreen(api = api) {
                    sessionEpoch++
                    navController.navigate(
                        if (TokenHolder.isConfigured) SupplierRoutes.DASHBOARD else SupplierRoutes.BILLING,
                    ) {
                        popUpTo(SupplierRoutes.LOGIN) { inclusive = true }
                    }
                }
            }
        }
        return
    }

    val tabs = listOf(
        SupplierTab(SupplierRoutes.DASHBOARD, "Dashboard", Icons.Default.Dashboard),
        SupplierTab(SupplierRoutes.ORDERS, "Orders", Icons.AutoMirrored.Filled.List),
        SupplierTab(SupplierRoutes.FLEET, "Fleet", Icons.Default.LocalShipping),
        SupplierTab(SupplierRoutes.MORE, "More", Icons.Default.MoreHoriz),
    )
    val backStack by navController.currentBackStackEntryAsState()
    val currentRoute = backStack?.destination?.route
    val showBottomBar = currentRoute in tabs.map { it.route }

    Scaffold(
        bottomBar = {
            if (showBottomBar) {
                NavigationBar {
                    tabs.forEach { tab ->
                        NavigationBarItem(
                            selected = currentRoute == tab.route,
                            onClick = {
                                navController.navigate(tab.route) {
                                    popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            icon = { Icon(tab.icon, contentDescription = tab.label) },
                            label = { Text(tab.label) },
                        )
                    }
                }
            }
        },
    ) { padding ->
        NavHost(
            navController = navController,
            startDestination = start,
            modifier = Modifier.padding(padding),
        ) {
            composable(SupplierRoutes.BILLING) {
                BillingScreen(
                    api = api,
                    onSkip = {
                        TokenHolder.isConfigured = true
                        navController.navigate(SupplierRoutes.DASHBOARD) {
                            popUpTo(SupplierRoutes.BILLING) { inclusive = true }
                        }
                    },
                    onComplete = {
                        TokenHolder.isConfigured = true
                        navController.navigate(SupplierRoutes.DASHBOARD) {
                            popUpTo(SupplierRoutes.BILLING) { inclusive = true }
                        }
                    },
                )
            }
            composable(SupplierRoutes.DASHBOARD) {
                key(refreshEpoch) {
                    DashboardScreen(
                        api = api,
                        ops = ops,
                        showBillingBanner = !TokenHolder.isConfigured,
                        onOpenBilling = { navController.navigate(SupplierRoutes.BILLING) },
                    )
                }
            }
            composable(SupplierRoutes.ORDERS) { key(refreshEpoch) { OrdersScreen(ops) } }
            composable(SupplierRoutes.FLEET) { key(refreshEpoch) { FleetScreen(api = api, ops = ops) } }
            composable(SupplierRoutes.MORE) {
                MoreScreen(
                    onExceptions = { navController.navigate(SupplierRoutes.EXCEPTIONS) },
                    onShopClosed = { navController.navigate(SupplierRoutes.SHOP_CLOSED) },
                    onNegotiations = { navController.navigate(SupplierRoutes.NEGOTIATIONS) },
                    onManifests = { navController.navigate(SupplierRoutes.MANIFESTS) },
                    onDispatch = { navController.navigate(SupplierRoutes.DISPATCH_PREVIEW) },
                    onActivity = { navController.navigate(SupplierRoutes.ACTIVITY) },
                    onFleetOrders = { navController.navigate(SupplierRoutes.FLEET_ORDERS) },
                    onLedger = { navController.navigate(SupplierRoutes.LEDGER) },
                    onOperations = { navController.navigate(SupplierRoutes.OPERATIONS) },
                    onInventory = { navController.navigate(SupplierRoutes.INVENTORY) },
                    onEarnings = { navController.navigate(SupplierRoutes.EARNINGS) },
                    onProfile = { navController.navigate(SupplierRoutes.PROFILE) },
                    onBilling = { navController.navigate(SupplierRoutes.BILLING) },
                    onSignOut = {
                        TokenHolder.clear()
                        sessionEpoch++
                    },
                )
            }
            composable(SupplierRoutes.EXCEPTIONS) {
                key(refreshEpoch) { ExceptionsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.SHOP_CLOSED) {
                key(refreshEpoch) { ShopClosedScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.NEGOTIATIONS) {
                key(refreshEpoch) { NegotiationsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.MANIFESTS) {
                key(refreshEpoch) { ManifestsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.DISPATCH_PREVIEW) {
                key(refreshEpoch) { DispatchPreviewScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.ACTIVITY) {
                key(refreshEpoch) { ActivityScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.FLEET_ORDERS) {
                key(refreshEpoch) { FleetOrdersScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.LEDGER) {
                key(refreshEpoch) { LedgerScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.OPERATIONS) {
                key(refreshEpoch) { OperationsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.INVENTORY) { key(refreshEpoch) { InventoryScreen(api) } }
            composable(SupplierRoutes.EARNINGS) { key(refreshEpoch) { EarningsScreen(api = api, ops = ops) } }
            composable(SupplierRoutes.PROFILE) { key(refreshEpoch) { ProfileScreen(api) } }
        }
    }
}
