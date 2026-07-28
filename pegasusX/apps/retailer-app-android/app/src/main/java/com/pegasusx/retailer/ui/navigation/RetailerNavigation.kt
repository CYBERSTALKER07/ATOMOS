package com.pegasusx.retailer.ui.navigation

import android.content.Intent
import android.net.Uri
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import com.pegasusx.retailer.ui.theme.MotionTokens
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.ui.components.ActiveDeliveriesSheet
import com.pegasusx.retailer.ui.components.DeliveryPaymentSheet
import com.pegasusx.retailer.ui.components.FloatingActiveOrdersBar
import com.pegasusx.retailer.ui.components.ShopClosedSheet
import com.pegasusx.retailer.ui.components.PegasusBottomBar
import com.pegasusx.retailer.ui.components.PegasusTab
import com.pegasusx.retailer.ui.components.PegasusTopBar
import com.pegasusx.retailer.ui.components.FileClaimHost
import com.pegasusx.retailer.ui.components.OrderDetailSheet
import com.pegasusx.retailer.ui.components.PaymentPhase
import com.pegasusx.retailer.ui.components.QROverlay
import androidx.compose.material3.windowsizeclass.WindowSizeClass
import androidx.compose.material3.windowsizeclass.WindowWidthSizeClass
import androidx.compose.foundation.layout.Row
import com.pegasusx.retailer.ui.components.PegasusNavigationRail
import com.pegasusx.retailer.ui.components.SidebarMenu
import com.pegasusx.retailer.ui.screens.cart.CartScreen
import com.pegasusx.retailer.ui.screens.cart.CartViewModel
import com.pegasusx.retailer.ui.screens.profile.AccountProfileScreen
import com.pegasusx.retailer.ui.screens.profile.ProfileScreen
import com.pegasusx.retailer.ui.screens.profile.FamilyMembersScreen
import com.pegasusx.retailer.ui.screens.profile.SavedCardsScreen
import com.pegasusx.retailer.ui.screens.catalog.CatalogScreen
import com.pegasusx.retailer.ui.screens.catalog.CategorySuppliersScreen
import com.pegasusx.retailer.ui.screens.dashboard.DashboardScreen
import com.pegasusx.retailer.ui.screens.orders.OrdersScreen
import com.pegasusx.retailer.ui.screens.procurement.ProcurementScreen
import com.pegasusx.retailer.ui.screens.suppliers.MySuppliersScreen
import com.pegasusx.retailer.ui.screens.analytics.AnalyticsScreen
import com.pegasusx.retailer.ui.screens.autoorder.AutoOrderScreen
import com.pegasusx.retailer.ui.screens.product.ProductDetailScreen
import com.pegasusx.retailer.ui.screens.suppliers.SupplierCatalogScreen
import com.pegasusx.retailer.ui.components.ClientPolicyBanner
import com.pegasusx.retailer.ui.screens.predictions.FutureDemandScreen
import com.pegasusx.retailer.ui.screens.tracking.DeliveriesHubScreen
import com.pegasusx.retailer.ui.screens.notifications.NotificationInboxScreen
import com.pegasusx.retailer.ui.controltower.ControlTowerScreen
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RetailerNavigation(
    windowSizeClass: WindowSizeClass,
    navigationViewModel: NavigationViewModel = hiltViewModel(),
) {
    val navController = rememberNavController()
    val navState by navigationViewModel.uiState.collectAsState()
    val cartViewModel: CartViewModel = hiltViewModel()
    val cartState by cartViewModel.uiState.collectAsState()
    var currentTab by rememberSaveable { mutableStateOf(PegasusTab.HOME) }
    val cartBadge = cartState.totalItems

    // Sidebar state
    var sidebarOpen by rememberSaveable { mutableStateOf(false) }
    var railExpanded by rememberSaveable { mutableStateOf(false) }

    // Active deliveries sheet (notification bell)
    var showActiveDeliveries by rememberSaveable { mutableStateOf(false) }

    // Global detail/QR state (hoisted so overlays render above sheets)
    var globalDetailOrder by remember { mutableStateOf<Order?>(null) }
    var globalQROrder by remember { mutableStateOf<Order?>(null) }
    var claimOrder by remember { mutableStateOf<Order?>(null) }

    // Payment sheet state
    var paymentPhase by remember { mutableStateOf(PaymentPhase.CHOOSE) }
    var paymentError by remember { mutableStateOf<String?>(null) }
    val coroutineScope = rememberCoroutineScope()

    // Show floating bar on primary operational tabs.
    val showFloatingBar = currentTab in listOf(PegasusTab.HOME, PegasusTab.CATALOG, PegasusTab.ORDERS, PegasusTab.MAP)
    val isCompact = windowSizeClass.widthSizeClass == WindowWidthSizeClass.Compact
    val topBarTitle = if (currentTab in PegasusTab.PrimaryTabs) currentTab.label else "Retailer"

    Box(modifier = Modifier.fillMaxSize()) {
        Row(modifier = Modifier.fillMaxSize()) {
            if (!isCompact) {
                PegasusNavigationRail(
                    isExpanded = railExpanded,
                    onToggleExpanded = { railExpanded = !railExpanded },
                    currentTab = currentTab,
                    userName = navState.userName,
                    companyName = navState.companyName,
                    onSidebarNavigate = { dest ->
                        when (dest) {
                            com.pegasusx.retailer.ui.components.SidebarDestination.DASHBOARD -> {
                                currentTab = PegasusTab.HOME
                                navController.navigate(PegasusTab.HOME.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.pegasusx.retailer.ui.components.SidebarDestination.PROCUREMENT -> {
                                navController.navigate("PROCUREMENT") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.pegasusx.retailer.ui.components.SidebarDestination.AI_PREDICTIONS -> {
                                navController.navigate("FUTURE_DEMAND") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.pegasusx.retailer.ui.components.SidebarDestination.CONTROL_TOWER -> {
                                navController.navigate("CONTROL_TOWER") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.pegasusx.retailer.ui.components.SidebarDestination.DOCK -> {
                                currentTab = PegasusTab.MAP
                                navController.navigate("DOCK") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.pegasusx.retailer.ui.components.SidebarDestination.INSIGHTS -> {
                                navController.navigate("ANALYTICS") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.pegasusx.retailer.ui.components.SidebarDestination.AUTO_ORDER -> {
                                navController.navigate("AUTO_ORDER") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.pegasusx.retailer.ui.components.SidebarDestination.INBOX -> {
                                navController.navigate("NOTIFICATIONS") {
                                    launchSingleTop = true
                                }
                            }
                            com.pegasusx.retailer.ui.components.SidebarDestination.PROFILE,
                            com.pegasusx.retailer.ui.components.SidebarDestination.SETTINGS -> {
                                navController.navigate(PegasusTab.PROFILE.name) {
                                    launchSingleTop = true
                                }
                            }
                            else -> { /* unhandled sidebar destination */ }
                        }
                        railExpanded = false // Collapse after selection if desired
                    },
                    onTabSelected = { tab ->
                        if (tab != currentTab) {
                            currentTab = tab
                            navController.navigate(tab.name) {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                                restoreState = true
                            }
                        }
                    }
                )
            }
            val navSyncMessage = when {
                navState.loadIssue != null -> navState.syncError ?: navState.syncMessage.orEmpty()
                navState.isRefreshing -> "Syncing active orders..."
                else -> null
            }

            Scaffold(
            topBar = {
                PegasusTopBar(
                    onAvatarCash = { sidebarOpen = true },
                    onCartCash = {
                        navController.navigate("CART") {
                            launchSingleTop = true
                        }
                    },
                    onNotificationClick = {
                        navController.navigate("NOTIFICATIONS") {
                            launchSingleTop = true
                        }
                    },
                    cartBadge = cartBadge,
                    notificationBadge = navState.unreadNotificationCount,
                    avatarInitial = navState.avatarInitial,
                    title = topBarTitle,
                )
            },
            bottomBar = {
                Column {
                    // Floating active orders bar above bottom nav
                    FloatingActiveOrdersBar(
                        visible = showFloatingBar && navState.activeOrderCount > 0,
                        orderCount = navState.activeOrderCount,
                        statusText = navState.floatingStatusText,
                        totalDisplay = navState.floatingTotalDisplay,
                        countdownIso = navState.floatingCountdownIso,
                        onClick = { showActiveDeliveries = true },
                    )
                    if (isCompact) {
                        PegasusBottomBar(
                            currentTab = currentTab,
                            onTabSelected = { tab ->
                                if (tab != currentTab) {
                                    currentTab = tab
                                    navController.navigate(tab.name) {
                                        popUpTo(navController.graph.startDestinationId) { saveState = true }
                                        launchSingleTop = true
                                        restoreState = true
                                    }
                                }
                            },
                        )
                    }
                }
            },
        ) { innerPadding ->
            Column(modifier = Modifier.fillMaxSize().padding(innerPadding)) {
                ClientPolicyBanner(
                    message = navState.clientPolicyMessage,
                    force = navState.clientPolicyForce,
                    onUpdate = if (navState.clientPolicyMessage != null) {
                        { navigationViewModel.onUpdateClick() }
                    } else {
                        null
                    },
                    onDismiss = if (!navState.clientPolicyForce) {
                        { navigationViewModel.dismissClientPolicyBanner() }
                    } else {
                        null
                    },
                )
                RetailerOperationsStrip(
                    navState = navState,
                    navSyncMessage = navSyncMessage,
                    onRetrySync = navigationViewModel::retrySync,
                    onOpenDeliveries = {
                        currentTab = PegasusTab.MAP
                        showActiveDeliveries = true
                        navController.navigate(PegasusTab.MAP.name) {
                            popUpTo(navController.graph.startDestinationId) { saveState = true }
                            launchSingleTop = true
                            restoreState = true
                        }
                    },
                    onReviewPayment = {
                        paymentPhase = PaymentPhase.CHOOSE
                        paymentError = null
                        if (navState.paymentEvent == null) {
                            navigationViewModel.loadPendingPayments()
                        }
                    },
                )

                NavHost(
                    navController = navController,
                    startDestination = PegasusTab.HOME.name,
                    modifier = Modifier.fillMaxSize().weight(1f),
                    enterTransition = {
                        slideInHorizontally(
                            initialOffsetX = { (it * 0.20).toInt() },
                            animationSpec = tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate),
                        ) + fadeIn(tween(MotionTokens.DurationShort4, easing = MotionTokens.EasingEmphasizedDecelerate))
                    },
                    exitTransition = {
                        fadeOut(tween(MotionTokens.DurationShort2, easing = MotionTokens.EasingEmphasizedAccelerate))
                    },
                    popEnterTransition = {
                        slideInHorizontally(
                            initialOffsetX = { -(it * 0.20).toInt() },
                            animationSpec = tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate),
                        ) + fadeIn(tween(MotionTokens.DurationShort4, easing = MotionTokens.EasingEmphasizedDecelerate))
                    },
                    popExitTransition = {
                        slideOutHorizontally(
                            targetOffsetX = { (it * 0.20).toInt() },
                            animationSpec = tween(MotionTokens.DurationShort4, easing = MotionTokens.EasingEmphasizedAccelerate),
                        ) + fadeOut(tween(MotionTokens.DurationShort2, easing = MotionTokens.EasingEmphasizedAccelerate))
                    },
                ) {
                composable(PegasusTab.HOME.name) {
                    Box(Modifier.fillMaxSize()) {
                        DashboardScreen(
                            onOpenCatalog = {
                                currentTab = PegasusTab.CATALOG
                                navController.navigate(PegasusTab.CATALOG.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            onOpenOrders = {
                                currentTab = PegasusTab.ORDERS
                                navController.navigate(PegasusTab.ORDERS.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            onOpenInsights = {
                                navController.navigate("ANALYTICS") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            },
                            onOpenSuppliers = {
                                currentTab = PegasusTab.CATALOG
                                navController.navigate(PegasusTab.SUPPLIERS.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            onOpenProcurement = {
                                navController.navigate("PROCUREMENT") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            },
                            onOpenProfile = {
                                navController.navigate(PegasusTab.PROFILE.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            onOpenDeliveries = {
                                currentTab = PegasusTab.MAP
                                navController.navigate(PegasusTab.MAP.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            onQuickReorder = { product ->
                                navController.navigate("PRODUCT_DETAIL/${Uri.encode(product.id)}")
                            },
                        )
                    }
                }
                composable(PegasusTab.CATALOG.name) {
                    Box(Modifier.fillMaxSize()) {
                        CatalogScreen(
                            onProductCash = { productId ->
                                navController.navigate("PRODUCT_DETAIL/$productId")
                            },
                            onCategoryCash = { categoryId, categoryName ->
                                navController.navigate("CATEGORY_SUPPLIERS/${Uri.encode(categoryId)}/${Uri.encode(categoryName)}")
                            },
                            onNavigateToSuppliers = {
                                currentTab = PegasusTab.SUPPLIERS
                                navController.navigate(PegasusTab.SUPPLIERS.name) {
                                    launchSingleTop = true
                                }
                            },
                        )
                    }
                }
                composable(PegasusTab.ORDERS.name) { Box(Modifier.fillMaxSize()) { OrdersScreen() } }
                composable(PegasusTab.MAP.name) {
                    Box(Modifier.fillMaxSize()) {
                        DeliveriesHubScreen()
                    }
                }
                composable("DOCK") {
                    Box(Modifier.fillMaxSize()) {
                        DeliveriesHubScreen(initialTabIndex = 1)
                    }
                }
                composable(PegasusTab.PROFILE.name) {
                    Box(Modifier.fillMaxSize()) {
                        ProfileScreen(
                            onAccountClick = { navController.navigate("ACCOUNT_PROFILE") },
                            onSavedCardsClick = { navController.navigate("SAVED_CARDS") },
                            onFamilyMembersClick = { navController.navigate("FAMILY_MEMBERS") },
                        )
                    }
                }
                composable("SAVED_CARDS") {
                    Box(Modifier.fillMaxSize()) {
                        SavedCardsScreen(onNavigateBack = { navController.popBackStack() })
                    }
                }
                composable("FAMILY_MEMBERS") {
                    Box(Modifier.fillMaxSize()) {
                        FamilyMembersScreen(onNavigateBack = { navController.popBackStack() })
                    }
                }
                composable("ACCOUNT_PROFILE") {
                    Box(Modifier.fillMaxSize()) {
                        AccountProfileScreen(onBack = { navController.popBackStack() })
                    }
                }
                composable("SAVED_CARDS_DELIVERY_PAYMENT/{orderId}/{sessionId}") {
                    Box(Modifier.fillMaxSize()) {
                        SavedCardsScreen(
                            returnTo = "delivery_payment",
                            onNavigateBack = { navController.popBackStack() },
                            onReturnToDeliveryPayment = {
                                navigationViewModel.loadPendingPayments()
                                paymentPhase = PaymentPhase.CHOOSE
                                paymentError = null
                                navController.popBackStack()
                            },
                        )
                    }
                }
                composable(PegasusTab.SUPPLIERS.name) {
                    Box(Modifier.fillMaxSize()) {
                        MySuppliersScreen(
                            onSupplierCash = { supplier ->
                                cartViewModel.setSupplierIsActive(supplier.isActive)
                                navController.navigate(
                                    "SUPPLIER_CATEGORY_CATALOG/${Uri.encode(supplier.id)}/${Uri.encode(supplier.name)}/${Uri.encode(supplier.displayCategory.orEmpty())}/${supplier.isActive}"
                                )
                            },
                        )
                    }
                }
                composable("CART") { Box(Modifier.fillMaxSize()) { CartScreen(viewModel = cartViewModel) } }
                composable("ANALYTICS") { Box(Modifier.fillMaxSize()) { AnalyticsScreen() } }
                composable("PROCUREMENT") { Box(Modifier.fillMaxSize()) { ProcurementScreen() } }
                composable("AUTO_ORDER") { Box(Modifier.fillMaxSize()) { AutoOrderScreen() } }
                composable("CONTROL_TOWER") { Box(Modifier.fillMaxSize()) { ControlTowerScreen() } }
                composable("FUTURE_DEMAND") {
                    Box(Modifier.fillMaxSize()) {
                        FutureDemandScreen(onBack = { navController.popBackStack() })
                    }
                }
                composable("NOTIFICATIONS") {
                    Box(Modifier.fillMaxSize()) {
                        NotificationInboxScreen(onBack = { navController.popBackStack() })
                    }
                }
                composable("PRODUCT_DETAIL/{productId}") { backStackEntry ->
                    val productId = backStackEntry.arguments?.getString("productId") ?: return@composable
                    Box(Modifier.fillMaxSize()) {
                        ProductDetailScreen(
                            productId = productId,
                            onBack = { navController.popBackStack() },
                            onAddToCart = { product, variant -> cartViewModel.addToCart(product, variant) },
                        )
                    }
                }
                composable("CATEGORY_SUPPLIERS/{categoryId}/{categoryName}") { backStackEntry ->
                    val categoryId = backStackEntry.arguments?.getString("categoryId") ?: return@composable
                    val categoryName = backStackEntry.arguments?.getString("categoryName") ?: "Category"
                    Box(Modifier.fillMaxSize()) {
                        CategorySuppliersScreen(
                            categoryId = categoryId,
                            categoryName = categoryName,
                            onBack = { navController.popBackStack() },
                            onSupplierCash = { supplier ->
                                cartViewModel.setSupplierIsActive(supplier.isActive)
                                navController.navigate(
                                    "SUPPLIER_CATEGORY_CATALOG/${Uri.encode(supplier.id)}/${Uri.encode(supplier.name)}/${Uri.encode(supplier.displayCategory.orEmpty())}/${supplier.isActive}"
                                )
                            },
                        )
                    }
                }
                composable("SUPPLIER_CATEGORY_CATALOG/{supplierId}/{supplierName}/{supplierCategory}/{supplierIsActive}") { backStackEntry ->
                    val supplierId = backStackEntry.arguments?.getString("supplierId") ?: return@composable
                    val supplierName = backStackEntry.arguments?.getString("supplierName") ?: "Supplier"
                    val supplierCategory = backStackEntry.arguments?.getString("supplierCategory") ?: ""
                    val supplierIsActive = backStackEntry.arguments?.getString("supplierIsActive")?.toBooleanStrictOrNull() ?: true
                    Box(Modifier.fillMaxSize()) {
                        SupplierCatalogScreen(
                            supplierId = supplierId,
                            supplierName = supplierName,
                            supplierCategory = supplierCategory,
                            supplierIsActive = supplierIsActive,
                            onBack = { navController.popBackStack() },
                            onProductCash = { productId ->
                                navController.navigate("PRODUCT_DETAIL/$productId")
                            },
                        )
                    }
                }
                composable("SETUP_WIZARD") {
                    Box(Modifier.fillMaxSize()) {
                        com.pegasusx.retailer.ui.screens.setup.SetupWizardScreen(
                            onSetupComplete = {
                                navController.navigate(PegasusTab.HOME.name) {
                                    popUpTo("SETUP_WIZARD") { inclusive = true }
                                }
                            }
                        )
                    }
                }
                }
            }
        }

        // ── Active Deliveries Half Sheet ──
        if (showActiveDeliveries) {
            ActiveDeliveriesSheet(
                activeOrders = navState.activeOrders,
                approachingOrderIds = navState.approachingOrderIds,
                onDismiss = { showActiveDeliveries = false },
                onShowDetail = { globalDetailOrder = it },
                onShowQR = { globalQROrder = it },
                isCompact = isCompact,
            )
        }

        // ── Order Detail Sheet (top-level, above everything) ──
        globalDetailOrder?.let { order ->
            OrderDetailSheet(
                order = order,
                onDismiss = { globalDetailOrder = null },
                onShowQR = {
                    globalQROrder = order
                    globalDetailOrder = null
                },
                isCompact = isCompact,
                onFileClaim = {
                    claimOrder = order
                    globalDetailOrder = null
                },
            )
        }
        claimOrder?.let { order ->
            FileClaimHost(
                order = order,
                onDismiss = { claimOrder = null },
            )
        }

        // ── QR Overlay (top-level, above everything) ──
        QROverlay(
            visible = globalQROrder != null,
            order = globalQROrder,
            onDismiss = { globalQROrder = null },
        )

        // ── Sidebar Overlay ──
} // Close Row
        if (isCompact) {
            SidebarMenu(
                isOpen = sidebarOpen,
                onDismiss = { sidebarOpen = false },
                userName = navState.userName,
                companyName = navState.companyName,
                onNavigate = { dest ->
                    // Navigate based on sidebar destination
                    when (dest) {
                        com.pegasusx.retailer.ui.components.SidebarDestination.DASHBOARD -> {
                            currentTab = PegasusTab.HOME
                            navController.navigate(PegasusTab.HOME.name) {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.pegasusx.retailer.ui.components.SidebarDestination.PROCUREMENT -> {
                            navController.navigate("PROCUREMENT") {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.pegasusx.retailer.ui.components.SidebarDestination.AI_PREDICTIONS -> {
                            navController.navigate("FUTURE_DEMAND") {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.pegasusx.retailer.ui.components.SidebarDestination.CONTROL_TOWER -> {
                            navController.navigate("CONTROL_TOWER") {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.pegasusx.retailer.ui.components.SidebarDestination.DOCK -> {
                            currentTab = PegasusTab.MAP
                            navController.navigate("DOCK") {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.pegasusx.retailer.ui.components.SidebarDestination.INSIGHTS -> {
                            navController.navigate("ANALYTICS") {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.pegasusx.retailer.ui.components.SidebarDestination.AUTO_ORDER -> {
                            navController.navigate("AUTO_ORDER") {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.pegasusx.retailer.ui.components.SidebarDestination.INBOX -> {
                            navController.navigate("NOTIFICATIONS") {
                                launchSingleTop = true
                            }
                        }
                        com.pegasusx.retailer.ui.components.SidebarDestination.PROFILE,
                        com.pegasusx.retailer.ui.components.SidebarDestination.SETTINGS -> {
                            navController.navigate(PegasusTab.PROFILE.name) {
                                launchSingleTop = true
                            }
                        }
                        else -> { /* unhandled sidebar destination */ }
                    }
                },
            )
        }

        // ── Delivery Payment Sheet (WebSocket-driven) ──
        val paymentEvent = navState.paymentEvent
        val context = LocalContext.current
        val navBackStackEntry by navController.currentBackStackEntryAsState()
        val hidePaymentForAddCard =
            navBackStackEntry?.destination?.route?.startsWith("SAVED_CARDS_DELIVERY_PAYMENT") == true

        // ADR-009: fiscalizing while OFD runs; SUCCESS only after fiscal/order complete.
        LaunchedEffect(navState.fiscalizing, navState.orderCompleted) {
            when {
                navState.orderCompleted -> paymentPhase = PaymentPhase.SUCCESS
                navState.fiscalizing -> paymentPhase = PaymentPhase.FISCALIZING
            }
        }

        LaunchedEffect(navState.approachingOrderIds, navState.activeOrders) {
            val approachingId = navState.approachingOrderIds.firstOrNull() ?: return@LaunchedEffect
            val order = navState.activeOrders.firstOrNull { it.id == approachingId && it.status.hasDeliveryToken }
                ?: return@LaunchedEffect
            if (globalQROrder?.id != order.id) {
                globalQROrder = order
            }
        }

        val shopClosedAlert = navState.shopClosedAlert
        var shopClosedSubmitting by remember(shopClosedAlert?.orderId) { mutableStateOf(false) }
        var shopClosedError by remember(shopClosedAlert?.orderId) { mutableStateOf<String?>(null) }

        LaunchedEffect(navState.reconnectEpoch) {
            if (navState.reconnectEpoch > 0L && shopClosedSubmitting) {
                shopClosedSubmitting = false
                shopClosedError = "Connection restored — verify response before retrying."
            }
        }

        if (shopClosedAlert != null) {
            ShopClosedSheet(
                alert = shopClosedAlert,
                isSubmitting = shopClosedSubmitting,
                errorMessage = shopClosedError,
                onRespond = { option ->
                    if (shopClosedSubmitting) return@ShopClosedSheet
                    shopClosedSubmitting = true
                    shopClosedError = null
                    coroutineScope.launch {
                        val result = navigationViewModel.respondToShopClosed(shopClosedAlert.orderId, option)
                        shopClosedSubmitting = false
                        if (result.isFailure) {
                            shopClosedError = result.exceptionOrNull()?.message ?: "Could not submit response"
                        }
                    }
                },
            )
        }

        if (paymentEvent != null && !hidePaymentForAddCard) {
            DeliveryPaymentSheet(
                event = paymentEvent,
                phase = paymentPhase,
                errorMessage = paymentError,
                isCompact = isCompact,
                onSelectCash = {
                    paymentError = null
                    paymentPhase = PaymentPhase.CASH_CONFIRM
                },
                onConfirmCash = {
                    paymentPhase = PaymentPhase.PROCESSING
                    coroutineScope.launch {
                        val result = navigationViewModel.confirmCash(paymentEvent.orderId)
                        if (result.isSuccess) {
                            paymentPhase = PaymentPhase.CASH_PENDING
                        } else {
                            paymentError = result.exceptionOrNull()?.message ?: "Cash confirmation failed"
                            paymentPhase = PaymentPhase.FAILED
                        }
                    }
                },
                onBackToPaymentChoice = {
                    paymentError = null
                    paymentPhase = PaymentPhase.CHOOSE
                },
                onAddCard = {
                    val sessionToken = paymentEvent.sessionId.orEmpty().ifBlank { "_" }
                    navController.navigate(
                        "SAVED_CARDS_DELIVERY_PAYMENT/${Uri.encode(paymentEvent.orderId)}/${Uri.encode(sessionToken)}"
                    )
                },
                onSelectCard = { gateway ->
                    paymentPhase = PaymentPhase.PROCESSING
                    coroutineScope.launch {
                        val result = navigationViewModel.cardCheckout(paymentEvent.orderId, gateway)
                        if (result.isSuccess) {
                            val checkout = result.getOrNull()
                            val url = checkout?.paymentUrl
                            if (!url.isNullOrBlank()) {
                                // Open deep-link in GlobalPay/Cash banking app
                                try {
                                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                                } catch (_: Exception) {
                                    paymentError = "Could not open $gateway app. Check it is installed."
                                    paymentPhase = PaymentPhase.FAILED
                                }
                            } else {
                                paymentError = "Payment gateway is not configured for this supplier."
                                paymentPhase = PaymentPhase.FAILED
                            }
                            // Stay on PROCESSING — the webhook settlement will trigger ORDER_COMPLETED via WS
                        } else {
                            paymentError = result.exceptionOrNull()?.message ?: "Card checkout failed"
                            paymentPhase = PaymentPhase.FAILED
                        }
                    }
                },
                onRetry = {
                    paymentPhase = PaymentPhase.CHOOSE
                    paymentError = null
                },
                onDismiss = {
                    paymentPhase = PaymentPhase.CHOOSE
                    paymentError = null
                    navigationViewModel.clearPaymentEvent()
                },
            )
        }
    }
}

@Composable
private fun RetailerOperationsStrip(
    navState: NavigationUiState,
    navSyncMessage: String?,
    onRetrySync: () -> Unit,
    onOpenDeliveries: () -> Unit,
    onReviewPayment: () -> Unit,
) {
    var message: String? = null
    var actionLabel: String? = null
    var onAction: (() -> Unit)? = null
    var containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
    var contentColor = MaterialTheme.colorScheme.onPrimaryContainer

    when {
        navState.paymentEvent != null -> {
            val orderRef = navState.paymentEvent.orderId.takeLast(6)
            message = "Settlement required for order #$orderRef"
            actionLabel = "Review"
            onAction = onReviewPayment
            containerColor = MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.55f)
            contentColor = MaterialTheme.colorScheme.onTertiaryContainer
        }

        navState.approachingOrderIds.isNotEmpty() -> {
            val count = navState.approachingOrderIds.size
            message = if (count == 1) {
                "Driver approaching now"
            } else {
                "Drivers approaching for $count deliveries"
            }
            actionLabel = "Track"
            onAction = onOpenDeliveries
            containerColor = MaterialTheme.colorScheme.secondaryContainer.copy(alpha = 0.55f)
            contentColor = MaterialTheme.colorScheme.onSecondaryContainer
        }

        navState.paymentFailed -> {
            message = navState.paymentFailureMessage.ifBlank { "Payment failed. Retry settlement." }
            actionLabel = "Retry"
            onAction = onReviewPayment
            containerColor = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
            contentColor = MaterialTheme.colorScheme.onErrorContainer
        }

        navSyncMessage != null -> {
            message = navSyncMessage
            val loadIssue = navState.loadIssue
            containerColor = when (loadIssue) {
                NavigationLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                NavigationLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                NavigationLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
            }
            contentColor = when (loadIssue) {
                NavigationLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                NavigationLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                NavigationLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
                null -> MaterialTheme.colorScheme.onPrimaryContainer
            }
            if (loadIssue != null) {
                actionLabel = "Retry"
                onAction = onRetrySync
            }
        }
    }

    if (message == null) return

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 8.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(containerColor)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = message,
            modifier = Modifier.weight(1f),
            style = MaterialTheme.typography.labelMedium,
            color = contentColor,
        )

        if (actionLabel != null && onAction != null) {
            TextButton(onClick = onAction) {
                Text(actionLabel, color = contentColor)
            }
        }
    }
}
