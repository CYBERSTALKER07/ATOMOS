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
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.TokenHolder
import com.pegasusx.factory.ui.screens.analytics.AnalyticsScreen
import com.pegasusx.factory.ui.screens.auth.LoginScreen
import com.pegasusx.factory.ui.screens.dashboard.DashboardScreen
import com.pegasusx.factory.ui.screens.fleet.FleetScreen
import com.pegasusx.factory.ui.screens.insights.InsightsScreen
import com.pegasusx.factory.ui.screens.loadingbay.LoadingBayScreen
import com.pegasusx.factory.ui.screens.manifest.ManifestDetailScreen
import com.pegasusx.factory.ui.screens.manifest.ManifestListScreen
import com.pegasusx.factory.ui.screens.exceptions.ManifestExceptionsScreen
import com.pegasusx.factory.ui.screens.override.PayloadOverrideScreen
import com.pegasusx.factory.ui.screens.staff.StaffDetailScreen
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
    const val INSIGHTS = "insights"
    const val ANALYTICS = "analytics"
    const val SUPPLY_REQUESTS = "supply_requests"
    const val PAYLOAD_OVERRIDE = "payload_override"
    const val MANIFEST_EXCEPTIONS = "manifest_exceptions"
    const val MANIFESTS = "manifests"
    const val MANIFEST_DETAIL = "manifests/{id}"
    const val STAFF_DETAIL = "staff/{id}"

    fun transferDetail(id: String) = "transfers/$id"
    fun manifestDetail(id: String) = "manifests/$id"
    fun staffDetail(id: String) = "staff/$id"
}

private const val MOTION_DURATION = 300

@Composable
fun FactoryNavigation(
    api: FactoryApi,
    navController: NavHostController = rememberNavController(),
) {
    val startDestination = if (TokenHolder.isLoggedIn) FactoryRoutes.DASHBOARD else FactoryRoutes.LOGIN
    val lifecycleOwner = LocalLifecycleOwner.current
    val context = LocalContext.current
    var refreshEpoch by remember { mutableIntStateOf(0) }
    var networkAvailable by remember { mutableStateOf(true) }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                refreshEpoch += 1
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
                    if (!networkAvailable) {
                        refreshEpoch += 1
                    }
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

    key(refreshEpoch) {
        NavHost(
            navController = navController,
            startDestination = startDestination,
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
                        navController.navigate(FactoryRoutes.DASHBOARD) {
                            popUpTo(FactoryRoutes.LOGIN) { inclusive = true }
                        }
                    }
                )
            }

            composable(FactoryRoutes.DASHBOARD) {
                DashboardScreen(
                    api = api,
                    onNavigate = { route -> navController.navigate(route) },
                    onSignOut = {
                        TokenHolder.clear()
                        navController.navigate(FactoryRoutes.LOGIN) {
                            popUpTo(0) { inclusive = true }
                        }
                    }
                )
            }

            composable(FactoryRoutes.LOADING_BAY) {
                LoadingBayScreen(
                    api = api,
                    onTransferClick = { id -> navController.navigate(FactoryRoutes.transferDetail(id)) },
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.TRANSFERS) {
                TransferListScreen(
                    api = api,
                    onTransferClick = { id -> navController.navigate(FactoryRoutes.transferDetail(id)) },
                    onCreateTransfer = { navController.navigate(FactoryRoutes.TRANSFER_CREATE) },
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.TRANSFER_CREATE) {
                CreateTransferScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
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
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.FLEET) {
                FleetScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.STAFF) {
                StaffScreen(
                    api = api,
                    onStaffClick = { id -> navController.navigate(FactoryRoutes.staffDetail(id)) },
                    onBack = { navController.popBackStack() },
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
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.MANIFESTS) {
                ManifestListScreen(
                    api = api,
                    onManifestClick = { id -> navController.navigate(FactoryRoutes.manifestDetail(id)) },
                    onBack = { navController.popBackStack() },
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
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.INSIGHTS) {
                InsightsScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.ANALYTICS) {
                AnalyticsScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.SUPPLY_REQUESTS) {
                SupplyRequestsScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.PAYLOAD_OVERRIDE) {
                PayloadOverrideScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
                )
            }

            composable(FactoryRoutes.MANIFEST_EXCEPTIONS) {
                ManifestExceptionsScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
                )
            }
        }
    }
}
