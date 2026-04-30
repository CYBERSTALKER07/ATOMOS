package com.pegasus.driver.ui.navigation

import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.runtime.Composable
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import java.net.URLEncoder
import com.pegasus.driver.data.remote.DriverApi
import com.pegasus.driver.data.remote.TokenHolder
import com.pegasus.driver.ui.screens.auth.LoginScreen
import com.pegasus.driver.ui.screens.home.HomeScreen
import com.pegasus.driver.ui.screens.manifest.DeliveryCorrectionScreen
import com.pegasus.driver.ui.screens.manifest.ManifestScreen
import com.pegasus.driver.ui.screens.manifest.ManifestViewModel
import com.pegasus.driver.ui.screens.map.MapScreen
import com.pegasus.driver.ui.screens.offload.CashCollectionScreen
import com.pegasus.driver.ui.screens.offload.OffloadReviewScreen
import com.pegasus.driver.ui.screens.offload.PaymentWaitingScreen
import com.pegasus.driver.ui.screens.offload.ShopClosedWaitingScreen
import com.pegasus.driver.ui.screens.profile.ProfileScreen
import com.pegasus.driver.ui.screens.scanner.ScannerScreen
import com.pegasus.driver.ui.screens.notifications.DriverNotificationInboxScreen
import com.pegasus.driver.ui.theme.MotionTokens

object DriverRoutes {
    const val LOGIN = "login"
    const val MAIN = "main"
    const val SCANNER = "scanner"
    const val NOTIFICATIONS = "notifications"
    const val CORRECTION = "correction/{orderId}/{retailerName}"
    const val OFFLOAD_REVIEW = "offload_review/{orderId}/{retailerName}"
    const val PAYMENT_WAITING = "payment_waiting/{orderId}/{amount}"
    const val CASH_COLLECTION = "cash_collection/{orderId}/{amount}"
    const val SHOP_CLOSED_WAITING = "shop_closed_waiting/{orderId}"

    fun correctionRoute(orderId: String, retailerName: String): String {
        val encodedName = URLEncoder.encode(retailerName.ifBlank { "_" }, "UTF-8")
        return "correction/$orderId/$encodedName"
    }

    fun offloadReviewRoute(orderId: String, retailerName: String): String {
        val encodedName = URLEncoder.encode(retailerName.ifBlank { "_" }, "UTF-8")
        return "offload_review/$orderId/$encodedName"
    }

    fun paymentWaitingRoute(orderId: String, amount: Long): String =
        "payment_waiting/$orderId/$amount"

    fun cashCollectionRoute(orderId: String, amount: Long): String =
        "cash_collection/$orderId/$amount"

    fun shopClosedWaitingRoute(orderId: String): String =
        "shop_closed_waiting/$orderId"
}

@Composable
fun DriverNavigation(api: DriverApi) {
    val navController = rememberNavController()
    val startDest = if (TokenHolder.token != null) DriverRoutes.MAIN else DriverRoutes.LOGIN

    NavHost(
        navController = navController,
        startDestination = startDest,
        enterTransition = {
            slideInHorizontally(
                initialOffsetX = { it / 5 },
                animationSpec = tween(MotionTokens.DurationMedium4, easing = MotionTokens.EasingEmphasizedDecelerate)
            ) + fadeIn(tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate))
        },
        exitTransition = {
            fadeOut(tween(MotionTokens.DurationShort3, easing = MotionTokens.EasingEmphasizedAccelerate))
        },
        popEnterTransition = {
            slideInHorizontally(
                initialOffsetX = { -it / 5 },
                animationSpec = tween(MotionTokens.DurationMedium4, easing = MotionTokens.EasingEmphasizedDecelerate)
            ) + fadeIn(tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate))
        },
        popExitTransition = {
            slideOutHorizontally(
                targetOffsetX = { it / 5 },
                animationSpec = tween(MotionTokens.DurationShort4, easing = MotionTokens.EasingEmphasizedAccelerate)
            ) + fadeOut(tween(MotionTokens.DurationShort3, easing = MotionTokens.EasingEmphasizedAccelerate))
        },
    ) {
        composable(DriverRoutes.LOGIN) {
            LoginScreen(
                api = api,
                onLoginSuccess = {
                    navController.navigate(DriverRoutes.MAIN) {
                        popUpTo(DriverRoutes.LOGIN) { inclusive = true }
                    }
                }
            )
        }

        composable(DriverRoutes.MAIN) {
            val manifestViewModel: ManifestViewModel = hiltViewModel()
            MainTabView(
                homeContent = {
                    HomeScreen(
                        viewModel = manifestViewModel,
                        onOpenMap = { /* Map tab handled internally by MainTabView */ },
                        onScanQR = { navController.navigate(DriverRoutes.SCANNER) },
                        onNotificationsClick = { navController.navigate(DriverRoutes.NOTIFICATIONS) { launchSingleTop = true } },
                    )
                },
                mapContent = {
                    MapScreen(viewModel = manifestViewModel)
                },
                ridesContent = {
                    ManifestScreen(viewModel = manifestViewModel)
                },
                profileContent = {
                    ProfileScreen(viewModel = manifestViewModel)
                }
            )
        }

        composable(DriverRoutes.SCANNER) {
            ScannerScreen(
                onClose = { navController.popBackStack() },
                onValidated = { validated ->
                    navController.popBackStack()
                    navController.navigate(
                        DriverRoutes.offloadReviewRoute(validated.orderId, validated.retailerName)
                    )
                }
            )
        }

        composable(DriverRoutes.NOTIFICATIONS) {
            DriverNotificationInboxScreen(onBack = { navController.popBackStack() })
        }

        composable(
            route = DriverRoutes.OFFLOAD_REVIEW,
            arguments = listOf(
                navArgument("orderId") { type = NavType.StringType },
                navArgument("retailerName") { type = NavType.StringType }
            )
        ) {
            OffloadReviewScreen(
                onClose = { navController.popBackStack() },
                onOffloadConfirmed = { response ->
                    navController.popBackStack()
                    if (response.paymentMethod == "cash") {
                        navController.navigate(
                            DriverRoutes.cashCollectionRoute(response.orderId, response.amount)
                        )
                    } else {
                        navController.navigate(
                            DriverRoutes.paymentWaitingRoute(response.orderId, response.amount)
                        )
                    }
                },
                onShopClosed = { orderId ->
                    navController.popBackStack()
                    navController.navigate(DriverRoutes.shopClosedWaitingRoute(orderId))
                }
            )
        }

        composable(
            route = DriverRoutes.PAYMENT_WAITING,
            arguments = listOf(
                navArgument("orderId") { type = NavType.StringType },
                navArgument("amount") { type = NavType.LongType }
            )
        ) {
            PaymentWaitingScreen(
                onComplete = {
                    navController.popBackStack(DriverRoutes.MAIN, inclusive = false)
                }
            )
        }

        composable(
            route = DriverRoutes.CASH_COLLECTION,
            arguments = listOf(
                navArgument("orderId") { type = NavType.StringType },
                navArgument("amount") { type = NavType.LongType }
            )
        ) {
            CashCollectionScreen(
                onComplete = {
                    navController.popBackStack(DriverRoutes.MAIN, inclusive = false)
                }
            )
        }

        composable(
            route = DriverRoutes.CORRECTION,
            arguments = listOf(
                navArgument("orderId") { type = NavType.StringType },
                navArgument("retailerName") { type = NavType.StringType }
            )
        ) {
            DeliveryCorrectionScreen(
                onClose = { navController.popBackStack() },
                onComplete = {
                    navController.popBackStack(DriverRoutes.MAIN, inclusive = false)
                }
            )
        }

        composable(
            route = DriverRoutes.SHOP_CLOSED_WAITING,
            arguments = listOf(
                navArgument("orderId") { type = NavType.StringType }
            )
        ) { backStackEntry ->
            val orderId = backStackEntry.arguments?.getString("orderId") ?: ""
            ShopClosedWaitingScreen(
                orderId = orderId,
                onClose = { navController.popBackStack() },
                onBypassComplete = {
                    navController.popBackStack(DriverRoutes.MAIN, inclusive = false)
                },
                onReturnToDepot = {
                    navController.popBackStack(DriverRoutes.MAIN, inclusive = false)
                }
            )
        }
    }
}
