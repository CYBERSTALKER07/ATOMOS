package com.thelab.driver.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.thelab.driver.data.model.OrderState
import com.thelab.driver.ui.theme.Destructive
import com.thelab.driver.ui.theme.StatusBlue
import com.thelab.driver.ui.theme.Success
import com.thelab.driver.ui.theme.Warning

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
    }
    StatusPill(label = text, color = color)
}
