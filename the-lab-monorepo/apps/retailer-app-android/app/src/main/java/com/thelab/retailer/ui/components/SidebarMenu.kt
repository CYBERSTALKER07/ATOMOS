package com.thelab.retailer.ui.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import com.thelab.retailer.ui.theme.MotionTokens
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import com.thelab.retailer.ui.theme.SquircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ExitToApp
import androidx.compose.material.icons.outlined.AutoAwesome
import androidx.compose.material.icons.outlined.BarChart
import androidx.compose.material.icons.outlined.GridView
import androidx.compose.material.icons.outlined.Inbox
import androidx.compose.material.icons.outlined.Insights
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.zIndex
import com.thelab.retailer.ui.components.modifiers.bounceClick
import com.thelab.retailer.ui.theme.StatusRed

enum class SidebarDestination(val label: String, val icon: ImageVector) {
    DASHBOARD("Dashboard", Icons.Outlined.GridView),
    PROCUREMENT("Procurement", Icons.Outlined.BarChart),
    INSIGHTS("Insights", Icons.Outlined.Insights),
    AUTO_ORDER("Auto-Order", Icons.Outlined.AutoAwesome),
    AI_PREDICTIONS("AI Predictions", Icons.Outlined.AutoAwesome),
    INBOX("Inbox", Icons.Outlined.Inbox),
    PROFILE("Profile", Icons.Outlined.Person),
    SETTINGS("Settings", Icons.Outlined.Settings),
}

@Composable
fun SidebarMenu(
    isOpen: Boolean,
    onDismiss: () -> Unit,
    onNavigate: (SidebarDestination) -> Unit = {},
    userName: String = "",
    companyName: String = "",
    modifier: Modifier = Modifier,
) {
    val screenWidth = LocalConfiguration.current.screenWidthDp.dp
    val menuWidth = minOf(screenWidth * 0.82f, 340.dp)

    // Drag-to-dismiss offset
    var dragOffset by remember { mutableFloatStateOf(0f) }

    Box(modifier = modifier.zIndex(100f)) {
        // Scrim
        AnimatedVisibility(
            visible = isOpen,
            enter = fadeIn(tween(MotionTokens.DurationShort4)),
            exit = fadeOut(tween(MotionTokens.DurationShort4)),
        ) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color.Black.copy(alpha = 0.5f))
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                    ) { onDismiss() },
            )
        }

        // Menu panel — MDC emphasized easing
        AnimatedVisibility(
            visible = isOpen,
            enter = slideInHorizontally(
                animationSpec = tween(
                    durationMillis = 400,
                    easing = MotionTokens.EasingEmphasizedDecelerate
                )
            ) { -it },
            exit = slideOutHorizontally(
                animationSpec = tween(
                    durationMillis = 200,
                    easing = MotionTokens.EasingEmphasizedAccelerate
                )
            ) { -it },
        ) {
            Box(
                modifier = Modifier
                    .fillMaxHeight()
                    .width(menuWidth)
                    .background(
                        MaterialTheme.colorScheme.surface,
                        RoundedCornerShape(topEnd = 20.dp, bottomEnd = 20.dp),
                    )
                    .pointerInput(Unit) {
                        detectHorizontalDragGestures(
                            onDragEnd = {
                                if (dragOffset < -80f) onDismiss()
                                dragOffset = 0f
                            },
                            onHorizontalDrag = { _, dragAmount ->
                                dragOffset += dragAmount
                            },
                        )
                    },
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(horizontal = 24.dp, vertical = 16.dp),
                ) {
                    Spacer(modifier = Modifier.height(48.dp))

                    // ── Header ──
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(
                            modifier = Modifier
                                .size(52.dp)
                                .clip(CircleShape)
                                .background(MaterialTheme.colorScheme.primary),
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                userName.firstOrNull()?.uppercase() ?: "?",
                                style = MaterialTheme.typography.titleMedium.copy(
                                    color = MaterialTheme.colorScheme.onPrimary,
                                    fontWeight = FontWeight.Bold,
                                ),
                            )
                        }
                        Spacer(modifier = Modifier.width(14.dp))
                        Column {
                            Text(
                                userName.ifBlank { "–" },
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                            )
                            Text(
                                companyName.ifBlank { "–" },
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(32.dp))
                    HorizontalDivider(
                        thickness = 0.5.dp,
                        color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.3f),
                    )
                    Spacer(modifier = Modifier.height(24.dp))

                    // ── Menu Items ──
                    SidebarDestination.entries.forEach { dest ->
                        SidebarMenuItem(
                            icon = dest.icon,
                            label = dest.label,
                            onClick = {
                                onNavigate(dest)
                                onDismiss()
                            },
                        )
                    }

                    Spacer(modifier = Modifier.weight(1f))

                    // ── Log Out ──
                    HorizontalDivider(
                        thickness = 0.5.dp,
                        color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.3f),
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    SidebarMenuItem(
                        icon = Icons.AutoMirrored.Outlined.ExitToApp,
                        label = "Log Out",
                        tint = StatusRed,
                        onClick = { onDismiss() },
                    )
                    Spacer(modifier = Modifier.height(16.dp))

                    // ── Version ──
                    Text(
                        "The Lab · v1.0.0",
                        style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                }
            }
        }
    }
}

@Composable
private fun SidebarMenuItem(
    icon: ImageVector,
    label: String,
    tint: Color = MaterialTheme.colorScheme.onSurface,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(SquircleShape)
            .bounceClick { onClick() }
            .padding(horizontal = 12.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Icon(
            icon,
            contentDescription = label,
            modifier = Modifier.size(22.dp),
            tint = tint.copy(alpha = 0.7f),
        )
        Text(
            label,
            style = MaterialTheme.typography.bodyLarge,
            fontWeight = FontWeight.Medium,
            color = tint,
        )
    }
}
