package com.pegasusx.warehouse.ui.navigation

import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Handler
import android.os.Looper
import androidx.compose.animation.AnimatedContentTransitionScope
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.material3.windowsizeclass.WindowSizeClass
import androidx.compose.material3.windowsizeclass.WindowWidthSizeClass
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.pegasusx.warehouse.BuildConfig
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.GeocodeApi
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.service.AutoUpdater
import com.pegasusx.warehouse.ui.components.ClientPolicyBanner
import com.pegasusx.warehouse.ui.components.WarehouseBottomBar
import com.pegasusx.warehouse.ui.components.WarehouseNavigationDrawer
import com.pegasusx.warehouse.ui.screens.analytics.AnalyticsScreen
import com.pegasusx.warehouse.ui.screens.auth.LoginScreen
import com.pegasusx.warehouse.ui.screens.crm.CRMScreen
import com.pegasusx.warehouse.ui.screens.dashboard.DashboardScreen
import com.pegasusx.warehouse.ui.screens.dispatch.DispatchScreen
import com.pegasusx.warehouse.ui.screens.dispatch.DispatchSettingsScreen
import com.pegasusx.warehouse.ui.screens.drivers.DriversScreen
import com.pegasusx.warehouse.ui.screens.fleet.FleetLiveMapScreen
import com.pegasusx.warehouse.ui.screens.forecast.DemandForecastScreen
import com.pegasusx.warehouse.ui.screens.inventory.InventoryScreen
import com.pegasusx.warehouse.ui.screens.inventory.LocationSettingsScreen
import com.pegasusx.warehouse.ui.screens.inventory.OpsSettingsScreen
import com.pegasusx.warehouse.ui.screens.preorders.PreordersScreen
import com.pegasusx.warehouse.ui.screens.preorders.StockCommitmentsScreen
import com.pegasusx.warehouse.ui.screens.manifests.ManifestsScreen
import com.pegasusx.warehouse.ui.screens.more.MoreHubScreen
import com.pegasusx.warehouse.ui.screens.notifications.NotificationInboxScreen
import com.pegasusx.warehouse.ui.screens.operations.OperationsScreen
import com.pegasusx.warehouse.ui.screens.orders.OrderDetailScreen
import com.pegasusx.warehouse.ui.screens.orders.OrdersScreen
import com.pegasusx.warehouse.ui.screens.payment.PaymentConfigScreen
import com.pegasusx.warehouse.ui.screens.portal.PortalHandoffScreen
import com.pegasusx.warehouse.ui.screens.products.ProductsScreen
import com.pegasusx.warehouse.ui.screens.replenishment.ReplenishmentScreen
import com.pegasusx.warehouse.ui.screens.returns.ReturnsScreen
import com.pegasusx.warehouse.ui.screens.setup.LocationSetupScreen
import com.pegasusx.warehouse.ui.screens.supply.SupplyRequestDetailScreen
import com.pegasusx.warehouse.ui.screens.supply.SupplyRequestsScreen
import com.pegasusx.warehouse.ui.screens.transfers.TransferActionsScreen
import com.pegasusx.warehouse.ui.screens.treasury.TreasuryScreen
import com.pegasusx.warehouse.ui.screens.vehicles.VehicleDetailScreen
import com.pegasusx.warehouse.ui.screens.vehicles.VehiclesScreen
import com.pegasusx.warehouse.ui.portal.WarehousePortalFeature
import kotlinx.coroutines.launch

object WarehouseRoutes {
    const val LOGIN = "login"
    const val DASHBOARD = "dashboard"
    const val ORDERS = "orders"
    const val ORDER_DETAIL = "orders/{id}"
    const val DRIVERS = "drivers"
    const val VEHICLES = "vehicles"
    const val VEHICLE_DETAIL = "vehicles/{id}"
    const val INVENTORY = "inventory"
    const val PRODUCTS = "products"
    const val MANIFESTS = "manifests"
    const val ANALYTICS = "analytics"
    const val CRM = "crm"
    const val RETURNS = "returns"
    const val TREASURY = "treasury"
    const val DISPATCH = "dispatch"
    const val FLEET_LIVE_MAP = "fleet_live_map"
    const val STAFF = "staff"
    const val MORE = "more"
    const val DEMAND_FORECAST = "demand_forecast"
    const val TRANSFER_ACTIONS = "transfer_actions"
    const val REPLENISHMENT = "replenishment"
    const val DISPATCH_SETTINGS = "dispatch_settings"
    const val OPS_SETTINGS = "ops_settings"
    const val LOCATION_SETUP = "location_setup"
    const val LOCATION_SETTINGS = "location_settings"
    const val PREORDERS = "preorders"
    const val STOCK_COMMITMENTS = "stock_commitments"
    const val PAYMENT_CONFIG = "payment_config"
    const val NOTIFICATIONS = "notifications"
    const val OPERATIONS = "operations"
    const val SUPPLY_REQUESTS = "supply_requests"
    const val SUPPLY_REQUEST_DETAIL = "supply_requests/{id}"
    const val PORTAL_HANDOFF = "portal/{feature}"

    fun orderDetail(id: String) = "orders/$id"
    fun vehicleDetail(id: String) = "vehicles/$id"
    fun supplyRequestDetail(id: String) = "supply_requests/$id"
    fun portalHandoff(feature: String) = "portal/$feature"
}

private const val MOTION_DURATION = 300

private val compactTabRoutes = WarehouseSection.compactTabs.map { it.route }.toSet()

@Composable
fun WarehouseNavigation(
    api: WarehouseApi,
    opsRepository: WarehouseOperationsRepository,
    geocodeApi: GeocodeApi,
    realtimeSignals: WarehouseRealtimeSignals,
    windowSizeClass: WindowSizeClass,
    onAuthenticated: () -> Unit = {},
    navController: NavHostController = rememberNavController(),
) {
    val startDestination = when {
        !TokenHolder.isLoggedIn -> WarehouseRoutes.LOGIN
        !TokenHolder.isConfigured -> WarehouseRoutes.LOCATION_SETUP
        else -> WarehouseRoutes.DASHBOARD
    }
    val lifecycleOwner = LocalLifecycleOwner.current
    val context = LocalContext.current
    var networkAvailable by remember { mutableStateOf(true) }
    var clientPolicyMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val autoUpdater = remember { AutoUpdater(context.applicationContext) }
    val useDrawer = windowSizeClass.widthSizeClass != WindowWidthSizeClass.Compact
    var isRailExpanded by remember { mutableStateOf(true) }
    val navBackStackEntry by navController.currentBackStackEntryAsState()
    val currentRoute = navBackStackEntry?.destination?.route
    val showShell = currentRoute != null &&
        currentRoute != WarehouseRoutes.LOGIN &&
        currentRoute != WarehouseRoutes.LOCATION_SETUP
    var lastNavWasTab by remember { mutableStateOf(true) }

    DisposableEffect(autoUpdater) {
        onDispose { autoUpdater.cleanup() }
    }

    fun loadClientPolicy() {
        scope.launch {
            try {
                val resp = api.getClientPolicy(
                    platform = "android",
                    version = BuildConfig.VERSION_NAME,
                )
                if (resp.isSuccessful && resp.body() != null) {
                    val policy = resp.body()!!
                    clientPolicyMessage = if (policy.outdated || policy.forceUpdate) {
                        buildString {
                            append(if (policy.forceUpdate) "Update required" else "Update available")
                            if (policy.minimumVersion.isNotBlank()) {
                                append(" — minimum version ${policy.minimumVersion}")
                            }
                            policy.deferReason?.takeIf { it.isNotBlank() }?.let { append(". $it") }
                        }
                    } else {
                        null
                    }
                    if (policy.outdated || policy.forceUpdate) {
                        autoUpdater.checkForUpdates(BuildConfig.VERSION_CODE)
                    }
                }
            } catch (_: Exception) {
                // Policy fetch is optional on local/dev stacks.
            }
        }
    }

    LaunchedEffect(Unit) {
        loadClientPolicy()
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                loadClientPolicy()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    DisposableEffect(context) {
        val mainHandler = Handler(Looper.getMainLooper())
        val connectivityManager = context.getSystemService(ConnectivityManager::class.java)
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                mainHandler.post {
                    if (!networkAvailable) { /* per-screen refresh */ }
                    networkAvailable = true
                }
            }

            override fun onLost(network: Network) {
                mainHandler.post { networkAvailable = false }
            }
        }

        runCatching { connectivityManager?.registerNetworkCallback(request, callback) }
        onDispose {
            runCatching { connectivityManager?.unregisterNetworkCallback(callback) }
        }
    }

    fun signOut() {
        TokenHolder.clear()
        navController.navigate(WarehouseRoutes.LOGIN) {
            popUpTo(0) { inclusive = true }
        }
    }

    fun navigateSection(section: WarehouseSection) {
        lastNavWasTab = true
        navController.navigate(section.route) {
            popUpTo(WarehouseRoutes.DASHBOARD) { saveState = true }
            launchSingleTop = true
            restoreState = true
        }
    }

    fun showBack(route: String): Boolean {
        if (route.contains("{")) return true
        if (useDrawer) return navController.previousBackStackEntry != null
        val base = route.substringBefore("/")
        if (base in compactTabRoutes && lastNavWasTab) return false
        return navController.previousBackStackEntry != null
    }

    fun backFor(route: String): (() -> Unit)? =
        if (showBack(route)) ({
            if (!navController.popBackStack()) {
                navigateSection(WarehouseSection.DASHBOARD)
            }
        }) else null

    val navHost: @Composable (Modifier) -> Unit = { modifier ->
        NavHost(
                navController = navController,
                startDestination = startDestination,
                modifier = modifier,
                enterTransition = {
                    slideIntoContainer(AnimatedContentTransitionScope.SlideDirection.Start, tween(MOTION_DURATION)) + fadeIn(tween(MOTION_DURATION))
                },
                exitTransition = {
                    slideOutOfContainer(AnimatedContentTransitionScope.SlideDirection.Start, tween(MOTION_DURATION)) + fadeOut(tween(MOTION_DURATION))
                },
                popEnterTransition = {
                    slideIntoContainer(AnimatedContentTransitionScope.SlideDirection.End, tween(MOTION_DURATION)) + fadeIn(tween(MOTION_DURATION))
                },
                popExitTransition = {
                    slideOutOfContainer(AnimatedContentTransitionScope.SlideDirection.End, tween(MOTION_DURATION)) + fadeOut(tween(MOTION_DURATION))
                },
            ) {
                composable(WarehouseRoutes.LOGIN) {
                    LoginScreen(
                        api = api,
                        onLoginSuccess = {
                            onAuthenticated()
                            val dest = if (TokenHolder.isConfigured) {
                                WarehouseRoutes.DASHBOARD
                            } else {
                                WarehouseRoutes.LOCATION_SETUP
                            }
                            navController.navigate(dest) {
                                popUpTo(WarehouseRoutes.LOGIN) { inclusive = true }
                            }
                        },
                    )
                }

                composable(WarehouseRoutes.LOCATION_SETUP) {
                    LocationSetupScreen(
                        api = api,
                        geocodeApi = geocodeApi,
                        onComplete = {
                            navController.navigate(WarehouseRoutes.DASHBOARD) {
                                popUpTo(WarehouseRoutes.LOCATION_SETUP) { inclusive = true }
                            }
                        },
                    )
                }

                composable(WarehouseRoutes.DASHBOARD) {
                    DashboardScreen(
                        api = api,
                        opsRepository = opsRepository,
                        realtimeSignals = realtimeSignals,
                        onNavigate = { route ->
                            lastNavWasTab = false
                            navController.navigate(route)
                        },
                        onSignOut = ::signOut,
                    )
                }

                composable(WarehouseRoutes.ORDERS) {
                    OrdersScreen(
                        api = api,
                        opsRepository = opsRepository,
                        realtimeSignals = realtimeSignals,
                        onOrderClick = { id -> navController.navigate(WarehouseRoutes.orderDetail(id)) },
                        onBack = backFor(WarehouseRoutes.ORDERS),
                    )
                }

                composable(
                    route = WarehouseRoutes.ORDER_DETAIL,
                    arguments = listOf(navArgument("id") { type = NavType.StringType }),
                ) { backStackEntry ->
                    val id = backStackEntry.arguments?.getString("id") ?: return@composable
                    OrderDetailScreen(
                        api = api,
                        opsRepository = opsRepository,
                        orderId = id,
                        onBack = backFor(WarehouseRoutes.ORDER_DETAIL),
                    )
                }

                composable(WarehouseRoutes.DRIVERS) {
                    DriversScreen(api = api, realtimeSignals = realtimeSignals, onBack = backFor(WarehouseRoutes.DRIVERS))
                }

                composable(WarehouseRoutes.VEHICLES) {
                    VehiclesScreen(
                        api = api,
                        realtimeSignals = realtimeSignals,
                        onVehicleClick = { id -> navController.navigate(WarehouseRoutes.vehicleDetail(id)) },
                        onBack = backFor(WarehouseRoutes.VEHICLES),
                    )
                }

                composable(
                    route = WarehouseRoutes.VEHICLE_DETAIL,
                    arguments = listOf(navArgument("id") { type = NavType.StringType }),
                ) { backStackEntry ->
                    val id = backStackEntry.arguments?.getString("id") ?: return@composable
                    VehicleDetailScreen(
                        api = api,
                        vehicleId = id,
                        realtimeSignals = realtimeSignals,
                        onBack = backFor(WarehouseRoutes.VEHICLE_DETAIL),
                    )
                }

                composable(WarehouseRoutes.INVENTORY) {
                    InventoryScreen(api = api, realtimeSignals = realtimeSignals, onBack = backFor(WarehouseRoutes.INVENTORY))
                }

                composable(WarehouseRoutes.PRODUCTS) {
                    ProductsScreen(api = api, onBack = backFor(WarehouseRoutes.PRODUCTS))
                }

                composable(WarehouseRoutes.MANIFESTS) {
                    ManifestsScreen(api = api, onBack = backFor(WarehouseRoutes.MANIFESTS))
                }

                composable(WarehouseRoutes.ANALYTICS) {
                    AnalyticsScreen(api = api, onBack = backFor(WarehouseRoutes.ANALYTICS))
                }

                composable(WarehouseRoutes.CRM) {
                    CRMScreen(api = api, onBack = backFor(WarehouseRoutes.CRM))
                }

                composable(WarehouseRoutes.RETURNS) {
                    ReturnsScreen(api = api, onBack = backFor(WarehouseRoutes.RETURNS))
                }

                composable(WarehouseRoutes.TREASURY) {
                    TreasuryScreen(api = api, onBack = backFor(WarehouseRoutes.TREASURY))
                }

                composable(WarehouseRoutes.DISPATCH) {
                    DispatchScreen(
                        api = api,
                        opsRepository = opsRepository,
                        realtimeSignals = realtimeSignals,
                        onVehicleClick = { id -> navController.navigate(WarehouseRoutes.vehicleDetail(id)) },
                        onOrderClick = { id -> navController.navigate(WarehouseRoutes.orderDetail(id)) },
                        onBack = backFor(WarehouseRoutes.DISPATCH),
                    )
                }

                composable(WarehouseRoutes.FLEET_LIVE_MAP) {
                    FleetLiveMapScreen(
                        ops = opsRepository,
                        realtimeSignals = realtimeSignals,
                        onBack = backFor(WarehouseRoutes.FLEET_LIVE_MAP),
                    )
                }

                composable(WarehouseRoutes.STAFF) {
                    StaffScreen(api = api, realtimeSignals = realtimeSignals, onBack = backFor(WarehouseRoutes.STAFF))
                }

                composable(WarehouseRoutes.MORE) {
                    MoreHubScreen(
                        onNavigate = { route ->
                            lastNavWasTab = false
                            navController.navigate(route)
                        },
                        onBack = backFor(WarehouseRoutes.MORE),
                    )
                }

                composable(WarehouseRoutes.DEMAND_FORECAST) {
                    DemandForecastScreen(
                        api = api,
                        onBack = backFor(WarehouseRoutes.DEMAND_FORECAST),
                    )
                }

                composable(WarehouseRoutes.TRANSFER_ACTIONS) {
                    TransferActionsScreen(
                        opsRepository = opsRepository,
                        realtimeSignals = realtimeSignals,
                        onBack = backFor(WarehouseRoutes.TRANSFER_ACTIONS),
                    )
                }

                composable(WarehouseRoutes.REPLENISHMENT) {
                    ReplenishmentScreen(
                        opsRepository = opsRepository,
                        realtimeSignals = realtimeSignals,
                        onBack = backFor(WarehouseRoutes.REPLENISHMENT),
                    )
                }

                composable(WarehouseRoutes.DISPATCH_SETTINGS) {
                    DispatchSettingsScreen(
                        opsRepository = opsRepository,
                        onBack = backFor(WarehouseRoutes.DISPATCH_SETTINGS),
                    )
                }

                composable(WarehouseRoutes.OPS_SETTINGS) {
                    OpsSettingsScreen(api = api, onBack = backFor(WarehouseRoutes.OPS_SETTINGS))
                }
                composable(WarehouseRoutes.LOCATION_SETTINGS) {
                    LocationSettingsScreen(
                        api = api,
                        geocodeApi = geocodeApi,
                        onBack = backFor(WarehouseRoutes.LOCATION_SETTINGS),
                    )
                }
                composable(WarehouseRoutes.PREORDERS) {
                    PreordersScreen(
                        api = api,
                        realtimeSignals = realtimeSignals,
                        onBack = backFor(WarehouseRoutes.PREORDERS),
                    )
                }
                composable(WarehouseRoutes.STOCK_COMMITMENTS) {
                    StockCommitmentsScreen(api = api, onBack = backFor(WarehouseRoutes.STOCK_COMMITMENTS))
                }

                composable(WarehouseRoutes.PAYMENT_CONFIG) {
                    PaymentConfigScreen(
                        opsRepository = opsRepository,
                        onBack = backFor(WarehouseRoutes.PAYMENT_CONFIG),
                    )
                }

                composable(WarehouseRoutes.NOTIFICATIONS) {
                    NotificationInboxScreen(
                        api = api,
                        onBack = backFor(WarehouseRoutes.NOTIFICATIONS),
                    )
                }

                composable(WarehouseRoutes.OPERATIONS) {
                    OperationsScreen(
                        api = api,
                        onBack = backFor(WarehouseRoutes.OPERATIONS),
                    )
                }

                composable(WarehouseRoutes.SUPPLY_REQUESTS) {
                    SupplyRequestsScreen(
                        api = api,
                        onRequestClick = { id -> navController.navigate(WarehouseRoutes.supplyRequestDetail(id)) },
                        onBack = backFor(WarehouseRoutes.SUPPLY_REQUESTS),
                    )
                }

                composable(
                    route = WarehouseRoutes.SUPPLY_REQUEST_DETAIL,
                    arguments = listOf(navArgument("id") { type = NavType.StringType }),
                ) { backStackEntry ->
                    val id = backStackEntry.arguments?.getString("id") ?: return@composable
                    SupplyRequestDetailScreen(
                        api = api,
                        opsRepository = opsRepository,
                        realtimeSignals = realtimeSignals,
                        requestId = id,
                        onBack = backFor(WarehouseRoutes.SUPPLY_REQUEST_DETAIL),
                    )
                }

                composable(
                    route = WarehouseRoutes.PORTAL_HANDOFF,
                    arguments = listOf(navArgument("feature") { type = NavType.StringType }),
                ) { backStackEntry ->
                    val key = backStackEntry.arguments?.getString("feature") ?: return@composable
                    val feature = WarehousePortalFeature.fromRouteKey(key) ?: return@composable
                    PortalHandoffScreen(
                        feature = feature,
                        onBack = backFor(WarehouseRoutes.PORTAL_HANDOFF),
                    )
                }
            }
    }

    if (showShell) {
        Column(Modifier.fillMaxSize()) {
            ClientPolicyBanner(clientPolicyMessage)
            Row(Modifier.weight(1f)) {
            if (useDrawer) {
                WarehouseNavigationDrawer(
                    isExpanded = isRailExpanded,
                    onToggleExpanded = { isRailExpanded = !isRailExpanded },
                    selectedRoute = currentRoute,
                    onSectionSelected = ::navigateSection,
                    onSignOut = ::signOut,
                )
            }
            Scaffold(
                bottomBar = {
                    if (!useDrawer) {
                        WarehouseBottomBar(
                            selectedRoute = currentRoute,
                            onSectionSelected = ::navigateSection,
                        )
                    }
                },
            ) { innerPadding ->
                navHost(Modifier.padding(innerPadding).fillMaxSize())
            }
            }
        }
    } else {
        Column(Modifier.fillMaxSize()) {
            ClientPolicyBanner(clientPolicyMessage)
            navHost(Modifier.weight(1f).fillMaxSize())
        }
    }
}
