package com.pegasusx.warehouse.ui.navigation

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Undo
import androidx.compose.material.icons.filled.*
import androidx.compose.ui.graphics.vector.ImageVector
import com.pegasusx.warehouse.ui.portal.WarehousePortalFeature

enum class WarehouseSection(
    val route: String,
    val label: String,
    val icon: ImageVector,
) {
    DASHBOARD(WarehouseRoutes.DASHBOARD, "Dashboard", Icons.Default.Dashboard),
    ORDERS(WarehouseRoutes.ORDERS, "Orders", Icons.Default.ShoppingCart),
    DRIVERS(WarehouseRoutes.DRIVERS, "Drivers", Icons.Default.Badge),
    VEHICLES(WarehouseRoutes.VEHICLES, "Trucks", Icons.Default.LocalShipping),
    INVENTORY(WarehouseRoutes.INVENTORY, "Inventory", Icons.Default.Inventory2),
    DISPATCH(WarehouseRoutes.DISPATCH, "Dispatch", Icons.Default.NearMe),
    ANALYTICS(WarehouseRoutes.ANALYTICS, "Analytics", Icons.Default.Analytics),
    TREASURY(WarehouseRoutes.TREASURY, "Treasury", Icons.Default.AccountBalance),
    STAFF(WarehouseRoutes.STAFF, "Staff", Icons.Default.People),
    MANIFESTS(WarehouseRoutes.MANIFESTS, "Manifests", Icons.Default.Description),
    DISPATCH_SETTINGS(WarehouseRoutes.DISPATCH_SETTINGS, "Dispatch settings", Icons.Default.Tune),
    FLEET_LIVE_MAP(WarehouseRoutes.FLEET_LIVE_MAP, "Live fleet", Icons.Default.Map),
    TRANSFER_ACTIONS(WarehouseRoutes.TRANSFER_ACTIONS, "Transfer actions", Icons.Default.SwapHoriz),
    PRODUCTS(WarehouseRoutes.PRODUCTS, "Products", Icons.Default.GridView),
    PREORDERS(WarehouseRoutes.PREORDERS, "Pre-orders", Icons.Default.Event),
    STOCK_COMMITMENTS(WarehouseRoutes.STOCK_COMMITMENTS, "Stock commitments", Icons.Default.Inventory),
    SUPPLY_REQUESTS(WarehouseRoutes.SUPPLY_REQUESTS, "Supply requests", Icons.Default.Sync),
    REPLENISHMENT(WarehouseRoutes.REPLENISHMENT, "Replenishment", Icons.Default.Inventory),
    DEMAND_FORECAST(WarehouseRoutes.DEMAND_FORECAST, "Demand forecast", Icons.Default.ShowChart),
    RETAILERS(WarehouseRoutes.CRM, "Retailers", Icons.Default.Store),
    RETURNS(WarehouseRoutes.RETURNS, "Returns", Icons.AutoMirrored.Filled.Undo),
    PAYMENT_CONFIG(WarehouseRoutes.PAYMENT_CONFIG, "Payment config", Icons.Default.Payment),
    OPS_SETTINGS(WarehouseRoutes.OPS_SETTINGS, "Ops settings", Icons.Default.Settings),
    NOTIFICATIONS(WarehouseRoutes.NOTIFICATIONS, "Notifications", Icons.Default.Notifications),
    PORTAL_SETUP(WarehouseRoutes.portalHandoff("setup"), "Warehouse setup", Icons.Default.Business),
    PORTAL_PROFILE(WarehouseRoutes.portalHandoff("profile"), "Profile", Icons.Default.Person),
    PORTAL_SEARCH(WarehouseRoutes.portalHandoff("search"), "Global search", Icons.Default.Search),
    MORE(WarehouseRoutes.MORE, "More", Icons.Default.Apps),
    ;

    companion object {
        val compactTabs: List<WarehouseSection> = listOf(DASHBOARD, ORDERS, DISPATCH, MORE)

        val primarySections: List<WarehouseSection> = listOf(
            DASHBOARD, ORDERS, DRIVERS, VEHICLES, INVENTORY, DISPATCH, ANALYTICS, TREASURY, STAFF,
        )

        val fulfillmentSections: List<WarehouseSection> = listOf(
            MANIFESTS, DISPATCH_SETTINGS, FLEET_LIVE_MAP, TRANSFER_ACTIONS,
        )

        val inventorySections: List<WarehouseSection> = listOf(
            PRODUCTS,
            PREORDERS,
            STOCK_COMMITMENTS,
            SUPPLY_REQUESTS,
            REPLENISHMENT,
            DEMAND_FORECAST,
            OPS_SETTINGS,
        )

        val operationsSections: List<WarehouseSection> = listOf(
            RETAILERS, RETURNS, PAYMENT_CONFIG, NOTIFICATIONS,
        )

        val portalSections: List<WarehouseSection> = listOf(
            PORTAL_SETUP, PORTAL_PROFILE, PORTAL_SEARCH,
        )

        val drawerSections: List<WarehouseSection> =
            primarySections + fulfillmentSections + inventorySections + operationsSections + portalSections

        fun fromRoute(route: String?): WarehouseSection? {
            if (route.isNullOrBlank()) return null
            val exact = entries.firstOrNull { it.route == route }
            if (exact != null) return exact
            val base = route.substringBefore("/").substringBefore("?")
            return when {
                base == "orders" && route.contains("/") -> ORDERS
                base == "vehicles" && route.contains("/") -> VEHICLES
                base == "supply_requests" && route.contains("/") -> SUPPLY_REQUESTS
                route.startsWith("portal/") -> entries.firstOrNull { it.route == route }
                    ?: when (route.removePrefix("portal/")) {
                        WarehousePortalFeature.SETUP.routeKey -> PORTAL_SETUP
                        WarehousePortalFeature.PROFILE.routeKey -> PORTAL_PROFILE
                        WarehousePortalFeature.SEARCH.routeKey -> PORTAL_SEARCH
                        else -> null
                    }
                else -> entries.firstOrNull { it.route == base }
            }
        }
    }
}
