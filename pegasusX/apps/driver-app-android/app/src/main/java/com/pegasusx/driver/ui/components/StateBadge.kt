package com.pegasusx.driver.ui.components

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.data.remote.ConnectionState
import com.pegasusx.driver.ui.theme.Destructive
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.MotionTokens
import com.pegasusx.driver.ui.theme.StatusBlue
import com.pegasusx.driver.ui.theme.Success
import com.pegasusx.driver.ui.theme.Warning

/**
 * StatusPill — monospaced capsule badge with tinted M3 surface.
 */
@Composable
fun StatusPill(
    label: String,
    color: Color,
    modifier: Modifier = Modifier
) {
    Box(
        modifier = modifier
            .clip(CircleShape)
            .background(color.copy(alpha = 0.12f))
            .padding(horizontal = 10.dp, vertical = 5.dp)
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall.copy(
                fontFamily = FontFamily.Monospace,
                fontWeight = FontWeight.Bold,
                color = color,
            ),
        )
    }
}

/**
 * StateBadge — wrapper over StatusPill for OrderState.
 */
@Composable
fun StateBadge(state: OrderState) {
    val colorScheme = MaterialTheme.colorScheme
    val (color, text) = when (state) {
        OrderState.PENDING -> colorScheme.onSurfaceVariant to "PENDING"
        OrderState.LOADED -> colorScheme.onSurface to "LOADED"
        OrderState.IN_TRANSIT -> StatusBlue to "IN TRANSIT"
        OrderState.ARRIVING -> Warning to "ARRIVING"
        OrderState.ARRIVED -> Success to "ARRIVED"
        OrderState.ARRIVED_SHOP_CLOSED -> Warning to "SHOP CLOSED"
        OrderState.COMPLETED -> Success to "COMPLETED"
        OrderState.CANCELLED -> Destructive to "CANCELLED"
        OrderState.AWAITING_PAYMENT -> Warning to "AWAITING PAYMENT"
        OrderState.DISPATCHED -> StatusBlue to "DISPATCHED"
        OrderState.PENDING_CASH_COLLECTION -> Warning to "CASH COLLECTION"
        OrderState.CANCEL_REQUESTED -> StatusBlue to "CANCEL REQUESTED"
        OrderState.NO_CAPACITY -> Destructive to "NO CAPACITY"
        OrderState.QUARANTINE -> Destructive to "QUARANTINE"
        OrderState.DELIVERED_ON_CREDIT -> Warning to "ON CREDIT"
        else -> colorScheme.onSurfaceVariant to state.name.replace("_", " ")
    }
    StatusPill(label = text, color = color)
}

/**
 * WsConnectionPill — manifest header indicator for driver command-socket state.
 * Mirrors iOS TelemetryBadge LIVE/OFFLINE semantics for manifest refresh channel.
 */
@Composable
fun WsConnectionPill(
    state: ConnectionState,
    modifier: Modifier = Modifier,
) {
    val lab = LocalPegasusColors.current
    when (state) {
        ConnectionState.CONNECTED -> LiveWsConnectionPill(color = lab.live, modifier = modifier)
        ConnectionState.RECONNECTING -> StatusPill(label = "SYNCING", color = lab.warning, modifier = modifier)
        ConnectionState.DISCONNECTED -> StatusPill(label = "OFFLINE", color = lab.fgTertiary, modifier = modifier)
    }
}

@Composable
private fun LiveWsConnectionPill(
    color: Color,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier
            .clip(CircleShape)
            .background(color.copy(alpha = 0.12f))
            .padding(horizontal = 10.dp, vertical = 5.dp)
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            PulsingConnectionDot(color = color)
            Text(
                text = "LIVE",
                style = MaterialTheme.typography.labelSmall.copy(
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                    color = color,
                ),
            )
        }
    }
}

@Composable
private fun PulsingConnectionDot(color: Color) {
    val transition = rememberInfiniteTransition(label = "ws_pulse")
    val alpha by transition.animateFloat(
        initialValue = 1f,
        targetValue = 0.35f,
        animationSpec = infiniteRepeatable(
            animation = tween(MotionTokens.DurationExtraLong4),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "ws_pulse_alpha",
    )
    Box(
        modifier = Modifier
            .size(7.dp)
            .alpha(alpha)
            .clip(CircleShape)
            .background(color),
    )
}
