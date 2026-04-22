package com.thelab.retailer.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.thelab.retailer.data.model.Order
import com.thelab.retailer.data.model.OrderStatus
import com.thelab.retailer.ui.theme.PillShape
import com.thelab.retailer.ui.theme.SoftSquircleShape
import com.thelab.retailer.ui.theme.StatusBlue
import com.thelab.retailer.ui.theme.StatusBlueSoft
import com.thelab.retailer.ui.theme.StatusGreen
import com.thelab.retailer.ui.theme.StatusGreenSoft
import com.thelab.retailer.ui.theme.StatusOrange
import com.thelab.retailer.ui.theme.StatusOrangeSoft
import com.thelab.retailer.ui.theme.StatusRed
import com.thelab.retailer.ui.theme.StatusRedSoft
import com.thelab.retailer.ui.theme.StatusTeal
import com.thelab.retailer.ui.theme.StatusTealSoft

/**
 * Order Card — B&W minimalist matching iOS activeOrderCard / pendingOrderCard.
 * White card, subtle shadow, status dot + badge capsule.
 */
@Composable
fun OrderCard(
    order: Order,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier
            .fillMaxWidth()
            .shadow(4.dp, SoftSquircleShape, spotColor = Color.Black.copy(alpha = 0.06f))
            .clip(SoftSquircleShape)
            .clickable { onClick() },
        color = MaterialTheme.colorScheme.surface,
        shape = SoftSquircleShape,
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.Top,
        ) {
            // Status circle
            val statusColor = order.status.statusColor()
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .clip(CircleShape)
                    .background(statusColor.copy(alpha = 0.1f)),
                contentAlignment = Alignment.Center,
            ) {
                Box(
                    modifier = Modifier
                        .size(10.dp)
                        .clip(CircleShape)
                        .background(statusColor),
                )
            }

            Spacer(Modifier.width(12.dp))

            Column(modifier = Modifier.weight(1f)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        "Order #${order.id.takeLast(3)}",
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                    )
                    OrderStatusBadge(order.status)
                }

                Spacer(Modifier.height(2.dp))

                Text(
                    "${order.itemCount} items · ${order.displayTotal}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
fun OrderStatusBadge(status: OrderStatus, modifier: Modifier = Modifier) {
    val (bg, fg) = status.badgeColors()
    Row(
        modifier = modifier
            .clip(PillShape)
            .background(bg)
            .padding(horizontal = 10.dp, vertical = 5.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Box(
            modifier = Modifier
                .size(6.dp)
                .clip(CircleShape)
                .background(fg),
        )
        Text(
            status.displayName,
            style = MaterialTheme.typography.labelSmall.copy(
                fontSize = 11.sp,
                fontWeight = FontWeight.Bold,
            ),
            color = fg,
        )
    }
}

fun OrderStatus.statusColor(): Color = when (this) {
    OrderStatus.PENDING -> StatusOrange
    OrderStatus.LOADED -> StatusBlue
    OrderStatus.DISPATCHED -> StatusTeal
    OrderStatus.IN_TRANSIT -> StatusGreen
    OrderStatus.ARRIVED -> StatusGreen
    OrderStatus.COMPLETED -> StatusGreen
    OrderStatus.CANCELLED -> StatusRed
    OrderStatus.AWAITING_PAYMENT -> StatusOrange
    OrderStatus.PENDING_CASH_COLLECTION -> StatusOrange
}

private fun OrderStatus.badgeColors(): Pair<Color, Color> = when (this) {
    OrderStatus.PENDING -> StatusOrangeSoft to StatusOrange
    OrderStatus.LOADED -> StatusBlueSoft to StatusBlue
    OrderStatus.DISPATCHED -> StatusTealSoft to StatusTeal
    OrderStatus.IN_TRANSIT -> StatusGreenSoft to StatusGreen
    OrderStatus.ARRIVED -> StatusGreenSoft to StatusGreen
    OrderStatus.COMPLETED -> StatusGreenSoft to StatusGreen
    OrderStatus.CANCELLED -> StatusRedSoft to StatusRed
    OrderStatus.AWAITING_PAYMENT -> StatusOrangeSoft to StatusOrange
    OrderStatus.PENDING_CASH_COLLECTION -> StatusOrangeSoft to StatusOrange
}
