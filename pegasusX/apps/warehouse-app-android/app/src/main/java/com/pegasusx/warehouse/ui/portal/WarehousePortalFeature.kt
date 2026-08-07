package com.pegasusx.warehouse.ui.portal

import androidx.compose.ui.res.stringResource

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Store
import androidx.compose.ui.graphics.vector.ImageVector

/** Portal-only surfaces — native apps hand off to warehouse-portal (port 3002). */
enum class WarehousePortalFeature(
    val routeKey: String,
    val title: String,
    val subtitle: String,
    val portalPath: String,
    val icon: ImageVector,
) {
    REGISTER(
        routeKey = "register",
        title = stringResource(R.string.mobile_warehouse_ui_register_warehouse),
        subtitle = "Create a new warehouse account",
        portalPath = "/auth/register",
        icon = Icons.Default.Store,
    ),
    SETUP(
        routeKey = "setup",
        title = stringResource(R.string.warehouse_portal_setup_setup_wizard_shell_text_warehouse_setup),
        subtitle = "Location, billing, and configuration",
        portalPath = "/setup/location",
        icon = Icons.Default.Settings,
    ),
    PROFILE(
        routeKey = "profile",
        title = stringResource(R.string.portal_nav_profile),
        subtitle = "Account and warehouse identity",
        portalPath = "/profile",
        icon = Icons.Default.Person,
    ),
    SEARCH(
        routeKey = "search",
        title = stringResource(R.string.mobile_warehouse_ui_global_search),
        subtitle = "Jump to any portal page",
        portalPath = "/",
        icon = Icons.Default.Search,
    ),
    ;

    val handoffMessage: String
        get() = when (this) {
            REGISTER -> "New warehouse registration is completed on the warehouse web portal."
            SETUP -> "Warehouse setup and onboarding run on the web portal after registration."
            PROFILE -> "Profile and account settings are managed on the warehouse web portal."
            SEARCH -> "Global search (⌘K) is available on the warehouse web portal desktop shell."
        }

    companion object {
        fun fromRouteKey(key: String): WarehousePortalFeature? =
            entries.firstOrNull { it.routeKey == key }
    }
}
