package com.pegasusx.driver.ui.navigation

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.automirrored.filled.ListAlt
import androidx.compose.material.icons.filled.Map
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.automirrored.outlined.ListAlt
import androidx.compose.material.icons.outlined.Map
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarDefaults
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.MenuOpen
import androidx.compose.material3.IconButton
import androidx.compose.material3.NavigationRail
import androidx.compose.material3.NavigationRailItem
import androidx.compose.material3.windowsizeclass.WindowWidthSizeClass
import com.pegasus.design.PegasusCollapsibleRail
import com.pegasus.design.PegasusRailGroup
import com.pegasus.design.PegasusRailItem
import androidx.compose.material3.windowsizeclass.WindowSizeClass
import com.pegasusx.driver.ui.theme.MotionTokens

// ── Tab Enum ──

enum class AppTab(
    val selectedIcon: ImageVector,
    val unselectedIcon: ImageVector,
    val label: String
) {
    HOME(Icons.Filled.Home, Icons.Outlined.Home, "Home"),
    MAP(Icons.Filled.Map, Icons.Outlined.Map, "Map"),
    RIDES(Icons.AutoMirrored.Filled.ListAlt, Icons.AutoMirrored.Outlined.ListAlt, "Rides"),
    PROFILE(Icons.Filled.Person, Icons.Outlined.Person, "Profile")
}

/**
 * MainTabView — M3 NavigationBar with MDC motion transitions.
 * Hosts the 4 main tabs: Home, Map, Rides, Profile.
 */
@Composable
fun MainTabView(
    homeContent: @Composable () -> Unit,
    mapContent: @Composable () -> Unit,
    ridesContent: @Composable () -> Unit,
    profileContent: @Composable () -> Unit,
    activeRideBar: (@Composable (onOpenMap: () -> Unit) -> Unit)? = null,
    windowSizeClass: WindowSizeClass? = null,
) {
    var selectedTab by remember { mutableStateOf(AppTab.HOME) }
    val openMapTab: () -> Unit = { selectedTab = AppTab.MAP }
    val useRail = windowSizeClass?.widthSizeClass != null &&
        windowSizeClass.widthSizeClass != WindowWidthSizeClass.Compact
    var isRailExpanded by remember { mutableStateOf(true) }

    val tabContent: @Composable () -> Unit = {
        Box(modifier = Modifier.fillMaxSize()) {
            AnimatedContent(
                targetState = selectedTab,
                transitionSpec = {
                    (fadeIn(
                        animationSpec = tween(
                            durationMillis = MotionTokens.DurationMedium2,
                            easing = MotionTokens.EasingEmphasizedDecelerate
                        )
                    ) + slideInVertically(
                        initialOffsetY = { it / 20 },
                        animationSpec = tween(
                            durationMillis = MotionTokens.DurationMedium2,
                            easing = MotionTokens.EasingEmphasizedDecelerate
                        )
                    )) togetherWith (fadeOut(
                        animationSpec = tween(
                            durationMillis = MotionTokens.DurationShort3,
                            easing = MotionTokens.EasingEmphasizedAccelerate
                        )
                    ))
                },
                label = "tab_content"
            ) { tab ->
                when (tab) {
                    AppTab.HOME -> homeContent()
                    AppTab.MAP -> mapContent()
                    AppTab.RIDES -> ridesContent()
                    AppTab.PROFILE -> profileContent()
                }
            }

            if (!useRail) {
                Column(
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .fillMaxWidth(),
                ) {
                    bottomChrome(selectedTab, openMapTab, activeRideBar) { selectedTab = it }
                }
            } else if (selectedTab != AppTab.MAP) {
                Column(
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .fillMaxWidth(),
                ) {
                    activeRideBar?.invoke(openMapTab)
                }
            }
        }
    }

    if (useRail) {
        Row(Modifier.fillMaxSize()) {
            PegasusCollapsibleRail(
                appTitle = "Pegasus Driver",
                isExpanded = isRailExpanded,
                onToggleExpanded = { isRailExpanded = !isRailExpanded },
                groups = listOf(
                    PegasusRailGroup(
                        "Primary",
                        AppTab.entries.map {
                            PegasusRailItem(it.name, it.label, it.selectedIcon)
                        },
                    ),
                ),
                selectedItemId = selectedTab.name,
                onItemSelected = { item ->
                    AppTab.entries.firstOrNull { it.name == item.id }?.let { selectedTab = it }
                },
            )
            tabContent()
        }
    } else {
        tabContent()
    }
}

@Composable
private fun bottomChrome(
    selectedTab: AppTab,
    openMapTab: () -> Unit,
    activeRideBar: (@Composable (onOpenMap: () -> Unit) -> Unit)?,
    onTabSelected: (AppTab) -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(0.dp)
    ) {
        if (selectedTab != AppTab.MAP) {
            activeRideBar?.invoke(openMapTab)
        }

        NavigationBar(
            modifier = Modifier.height(88.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            tonalElevation = 0.dp,
            windowInsets = NavigationBarDefaults.windowInsets,
        ) {
            AppTab.entries.forEach { tab ->
                val selected = tab == selectedTab
                NavigationBarItem(
                    selected = selected,
                    onClick = { onTabSelected(tab) },
                    icon = {
                        Icon(
                            imageVector = if (selected) tab.selectedIcon else tab.unselectedIcon,
                            contentDescription = tab.label,
                        )
                    },
                    label = {
                        Text(
                            tab.label,
                            style = MaterialTheme.typography.labelMedium,
                        )
                    },
                    colors = NavigationBarItemDefaults.colors(
                        selectedIconColor = MaterialTheme.colorScheme.onSurface,
                        selectedTextColor = MaterialTheme.colorScheme.onSurface,
                        indicatorColor = MaterialTheme.colorScheme.secondaryContainer,
                        unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    ),
                )
            }
        }
    }
}
