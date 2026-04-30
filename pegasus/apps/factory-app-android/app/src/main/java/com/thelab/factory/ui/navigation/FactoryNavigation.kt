package com.thelab.factory.ui.navigation

import androidx.compose.animation.AnimatedContentTransitionScope
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.runtime.Composable
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.thelab.factory.data.remote.FactoryApi
import com.thelab.factory.data.remote.TokenHolder
import com.thelab.factory.ui.screens.auth.LoginScreen
import com.thelab.factory.ui.screens.dashboard.DashboardScreen
import com.thelab.factory.ui.screens.fleet.FleetScreen
import com.thelab.factory.ui.screens.insights.InsightsScreen
import com.thelab.factory.ui.screens.loadingbay.LoadingBayScreen
import com.thelab.factory.ui.screens.staff.StaffScreen
import com.thelab.factory.ui.screens.transfer.TransferDetailScreen
import com.thelab.factory.ui.screens.transfer.TransferListScreen

object FactoryRoutes {
    const val LOGIN = "login"
    const val DASHBOARD = "dashboard"
    const val LOADING_BAY = "loading_bay"
    const val TRANSFERS = "transfers"
    const val TRANSFER_DETAIL = "transfers/{id}"
    const val FLEET = "fleet"
    const val STAFF = "staff"
    const val INSIGHTS = "insights"

    fun transferDetail(id: String) = "transfers/$id"
}

private const val MOTION_DURATION = 300

@Composable
fun FactoryNavigation(
    api: FactoryApi,
    navController: NavHostController = rememberNavController(),
) {
    val startDestination = if (TokenHolder.isLoggedIn) FactoryRoutes.DASHBOARD else FactoryRoutes.LOGIN

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
                onBack = { navController.popBackStack() },
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
                onBack = { navController.popBackStack() },
            )
        }

        composable(FactoryRoutes.INSIGHTS) {
            InsightsScreen(
                api = api,
                onBack = { navController.popBackStack() },
            )
        }
    }
}
