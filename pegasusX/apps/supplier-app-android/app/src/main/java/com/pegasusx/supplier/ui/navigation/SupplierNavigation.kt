package com.pegasusx.supplier.ui.navigation

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.windowsizeclass.WindowSizeClass
import androidx.compose.material3.windowsizeclass.WindowWidthSizeClass
import com.pegasusx.supplier.ui.components.SupplierNavigationDrawer
import com.pegasusx.supplier.ui.navigation.SupplierSection
import com.pegasusx.supplier.BuildConfig
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.ui.components.ClientPolicyBanner
import com.pegasusx.supplier.service.AutoUpdater
import com.pegasusx.supplier.service.EnterpriseUpdateConfig
import androidx.compose.ui.platform.LocalContext
import kotlinx.coroutines.launch
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
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import kotlinx.coroutines.flow.collectLatest
import com.pegasusx.supplier.ui.screens.auth.LoginScreen
import com.pegasusx.supplier.ui.screens.auth.RegisterScreen
import com.pegasusx.supplier.ui.screens.onboarding.BusinessSetupScreen
import com.pegasusx.supplier.ui.screens.billing.BillingScreen
import com.pegasusx.supplier.ui.screens.dashboard.DashboardScreen
import com.pegasusx.supplier.ui.screens.earnings.EarningsScreen
import com.pegasusx.supplier.ui.screens.fleet.FleetLiveMapScreen
import com.pegasusx.supplier.ui.screens.fleet.FleetScreen
import com.pegasusx.supplier.ui.screens.catalog.CatalogScreen
import com.pegasusx.supplier.ui.screens.catalog.CatalogDetailScreen
import com.pegasusx.supplier.ui.screens.inventory.InventoryScreen
import com.pegasusx.supplier.ui.screens.inventory.InventoryImportScreen
import com.pegasusx.supplier.ui.screens.activity.ActivityScreen
import com.pegasusx.supplier.ui.screens.dispatch.DispatchPreviewScreen
import com.pegasusx.supplier.ui.screens.exceptions.ClaimsScreen
import com.pegasusx.supplier.ui.screens.exceptions.EarlyCompleteScreen
import com.pegasusx.supplier.ui.screens.exceptions.ExceptionsScreen
// NegotiationsScreen removed — quantity negotiation disabled ecosystem-wide.
import com.pegasusx.supplier.ui.screens.exceptions.ShopClosedScreen
import com.pegasusx.supplier.ui.screens.fleet.FleetOrdersScreen
import com.pegasusx.supplier.ui.screens.manifests.ManifestDetailScreen
import com.pegasusx.supplier.ui.screens.manifests.ManifestExceptionsScreen
import com.pegasusx.supplier.ui.screens.manifests.ManifestsScreen
import com.pegasusx.supplier.ui.screens.more.MoreScreen
import com.pegasusx.supplier.ui.screens.settings.NotificationPreferencesScreen
import com.pegasusx.supplier.ui.screens.notifications.NotificationInboxScreen
import com.pegasusx.supplier.ui.screens.operations.OperationsScreen
import com.pegasusx.supplier.ui.screens.orders.OrderDetailScreen
import com.pegasusx.supplier.ui.screens.orders.OrdersHubScreen
import com.pegasusx.supplier.ui.screens.profile.ProfileScreen
import com.pegasusx.supplier.ui.screens.orgfleet.OrgFleetScreen
import com.pegasusx.supplier.ui.screens.pricing.PricingScreen
import com.pegasusx.supplier.ui.screens.promotions.PromotionsScreen
import com.pegasusx.supplier.ui.screens.returns.ReturnsScreen
import com.pegasusx.supplier.ui.screens.treasury.ClaimChargebacksScreen
import com.pegasusx.supplier.ui.screens.analytics.RoutePerformanceScreen
import com.pegasusx.supplier.ui.screens.treasury.CashReconciliationsScreen
import com.pegasusx.supplier.ui.screens.treasury.CreditNotesScreen
import com.pegasusx.supplier.ui.screens.treasury.CreditProfilesScreen
import com.pegasusx.supplier.ui.screens.treasury.ComplianceScreen
import com.pegasusx.supplier.ui.screens.treasury.LedgerScreen
import com.pegasusx.supplier.ui.screens.treasury.PaymentsScreen
import com.pegasusx.supplier.ui.screens.treasury.ReconciliationScreen
import com.pegasusx.supplier.ui.screens.analytics.AnalyticsScreen
import com.pegasusx.supplier.ui.screens.analytics.DemandHistoryScreen
import com.pegasusx.supplier.ui.screens.planning.KnowledgeGraphScreen
import com.pegasusx.supplier.ui.screens.planning.PlanningBrainScreen
import com.pegasusx.supplier.ui.screens.planning.PlanningSettingsScreen
import com.pegasusx.supplier.ui.screens.planning.ReplenishmentPoliciesScreen
import com.pegasusx.supplier.ui.screens.ai.AIRecommendationsScreen
import com.pegasusx.supplier.ui.screens.network.DeliveryZonesScreen
import com.pegasusx.supplier.ui.screens.network.FactoriesScreen
import com.pegasusx.supplier.ui.screens.network.GeoReportScreen
import com.pegasusx.supplier.ui.screens.network.SupplyLanesScreen
import com.pegasusx.supplier.ui.screens.network.TopologyScreen
import com.pegasusx.supplier.ui.screens.network.WarehousesScreen
import com.pegasusx.supplier.ui.screens.pricing.RetailerOverridesScreen
import com.pegasusx.supplier.ui.screens.treasury.ChargebacksScreen
import com.pegasusx.supplier.ui.screens.treasury.TreasuryHubScreen

object SupplierRoutes {
    const val LOGIN = "login"
    const val REGISTER = "register"
    const val BUSINESS_SETUP = "business_setup"
    const val BILLING = "billing"
    const val DASHBOARD = "dashboard"
    const val ORDERS = "orders"
    const val ORDER_DETAIL = "orders/{orderId}"
    const val FLEET = "fleet"
    const val MORE = "more"
    const val INVENTORY = "inventory"
    const val CATALOG = "catalog"
    const val CATALOG_DETAIL = "catalog_detail/{productId}"
    const val PROMOTIONS = "promotions"
    const val PRICING = "pricing"
    const val RETURNS = "returns"
    const val RECONCILIATION = "reconciliation"
    const val CHARGEBACKS = "chargebacks"
    const val CLAIMS = "claims"
    const val CLAIM_CHARGEBACKS = "claim_chargebacks"
    const val RETAILER_OVERRIDES = "retailer_overrides"
    const val INVENTORY_IMPORT = "inventory_import"
    const val TREASURY_HUB = "treasury_hub"
    const val DEMAND_HISTORY = "demand_history"
    const val PLANNING_BRAIN = "planning_brain"
    const val PLANNING_SETTINGS = "planning_settings"
    const val KNOWLEDGE_GRAPH = "knowledge_graph"
    const val REPLENISHMENT_POLICIES = "replenishment_policies"
    const val FACTORIES = "factories"
    const val WAREHOUSES = "warehouses"
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
    fun orderDetail(orderId: String) = "orders/$orderId"
    fun catalogDetail(productId: String) = "catalog_detail/$productId"
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
    const val COMPLIANCE = "compliance"
    const val CASH_RECONCILIATIONS = "cash_reconciliations"
    const val CREDIT_NOTES = "credit_notes"
    const val CREDIT_PROFILES = "credit_profiles"
    const val ROUTE_PERFORMANCE = "route_performance"
    const val NOTIFICATION_PREFS = "notification_prefs"
    const val NOTIFICATIONS = "notifications"
}

private data class SupplierTab(val route: String, val label: String, val icon: ImageVector)

@Composable
fun SupplierNavigation(
    api: SupplierApi,
    ops: SupplierOperationsRepository,
    geocodeApi: GeocodeApi,
    realtimeSignals: SupplierRealtimeSignals,
    windowSizeClass: WindowSizeClass,
) {
    var sessionEpoch by remember { mutableIntStateOf(0) }
    var pendingBusinessSetup by remember { mutableStateOf(false) }
    val loggedIn = remember(sessionEpoch) { TokenHolder.isLoggedIn }
    val navController = rememberNavController()
    val start = when {
        pendingBusinessSetup -> SupplierRoutes.BUSINESS_SETUP
        !TokenHolder.isConfigured -> SupplierRoutes.BILLING
        else -> SupplierRoutes.DASHBOARD
    }

    if (!loggedIn) {
        NavHost(navController = navController, startDestination = SupplierRoutes.LOGIN) {
            composable(SupplierRoutes.LOGIN) {
                LoginScreen(
                    api = api,
                    onLoginSuccess = { sessionEpoch++ },
                    onRegister = { navController.navigate(SupplierRoutes.REGISTER) },
                )
            }
            composable(SupplierRoutes.REGISTER) {
                RegisterScreen(
                    onBack = { navController.popBackStack() },
                    onRegistered = {
                        pendingBusinessSetup = true
                        sessionEpoch++
                    },
                )
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
    val useDrawer = windowSizeClass.widthSizeClass != WindowWidthSizeClass.Compact
    val showBottomBar = !useDrawer && currentRoute in tabs.map { it.route }
    var isRailExpanded by remember { mutableStateOf(true) }
    var clientPolicyMessage by remember { mutableStateOf<String?>(null) }
    var clientPolicyForce by remember { mutableStateOf(false) }
    var pendingManifest by remember { mutableStateOf<AutoUpdater.Manifest?>(null) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val autoUpdater = remember(context) { AutoUpdater(context.applicationContext) }

    DisposableEffect(autoUpdater) {
        autoUpdater.register()
        onDispose { autoUpdater.cleanup() }
    }

    fun loadClientPolicy() {
        scope.launch {
            try {
                val resp = api.getClientPolicy(
                    role = EnterpriseUpdateConfig.POLICY_ROLE,
                    platform = "android",
                    version = BuildConfig.VERSION_NAME,
                    channel = EnterpriseUpdateConfig.CHANNEL,
                )
                if (resp.isSuccessful && resp.body() != null) {
                    val state = autoUpdater.checkFromPolicy(resp.body()!!, autoDownload = false)
                    clientPolicyMessage = state.message
                    clientPolicyForce = state.force
                    pendingManifest = state.manifest
                }
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

    LaunchedEffect(sessionEpoch) {
        if (loggedIn) loadClientPolicy()
    }
    LaunchedEffect(realtimeSignals, loggedIn) {
        if (!loggedIn) return@LaunchedEffect
        realtimeSignals.reconnectTick.collectLatest { loadClientPolicy() }
    }


    fun navigateSection(section: SupplierSection) {
        navController.navigate(section.route) {
            popUpTo(navController.graph.findStartDestination().id) { saveState = true }
            launchSingleTop = true
            restoreState = true
        }
    }

    val navHost: @Composable (Modifier) -> Unit = { modifier ->
        NavHost(
            navController = navController,
            startDestination = start,
            modifier = modifier,
        ) {
            composable(SupplierRoutes.BUSINESS_SETUP) {
                val canGoBack = navController.previousBackStackEntry != null
                BusinessSetupScreen(
                    onComplete = {
                        pendingBusinessSetup = false
                        if (!navController.popBackStack()) {
                            navController.navigate(SupplierRoutes.BILLING) {
                                popUpTo(SupplierRoutes.BUSINESS_SETUP) { inclusive = true }
                            }
                        }
                    },
                    onBack = if (canGoBack) {{ navController.popBackStack() }} else null,
                )
            }
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
                
                    DashboardScreen(
                        api = api,
                        ops = ops,
                        realtimeSignals = realtimeSignals,
                        showBillingBanner = !TokenHolder.isConfigured,
                        onOpenBilling = { navController.navigate(SupplierRoutes.BILLING) },
                        onOpenNotifications = { navController.navigate(SupplierRoutes.NOTIFICATIONS) },
                    )
                
            }
            composable(SupplierRoutes.ORDERS) {
                OrdersHubScreen(
                    ops = ops,
                    realtimeSignals = realtimeSignals,
                    onOrderClick = { order -> navController.navigate(SupplierRoutes.orderDetail(order.orderId)) },
                )
            }

            composable(
                route = SupplierRoutes.ORDER_DETAIL,
                arguments = listOf(navArgument("orderId") { type = NavType.StringType }),
            ) { backStackEntry ->
                val orderId = backStackEntry.arguments?.getString("orderId") ?: return@composable
                OrderDetailScreen(
                    orderId = orderId,
                    ops = ops,
                    onBack = { navController.popBackStack() },
                )
            }
            composable(SupplierRoutes.FLEET) {
                
                    FleetScreen(
                        api = api,
                        ops = ops,
                        realtimeSignals = realtimeSignals,
                        onOpenLiveMap = { navController.navigate(SupplierRoutes.FLEET_LIVE_MAP) },
                    )
                
            }
            composable(SupplierRoutes.MORE) {
                MoreScreen(
                    onManifests = { navController.navigate(SupplierRoutes.MANIFESTS) },
                    onDispatch = { navController.navigate(SupplierRoutes.DISPATCH_PREVIEW) },
                    onActivity = { navController.navigate(SupplierRoutes.ACTIVITY) },
                    onFleetOrders = { navController.navigate(SupplierRoutes.FLEET_ORDERS) },
                    onLedger = { navController.navigate(SupplierRoutes.LEDGER) },
                    onOperations = { navController.navigate(SupplierRoutes.OPERATIONS) },
                    onReplenishmentPolicies = { navController.navigate(SupplierRoutes.REPLENISHMENT_POLICIES) },
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
                    onOrgFleet = { navController.navigate(SupplierRoutes.ORG_FLEET) },
                    onEarnings = { navController.navigate(SupplierRoutes.EARNINGS) },
                    onProfile = { navController.navigate(SupplierRoutes.PROFILE) },
                    onNotifications = { navController.navigate(SupplierRoutes.NOTIFICATIONS) },
                    onNotificationPrefs = { navController.navigate(SupplierRoutes.NOTIFICATION_PREFS) },
                    onBilling = { navController.navigate(SupplierRoutes.BILLING) },
                    onBusinessSetup = { navController.navigate(SupplierRoutes.BUSINESS_SETUP) },
                    onChargebacks = { navController.navigate(SupplierRoutes.CHARGEBACKS) },
                    onClaims = { navController.navigate(SupplierRoutes.CLAIMS) },
                    onClaimChargebacks = { navController.navigate(SupplierRoutes.CLAIM_CHARGEBACKS) },
                    onRetailerOverrides = { navController.navigate(SupplierRoutes.RETAILER_OVERRIDES) },
                    onInventoryImport = { navController.navigate(SupplierRoutes.INVENTORY_IMPORT) },
                    onTreasuryHub = { navController.navigate(SupplierRoutes.TREASURY_HUB) },
                    onDemandHistory = { navController.navigate(SupplierRoutes.DEMAND_HISTORY) },
                    onFactories = { navController.navigate(SupplierRoutes.FACTORIES) },
                    onWarehouses = { navController.navigate(SupplierRoutes.WAREHOUSES) },
                    onSignOut = {
                        TokenHolder.clear()
                        pendingBusinessSetup = false
                        sessionEpoch++
                    },
                )
            }
            composable(SupplierRoutes.EXCEPTIONS) {
                 ExceptionsScreen(
                     ops = ops,
                     onBack = { navController.popBackStack() },
                     onOpenClaims = { navController.navigate(SupplierRoutes.CLAIMS) },
                 )
            }
            composable(SupplierRoutes.CLAIMS) {
                ClaimsScreen(
                    ops = ops,
                    onBack = { navController.popBackStack() },
                    onOpenClaimChargebacks = { navController.navigate(SupplierRoutes.CLAIM_CHARGEBACKS) },
                )
            }
            composable(SupplierRoutes.CLAIM_CHARGEBACKS) {
                ClaimChargebacksScreen(
                    ops = ops,
                    onBack = { navController.popBackStack() },
                )
            }
            composable(SupplierRoutes.SHOP_CLOSED) {
                 ShopClosedScreen(ops, realtimeSignals) { navController.popBackStack() } 
            }
            // Quantity negotiation disabled ecosystem-wide.
            // composable(SupplierRoutes.NEGOTIATIONS) { ... }
            composable(SupplierRoutes.MANIFESTS) {
                ManifestsScreen(
                    ops = ops,
                    realtimeSignals = realtimeSignals,
                    onBack = {
                        if (!navController.popBackStack()) {
                            navigateSection(SupplierSection.DASHBOARD)
                        }
                    },
                    onOpenManifest = { manifestId -> navController.navigate(SupplierRoutes.manifestDetail(manifestId)) },
                    onOpenGateExceptions = { navController.navigate(SupplierRoutes.MANIFEST_EXCEPTIONS) },
                )
            }
            composable(
                SupplierRoutes.MANIFEST_DETAIL,
                arguments = listOf(navArgument("manifestId") { type = NavType.StringType }),
            ) { entry ->
                val manifestId = entry.arguments?.getString("manifestId").orEmpty()
                
                    ManifestDetailScreen(
                        manifestId = manifestId,
                        ops = ops,
                        realtimeSignals = realtimeSignals,
                        onBack = { navController.popBackStack() },
                    )
                
            }
            composable(SupplierRoutes.MANIFEST_EXCEPTIONS) {
                
                    ManifestExceptionsScreen(
                        ops = ops,
                        onBack = { navController.popBackStack() },
                        onOpenManifest = { manifestId -> navController.navigate(SupplierRoutes.manifestDetail(manifestId)) },
                    )
                
            }
            composable(SupplierRoutes.DISPATCH_PREVIEW) {
                
                    DispatchPreviewScreen(
                        ops = ops,
                        realtimeSignals = realtimeSignals,
                        onBack = { navController.popBackStack() },
                    )
                
            }
            composable(SupplierRoutes.ACTIVITY) {
                 ActivityScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.FLEET_ORDERS) {
                 FleetOrdersScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.FLEET_LIVE_MAP) {
                
                    FleetLiveMapScreen(
                        ops = ops,
                        realtimeSignals = realtimeSignals,
                    ) { navController.popBackStack() }
                
            }
            composable(SupplierRoutes.LEDGER) {
                 LedgerScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.OPERATIONS) {
                 OperationsScreen(
                    ops = ops,
                    onBack = { navController.popBackStack() },
                    onOpenReplenishmentPolicies = { navController.navigate(SupplierRoutes.REPLENISHMENT_POLICIES) },
                 ) 
            }
            composable(SupplierRoutes.ANALYTICS) {
                 AnalyticsScreen(
                    ops = ops,
                    realtimeSignals = realtimeSignals,
                    onBack = { navController.popBackStack() },
                    onOpenPlanningBrain = { navController.navigate(SupplierRoutes.PLANNING_BRAIN) },
                    onOpenKnowledgeGraph = { navController.navigate(SupplierRoutes.KNOWLEDGE_GRAPH) },
                    onOpenPlanningSettings = { navController.navigate(SupplierRoutes.PLANNING_SETTINGS) },
                    onOpenRoutePerformance = { navController.navigate(SupplierRoutes.ROUTE_PERFORMANCE) },
                 )
            }
            composable(SupplierRoutes.AI_RECOMMENDATIONS) {
                 AIRecommendationsScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.GEO_REPORT) {
                 GeoReportScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.TOPOLOGY) {
                 TopologyScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.DELIVERY_ZONES) {
                 DeliveryZonesScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.SUPPLY_LANES) {
                 SupplyLanesScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.PAYMENTS) {
                 PaymentsScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.INVENTORY) {  InventoryScreen()  }
            composable(SupplierRoutes.CATALOG) {
                
                    CatalogScreen(api, realtimeSignals) { productId ->
                        navController.navigate(SupplierRoutes.catalogDetail(productId))
                    }
                
            }
            composable(
                SupplierRoutes.CATALOG_DETAIL,
                arguments = listOf(navArgument("productId") { type = NavType.StringType }),
            ) { entry ->
                val productId = entry.arguments?.getString("productId").orEmpty()
                
                    CatalogDetailScreen(
                        productId = productId,
                        ops = ops,
                        onBack = { navController.popBackStack() },
                    )
                
            }
            composable(SupplierRoutes.PROMOTIONS) {  PromotionsScreen(api, realtimeSignals)  }
            composable(SupplierRoutes.PRICING) {
                 PricingScreen(api) { navController.popBackStack() }
            }
            composable(SupplierRoutes.RETURNS) {
                 ReturnsScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.RECONCILIATION) {
                 ReconciliationScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.COMPLIANCE) {
                ComplianceScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.CASH_RECONCILIATIONS) {
                CashReconciliationsScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.CREDIT_NOTES) {
                CreditNotesScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.CREDIT_PROFILES) {
                CreditProfilesScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.ROUTE_PERFORMANCE) {
                RoutePerformanceScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.NOTIFICATION_PREFS) {
                NotificationPreferencesScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.CHARGEBACKS) {
                
                    ChargebacksScreen(onBack = { navController.popBackStack() })
                
            }
            composable(SupplierRoutes.TREASURY_HUB) {
                
                    TreasuryHubScreen(
                        onBack = { navController.popBackStack() },
                        onLedger = { navController.navigate(SupplierRoutes.LEDGER) },
                        onPayments = { navController.navigate(SupplierRoutes.PAYMENTS) },
                        onReconciliation = { navController.navigate(SupplierRoutes.RECONCILIATION) },
                        onCompliance = { navController.navigate(SupplierRoutes.COMPLIANCE) },
                        onCashReconciliations = { navController.navigate(SupplierRoutes.CASH_RECONCILIATIONS) },
                        onCreditNotes = { navController.navigate(SupplierRoutes.CREDIT_NOTES) },
                        onCreditProfiles = { navController.navigate(SupplierRoutes.CREDIT_PROFILES) },
                        onEarnings = { navController.navigate(SupplierRoutes.EARNINGS) },
                        onChargebacks = { navController.navigate(SupplierRoutes.CHARGEBACKS) },
                        onClaimChargebacks = { navController.navigate(SupplierRoutes.CLAIM_CHARGEBACKS) },
                        onClaims = { navController.navigate(SupplierRoutes.CLAIMS) },
                    )
                
            }
            composable(SupplierRoutes.RETAILER_OVERRIDES) {
                 RetailerOverridesScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.INVENTORY_IMPORT) {
                 InventoryImportScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.DEMAND_HISTORY) {
                 DemandHistoryScreen(ops) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.PLANNING_BRAIN) {
                PlanningBrainScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.PLANNING_SETTINGS) {
                PlanningSettingsScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.KNOWLEDGE_GRAPH) {
                KnowledgeGraphScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.REPLENISHMENT_POLICIES) {
                ReplenishmentPoliciesScreen(ops) { navController.popBackStack() }
            }
            composable(SupplierRoutes.FACTORIES) {
                 FactoriesScreen(ops, geocodeApi) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.WAREHOUSES) {
                 WarehousesScreen(ops, geocodeApi) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.EARLY_COMPLETE) {
                 EarlyCompleteScreen(ops, realtimeSignals) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.ORG_FLEET) {
                 OrgFleetScreen(api, ops, realtimeSignals) { navController.popBackStack() } 
            }
            composable(SupplierRoutes.EARNINGS) {  EarningsScreen(api = api, ops = ops)  }
            composable(SupplierRoutes.PROFILE) {  ProfileScreen(api)  }
            composable(SupplierRoutes.NOTIFICATIONS) {
                NotificationInboxScreen(
                    api = api,
                    onBack = { navController.popBackStack() },
                )
            }
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
        if (useDrawer) {
            Row(Modifier.weight(1f).fillMaxSize()) {
                SupplierNavigationDrawer(
                    isExpanded = isRailExpanded,
                    onToggleExpanded = { isRailExpanded = !isRailExpanded },
                    selectedRoute = currentRoute,
                    onSectionSelected = ::navigateSection,
                )
                navHost(Modifier.weight(1f).fillMaxSize())
            }
        } else {
            Scaffold(
                bottomBar = {
                    if (showBottomBar) {
                        NavigationBar {
                            tabs.forEach { tab ->
                                NavigationBarItem(
                                    selected = currentRoute == tab.route,
                                    onClick = { navigateSection(SupplierSection.entries.first { it.route == tab.route }) },
                                    icon = { Icon(tab.icon, contentDescription = tab.label) },
                                    label = { Text(tab.label) },
                                )
                            }
                        }
                    }
                },
            ) { padding ->
                navHost(Modifier.padding(padding).fillMaxSize())
            }
        }
    }
}
