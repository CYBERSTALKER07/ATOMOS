package com.pegasus.design

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.SizeTransform
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.MenuOpen
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationDrawerItem
import androidx.compose.material3.NavigationDrawerItemDefaults
import androidx.compose.material3.NavigationRail
import androidx.compose.material3.NavigationRailItem
import androidx.compose.material3.NavigationRailItemDefaults
import androidx.compose.material3.PermanentDrawerSheet
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp

data class PegasusRailItem(
    val id: String,
    val label: String,
    val icon: ImageVector,
)

data class PegasusRailGroup(
    val title: String,
    val items: List<PegasusRailItem>,
)

@Composable
fun PegasusCollapsibleRail(
    appTitle: String,
    isExpanded: Boolean,
    onToggleExpanded: () -> Unit,
    groups: List<PegasusRailGroup>,
    selectedItemId: String?,
    onItemSelected: (PegasusRailItem) -> Unit,
    modifier: Modifier = Modifier,
    footer: @Composable (() -> Unit)? = null,
) {
    AnimatedContent(
        targetState = isExpanded,
        transitionSpec = {
            fadeIn(animationSpec = tween(220)) togetherWith
                fadeOut(animationSpec = tween(180)) using SizeTransform(clip = false)
        },
        label = "pegasus_rail",
        modifier = modifier,
    ) { expanded ->
        if (expanded) {
            ExpandedRailDrawer(
                appTitle = appTitle,
                onToggleExpanded = onToggleExpanded,
                groups = groups,
                selectedItemId = selectedItemId,
                onItemSelected = onItemSelected,
                footer = footer,
            )
        } else {
            CollapsedRail(
                onToggleExpanded = onToggleExpanded,
                groups = groups,
                selectedItemId = selectedItemId,
                onItemSelected = onItemSelected,
                footer = footer,
            )
        }
    }
}

@Composable
private fun ExpandedRailDrawer(
    appTitle: String,
    onToggleExpanded: () -> Unit,
    groups: List<PegasusRailGroup>,
    selectedItemId: String?,
    onItemSelected: (PegasusRailItem) -> Unit,
    footer: @Composable (() -> Unit)?,
) {
    PermanentDrawerSheet(
        modifier = Modifier
            .width(280.dp)
            .fillMaxHeight(),
        drawerContainerColor = MaterialTheme.colorScheme.surface,
        drawerContentColor = MaterialTheme.colorScheme.onSurfaceVariant,
    ) {
        Column(modifier = Modifier.fillMaxHeight()) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp, vertical = 16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                IconButton(onClick = onToggleExpanded) {
                    Icon(Icons.Default.MenuOpen, contentDescription = "Collapse sidebar")
                }
                Text(
                    text = appTitle,
                    style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Column(
                modifier = Modifier
                    .weight(1f)
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 8.dp),
            ) {
                groups.forEach { group ->
                    Text(
                        text = group.title.uppercase(),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                    )
                    group.items.forEach { item ->
                        val selected = selectedItemId == item.id
                        NavigationDrawerItem(
                            label = { Text(item.label) },
                            icon = { Icon(item.icon, contentDescription = item.label) },
                            selected = selected,
                            onClick = { onItemSelected(item) },
                            colors = NavigationDrawerItemDefaults.colors(
                                selectedContainerColor = MaterialTheme.colorScheme.primaryContainer,
                                selectedIconColor = MaterialTheme.colorScheme.onPrimaryContainer,
                                selectedTextColor = MaterialTheme.colorScheme.onPrimaryContainer,
                            ),
                        )
                    }
                    Spacer(Modifier.height(8.dp))
                }
            }
            footer?.invoke()
        }
    }
}

@Composable
private fun CollapsedRail(
    onToggleExpanded: () -> Unit,
    groups: List<PegasusRailGroup>,
    selectedItemId: String?,
    onItemSelected: (PegasusRailItem) -> Unit,
    footer: @Composable (() -> Unit)?,
) {
    NavigationRail(
        modifier = Modifier
            .width(88.dp)
            .fillMaxHeight(),
        containerColor = MaterialTheme.colorScheme.surface,
        header = {
            IconButton(onClick = onToggleExpanded, modifier = Modifier.padding(top = 8.dp)) {
                Icon(Icons.Default.Menu, contentDescription = "Expand sidebar")
            }
        },
    ) {
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            groups.flatMap { it.items }.forEach { item ->
                val selected = selectedItemId == item.id
                NavigationRailItem(
                    selected = selected,
                    onClick = { onItemSelected(item) },
                    icon = { Icon(item.icon, contentDescription = item.label) },
                    label = {
                        Text(
                            text = item.label,
                            style = MaterialTheme.typography.labelSmall,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    },
                    colors = NavigationRailItemDefaults.colors(
                        selectedIconColor = MaterialTheme.colorScheme.onPrimaryContainer,
                        selectedTextColor = MaterialTheme.colorScheme.onPrimaryContainer,
                        indicatorColor = MaterialTheme.colorScheme.primaryContainer,
                    ),
                )
            }
        }
        footer?.invoke()
        HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
    }
}
