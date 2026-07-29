package com.pegasusx.supplier.ui.screens.more

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import com.pegasusx.supplier.ui.components.SupplierLeadingIcon
import com.pegasusx.supplier.ui.theme.PegasusSpacing

private data class MoreDestination(
    val title: String,
    val subtitle: String,
    val icon: ImageVector,
    val onClick: () -> Unit,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MoreScreen(
    onManifests: () -> Unit,
    onDispatch: () -> Unit,
    onActivity: () -> Unit,
    onFleetOrders: () -> Unit,
    onLedger: () -> Unit,
    onOperations: () -> Unit,
    onReplenishmentPolicies: () -> Unit,
    onAnalytics: () -> Unit,
    onAiRecommendations: () -> Unit,
    onGeoReport: () -> Unit,
    onTopology: () -> Unit,
    onDeliveryZones: () -> Unit,
    onSupplyLanes: () -> Unit,
    onPayments: () -> Unit,
    onInventory: () -> Unit,
    onCatalog: () -> Unit,
    onPromotions: () -> Unit,
    onPricing: () -> Unit,
    onReturns: () -> Unit,
    onReconciliation: () -> Unit,
    onOrgFleet: () -> Unit,
    onEarnings: () -> Unit,
    onProfile: () -> Unit,
    onNotifications: () -> Unit,
    onBilling: () -> Unit,
    onBusinessSetup: () -> Unit,
    onChargebacks: () -> Unit,
    onClaims: () -> Unit = {},
    onClaimChargebacks: () -> Unit = {},
    onRetailerOverrides: () -> Unit,
    onInventoryImport: () -> Unit,
    onTreasuryHub: () -> Unit,
    onDemandHistory: () -> Unit,
    onFactories: () -> Unit,
    onWarehouses: () -> Unit,
    onSignOut: () -> Unit,
) {
    val fulfillment = listOf(
        MoreDestination("Manifests", "Loading manifests", Icons.Default.Description, onManifests),
        MoreDestination("Dispatch preview", "Pending orders & drivers", Icons.Default.LocalShipping, onDispatch),
        MoreDestination("Fleet orders", "In-flight assignments", Icons.Default.Route, onFleetOrders),
    )
    val intelligence = listOf(
        MoreDestination("Analytics", "Operational metrics", Icons.Default.BarChart, onAnalytics),
        MoreDestination("Demand history", "14-day forecast vs actual", Icons.Default.History, onDemandHistory),
        MoreDestination("AI recommendations", "Advisory queue & decisions", Icons.Default.AutoAwesome, onAiRecommendations),
        MoreDestination("Geo report", "H3 coverage", Icons.Default.Public, onGeoReport),
    )
    val network = listOf(
        MoreDestination("Factories & warehouses", "Topology overview", Icons.Default.Apartment, onTopology),
        MoreDestination("Factories", "Production nodes", Icons.Default.PrecisionManufacturing, onFactories),
        MoreDestination("Warehouses", "Distribution nodes", Icons.Default.Store, onWarehouses),
        MoreDestination("Delivery zones", "Coverage areas", Icons.Default.Map, onDeliveryZones),
        MoreDestination("Supply lanes", "Lane utilization", Icons.Default.AltRoute, onSupplyLanes),
    )
    val treasury = listOf(
        MoreDestination("Treasury hub", "KPIs and finance modules", Icons.Default.AccountBalance, onTreasuryHub),
        MoreDestination("Payment ledger", "Treasury entries", Icons.Default.AccountBalance, onLedger),
        MoreDestination("Payments", "Settlement authority", Icons.Default.CreditCard, onPayments),
        MoreDestination("Chargebacks", "Record chargeback or reversal", Icons.Default.Payments, onChargebacks),
        MoreDestination("Claims queue", "Approve post-delivery OS&D claims", Icons.Default.Warning, onClaims),
        MoreDestination("Claim chargebacks", "Logistics claim settlements", Icons.Default.Description, onClaimChargebacks),
        MoreDestination("Reconciliation", "Settlement mismatches", Icons.Default.Balance, onReconciliation),
        MoreDestination("Operations", "Bypass, broadcast & replenishment", Icons.Default.Build, onOperations),
        MoreDestination("Replenishment policies", "Warehouse supply rules", Icons.Default.Description, onReplenishmentPolicies),
    )
    val insights = listOf(
        MoreDestination("Activity", "Recent events", Icons.Default.Timeline, onActivity),
    )
    val account = listOf(
        MoreDestination("Notifications", "Inbox & alerts", Icons.Default.Notifications, onNotifications),
        MoreDestination("Catalog", "Product unit VU for dispatch", Icons.Default.Category, onCatalog),
        MoreDestination("Inventory", "SKU levels", Icons.Default.Inventory2, onInventory),
        MoreDestination("Inventory import", "Bulk CSV wizard", Icons.Default.Upload, onInventoryImport),
        MoreDestination("Retailer overrides", "Per-retailer pricing", Icons.Default.PriceCheck, onRetailerOverrides),
        MoreDestination("Pricing", "Catalog list and sale pricing", Icons.Default.PriceChange, onPricing),
        MoreDestination("Promotions", "Sales and discounts", Icons.Default.LocalOffer, onPromotions),
        MoreDestination("Returns", "Cancelled and rejected orders", Icons.Default.Undo, onReturns),
        MoreDestination("Org & fleet", "Drivers, vehicles, staff", Icons.Default.Groups, onOrgFleet),
        MoreDestination("Earnings", "Revenue summary", Icons.Default.Payments, onEarnings),
        MoreDestination("Profile", "Company profile", Icons.Default.Person, onProfile),
        MoreDestination("Business setup", "Tax and headquarters info", Icons.Default.Business, onBusinessSetup),
        MoreDestination("Billing setup", "Banking & gateway", Icons.Default.CreditCard, onBilling),
    )

    Scaffold(topBar = { TopAppBar(title = { Text("More") }) }) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            item { SectionHeader("Fulfillment") }
            fulfillment.forEach { item { MoreRow(it) } }
            item { SectionHeader("Intelligence") }
            intelligence.forEach { item { MoreRow(it) } }
            item { SectionHeader("Network") }
            network.forEach { item { MoreRow(it) } }
            item { SectionHeader("Treasury") }
            treasury.forEach { item { MoreRow(it) } }
            item { SectionHeader("Insights") }
            insights.forEach { item { MoreRow(it) } }
            item { SectionHeader("Account") }
            account.forEach { item { MoreRow(it) } }
            item {
                HorizontalDivider(Modifier.padding(vertical = PegasusSpacing.md))
                MoreRow(
                    MoreDestination("Sign out", "End session", Icons.AutoMirrored.Filled.ExitToApp, onSignOut),
                )
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
private fun MoreRow(dest: MoreDestination) {
    ListItem(
        headlineContent = { Text(dest.title, style = MaterialTheme.typography.titleSmall) },
        supportingContent = {
            Text(
                dest.subtitle,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        },
        leadingContent = { SupplierLeadingIcon(icon = dest.icon) },
        modifier = Modifier.clickable(onClick = dest.onClick),
    )
    HorizontalDivider()
}
