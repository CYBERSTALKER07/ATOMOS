package com.pegasusx.driver.ui.navigation

import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Handler
import android.os.Looper
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.collectAsState
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import java.net.URLEncoder
import androidx.compose.material3.Button
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.windowsizeclass.WindowSizeClass
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.BuildConfig
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.DriverOutdatedState
import com.pegasusx.driver.data.remote.DriverWebSocket
import com.pegasusx.driver.services.OfflineSyncScheduler
import com.pegasusx.driver.service.AutoUpdater
import com.pegasusx.driver.service.EnterpriseUpdateConfig
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.ui.components.ClientPolicyBanner
import com.pegasusx.driver.ui.screens.auth.LoginScreen
import com.pegasusx.driver.ui.screens.home.HomeScreen
import com.pegasusx.driver.ui.screens.manifest.DeliveryCorrectionScreen
import com.pegasusx.driver.ui.screens.manifest.ManifestScreen
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.ui.components.ActiveRideBar
import com.pegasusx.driver.ui.screens.manifest.ManifestViewModel
import com.pegasusx.driver.ui.screens.map.MapScreen
import com.pegasusx.driver.ui.screens.offload.CashCollectionScreen
import com.pegasusx.driver.ui.screens.offload.OffloadReviewScreen
import com.pegasusx.driver.ui.screens.offload.PaymentWaitingScreen
import com.pegasusx.driver.ui.screens.offload.ShopClosedWaitingScreen
import com.pegasusx.driver.ui.screens.profile.ProfileScreen
import com.pegasusx.driver.ui.screens.scanner.ScannerScreen
import com.pegasusx.driver.ui.screens.notifications.DriverNotificationInboxScreen
import com.pegasusx.driver.ui.screens.offline.OfflineVerifierScreen
import com.pegasusx.driver.ui.screens.supply.SupplyTransfersScreen
import com.pegasusx.driver.ui.theme.MotionTokens
import kotlinx.coroutines.launch

object DriverRoutes {
    const val LOGIN = "login"
    const val MAIN = "main"
    const val SCANNER = "scanner"
    const val NOTIFICATIONS = "notifications"
    const val CORRECTION = "correction/{orderId}/{retailerName}"
    const val OFFLOAD_REVIEW = "offload_review/{orderId}/{retailerName}?scannedToken={scannedToken}"
    const val PAYMENT_WAITING = "payment_waiting/{orderId}/{amount}"
    const val CASH_COLLECTION = "cash_collection/{orderId}/{amount}"
    const val SHOP_CLOSED_WAITING = "shop_closed_waiting/{orderId}"
    const val OFFLINE_VERIFIER = "offline_verifier"
    const val SUPPLY_TRANSFERS = "supply_transfers"

    fun correctionRoute(orderId: String, retailerName: String): String {
        val encodedName = URLEncoder.encode(retailerName.ifBlank { "_" }, "UTF-8")
        return "correction/$orderId/$encodedName"
    }

    fun offloadReviewRoute(orderId: String, retailerName: String, scannedToken: String = ""): String {
        val encodedName = URLEncoder.encode(retailerName.ifBlank { "_" }, "UTF-8")
        val encodedToken = URLEncoder.encode(scannedToken, "UTF-8")
        return "offload_review/$orderId/$encodedName?scannedToken=$encodedToken"
    }

    fun paymentWaitingRoute(orderId: String, amount: Long): String =
        "payment_waiting/$orderId/$amount"

    fun cashCollectionRoute(orderId: String, amount: Long): String =
        "cash_collection/$orderId/$amount"

    fun shopClosedWaitingRoute(orderId: String): String =
        "shop_closed_waiting/$orderId"
}

@Composable
fun DriverNavigation(
    api: DriverApi,
    driverWebSocket: DriverWebSocket,
    windowSizeClass: WindowSizeClass? = null,
) {
    val navController = rememberNavController()
    val snackbarHostState = remember { SnackbarHostState() }
    var handoffTabRequest by remember { mutableStateOf<AppTab?>(null) }
    val startDest = if (TokenHolder.token != null) DriverRoutes.MAIN else DriverRoutes.LOGIN
    val lifecycleOwner = LocalLifecycleOwner.current
    val context = LocalContext.current
    val outdatedState by driverWebSocket.outdatedState.collectAsState()
    var networkAvailable by remember { mutableStateOf(true) }
    var clientPolicyMessage by remember { mutableStateOf<String?>(null) }
    var clientPolicyForce by remember { mutableStateOf(false) }
    var pendingManifest by remember { mutableStateOf<AutoUpdater.Manifest?>(null) }
    val scope = rememberCoroutineScope()
    val autoUpdater = remember { AutoUpdater(context.applicationContext) }

    DisposableEffect(autoUpdater) {
        autoUpdater.register()
        onDispose { autoUpdater.cleanup() }
    }

    fun loadClientPolicy() {
        scope.launch {
            try {
                val policy = api.getClientPolicy(
                    role = EnterpriseUpdateConfig.POLICY_ROLE,
                    platform = "android",
                    version = BuildConfig.VERSION_NAME,
                    channel = EnterpriseUpdateConfig.CHANNEL,
                )
                val state = autoUpdater.checkFromPolicy(policy, autoDownload = false)
                clientPolicyMessage = state.message
                clientPolicyForce = state.force
                pendingManifest = state.manifest
            } catch (_: Exception) {
                // Policy fetch is optional on local/dev stacks.
            }
        }
    }

    fun onUpdateClick() {
        scope.launch {
            autoUpdater.startUpdate(pendingManifest)
        }
    }

    LaunchedEffect(Unit) {
        loadClientPolicy()
    }

    DisposableEffect(Unit) {
        connectDriverSocketIfPossible(driverWebSocket)
        onDispose {
            driverWebSocket.disconnect()
            driverWebSocket.clearOutdatedState()
        }
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                connectDriverSocketIfPossible(driverWebSocket)
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
                        OfflineSyncScheduler.enqueue(context)
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

    Column(modifier = Modifier.fillMaxSize()) {
        ClientPolicyBanner(
            message = clientPolicyMessage,
            force = clientPolicyForce,
            onUpdate = if (clientPolicyMessage != null) {
                { onUpdateClick() }
            } else {
                null
            },
            onDismiss = if (!clientPolicyForce) {
                { clientPolicyMessage = null }
            } else {
                null
            },
        )
        Box(modifier = Modifier.weight(1f)) {
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
            val manifestState by manifestViewModel.state.collectAsState()
            val activeRideOrder = manifestState.orders.firstOrNull { order ->
                order.state != OrderState.COMPLETED &&
                    order.state != OrderState.CANCELLED &&
                    order.latitude != null &&
                    order.longitude != null
            }
            MainTabView(
                requestedTab = handoffTabRequest,
                onRequestedTabConsumed = { handoffTabRequest = null },
                homeContent = {
                    HomeScreen(
                        api = api,
                        viewModel = manifestViewModel,
                        onOpenMap = { /* Map tab handled internally by MainTabView */ },
                        onScanQR = { navController.navigate(DriverRoutes.SCANNER) },
                        onOfflineVerify = { navController.navigate(DriverRoutes.OFFLINE_VERIFIER) },
                        onResumeCashCollection = { orderId, amount ->
                            navController.navigate(DriverRoutes.cashCollectionRoute(orderId, amount))
                        },
                        onNotificationsClick = { navController.navigate(DriverRoutes.NOTIFICATIONS) { launchSingleTop = true } },
                        onOpenSupplyTransfers = { navController.navigate(DriverRoutes.SUPPLY_TRANSFERS) },
                    )
                },
                mapContent = {
                    MapScreen(
                        api = api,
                        viewModel = manifestViewModel,
                        onOpenScanner = { navController.navigate(DriverRoutes.SCANNER) },
                        onOpenCorrection = { orderId, retailerName ->
                            navController.navigate(DriverRoutes.correctionRoute(orderId, retailerName))
                        },
                    )
                },
                ridesContent = {
                    ManifestScreen(
                        viewModel = manifestViewModel,
                        onOrderClick = { order ->
                            navController.navigate(
                                DriverRoutes.correctionRoute(order.id, order.retailerName)
                            )
                        },
                        onRequestEarlyComplete = { reason, note ->
                            manifestViewModel.requestEarlyComplete(reason, note)
                        },
                    )
                },
                profileContent = {
                    ProfileScreen(viewModel = manifestViewModel)
                },
                activeRideBar = { onOpenMap ->
                    ActiveRideBar(
                        visible = activeRideOrder != null,
                        order = activeRideOrder,
                        onClick = onOpenMap,
                    )
                },
                windowSizeClass = windowSizeClass,
            )
        }

        composable(DriverRoutes.OFFLINE_VERIFIER) {
            OfflineVerifierScreen(onBack = { navController.popBackStack() })
        }

        composable(DriverRoutes.SUPPLY_TRANSFERS) {
            SupplyTransfersScreen(onBack = { navController.popBackStack() })
        }

        composable(DriverRoutes.SCANNER) {
            ScannerScreen(
                onClose = { navController.popBackStack() },
                onValidated = { validated, scannedToken ->
                    navController.popBackStack()
                    navController.navigate(
                        DriverRoutes.offloadReviewRoute(
                            validated.orderId,
                            validated.retailerName,
                            scannedToken,
                        )
                    )
                }
            )
        }

        composable(DriverRoutes.NOTIFICATIONS) {
            DriverNotificationInboxScreen(
                onBack = { navController.popBackStack() },
                onHandoffAction = { link ->
                    when (val dest = HandoffPathResolver.resolve(link)) {
                        HandoffDestination.FleetMap -> {
                            handoffTabRequest = AppTab.MAP
                            navController.popBackStack()
                        }
                        HandoffDestination.Home -> {
                            handoffTabRequest = AppTab.HOME
                            navController.popBackStack()
                        }
                        HandoffDestination.ManifestList -> {
                            handoffTabRequest = AppTab.RIDES
                            navController.popBackStack()
                        }
                        is HandoffDestination.ManifestDetail -> {
                            handoffTabRequest = AppTab.RIDES
                            navController.popBackStack()
                        }
                        is HandoffDestination.OrderDetail -> {
                            navController.popBackStack()
                            navController.navigate(DriverRoutes.correctionRoute(dest.orderId, ""))
                        }
                        HandoffDestination.Unresolved -> {
                            scope.launch {
                                snackbarHostState.showSnackbar("Open in portal — no native route for this link")
                            }
                        }
                    }
                },
                onHandoffUnresolved = {
                    scope.launch {
                        snackbarHostState.showSnackbar("Open in portal — no native route for this link")
                    }
                },
            )
        }

        composable(
            route = DriverRoutes.OFFLOAD_REVIEW,
            arguments = listOf(
                navArgument("orderId") { type = NavType.StringType },
                navArgument("retailerName") { type = NavType.StringType },
                navArgument("scannedToken") {
                    type = NavType.StringType
                    defaultValue = ""
                },
            )
        ) {
            OffloadReviewScreen(
                onClose = { navController.popBackStack() },
                onOffloadConfirmed = { response ->
                    navController.popBackStack()
                    navController.navigate(
                        DriverRoutes.paymentWaitingRoute(response.orderId, response.amount)
                    )
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
                },
                onCashCollectionRequired = { orderId, amount ->
                    navController.popBackStack()
                    navController.navigate(DriverRoutes.cashCollectionRoute(orderId, amount))
                },
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

        if (TokenHolder.token != null && outdatedState != null) {
            DriverOutdatedOverlay(
                outdatedState = outdatedState!!,
                onSignOut = {
                    TokenHolder.clear()
                    driverWebSocket.disconnect()
                    driverWebSocket.clearOutdatedState()
                    navController.navigate(DriverRoutes.LOGIN) {
                        popUpTo(navController.graph.startDestinationId) { inclusive = true }
                        launchSingleTop = true
                    }
                }
            )
        }
    }
        SnackbarHost(hostState = snackbarHostState)
    }
}

private fun connectDriverSocketIfPossible(driverWebSocket: DriverWebSocket) {
    val token = TokenHolder.token
    val driverId = TokenHolder.userId
    if (!token.isNullOrBlank() && !driverId.isNullOrBlank()) {
        driverWebSocket.connect(BuildConfig.API_BASE_URL, driverId, token)
    }
}

@Composable
private fun DriverOutdatedOverlay(
    outdatedState: DriverOutdatedState,
    onSignOut: () -> Unit
) {
    val blocked = outdatedState.blockedEventType ?: "this operation"
    val required = outdatedState.requiredSchemaVersion?.toString() ?: "latest"

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.surface.copy(alpha = 0.98f))
            .padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "App Update Required",
                style = MaterialTheme.typography.headlineSmall,
                color = MaterialTheme.colorScheme.onSurface,
            )
            Text(
                text = outdatedState.message,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                text = "Blocked event: $blocked | Required schema: $required",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Button(onClick = onSignOut) {
                Text(text = "Sign Out")
            }
        }
    }
}
