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
    onExceptions: () -> Unit,
    onShopClosed: () -> Unit,
    onNegotiations: () -> Unit,
    onManifests: () -> Unit,
    onDispatch: () -> Unit,
    onActivity: () -> Unit,
    onFleetOrders: () -> Unit,
    onLedger: () -> Unit,
    onOperations: () -> Unit,
    onInventory: () -> Unit,
    onPromotions: () -> Unit,
    onEarnings: () -> Unit,
    onProfile: () -> Unit,
    onBilling: () -> Unit,
    onSignOut: () -> Unit,
) {
    val fulfillment = listOf(
        MoreDestination("Manifests", "Loading manifests", Icons.Default.Description, onManifests),
        MoreDestination("Dispatch preview", "Pending orders & drivers", Icons.Default.LocalShipping, onDispatch),
        MoreDestination("Fleet orders", "In-flight assignments", Icons.Default.Route, onFleetOrders),
    )
    val exceptions = listOf(
        MoreDestination("Exceptions", "Operational queue", Icons.Default.Warning, onExceptions),
        MoreDestination("Shop closed", "Driver wait cases", Icons.Default.Store, onShopClosed),
        MoreDestination("Negotiations", "Quantity proposals", Icons.Default.Handshake, onNegotiations),
    )
    val insights = listOf(
        MoreDestination("Activity", "Recent events", Icons.Default.Timeline, onActivity),
        MoreDestination("Payment ledger", "Treasury entries", Icons.Default.AccountBalance, onLedger),
        MoreDestination("Operations", "Replenishment trigger", Icons.Default.Build, onOperations),
    )
    val account = listOf(
        MoreDestination("Inventory", "SKU levels", Icons.Default.Inventory2, onInventory),
        MoreDestination("Promotions", "Sales and discounts", Icons.Default.LocalOffer, onPromotions),
        MoreDestination("Earnings", "Revenue summary", Icons.Default.Payments, onEarnings),
        MoreDestination("Profile", "Company profile", Icons.Default.Person, onProfile),
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
            item { SectionHeader("Exceptions") }
            exceptions.forEach { item { MoreRow(it) } }
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
        headlineContent = { Text(dest.title) },
        supportingContent = { Text(dest.subtitle) },
        leadingContent = { Icon(dest.icon, contentDescription = null) },
        modifier = Modifier.clickable(onClick = dest.onClick),
    )
    HorizontalDivider()
}
