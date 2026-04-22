package com.thelab.retailer.ui.navigation

import android.content.Intent
import android.net.Uri
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import com.thelab.retailer.ui.theme.MotionTokens
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.platform.LocalContext
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.thelab.retailer.data.model.Order
import com.thelab.retailer.ui.components.ActiveDeliveriesSheet
import com.thelab.retailer.ui.components.DeliveryPaymentSheet
import com.thelab.retailer.ui.components.FloatingActiveOrdersBar
import com.thelab.retailer.ui.components.LabBottomBar
import com.thelab.retailer.ui.components.LabTab
import com.thelab.retailer.ui.components.LabTopBar
import com.thelab.retailer.ui.components.OrderDetailSheet
import com.thelab.retailer.ui.components.PaymentPhase
import com.thelab.retailer.ui.components.QROverlay
import androidx.compose.material3.windowsizeclass.WindowSizeClass
import androidx.compose.material3.windowsizeclass.WindowWidthSizeClass
import androidx.compose.foundation.layout.Row
import com.thelab.retailer.ui.components.LabNavigationRail
import com.thelab.retailer.ui.components.SidebarMenu
import com.thelab.retailer.ui.screens.cart.CartScreen
import com.thelab.retailer.ui.screens.cart.CartViewModel
import com.thelab.retailer.ui.screens.profile.ProfileScreen
import com.thelab.retailer.ui.screens.catalog.CatalogScreen
import com.thelab.retailer.ui.screens.catalog.CategorySuppliersScreen
import com.thelab.retailer.ui.screens.dashboard.DashboardScreen
import com.thelab.retailer.ui.screens.orders.OrdersScreen
import com.thelab.retailer.ui.screens.suppliers.MySuppliersScreen
import com.thelab.retailer.ui.screens.analytics.AnalyticsScreen
import com.thelab.retailer.ui.screens.autoorder.AutoOrderScreen
import com.thelab.retailer.ui.screens.product.ProductDetailScreen
import com.thelab.retailer.ui.screens.suppliers.SupplierCatalogScreen
import com.thelab.retailer.ui.screens.tracking.DeliveryMapScreen
import com.thelab.retailer.ui.screens.notifications.NotificationInboxScreen
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
    var currentTab by rememberSaveable { mutableStateOf(LabTab.HOME) }
    val cartBadge = cartState.totalItems

    // Sidebar state
    var sidebarOpen by rememberSaveable { mutableStateOf(false) }
    var railExpanded by rememberSaveable { mutableStateOf(false) }

    // Active deliveries sheet (notification bell)
    var showActiveDeliveries by rememberSaveable { mutableStateOf(false) }

    // Global detail/QR state (hoisted so overlays render above sheets)
    var globalDetailOrder by remember { mutableStateOf<Order?>(null) }
    var globalQROrder by remember { mutableStateOf<Order?>(null) }

    // Payment sheet state
    var paymentPhase by remember { mutableStateOf(PaymentPhase.CHOOSE) }
    var paymentError by remember { mutableStateOf<String?>(null) }
    val coroutineScope = rememberCoroutineScope()

    // Show floating bar on Home, Orders, Suppliers tabs
    val showFloatingBar = currentTab in listOf(LabTab.HOME, LabTab.ORDERS, LabTab.SUPPLIERS)
    val isCompact = windowSizeClass.widthSizeClass == WindowWidthSizeClass.Compact

    Box(modifier = Modifier.fillMaxSize()) {
        Row(modifier = Modifier.fillMaxSize()) {
            if (!isCompact) {
                LabNavigationRail(
                    isExpanded = railExpanded,
                    onToggleExpanded = { railExpanded = !railExpanded },
                    currentTab = currentTab,
                    userName = navState.userName,
                    companyName = navState.companyName,
                    onSidebarNavigate = { dest ->
                        when (dest) {
                            com.thelab.retailer.ui.components.SidebarDestination.DASHBOARD -> {
                                currentTab = LabTab.HOME
                                navController.navigate(LabTab.HOME.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.thelab.retailer.ui.components.SidebarDestination.PROCUREMENT -> {
                                currentTab = LabTab.CATALOG
                                navController.navigate(LabTab.CATALOG.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.thelab.retailer.ui.components.SidebarDestination.AI_PREDICTIONS -> {
                                currentTab = LabTab.ORDERS
                                navController.navigate(LabTab.ORDERS.name) {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.thelab.retailer.ui.components.SidebarDestination.INSIGHTS -> {
                                navController.navigate("ANALYTICS") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            com.thelab.retailer.ui.components.SidebarDestination.AUTO_ORDER -> {
                                navController.navigate("AUTO_ORDER") {
                                    popUpTo(navController.graph.startDestinationId) { saveState = true }
                                    launchSingleTop = true
                                }
                            }
                            else -> { /* Profile, Settings, Inbox — future */ }
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
            Scaffold(
            topBar = {
                LabTopBar(
                    onAvatarClick = { sidebarOpen = true },
                    onCartClick = {
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
                    notificationBadge = navState.activeOrderCount,
                    avatarInitial = navState.avatarInitial,
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
                        LabBottomBar(
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
            NavHost(
                navController = navController,
                startDestination = LabTab.HOME.name,
                modifier = Modifier.fillMaxSize().padding(innerPadding),
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
                composable(LabTab.HOME.name) { Box(Modifier.fillMaxSize()) { DashboardScreen() } }
                composable(LabTab.CATALOG.name) {
                    Box(Modifier.fillMaxSize()) {
                        CatalogScreen(
                            onProductClick = { productId ->
                                navController.navigate("PRODUCT_DETAIL/$productId")
                            },
                            onCategoryClick = { categoryId, categoryName ->
                                navController.navigate("CATEGORY_SUPPLIERS/${Uri.encode(categoryId)}/${Uri.encode(categoryName)}")
                            },
                        )
                    }
                }
                composable(LabTab.ORDERS.name) { Box(Modifier.fillMaxSize()) { OrdersScreen() } }
                composable(LabTab.MAP.name) {
                    Box(Modifier.fillMaxSize()) {
                        DeliveryMapScreen(viewModel = hiltViewModel(), onBack = { navController.popBackStack() })
                    }
                }
                composable(LabTab.PROFILE.name) { Box(Modifier.fillMaxSize()) { ProfileScreen() } }
                composable(LabTab.SUPPLIERS.name) {
                    Box(Modifier.fillMaxSize()) {
                        MySuppliersScreen(
                            onSupplierClick = { supplier ->
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
                composable("AUTO_ORDER") { Box(Modifier.fillMaxSize()) { AutoOrderScreen() } }
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
                            onSupplierClick = { supplier ->
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
                            onProductClick = { productId ->
                                navController.navigate("PRODUCT_DETAIL/$productId")
                            },
                        )
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
                        com.thelab.retailer.ui.components.SidebarDestination.DASHBOARD -> {
                            currentTab = LabTab.HOME
                            navController.navigate(LabTab.HOME.name) {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.thelab.retailer.ui.components.SidebarDestination.PROCUREMENT -> {
                            currentTab = LabTab.CATALOG
                            navController.navigate(LabTab.CATALOG.name) {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.thelab.retailer.ui.components.SidebarDestination.AI_PREDICTIONS -> {
                            currentTab = LabTab.ORDERS
                            navController.navigate(LabTab.ORDERS.name) {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.thelab.retailer.ui.components.SidebarDestination.INSIGHTS -> {
                            navController.navigate("ANALYTICS") {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        com.thelab.retailer.ui.components.SidebarDestination.AUTO_ORDER -> {
                            navController.navigate("AUTO_ORDER") {
                                popUpTo(navController.graph.startDestinationId) { saveState = true }
                                launchSingleTop = true
                            }
                        }
                        else -> { /* Profile, Settings, Inbox — future */ }
                    }
                },
            )
        }

        // ── Delivery Payment Sheet (WebSocket-driven) ──
        val paymentEvent = navState.paymentEvent
        val context = LocalContext.current

        // Auto-transition to SUCCESS when ORDER_COMPLETED arrives via WebSocket
        LaunchedEffect(navState.orderCompleted) {
            if (navState.orderCompleted) {
                paymentPhase = PaymentPhase.SUCCESS
            }
        }

        if (paymentEvent != null) {
            DeliveryPaymentSheet(
                event = paymentEvent,
                phase = paymentPhase,
                errorMessage = paymentError,
                isCompact = isCompact,
                onSelectCash = {
                    paymentPhase = PaymentPhase.PROCESSING
                    coroutineScope.launch {
                        val result = navigationViewModel.cashCheckout(paymentEvent.orderId)
                        if (result.isSuccess) {
                            paymentPhase = PaymentPhase.CASH_PENDING
                        } else {
                            paymentError = result.exceptionOrNull()?.message ?: "Cash checkout failed"
                            paymentPhase = PaymentPhase.FAILED
                        }
                    }
                },
                onSelectCard = { gateway ->
                    paymentPhase = PaymentPhase.PROCESSING
                    coroutineScope.launch {
                        val result = navigationViewModel.cardCheckout(paymentEvent.orderId, gateway)
                        if (result.isSuccess) {
                            val checkout = result.getOrNull()
                            val url = checkout?.paymentUrl
                            if (!url.isNullOrBlank()) {
                                // Open deep-link in Payme/Click banking app
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
