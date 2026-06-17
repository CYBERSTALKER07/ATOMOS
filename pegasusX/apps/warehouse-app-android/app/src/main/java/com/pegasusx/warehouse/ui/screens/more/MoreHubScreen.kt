package com.pegasusx.warehouse.ui.screens.more

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import com.pegasusx.warehouse.ui.navigation.WarehouseRoutes
import com.pegasusx.warehouse.ui.portal.WarehousePortalFeature
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

private data class MoreDestination(
    val title: String,
    val subtitle: String,
    val icon: ImageVector,
    val route: String,
)

private val fulfillment = listOf(
    MoreDestination("Manifests", "Loading manifests", Icons.Default.Description, WarehouseRoutes.MANIFESTS),
    MoreDestination("Dispatch", "Preview, supply, locks", Icons.Default.LocalShipping, WarehouseRoutes.DISPATCH),
    MoreDestination("Dispatch settings", "Auto-dispatch policy", Icons.Default.Tune, WarehouseRoutes.DISPATCH_SETTINGS),
    MoreDestination("Live fleet map", "Route geometry + drivers", Icons.Default.Map, WarehouseRoutes.FLEET_LIVE_MAP),
    MoreDestination("Transfer actions", "Emergency / receive", Icons.Default.SwapHoriz, WarehouseRoutes.TRANSFER_ACTIONS),
)

private val inventory = listOf(
    MoreDestination("Products", "Catalog SKUs", Icons.Default.GridView, WarehouseRoutes.PRODUCTS),
    MoreDestination("Supply requests", "Factory restock queue", Icons.Default.Sync, WarehouseRoutes.SUPPLY_REQUESTS),
    MoreDestination("Replenishment", "Stock velocity insights", Icons.Default.Inventory2, WarehouseRoutes.REPLENISHMENT),
    MoreDestination("Demand Forecast", "Projected demand", Icons.Default.ShowChart, WarehouseRoutes.DEMAND_FORECAST),
    MoreDestination("Ops settings", "Stock policy + hours", Icons.Default.Settings, WarehouseRoutes.OPS_SETTINGS),
)

private val operations = listOf(
    MoreDestination("Retailers", "CRM", Icons.Default.Store, WarehouseRoutes.CRM),
    MoreDestination("Returns", "Return queue", Icons.Default.Undo, WarehouseRoutes.RETURNS),
    MoreDestination("Analytics", "KPI trends", Icons.Default.Analytics, WarehouseRoutes.ANALYTICS),
    MoreDestination("Treasury", "Ledger overview", Icons.Default.AccountBalance, WarehouseRoutes.TREASURY),
    MoreDestination("Payment config", "Gateway visibility", Icons.Default.Payment, WarehouseRoutes.PAYMENT_CONFIG),
    MoreDestination("Notifications", "Native alert inbox", Icons.Default.Notifications, WarehouseRoutes.NOTIFICATIONS),
)

private val portalAccount = listOf(
    MoreDestination("Warehouse setup", "Onboarding wizard", Icons.Default.Business, WarehouseRoutes.portalHandoff(WarehousePortalFeature.SETUP.routeKey)),
    MoreDestination("Profile", "Account settings", Icons.Default.Person, WarehouseRoutes.portalHandoff(WarehousePortalFeature.PROFILE.routeKey)),
    MoreDestination("Global search", "Portal navigation", Icons.Default.Search, WarehouseRoutes.portalHandoff(WarehousePortalFeature.SEARCH.routeKey)),
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MoreHubScreen(
    onNavigate: (String) -> Unit,
    onBack: (() -> Unit)? = null,
) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("More") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") } } },
            )
        },
    ) { innerPadding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            item { SectionHeader("Fulfillment") }
            fulfillment.forEach { dest ->
                item { MoreRow(dest, onNavigate) }
            }
            item { SectionHeader("Inventory") }
            inventory.forEach { dest ->
                item { MoreRow(dest, onNavigate) }
            }
            item { SectionHeader("Operations") }
            operations.forEach { dest ->
                item { MoreRow(dest, onNavigate) }
            }
            item { SectionHeader("Portal only") }
            portalAccount.forEach { dest ->
                item { MoreRow(dest, onNavigate) }
            }
        }
    }
}

@Composable
private fun SectionHeader(title: String) {
    Text(
        text = title,
        style = MaterialTheme.typography.titleSmall,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier.padding(
            horizontal = PegasusSpacing.lg,
            vertical = PegasusSpacing.md,
        ),
    )
}

@Composable
private fun MoreRow(
    dest: MoreDestination,
    onNavigate: (String) -> Unit,
) {
    ListItem(
        headlineContent = { Text(dest.title) },
        supportingContent = { Text(dest.subtitle) },
        leadingContent = { Icon(dest.icon, contentDescription = null) },
        modifier = Modifier.clickable { onNavigate(dest.route) },
    )
    HorizontalDivider()
}
