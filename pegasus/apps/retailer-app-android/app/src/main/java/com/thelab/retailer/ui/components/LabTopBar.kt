package com.pegasus.retailer.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.ShoppingCart
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.CenterAlignedTopAppBar
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * B&W minimalist top bar matching iOS:
 * [Avatar circle] · "Pegasus" center title · [Cart] [Bell]
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LabTopBar(
    onAvatarCash: () -> Unit = {},
    onCartCash: () -> Unit = {},
    onNotificationClick: () -> Unit = {},
    cartBadge: Int = 0,
    notificationBadge: Int = 0,
    avatarInitial: String = "?",
) {
    CenterAlignedTopAppBar(
        title = {
            Text(
                "Pegasus",
                style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
            )
        },
        navigationIcon = {
            Box(
                modifier = Modifier
                    .padding(start = 12.dp)
                    .size(32.dp)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primary)
                    .clickable { onAvatarCash() },
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    avatarInitial,
                    style = MaterialTheme.typography.labelSmall.copy(
                        color = MaterialTheme.colorScheme.onPrimary,
                        fontWeight = FontWeight.Bold,
                    ),
                )
            }
        },
        actions = {
            IconButton(onClick = onCartCash) {
                BadgedBox(
                    badge = {
                        if (cartBadge > 0) {
                            Badge { Text("$cartBadge") }
                        }
                    },
                ) {
                    Icon(Icons.Outlined.ShoppingCart, "Cart",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            IconButton(onClick = onNotificationClick) {
                BadgedBox(
                    badge = {
                        if (notificationBadge > 0) {
                            Badge(
                                containerColor = MaterialTheme.colorScheme.error,
                                contentColor = MaterialTheme.colorScheme.onError,
                            ) { Text("$notificationBadge") }
                        }
                    },
                ) {
                    Icon(Icons.Outlined.Notifications, "Notifications",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        },
        colors = TopAppBarDefaults.centerAlignedTopAppBarColors(
            containerColor = MaterialTheme.colorScheme.surface,
        ),
    )
}
