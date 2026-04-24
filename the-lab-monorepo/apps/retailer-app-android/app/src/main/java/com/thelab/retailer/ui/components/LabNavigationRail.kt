package com.thelab.retailer.ui.components
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.MenuOpen
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.SizeTransform
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import com.thelab.retailer.ui.theme.MotionTokens
import com.thelab.retailer.ui.components.modifiers.bounceCash

@Composable
fun LabNavigationRail(
    isExpanded: Boolean,
    onToggleExpanded: () -> Unit,
    currentTab: LabTab,
    onTabSelected: (LabTab) -> Unit,
    onSidebarNavigate: (SidebarDestination) -> Unit,
    userName: String,
    companyName: String,
    modifier: Modifier = Modifier
) {
    AnimatedContent(
        targetState = isExpanded,
        transitionSpec = {
            fadeIn(animationSpec = tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate)) togetherWith
            fadeOut(animationSpec = tween(MotionTokens.DurationShort4, easing = MotionTokens.EasingEmphasizedAccelerate)) using
            SizeTransform(clip = false)
        },
        label = "rail_expansion"
    ) { expanded ->
        if (expanded) {
            ExpandedDrawer(
                onToggleExpanded = onToggleExpanded,
                currentTab = currentTab,
                onTabSelected = onTabSelected,
                onSidebarNavigate = onSidebarNavigate,
                userName = userName,
                companyName = companyName,
                modifier = modifier
            )
        } else {
            CollapsedRail(
                onToggleExpanded = onToggleExpanded,
                currentTab = currentTab,
                onTabSelected = onTabSelected,
                onSidebarNavigate = onSidebarNavigate,
                userName = userName,
                modifier = modifier
            )
        }
    }
}

@Composable
private fun ExpandedDrawer(
    onToggleExpanded: () -> Unit,
    currentTab: LabTab,
    onTabSelected: (LabTab) -> Unit,
    onSidebarNavigate: (SidebarDestination) -> Unit,
    userName: String,
    companyName: String,
    modifier: Modifier = Modifier
) {
    PermanentDrawerSheet(
        modifier = modifier.width(280.dp),
        drawerContainerColor = MaterialTheme.colorScheme.surface,
        drawerContentColor = MaterialTheme.colorScheme.onSurfaceVariant
    ) {
        // Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            IconButton(onClick = onToggleExpanded) {
                Icon(
                    imageVector = Icons.Default.MenuOpen,
                    contentDescription = "Collapse menu",
                    tint = MaterialTheme.colorScheme.onSurface
                )
            }
            Spacer(modifier = Modifier.width(12.dp))
            Text(
                text = companyName.ifBlank { "The Lab" },
                style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                color = MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
        }

        // Content
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 12.dp)
        ) {
            LabTab.entries.forEach { tab ->
                val selected = currentTab == tab
                NavigationDrawerItem(
                    label = { Text(tab.label) },
                    icon = { Icon(if (selected) tab.selectedIcon else tab.unselectedIcon, contentDescription = tab.label) },
                    selected = selected,
                    onClick = { onTabSelected(tab) },
                    colors = NavigationDrawerItemDefaults.colors(
                        unselectedContainerColor = MaterialTheme.colorScheme.surface,
                        selectedContainerColor = MaterialTheme.colorScheme.primaryContainer,
                        selectedIconColor = MaterialTheme.colorScheme.onPrimaryContainer,
                        selectedTextColor = MaterialTheme.colorScheme.onPrimaryContainer,
                        unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                )
            }

            Spacer(modifier = Modifier.height(16.dp))
            HorizontalDivider(modifier = Modifier.padding(horizontal = 16.dp))
            Spacer(modifier = Modifier.height(16.dp))

            // Sidebar Destinations
            SidebarDestination.entries.forEach { destination ->
                NavigationDrawerItem(
                    label = { Text(destination.label) },
                    icon = { Icon(destination.icon, contentDescription = destination.label) },
                    selected = false,
                    onClick = { onSidebarNavigate(destination) },
                    colors = NavigationDrawerItemDefaults.colors(
                        unselectedContainerColor = MaterialTheme.colorScheme.surface,
                        unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                )
            }
        }

        // Footer (User Profile)
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primary),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = userName.firstOrNull()?.uppercase() ?: "?",
                    style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                    color = MaterialTheme.colorScheme.onPrimary
                )
            }
            Spacer(modifier = Modifier.width(16.dp))
            Column {
                Text(
                    text = userName.ifBlank { "User" },
                    style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.Bold),
                    color = MaterialTheme.colorScheme.onSurface,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    text = "View Profile",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

@Composable
private fun CollapsedRail(
    onToggleExpanded: () -> Unit,
    currentTab: LabTab,
    onTabSelected: (LabTab) -> Unit,
    onSidebarNavigate: (SidebarDestination) -> Unit,
    userName: String,
    modifier: Modifier = Modifier
) {
    NavigationRail(
        modifier = modifier.width(80.dp),
        containerColor = MaterialTheme.colorScheme.surface,
        contentColor = MaterialTheme.colorScheme.onSurfaceVariant,
        header = {
            IconButton(
                onClick = onToggleExpanded,
                modifier = Modifier.padding(top = 16.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Menu,
                    contentDescription = "Expand menu",
                    tint = MaterialTheme.colorScheme.onSurface
                )
            }
        }
    ) {
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(top = 16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            LabTab.entries.forEach { tab ->
                val selected = currentTab == tab
                NavigationRailItem(
                    selected = selected,
                    onClick = { onTabSelected(tab) },
                    icon = { 
                        Icon(
                            imageVector = if (selected) tab.selectedIcon else tab.unselectedIcon, 
                            contentDescription = tab.label,
                            modifier = Modifier.size(32.dp)
                        ) 
                    },
                    label = { 
                        Text(
                            text = tab.label,
                            style = MaterialTheme.typography.labelMedium,
                            maxLines = 1, 
                            overflow = TextOverflow.Ellipsis
                        ) 
                    },
                    colors = NavigationRailItemDefaults.colors(
                        unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        selectedIconColor = MaterialTheme.colorScheme.onPrimaryContainer,
                        selectedTextColor = MaterialTheme.colorScheme.onPrimaryContainer,
                        indicatorColor = MaterialTheme.colorScheme.primaryContainer
                    )
                )
            }

            Spacer(modifier = Modifier.height(8.dp))
            HorizontalDivider(modifier = Modifier.padding(horizontal = 16.dp))
            Spacer(modifier = Modifier.height(8.dp))

            SidebarDestination.entries.forEach { destination ->
                NavigationRailItem(
                    selected = false,
                    onClick = { onSidebarNavigate(destination) },
                    icon = { 
                        Icon(
                            imageVector = destination.icon, 
                            contentDescription = destination.label,
                            modifier = Modifier.size(32.dp)
                        ) 
                    },
                    label = { 
                        Text(
                            text = destination.label,
                            style = MaterialTheme.typography.labelMedium,
                            maxLines = 1, 
                            overflow = TextOverflow.Ellipsis
                        ) 
                    },
                    colors = NavigationRailItemDefaults.colors(
                        unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                )
            }
        }

        // Small Profile Avatar at Bottom
        Box(
            modifier = Modifier
                .padding(bottom = 24.dp)
                .size(40.dp)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.primary)
                .bounceCash { /* Handle Profile Cash if needed */ },
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = userName.firstOrNull()?.uppercase() ?: "?",
                style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                color = MaterialTheme.colorScheme.onPrimary
            )
        }
    }
}
