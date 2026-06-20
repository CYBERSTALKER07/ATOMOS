package com.pegasusx.factory.ui.navigation

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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.pegasusx.factory.ui.components.FactoryBottomBar
import com.pegasusx.factory.ui.components.FactoryNavigationDrawer
import com.pegasusx.factory.ui.screens.more.FactoryMoreHubScreen
import androidx.compose.ui.Modifier
import com.pegasusx.factory.ui.navigation.FactorySection
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.GeocodeApi
import com.pegasusx.factory.data.remote.TokenHolder
import com.pegasusx.factory.ui.screens.analytics.AnalyticsScreen
import com.pegasusx.factory.ui.screens.auth.LoginScreen
import com.pegasusx.factory.ui.screens.dashboard.DashboardScreen
import com.pegasusx.factory.ui.screens.fleet.FleetScreen
import com.pegasusx.factory.ui.screens.insights.InsightsScreen
import com.pegasusx.factory.ui.screens.location.LocationSettingsScreen
import com.pegasusx.factory.ui.screens.loadingbay.LoadingBayScreen
import com.pegasusx.factory.ui.screens.manifest.ManifestDetailScreen
import com.pegasusx.factory.ui.screens.manifest.ManifestListScreen
import com.pegasusx.factory.ui.screens.exceptions.ManifestExceptionsScreen
import com.pegasusx.factory.ui.screens.notifications.NotificationInboxScreen
import com.pegasusx.factory.ui.screens.override.PayloadOverrideScreen
import com.pegasusx.factory.ui.screens.setup.LocationSetupScreen
import com.pegasusx.factory.ui.screens.staff.StaffScreen
import com.pegasusx.factory.ui.screens.supply.SupplyRequestsScreen
import com.pegasusx.factory.ui.screens.transfer.CreateTransferScreen
import com.pegasusx.factory.ui.screens.transfer.TransferDetailScreen
import com.pegasusx.factory.ui.screens.transfer.TransferListScreen

object FactoryRoutes {
    const val LOGIN = "login"
    const val DASHBOARD = "dashboard"
    const val LOADING_BAY = "loading_bay"
    const val TRANSFERS = "transfers"
    const val TRANSFER_DETAIL = "transfers/{id}"
    const val TRANSFER_CREATE = "transfers/create"
    const val FLEET = "fleet"
    const val STAFF = "staff"
    const val LOCATION_SETUP = "location_setup"
    const val LOCATION_SETTINGS = "location_settings"
    const val INSIGHTS = "insights"
    const val ANALYTICS = "analytics"
    const val SUPPLY_REQUESTS = "supply_requests"
    const val PAYLOAD_OVERRIDE = "payload_override"
    const val MANIFEST_EXCEPTIONS = "manifest_exceptions"
    const val MANIFESTS = "manifests"
    const val MANIFEST_DETAIL = "manifests/{id}"
    const val STAFF_DETAIL = "staff/{id}"
    const val NOTIFICATIONS = "notifications"
    const val MORE = "more"

    fun transferDetail(id: String) = "transfers/$id"
    fun manifestDetail(id: String) = "manifests/$id"
    fun staffDetail(id: String) = "staff/$id"
}

private const val MOTION_DURATION = 300

private val compactTabRoutes = FactorySection.compactTabs
    .filter { it != FactorySection.MORE }
    .map { it.route }
    .toSet()

@Composable
fun FactoryNavigation(
    api: FactoryApi,
    geocodeApi: GeocodeApi,
    windowSizeClass: WindowSizeClass,
    navController: NavHostController = rememberNavController(),
) {
    val startDestination = when {
        !TokenHolder.isLoggedIn -> FactoryRoutes.LOGIN
        !TokenHolder.isConfigured -> FactoryRoutes.LOCATION_SETUP
        else -> FactoryRoutes.DASHBOARD
    }
    val context = LocalContext.current
    var networkAvailable by remember { mutableStateOf(true) }
    val useDrawer = windowSizeClass.widthSizeClass != WindowWidthSizeClass.Compact
    var isRailExpanded by remember { mutableStateOf(true) }
    val navBackStackEntry by navController.currentBackStackEntryAsState()
    val currentRoute = navBackStackEntry?.destination?.route
    val showShell = currentRoute != null &&
        currentRoute != FactoryRoutes.LOGIN &&
        currentRoute != FactoryRoutes.LOCATION_SETUP

    fun navigateSection(section: FactorySection) {
        navController.navigate(section.route) {
            popUpTo(FactoryRoutes.DASHBOARD) { saveState = true }
            launchSingleTop = true
            restoreState = true
        }
    }

    fun popOrDashboard() {
        if (!navController.popBackStack()) {
            navigateSection(FactorySection.DASHBOARD)
        }
    }

    fun showBack(route: String): Boolean {
        if (route.contains("{")) return true
        if (useDrawer) return navController.previousBackStackEntry != null
        if (route == FactoryRoutes.TRANSFER_CREATE) return true
        val base = route.substringBefore("/")
        if (base in compactTabRoutes && route == base) return false
        return navController.previousBackStackEntry != null
    }

    fun backFor(route: String): (() -> Unit)? =
        if (showBack(route)) ({ popOrDashboard() }) else null

    fun requireBack(route: String): () -> Unit = backFor(route) ?: { popOrDashboard() }

    fun signOut() {
        TokenHolder.clear()
        navController.navigate(FactoryRoutes.LOGIN) {
            popUpTo(0) { inclusive = true }
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
            composable(FactoryRoutes.LOGIN) {
                LoginScreen(
                    api = api,
                    onLoginSuccess = {
                        val dest = if (TokenHolder.isConfigured) {
                            FactoryRoutes.DASHBOARD
                        } else {
                            FactoryRoutes.LOCATION_SETUP
                        }
                        navController.navigate(dest) {
                            popUpTo(FactoryRoutes.LOGIN) { inclusive = true }
                        }
                    },
                )
            }

            composable(FactoryRoutes.LOCATION_SETUP) {
                LocationSetupScreen(
                    api = api,
                    geocodeApi = geocodeApi,
                    onComplete = {
                        navController.navigate(FactoryRoutes.DASHBOARD) {
                            popUpTo(FactoryRoutes.LOCATION_SETUP) { inclusive = true }
                        }
                    },
                )
            }

            composable(FactoryRoutes.DASHBOARD) {
                DashboardScreen(
                    api = api,
                    onNavigate = { route -> navController.navigate(route) },
                    onSignOut = { signOut() }
                )
            }

            composable(FactoryRoutes.LOADING_BAY) {
                LoadingBayScreen(
                    api = api,
                    onTransferClick = { id -> navController.navigate(FactoryRoutes.transferDetail(id)) },
                    onBack = requireBack(FactoryRoutes.LOADING_BAY),
                )
            }

            composable(FactoryRoutes.TRANSFERS) {
                TransferListScreen(
                    api = api,
                    onTransferClick = { id -> navController.navigate(FactoryRoutes.transferDetail(id)) },
                    onCreateTransfer = { navController.navigate(FactoryRoutes.TRANSFER_CREATE) },
                    onBack = requireBack(FactoryRoutes.TRANSFERS),
                )
            }

            composable(FactoryRoutes.TRANSFER_CREATE) {
                CreateTransferScreen(
                    api = api,
                    onBack = requireBack(FactoryRoutes.TRANSFER_CREATE),
                    onCreated = { id ->
                        navController.navigate(FactoryRoutes.transferDetail(id)) {
                            popUpTo(FactoryRoutes.TRANSFER_CREATE) { inclusive = true }
                        }
                    },
                )
            }

            composable(
                route = FactoryRoutes.TRANSFER_DETAIL,
                arguments = listOf(navArgument("id") { type = NavType.StringType }),
            ) { backStackEntry ->
                val id = backStackEntry.arguments?.getString("id") ?: return@composable
                TransferDetailScreen(
                    api = api,
                    transferId = id,
                    onBack = requireBack(FactoryRoutes.TRANSFER_DETAIL),
                )
            }

            composable(FactoryRoutes.FLEET) {
                FleetScreen(
                    api = api,
                    onBack = requireBack(FactoryRoutes.FLEET),
                )
            }

            composable(FactoryRoutes.STAFF) {
                StaffScreen(
                    api = api,
                    onStaffClick = { id -> navController.navigate(FactoryRoutes.staffDetail(id)) },
                    onBack = requireBack(FactoryRoutes.STAFF),
                )
            }

            composable(FactoryRoutes.LOCATION_SETTINGS) {
                LocationSettingsScreen(
                    api = api,
                    geocodeApi = geocodeApi,
                    onBack = requireBack(FactoryRoutes.LOCATION_SETTINGS),
                )
            }

            composable(
                route = FactoryRoutes.STAFF_DETAIL,
                arguments = listOf(navArgument("id") { type = NavType.StringType }),
            ) { backStackEntry ->
                val id = backStackEntry.arguments?.getString("id") ?: return@composable
                StaffDetailScreen(
                    api = api,
                    staffId = id,
                    onBack = requireBack(FactoryRoutes.STAFF_DETAIL),
                )
            }

            composable(FactoryRoutes.MANIFESTS) {
                ManifestListScreen(
                    api = api,
                    onManifestClick = { id -> navController.navigate(FactoryRoutes.manifestDetail(id)) },
                    onBack = requireBack(FactoryRoutes.MANIFESTS),
                )
            }

            composable(
                route = FactoryRoutes.MANIFEST_DETAIL,
                arguments = listOf(navArgument("id") { type = NavType.StringType }),
            ) { backStackEntry ->
                val id = backStackEntry.arguments?.getString("id") ?: return@composable
                ManifestDetailScreen(
                    api = api,
                    manifestId = id,
                    onBack = requireBack(FactoryRoutes.MANIFEST_DETAIL),
                )
            }

            composable(FactoryRoutes.INSIGHTS) {
                InsightsScreen(
                    api = api,
                    onBack = requireBack(FactoryRoutes.INSIGHTS),
                )
            }

            composable(FactoryRoutes.ANALYTICS) {
                AnalyticsScreen(
                    api = api,
                    onBack = requireBack(FactoryRoutes.ANALYTICS),
                )
            }

            composable(FactoryRoutes.SUPPLY_REQUESTS) {
                SupplyRequestsScreen(
                    api = api,
                    onBack = requireBack(FactoryRoutes.SUPPLY_REQUESTS),
                )
            }

            composable(FactoryRoutes.PAYLOAD_OVERRIDE) {
                PayloadOverrideScreen(
                    api = api,
                    onBack = requireBack(FactoryRoutes.PAYLOAD_OVERRIDE),
                )
            }

            composable(FactoryRoutes.MANIFEST_EXCEPTIONS) {
                ManifestExceptionsScreen(
                    api = api,
                    onBack = requireBack(FactoryRoutes.MANIFEST_EXCEPTIONS),
                )
            }

            composable(FactoryRoutes.NOTIFICATIONS) {
                NotificationInboxScreen(
                    api = api,
                    onBack = requireBack(FactoryRoutes.NOTIFICATIONS),
                )
            }

            composable(FactoryRoutes.MORE) {
                FactoryMoreHubScreen(
                    onNavigate = { route -> navController.navigate(route) },
                )
            }
        }
    }

    if (showShell) {
        Row(Modifier.fillMaxSize()) {
                if (useDrawer) {
                    FactoryNavigationDrawer(
                        isExpanded = isRailExpanded,
                        onToggleExpanded = { isRailExpanded = !isRailExpanded },
                        selectedRoute = currentRoute,
                        onSectionSelected = ::navigateSection,
                        onSignOut = { signOut() },
                    )
                }
                Scaffold(
                    bottomBar = {
                        if (!useDrawer) {
                            FactoryBottomBar(
                                selectedRoute = currentRoute,
                                onSectionSelected = ::navigateSection,
                            )
                        }
                    },
                ) { innerPadding ->
                    navHost(Modifier.padding(innerPadding).fillMaxSize())
                }
            }
        } else {
            navHost(Modifier.fillMaxSize())
        }
}
