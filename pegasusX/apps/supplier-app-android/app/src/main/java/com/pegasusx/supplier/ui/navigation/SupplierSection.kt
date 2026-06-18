package com.pegasusx.supplier.ui.navigation

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.automirrored.filled.ShowChart
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
    MANIFESTS(SupplierRoutes.MANIFESTS, "Manifests", Icons.Default.Description),
    DISPATCH_PREVIEW(SupplierRoutes.DISPATCH_PREVIEW, "Dispatch", Icons.Default.NearMe),
    ACTIVITY(SupplierRoutes.ACTIVITY, "Activity", Icons.Default.History),
    FLEET_ORDERS(SupplierRoutes.FLEET_ORDERS, "Fleet orders", Icons.Default.LocalShipping),
    ORG_FLEET(SupplierRoutes.ORG_FLEET, "Org & fleet", Icons.Default.Groups),
    TREASURY_HUB(SupplierRoutes.TREASURY_HUB, "Treasury", Icons.Default.AccountBalance),
    LEDGER(SupplierRoutes.LEDGER, "Ledger", Icons.Default.AccountBalance),
    PAYMENTS(SupplierRoutes.PAYMENTS, "Payments", Icons.Default.Payment),
    CHARGEBACKS(SupplierRoutes.CHARGEBACKS, "Chargebacks", Icons.Default.Payments),
    RECONCILIATION(SupplierRoutes.RECONCILIATION, "Reconciliation", Icons.Default.Balance),
    OPERATIONS(SupplierRoutes.OPERATIONS, "Operations", Icons.Default.Build),
    ANALYTICS(SupplierRoutes.ANALYTICS, "Analytics", Icons.Default.Analytics),
    AI_RECOMMENDATIONS(SupplierRoutes.AI_RECOMMENDATIONS, "AI recommendations", Icons.Default.AutoAwesome),
    GEO_REPORT(SupplierRoutes.GEO_REPORT, "Geo report", Icons.Default.Map),
    DEMAND_HISTORY(SupplierRoutes.DEMAND_HISTORY, "Demand forecast", Icons.Default.Timeline),
    TOPOLOGY(SupplierRoutes.TOPOLOGY, "Topology", Icons.Default.Hub),
    FACTORIES(SupplierRoutes.FACTORIES, "Factories", Icons.Default.Business),
    WAREHOUSES(SupplierRoutes.WAREHOUSES, "Warehouses", Icons.Default.Inventory2),
    DELIVERY_ZONES(SupplierRoutes.DELIVERY_ZONES, "Delivery zones", Icons.Default.Place),
    SUPPLY_LANES(SupplierRoutes.SUPPLY_LANES, "Supply lanes", Icons.Default.SwapHoriz),
    CATALOG(SupplierRoutes.CATALOG, "Catalog", Icons.Default.GridView),
    INVENTORY(SupplierRoutes.INVENTORY, "Inventory", Icons.Default.Inventory2),
    INVENTORY_IMPORT(SupplierRoutes.INVENTORY_IMPORT, "Import inventory", Icons.Default.Upload),
    PROMOTIONS(SupplierRoutes.PROMOTIONS, "Promotions", Icons.Default.LocalOffer),
    PRICING(SupplierRoutes.PRICING, "Pricing", Icons.Default.AttachMoney),
    RETAILER_OVERRIDES(SupplierRoutes.RETAILER_OVERRIDES, "Retailer overrides", Icons.Default.PriceCheck),
    RETURNS(SupplierRoutes.RETURNS, "Returns", Icons.AutoMirrored.Filled.Undo),
    NOTIFICATIONS(SupplierRoutes.NOTIFICATIONS, "Notifications", Icons.Default.Notifications),
    EARNINGS(SupplierRoutes.EARNINGS, "Earnings", Icons.AutoMirrored.Filled.ShowChart),
    PROFILE(SupplierRoutes.PROFILE, "Profile", Icons.Default.Person),
    MORE(SupplierRoutes.MORE, "More", Icons.Default.MoreHoriz),
    ;

    companion object {
        val compactTabs = listOf(DASHBOARD, ORDERS, FLEET, MORE)

        /** Tablet / expanded rail — aligned with iOS sidebar + portal shell. */
        val opsSections = listOf(
            MANIFESTS,
            DISPATCH_PREVIEW,
            ACTIVITY,
            FLEET_ORDERS,
            ORG_FLEET,
            TREASURY_HUB,
            LEDGER,
            PAYMENTS,
            CHARGEBACKS,
            RECONCILIATION,
            OPERATIONS,
        )

        val intelligenceSections = listOf(
            ANALYTICS,
            AI_RECOMMENDATIONS,
            GEO_REPORT,
            DEMAND_HISTORY,
        )

        val networkSections = listOf(
            TOPOLOGY,
            FACTORIES,
            WAREHOUSES,
            DELIVERY_ZONES,
            SUPPLY_LANES,
        )

        val accountSections = listOf(
            CATALOG,
            INVENTORY,
            INVENTORY_IMPORT,
            PROMOTIONS,
            PRICING,
            RETAILER_OVERRIDES,
            RETURNS,
            NOTIFICATIONS,
            EARNINGS,
            PROFILE,
        )

        /** All destinations reachable from the tablet rail (excludes phone-only More hub). */
        val railSections = compactTabs.filter { it != MORE } +
            opsSections +
            intelligenceSections +
            networkSections +
            accountSections

        fun fromRoute(route: String?): SupplierSection? {
            if (route.isNullOrBlank()) return null
            val base = route.substringBefore("/").substringBefore("?")
            return when (base) {
                "catalog_detail" -> CATALOG
                "manifest_detail", "manifest_exceptions" -> MANIFESTS
                "fleet_live_map" -> FLEET
                "portal_handoff" -> null
                else -> entries.firstOrNull { it.route == base }
            }
        }
    }
}
