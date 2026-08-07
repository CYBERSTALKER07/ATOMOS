package com.pegasusx.retailer.ui.screens.dashboard.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.TrendingUp
import androidx.compose.material.icons.outlined.DeviceHub
import androidx.compose.material.icons.outlined.Insights
import androidx.compose.material.icons.rounded.History
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material.icons.rounded.Map
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.ShoppingCart
import androidx.compose.material.icons.rounded.Storefront
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.ui.components.modifiers.bounceCash
import com.pegasusx.retailer.ui.theme.SquircleShape

@Composable
fun ServiceGrid(
    activeOrderCount: Int,
    predictionCount: Int,
    onOpenCatalog: () -> Unit = {},
    onOpenOrders: () -> Unit = {},
    onOpenDeliveries: () -> Unit = {},
    onOpenInsights: () -> Unit = {},
    onOpenSuppliers: () -> Unit = {},
    onOpenProcurement: () -> Unit = {},
    onOpenProfile: () -> Unit = {},
    onOpenControlTower: () -> Unit = {},
) {
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            ServiceTile(
                title = stringResource(R.string.mobile_retailer_ui_buy_workspace),
                subtitle = "Browse products and restock",
                icon = Icons.Rounded.ShoppingCart,
                onClick = onOpenCatalog,
                modifier = Modifier
                    .weight(1f)
                    .height(152.dp),
            )
            ServiceTile(
                title = stringResource(R.string.portal_nav_orders),
                subtitle = "$activeOrderCount active now",
                icon = Icons.Rounded.Inventory2,
                onClick = onOpenOrders,
                modifier = Modifier
                    .weight(1f)
                    .height(152.dp),
            )
        }

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            ServiceTile(
                title = stringResource(R.string.mobile_retailer_ui_deliveries),
                subtitle = "Track inbound on map",
                icon = Icons.Rounded.Map,
                onClick = onOpenDeliveries,
                modifier = Modifier
                    .weight(1f)
                    .height(128.dp),
            )
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                    DashboardQuickAction(
                        icon = Icons.Outlined.Insights,
                        label = stringResource(R.string.portal_nav_analytics),
                        onClick = onOpenInsights,
                        modifier = Modifier.weight(1f),
                    )
                    DashboardQuickAction(
                        icon = Icons.Outlined.DeviceHub,
                        label = stringResource(R.string.portal_nav_control_tower),
                        onClick = onOpenControlTower,
                        modifier = Modifier.weight(1f),
                    )
                }
                ServiceTileCompact(
                    title = stringResource(R.string.retailer_desktop_residual_text_suppliers),
                    icon = Icons.Rounded.Storefront,
                    onClick = onOpenSuppliers,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            ServiceTileCompact(
                title = stringResource(R.string.portal_nav_procurement),
                icon = Icons.AutoMirrored.Rounded.TrendingUp,
                onClick = onOpenProcurement,
                modifier = Modifier.weight(1f),
            )
            ServiceTileCompact(
                title = stringResource(R.string.warehouse_portal_returns_text_history),
                icon = Icons.Rounded.History,
                onClick = onOpenOrders,
                modifier = Modifier.weight(1f),
            )
            ServiceTileCompact(
                title = stringResource(R.string.supplier_portal_supplier_shell_text_search),
                icon = Icons.Rounded.Search,
                onClick = onOpenCatalog,
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
fun ServiceTile(
    title: String,
    subtitle: String,
    icon: ImageVector,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier.bounceCash(onClick = onClick),
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surfaceContainer,
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(16.dp),
            verticalArrangement = Arrangement.SpaceBetween,
        ) {
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .background(
                        color = MaterialTheme.colorScheme.secondaryContainer,
                        shape = CircleShape,
                    ),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSecondaryContainer,
                )
            }

            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
fun ServiceTileCompact(
    title: String,
    icon: ImageVector,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier
            .height(88.dp)
            .bounceCash(onClick = onClick),
        shape = MaterialTheme.shapes.large,
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
    ) {
        Row(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .background(
                        color = MaterialTheme.colorScheme.primaryContainer,
                        shape = CircleShape,
                    ),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onPrimaryContainer,
                )
            }

            Text(
                text = title,
                style = MaterialTheme.typography.titleSmall,
                color = MaterialTheme.colorScheme.onSurface,
                fontWeight = FontWeight.Medium,
            )
        }
    }
}

@Composable
fun DashboardQuickAction(
    icon: ImageVector,
    label: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Surface(
        onClick = onClick,
        modifier = modifier.height(60.dp),
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant,
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = label,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(24.dp)
            )
            Text(
                text = label,
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}
