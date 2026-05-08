package com.pegasus.warehouse.ui.navigation

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
import com.pegasus.warehouse.data.remote.TokenHolder
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.ui.screens.analytics.AnalyticsScreen
import com.pegasus.warehouse.ui.screens.auth.LoginScreen
import com.pegasus.warehouse.ui.screens.crm.CRMScreen
import com.pegasus.warehouse.ui.screens.dashboard.DashboardScreen
import com.pegasus.warehouse.ui.screens.dispatch.DispatchScreen
import com.pegasus.warehouse.ui.screens.drivers.DriversScreen
import com.pegasus.warehouse.ui.screens.inventory.InventoryScreen
import com.pegasus.warehouse.ui.screens.manifests.ManifestsScreen
import com.pegasus.warehouse.ui.screens.orders.OrderDetailScreen
import com.pegasus.warehouse.ui.screens.orders.OrdersScreen
import com.pegasus.warehouse.ui.screens.products.ProductsScreen
import com.pegasus.warehouse.ui.screens.returns.ReturnsScreen
import com.pegasus.warehouse.ui.screens.staff.StaffScreen
import com.pegasus.warehouse.ui.screens.treasury.TreasuryScreen
import com.pegasus.warehouse.ui.screens.vehicles.VehiclesScreen

object WarehouseRoutes {
    const val LOGIN = "login"
    const val DASHBOARD = "dashboard"
    const val ORDERS = "orders"
    const val ORDER_DETAIL = "orders/{id}"
    const val DRIVERS = "drivers"
    const val VEHICLES = "vehicles"
    const val INVENTORY = "inventory"
    const val PRODUCTS = "products"
    const val MANIFESTS = "manifests"
    const val ANALYTICS = "analytics"
    const val CRM = "crm"
    const val RETURNS = "returns"
    const val TREASURY = "treasury"
    const val DISPATCH = "dispatch"
    const val STAFF = "staff"

    fun orderDetail(id: String) = "orders/$id"
}

private const val MOTION_DURATION = 300

@Composable
fun WarehouseNavigation(
    api: WarehouseApi,
    navController: NavHostController = rememberNavController(),
) {
    val startDestination = if (TokenHolder.isLoggedIn) WarehouseRoutes.DASHBOARD else WarehouseRoutes.LOGIN
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
            composable(WarehouseRoutes.LOGIN) {
                LoginScreen(
                    api = api,
                    onLoginSuccess = {
                        navController.navigate(WarehouseRoutes.DASHBOARD) {
                            popUpTo(WarehouseRoutes.LOGIN) { inclusive = true }
                        }
                    }
                )
            }

            composable(WarehouseRoutes.DASHBOARD) {
                DashboardScreen(
                    api = api,
                    onNavigate = { route -> navController.navigate(route) },
                    onSignOut = {
                        TokenHolder.clear()
                        navController.navigate(WarehouseRoutes.LOGIN) {
                            popUpTo(0) { inclusive = true }
                        }
                    }
                )
            }

            composable(WarehouseRoutes.ORDERS) {
                OrdersScreen(
                    api = api,
                    onOrderClick = { id -> navController.navigate(WarehouseRoutes.orderDetail(id)) },
                    onBack = { navController.popBackStack() },
                )
            }

            composable(
                route = WarehouseRoutes.ORDER_DETAIL,
                arguments = listOf(navArgument("id") { type = NavType.StringType }),
            ) { backStackEntry ->
                val id = backStackEntry.arguments?.getString("id") ?: return@composable
                OrderDetailScreen(
                    api = api,
                    orderId = id,
                    onBack = { navController.popBackStack() },
                )
            }

            composable(WarehouseRoutes.DRIVERS) {
                DriversScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.VEHICLES) {
                VehiclesScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.INVENTORY) {
                InventoryScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.PRODUCTS) {
                ProductsScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.MANIFESTS) {
                ManifestsScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.ANALYTICS) {
                AnalyticsScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.CRM) {
                CRMScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.RETURNS) {
                ReturnsScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.TREASURY) {
                TreasuryScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.DISPATCH) {
                DispatchScreen(api = api, onBack = { navController.popBackStack() })
            }

            composable(WarehouseRoutes.STAFF) {
                StaffScreen(api = api, onBack = { navController.popBackStack() })
            }
        }
    }
}
