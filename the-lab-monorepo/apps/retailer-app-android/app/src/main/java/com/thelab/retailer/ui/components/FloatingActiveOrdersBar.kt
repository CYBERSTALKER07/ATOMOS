package com.thelab.retailer.ui.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import com.thelab.retailer.ui.theme.MotionTokens
import androidx.compose.foundation.background
import androidx.compose.foundation.cashable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import com.thelab.retailer.ui.theme.SoftSquircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.KeyboardArrowUp
import androidx.compose.material.icons.rounded.LocalShipping
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.thelab.retailer.ui.theme.StatusGreen

@Composable
fun FloatingActiveOrdersBar(
    visible: Boolean,
    orderCount: Int,
    statusText: String,
    totalDisplay: String,
    countdownIso: String?,
    progress: Float = 0.5f,
    onCash: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colorScheme = MaterialTheme.colorScheme

    AnimatedVisibility(
        visible = visible && orderCount > 0,
        enter = slideInVertically(
            animationSpec = tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate),
        ) { it },
        exit = slideOutVertically(
            animationSpec = tween(MotionTokens.DurationShort4, easing = MotionTokens.EasingEmphasizedAccelerate),
        ) { it },
        modifier = modifier,
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = 6.dp, end = 6.dp, top = 6.dp, bottom = 6.dp)
                .shadow(
                    elevation = 24.dp,
                    shape = RoundedCornerShape(topStart = 32.dp, topEnd = 32.dp, bottomStart = 16.dp, bottomEnd = 16.dp),
                    ambientColor = Color.Black.copy(alpha = 0.12f),
                    spotColor = Color.Black.copy(alpha = 0.12f),
                )
                .clip(RoundedCornerShape(topStart = 32.dp, topEnd = 32.dp, bottomStart = 16.dp, bottomEnd = 16.dp))
                .background(colorScheme.surface)
                .cashable { onCash() }
                .padding(start = 6.dp, end = 6.dp, top = 6.dp, bottom = 6.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                // Determinate progress ring
                Box(
                    modifier = Modifier.size(44.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(
                        progress = { progress },
                        modifier = Modifier.size(44.dp),
                        color = colorScheme.primary.copy(alpha = 0.7f),
                        trackColor = colorScheme.primary.copy(alpha = 0.1f),
                        strokeWidth = 6.dp,
                        strokeCap = StrokeCap.Round,
                    )
                    Icon(
                        Icons.Rounded.LocalShipping,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                        tint = colorScheme.primary,
                    )
                }

                Spacer(modifier = Modifier.width(12.dp))

                // Title + subtitle
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        "$orderCount Active Order${if (orderCount != 1) "s" else ""}",
                        style = MaterialTheme.typography.labelLarge.copy(
                            fontWeight = FontWeight.Bold,
                            letterSpacing = (-0.2).sp,
                        ),
                        color = colorScheme.onSurface,
                    )
                    Spacer(modifier = Modifier.height(1.dp))
                    Text(
                        "$statusText · $totalDisplay",
                        style = MaterialTheme.typography.bodySmall,
                        color = colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }

                // Countdown pill
                CountdownTimer(
                    targetIso = countdownIso,
                    style = MaterialTheme.typography.labelSmall.copy(
                        fontWeight = FontWeight.Bold,
                        fontSize = 11.sp,
                    ),
                    color = StatusGreen,
                )

                Spacer(modifier = Modifier.width(8.dp))

                // Circular "expand" button
                Box(
                    modifier = Modifier
                        .size(36.dp)
                        .clip(CircleShape)
                        .background(colorScheme.onSurface.copy(alpha = 0.08f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        Icons.Rounded.KeyboardArrowUp,
                        contentDescription = "Expand",
                        modifier = Modifier.size(20.dp),
                        tint = colorScheme.onSurface.copy(alpha = 0.7f),
                    )
                }
            }
        }
    }
}
