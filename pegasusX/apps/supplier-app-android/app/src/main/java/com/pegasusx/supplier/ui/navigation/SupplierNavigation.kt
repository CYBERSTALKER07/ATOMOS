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
import androidx.navigation.NavType
import androidx.navigation.navArgument
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.data.remote.TokenHolder
import kotlinx.coroutines.flow.collectLatest
import com.pegasusx.supplier.ui.screens.auth.LoginScreen
import com.pegasusx.supplier.ui.screens.billing.BillingScreen
import com.pegasusx.supplier.ui.screens.dashboard.DashboardScreen
import com.pegasusx.supplier.ui.screens.earnings.EarningsScreen
import com.pegasusx.supplier.ui.screens.fleet.FleetLiveMapScreen
import com.pegasusx.supplier.ui.screens.fleet.FleetScreen
import com.pegasusx.supplier.ui.screens.catalog.CatalogScreen
import com.pegasusx.supplier.ui.screens.inventory.InventoryScreen
import com.pegasusx.supplier.ui.screens.activity.ActivityScreen
import com.pegasusx.supplier.ui.screens.dispatch.DispatchPreviewScreen
import com.pegasusx.supplier.ui.screens.exceptions.EarlyCompleteScreen
import com.pegasusx.supplier.ui.screens.exceptions.ExceptionsScreen
// NegotiationsScreen removed — quantity negotiation disabled ecosystem-wide.
import com.pegasusx.supplier.ui.screens.exceptions.ShopClosedScreen
import com.pegasusx.supplier.ui.screens.fleet.FleetOrdersScreen
import com.pegasusx.supplier.ui.screens.manifests.ManifestDetailScreen
import com.pegasusx.supplier.ui.screens.manifests.ManifestExceptionsScreen
import com.pegasusx.supplier.ui.screens.manifests.ManifestsScreen
import com.pegasusx.supplier.ui.screens.more.MoreScreen
import com.pegasusx.supplier.ui.screens.notifications.NotificationInboxScreen
import com.pegasusx.supplier.ui.screens.operations.OperationsScreen
import com.pegasusx.supplier.ui.screens.orders.OrdersScreen
import com.pegasusx.supplier.ui.screens.profile.ProfileScreen
import com.pegasusx.supplier.ui.screens.orgfleet.OrgFleetScreen
import com.pegasusx.supplier.ui.screens.pricing.PricingScreen
import com.pegasusx.supplier.ui.screens.promotions.PromotionsScreen
import com.pegasusx.supplier.ui.screens.returns.ReturnsScreen
import com.pegasusx.supplier.ui.screens.treasury.LedgerScreen
import com.pegasusx.supplier.ui.screens.treasury.PaymentsScreen
import com.pegasusx.supplier.ui.screens.treasury.ReconciliationScreen
import com.pegasusx.supplier.ui.screens.analytics.AnalyticsScreen
import com.pegasusx.supplier.ui.screens.ai.AIRecommendationsScreen
import com.pegasusx.supplier.ui.screens.network.DeliveryZonesScreen
import com.pegasusx.supplier.ui.screens.network.GeoReportScreen
import com.pegasusx.supplier.ui.screens.network.SupplyLanesScreen
import com.pegasusx.supplier.ui.screens.network.TopologyScreen
import com.pegasusx.supplier.ui.screens.portal.PortalHandoffScreen
import com.pegasusx.supplier.ui.portal.SupplierPortalFeature

object SupplierRoutes {
    const val LOGIN = "login"
    const val BILLING = "billing"
    const val DASHBOARD = "dashboard"
    const val ORDERS = "orders"
    const val FLEET = "fleet"
    const val MORE = "more"
    const val INVENTORY = "inventory"
    const val CATALOG = "catalog"
    const val PROMOTIONS = "promotions"
    const val PRICING = "pricing"
    const val RETURNS = "returns"
    const val RECONCILIATION = "reconciliation"
    const val EARLY_COMPLETE = "early_complete"
    const val ORG_FLEET = "org_fleet"
    const val EARNINGS = "earnings"
    const val PROFILE = "profile"
    const val EXCEPTIONS = "exceptions"
    const val SHOP_CLOSED = "shop_closed"
    const val NEGOTIATIONS = "negotiations"
    const val MANIFESTS = "manifests"
    const val MANIFEST_DETAIL = "manifest_detail/{manifestId}"
    const val MANIFEST_EXCEPTIONS = "manifest_exceptions"

    fun manifestDetail(manifestId: String) = "manifest_detail/$manifestId"
    const val DISPATCH_PREVIEW = "dispatch_preview"
    const val ACTIVITY = "activity"
    const val FLEET_ORDERS = "fleet_orders"
    const val FLEET_LIVE_MAP = "fleet_live_map"
    const val LEDGER = "ledger"
    const val OPERATIONS = "operations"
    const val ANALYTICS = "analytics"
    const val AI_RECOMMENDATIONS = "ai_recommendations"
    const val GEO_REPORT = "geo_report"
    const val TOPOLOGY = "topology"
    const val DELIVERY_ZONES = "delivery_zones"
    const val SUPPLY_LANES = "supply_lanes"
    const val PAYMENTS = "payments"
    const val NOTIFICATIONS = "notifications"
    const val PORTAL_HANDOFF = "portal_handoff/{feature}"

    fun portalHandoff(feature: SupplierPortalFeature) = "portal_handoff/${feature.routeKey}"
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
                LoginScreen(
                    api = api,
                    onRegisterPortal = {
                        navController.navigate(SupplierRoutes.portalHandoff(SupplierPortalFeature.REGISTER))
                    },
                ) {
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
                        onOpenNotifications = { navController.navigate(SupplierRoutes.NOTIFICATIONS) },
                    )
                }
            }
            composable(SupplierRoutes.ORDERS) { key(refreshEpoch) { OrdersScreen(ops) } }
            composable(SupplierRoutes.FLEET) {
                key(refreshEpoch) {
                    FleetScreen(
                        api = api,
                        ops = ops,
                        onOpenLiveMap = { navController.navigate(SupplierRoutes.FLEET_LIVE_MAP) },
                    )
                }
            }
            composable(SupplierRoutes.MORE) {
                MoreScreen(
                    onExceptions = { navController.navigate(SupplierRoutes.EXCEPTIONS) },
                    onShopClosed = { navController.navigate(SupplierRoutes.SHOP_CLOSED) },
                    onNegotiations = { /* quantity negotiation disabled */ },
                    onManifests = { navController.navigate(SupplierRoutes.MANIFESTS) },
                    onDispatch = { navController.navigate(SupplierRoutes.DISPATCH_PREVIEW) },
                    onActivity = { navController.navigate(SupplierRoutes.ACTIVITY) },
                    onFleetOrders = { navController.navigate(SupplierRoutes.FLEET_ORDERS) },
                    onLedger = { navController.navigate(SupplierRoutes.LEDGER) },
                    onOperations = { navController.navigate(SupplierRoutes.OPERATIONS) },
                    onAnalytics = { navController.navigate(SupplierRoutes.ANALYTICS) },
                    onAiRecommendations = { navController.navigate(SupplierRoutes.AI_RECOMMENDATIONS) },
                    onGeoReport = { navController.navigate(SupplierRoutes.GEO_REPORT) },
                    onTopology = { navController.navigate(SupplierRoutes.TOPOLOGY) },
                    onDeliveryZones = { navController.navigate(SupplierRoutes.DELIVERY_ZONES) },
                    onSupplyLanes = { navController.navigate(SupplierRoutes.SUPPLY_LANES) },
                    onPayments = { navController.navigate(SupplierRoutes.PAYMENTS) },
                    onInventory = { navController.navigate(SupplierRoutes.INVENTORY) },
                    onCatalog = { navController.navigate(SupplierRoutes.CATALOG) },
                    onPromotions = { navController.navigate(SupplierRoutes.PROMOTIONS) },
                    onPricing = { navController.navigate(SupplierRoutes.PRICING) },
                    onReturns = { navController.navigate(SupplierRoutes.RETURNS) },
                    onReconciliation = { navController.navigate(SupplierRoutes.RECONCILIATION) },
                    onEarlyComplete = { navController.navigate(SupplierRoutes.EARLY_COMPLETE) },
                    onOrgFleet = { navController.navigate(SupplierRoutes.ORG_FLEET) },
                    onEarnings = { navController.navigate(SupplierRoutes.EARNINGS) },
                    onProfile = { navController.navigate(SupplierRoutes.PROFILE) },
                    onNotifications = { navController.navigate(SupplierRoutes.NOTIFICATIONS) },
                    onBilling = { navController.navigate(SupplierRoutes.portalHandoff(SupplierPortalFeature.BUSINESS_SETUP)) },
                    onChargebacks = { navController.navigate(SupplierRoutes.portalHandoff(SupplierPortalFeature.CHARGEBACKS)) },
                    onPaymentBypass = { navController.navigate(SupplierRoutes.OPERATIONS) },
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
                key(refreshEpoch) { ShopClosedScreen(ops, realtimeSignals) { navController.popBackStack() } }
            }
            // Quantity negotiation disabled ecosystem-wide.
            // composable(SupplierRoutes.NEGOTIATIONS) { ... }
            composable(SupplierRoutes.MANIFESTS) {
                key(refreshEpoch) {
                    ManifestsScreen(
                        ops = ops,
                        onBack = { navController.popBackStack() },
                        onOpenManifest = { manifestId -> navController.navigate(SupplierRoutes.manifestDetail(manifestId)) },
                        onOpenGateExceptions = { navController.navigate(SupplierRoutes.MANIFEST_EXCEPTIONS) },
                    )
                }
            }
            composable(
                SupplierRoutes.MANIFEST_DETAIL,
                arguments = listOf(navArgument("manifestId") { type = NavType.StringType }),
            ) { entry ->
                val manifestId = entry.arguments?.getString("manifestId").orEmpty()
                key(refreshEpoch) {
                    ManifestDetailScreen(
                        manifestId = manifestId,
                        ops = ops,
                        realtimeSignals = realtimeSignals,
                        onBack = { navController.popBackStack() },
                    )
                }
            }
            composable(SupplierRoutes.MANIFEST_EXCEPTIONS) {
                key(refreshEpoch) {
                    ManifestExceptionsScreen(
                        ops = ops,
                        onBack = { navController.popBackStack() },
                        onOpenManifest = { manifestId -> navController.navigate(SupplierRoutes.manifestDetail(manifestId)) },
                    )
                }
            }
            composable(SupplierRoutes.DISPATCH_PREVIEW) {
                key(refreshEpoch) {
                    DispatchPreviewScreen(
                        ops = ops,
                        realtimeSignals = realtimeSignals,
                    ) { navController.popBackStack() }
                }
            }
            composable(SupplierRoutes.ACTIVITY) {
                key(refreshEpoch) { ActivityScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.FLEET_ORDERS) {
                key(refreshEpoch) { FleetOrdersScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.FLEET_LIVE_MAP) {
                key(refreshEpoch) {
                    FleetLiveMapScreen(
                        ops = ops,
                        realtimeSignals = realtimeSignals,
                    ) { navController.popBackStack() }
                }
            }
            composable(SupplierRoutes.LEDGER) {
                key(refreshEpoch) { LedgerScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.OPERATIONS) {
                key(refreshEpoch) { OperationsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.ANALYTICS) {
                key(refreshEpoch) { AnalyticsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.AI_RECOMMENDATIONS) {
                key(refreshEpoch) { AIRecommendationsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.GEO_REPORT) {
                key(refreshEpoch) { GeoReportScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.TOPOLOGY) {
                key(refreshEpoch) { TopologyScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.DELIVERY_ZONES) {
                key(refreshEpoch) { DeliveryZonesScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.SUPPLY_LANES) {
                key(refreshEpoch) { SupplyLanesScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.PAYMENTS) {
                key(refreshEpoch) { PaymentsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.INVENTORY) { key(refreshEpoch) { InventoryScreen(api) } }
            composable(SupplierRoutes.CATALOG) { key(refreshEpoch) { CatalogScreen(api) } }
            composable(SupplierRoutes.PROMOTIONS) { key(refreshEpoch) { PromotionsScreen(api) } }
            composable(SupplierRoutes.PRICING) {
                key(refreshEpoch) { PricingScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.RETURNS) {
                key(refreshEpoch) { ReturnsScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.RECONCILIATION) {
                key(refreshEpoch) { ReconciliationScreen(ops) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.EARLY_COMPLETE) {
                key(refreshEpoch) { EarlyCompleteScreen(ops, realtimeSignals) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.ORG_FLEET) {
                key(refreshEpoch) { OrgFleetScreen(api, ops, realtimeSignals) { navController.popBackStack() } }
            }
            composable(SupplierRoutes.EARNINGS) { key(refreshEpoch) { EarningsScreen(api = api, ops = ops) } }
            composable(SupplierRoutes.PROFILE) { key(refreshEpoch) { ProfileScreen(api) } }
            composable(SupplierRoutes.NOTIFICATIONS) {
                NotificationInboxScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
                )
            }
            composable(
                route = SupplierRoutes.PORTAL_HANDOFF,
                arguments = listOf(navArgument("feature") { type = NavType.StringType }),
            ) { entry ->
                val key = entry.arguments?.getString("feature").orEmpty()
                val feature = SupplierPortalFeature.fromRouteKey(key)
                if (feature != null) {
                    PortalHandoffScreen(feature) { navController.popBackStack() }
                }
            }
        }
    }
}
