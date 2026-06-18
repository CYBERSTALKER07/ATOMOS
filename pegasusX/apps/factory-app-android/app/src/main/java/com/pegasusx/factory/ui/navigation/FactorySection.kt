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
    SUPPLY_REQUESTS(FactoryRoutes.SUPPLY_REQUESTS, "Supply requests", Icons.Default.Sync),
    PAYLOAD_OVERRIDE(FactoryRoutes.PAYLOAD_OVERRIDE, "Override", Icons.Default.SwapHoriz),
    MANIFESTS(FactoryRoutes.MANIFESTS, "Manifests", Icons.Default.Description),
    MANIFEST_EXCEPTIONS(FactoryRoutes.MANIFEST_EXCEPTIONS, "Exceptions", Icons.Default.Warning),
    INSIGHTS(FactoryRoutes.INSIGHTS, "Insights", Icons.Default.Insights),
    ANALYTICS(FactoryRoutes.ANALYTICS, "Analytics", Icons.Default.Analytics),
    NOTIFICATIONS(FactoryRoutes.NOTIFICATIONS, "Notifications", Icons.Default.Notifications),
    MORE(FactoryRoutes.MORE, "More", Icons.Default.Apps),
    ;

    companion object {
        val compactTabs: List<FactorySection> = listOf(DASHBOARD, LOADING_BAY, TRANSFERS, MORE)

        val primarySections: List<FactorySection> = listOf(
            DASHBOARD, LOADING_BAY, TRANSFERS, FLEET, STAFF,
        )

        val operationsSections: List<FactorySection> = listOf(
            SUPPLY_REQUESTS, PAYLOAD_OVERRIDE, MANIFESTS, MANIFEST_EXCEPTIONS,
        )

        val intelligenceSections: List<FactorySection> = listOf(
            INSIGHTS, ANALYTICS, NOTIFICATIONS,
        )

        fun fromRoute(route: String?): FactorySection? {
            if (route.isNullOrBlank()) return null
            val base = route.substringBefore("/")
            return entries.firstOrNull { it.route == base }
        }
    }
}
