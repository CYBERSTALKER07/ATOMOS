package com.pegasusx.factory.ui.navigation

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.ui.graphics.vector.ImageVector

enum class FactorySection(
    val route: String,
    val label: String,
    val icon: ImageVector,
) {
    DASHBOARD(FactoryRoutes.DASHBOARD, "Dashboard", Icons.Default.Dashboard),
    LOADING_BAY(FactoryRoutes.LOADING_BAY, "Loading Bay", Icons.Default.Inventory),
    TRANSFERS(FactoryRoutes.TRANSFERS, "Transfers", Icons.Default.SwapHoriz),
    FLEET(FactoryRoutes.FLEET, "Fleet", Icons.Default.LocalShipping),
    STAFF(FactoryRoutes.STAFF, "Staff", Icons.Default.People),
    LOCATION(FactoryRoutes.LOCATION_SETTINGS, "Location", Icons.Default.Place),
    SUPPLY_REQUESTS(FactoryRoutes.SUPPLY_REQUESTS, "Supply requests", Icons.Default.Sync),
    PAYLOAD_OVERRIDE(FactoryRoutes.PAYLOAD_OVERRIDE, "Payload override", Icons.Default.SwapHoriz),
    PAYLOAD_LOAD(FactoryRoutes.PAYLOAD_LOAD, "Payload / Load", Icons.Default.Inventory),
    MANIFESTS(FactoryRoutes.MANIFESTS, "Manifests", Icons.Default.Description),
    MANIFEST_EXCEPTIONS(FactoryRoutes.MANIFEST_EXCEPTIONS, "Gate exceptions", Icons.Default.Warning),
    INSIGHTS(FactoryRoutes.INSIGHTS, "Insights", Icons.Default.Insights),
    ANALYTICS(FactoryRoutes.ANALYTICS, "Analytics", Icons.Default.Analytics),
    NOTIFICATIONS(FactoryRoutes.NOTIFICATIONS, "Notifications", Icons.Default.Notifications),
    MORE(FactoryRoutes.MORE, "More", Icons.Default.Apps),
    ;

    companion object {
        val compactTabs: List<FactorySection> = listOf(DASHBOARD, LOADING_BAY, TRANSFERS, MORE)

        val primarySections: List<FactorySection> = listOf(
            DASHBOARD, LOADING_BAY, TRANSFERS, FLEET, STAFF, LOCATION,
        )

        val operationsSections: List<FactorySection> = listOf(
            SUPPLY_REQUESTS, PAYLOAD_LOAD, PAYLOAD_OVERRIDE, MANIFESTS, MANIFEST_EXCEPTIONS,
        )

        val intelligenceSections: List<FactorySection> = listOf(
            INSIGHTS, ANALYTICS, NOTIFICATIONS,
        )

        /** Tablet rail — mirrors iOS sidebar + factory-portal nav. */
        val drawerSections: List<FactorySection> =
            primarySections + operationsSections + intelligenceSections

        /** Phone More hub: everything not on the bottom tab bar (except More itself). */
        val moreHubSections: List<FactorySection> =
            primarySections.filter { it !in compactTabs.filter { tab -> tab != MORE } } +
                operationsSections +
                intelligenceSections

        fun fromRoute(route: String?): FactorySection? {
            if (route.isNullOrBlank()) return null
            val exact = entries.firstOrNull { it.route == route }
            if (exact != null) return exact
            return when {
                route == FactoryRoutes.TRANSFER_CREATE -> TRANSFERS
                route.startsWith("transfers/") -> TRANSFERS
                route.startsWith("manifests/") -> MANIFESTS
                route.startsWith("staff/") -> STAFF
                else -> {
                    val base = route.substringBefore("/").substringBefore("?")
                    entries.firstOrNull { it.route == base }
                }
            }
        }
    }
}
