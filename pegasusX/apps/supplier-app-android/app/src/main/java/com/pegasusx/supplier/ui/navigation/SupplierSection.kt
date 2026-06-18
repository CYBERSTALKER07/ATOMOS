package com.pegasusx.supplier.ui.navigation

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.automirrored.filled.Undo
import androidx.compose.material.icons.filled.*
import androidx.compose.ui.graphics.vector.ImageVector

enum class SupplierSection(
    val route: String,
    val label: String,
    val icon: ImageVector,
) {
    DASHBOARD(SupplierRoutes.DASHBOARD, "Dashboard", Icons.Default.Dashboard),
    ORDERS(SupplierRoutes.ORDERS, "Orders", Icons.AutoMirrored.Filled.List),
    FLEET(SupplierRoutes.FLEET, "Fleet", Icons.Default.LocalShipping),
    EXCEPTIONS(SupplierRoutes.EXCEPTIONS, "Exceptions", Icons.Default.Warning),
    SHOP_CLOSED(SupplierRoutes.SHOP_CLOSED, "Shop closed", Icons.Default.Store),
    MANIFESTS(SupplierRoutes.MANIFESTS, "Manifests", Icons.Default.Description),
    DISPATCH_PREVIEW(SupplierRoutes.DISPATCH_PREVIEW, "Dispatch", Icons.Default.NearMe),
    ACTIVITY(SupplierRoutes.ACTIVITY, "Activity", Icons.Default.History),
    FLEET_ORDERS(SupplierRoutes.FLEET_ORDERS, "Fleet orders", Icons.Default.LocalShipping),
    LEDGER(SupplierRoutes.LEDGER, "Ledger", Icons.Default.AccountBalance),
    PAYMENTS(SupplierRoutes.PAYMENTS, "Payments", Icons.Default.Payment),
    OPERATIONS(SupplierRoutes.OPERATIONS, "Operations", Icons.Default.Build),
    ANALYTICS(SupplierRoutes.ANALYTICS, "Analytics", Icons.Default.Analytics),
    AI_RECOMMENDATIONS(SupplierRoutes.AI_RECOMMENDATIONS, "AI recommendations", Icons.Default.AutoAwesome),
    GEO_REPORT(SupplierRoutes.GEO_REPORT, "Geo report", Icons.Default.Map),
    TOPOLOGY(SupplierRoutes.TOPOLOGY, "Topology", Icons.Default.Hub),
    DELIVERY_ZONES(SupplierRoutes.DELIVERY_ZONES, "Delivery zones", Icons.Default.Place),
    SUPPLY_LANES(SupplierRoutes.SUPPLY_LANES, "Supply lanes", Icons.Default.SwapHoriz),
    CATALOG(SupplierRoutes.CATALOG, "Catalog", Icons.Default.GridView),
    INVENTORY(SupplierRoutes.INVENTORY, "Inventory", Icons.Default.Inventory2),
    PROMOTIONS(SupplierRoutes.PROMOTIONS, "Promotions", Icons.Default.LocalOffer),
    PRICING(SupplierRoutes.PRICING, "Pricing", Icons.Default.AttachMoney),
    RETURNS(SupplierRoutes.RETURNS, "Returns", Icons.AutoMirrored.Filled.Undo),
    NOTIFICATIONS(SupplierRoutes.NOTIFICATIONS, "Notifications", Icons.Default.Notifications),
    EARNINGS(SupplierRoutes.EARNINGS, "Earnings", Icons.Default.ShowChart),
    PROFILE(SupplierRoutes.PROFILE, "Profile", Icons.Default.Person),
    MORE(SupplierRoutes.MORE, "More", Icons.Default.MoreHoriz),
    ;

    companion object {
        val compactTabs = listOf(DASHBOARD, ORDERS, FLEET, MORE)

        val opsSections = listOf(
            EXCEPTIONS, SHOP_CLOSED, MANIFESTS, DISPATCH_PREVIEW, ACTIVITY, FLEET_ORDERS,
            LEDGER, PAYMENTS, OPERATIONS,
        )

        val intelligenceSections = listOf(ANALYTICS, AI_RECOMMENDATIONS, GEO_REPORT)

        val networkSections = listOf(TOPOLOGY, DELIVERY_ZONES, SUPPLY_LANES)

        val accountSections = listOf(
            CATALOG, INVENTORY, PROMOTIONS, PRICING, RETURNS, NOTIFICATIONS, EARNINGS, PROFILE,
        )

        fun fromRoute(route: String?): SupplierSection? {
            if (route.isNullOrBlank()) return null
            val base = route.substringBefore("/").substringBefore("?")
            return entries.firstOrNull { it.route == base }
        }
    }
}
