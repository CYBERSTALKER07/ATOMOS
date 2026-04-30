package com.pegasus.driver.ui.navigation

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
import com.pegasus.driver.ui.theme.MotionTokens

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
    activeRideBar: @Composable (() -> Unit)? = null
) {
    var selectedTab by remember { mutableStateOf(AppTab.HOME) }

    Box(modifier = Modifier.fillMaxSize()) {
        // Tab content with M3 emphasized decelerate enter / accelerate exit
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

        // Bottom overlay: active ride bar + M3 NavigationBar
        Column(
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(0.dp)
        ) {
            // Active ride bar (shown when route active)
            activeRideBar?.invoke()

            // M3 NavigationBar
            NavigationBar(
                modifier = Modifier.height(80.dp),
                containerColor = MaterialTheme.colorScheme.surfaceContainer,
                tonalElevation = 0.dp,
                windowInsets = WindowInsets(0, 0, 0, 0),
            ) {
                AppTab.entries.forEach { tab ->
                    val selected = tab == selectedTab
                    NavigationBarItem(
                        selected = selected,
                        onClick = { selectedTab = tab },
                        icon = {
                            Icon(
                                imageVector = if (selected) tab.selectedIcon else tab.unselectedIcon,
                                contentDescription = tab.label,
                            )
                        },
                        label = {
                            Text(
                                tab.label,
                                style = MaterialTheme.typography.labelSmall,
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
}
