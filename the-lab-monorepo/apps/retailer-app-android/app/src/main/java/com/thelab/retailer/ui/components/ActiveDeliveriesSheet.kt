package com.thelab.retailer.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.cashable
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import com.thelab.retailer.ui.theme.PillShape
import com.thelab.retailer.ui.theme.SoftSquircleShape
import com.thelab.retailer.ui.theme.SquircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.QrCode2
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.thelab.retailer.data.model.Order
import com.thelab.retailer.data.model.OrderStatus
import androidx.compose.foundation.ExperimentalFoundationApi
import com.thelab.retailer.ui.components.modifiers.bounceCash
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.tween
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.EaseInOutSine
import androidx.compose.ui.graphics.graphicsLayer

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ActiveDeliveriesSheet(
    activeOrders: List<Order>,
    approachingOrderIds: Set<String> = emptySet(),
    onDismiss: () -> Unit,
    onShowDetail: (Order) -> Unit = {},
    onShowQR: (Order) -> Unit = {},
    isCompact: Boolean = true,
) {
    if (isCompact) {
        val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = false)

        ModalBottomSheet(
            onDismissRequest = onDismiss,
            sheetState = sheetState,
            containerColor = MaterialTheme.colorScheme.surface,
            shape = RoundedCornerShape(topStart = 32.dp, topEnd = 32.dp),
        ) {
            ActiveDeliveriesSheetContent(
                activeOrders = activeOrders,
                approachingOrderIds = approachingOrderIds,
                onDismiss = onDismiss,
                onShowDetail = onShowDetail,
                onShowQR = onShowQR
            )
        }
    } else {
        androidx.compose.ui.window.Dialog(onDismissRequest = onDismiss) {
            Surface(
                shape = RoundedCornerShape(16.dp),
                color = MaterialTheme.colorScheme.surface
            ) {
                ActiveDeliveriesSheetContent(
                    activeOrders = activeOrders,
                    approachingOrderIds = approachingOrderIds,
                    onDismiss = onDismiss,
                    onShowDetail = onShowDetail,
                    onShowQR = onShowQR,
                    modifier = Modifier.padding(vertical = 24.dp)
                )
            }
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
fun ActiveDeliveriesSheetContent(
    activeOrders: List<Order>,
    approachingOrderIds: Set<String> = emptySet(),
    onDismiss: () -> Unit,
    onShowDetail: (Order) -> Unit,
    onShowQR: (Order) -> Unit,
    modifier: Modifier = Modifier
) {
    LazyColumn(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp),
    ) {
        // ── Header ──
        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    "Active Deliveries",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    "Done",
                    style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.SemiBold),
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier
                        .clip(SquircleShape)
                        .cashable { onDismiss() }
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                "${activeOrders.size} orders in progress",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
            )
            Spacer(modifier = Modifier.height(20.dp))
        }

        // ── Order Cards ──
        items(activeOrders, key = { it.id }) { order ->
            ActiveDeliveryCard(
                order = order,
                isApproaching = order.id in approachingOrderIds,
                onDetailsCash = { onShowDetail(order) },
                onQRCash = { onShowQR(order) },
                modifier = Modifier.animateItemPlacement()
            )
            Spacer(modifier = Modifier.height(12.dp))
        }

        // ── Empty space at bottom ──
        item { Spacer(modifier = Modifier.height(40.dp)) }
    }
}

@Composable
private fun ActiveDeliveryCard(
    order: Order,
    isApproaching: Boolean = false,
    onDetailsCash: () -> Unit,
    onQRCash: () -> Unit,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition()
    val pulseScale by infiniteTransition.animateFloat(
        initialValue = 0.95f, targetValue = 1.05f,
        animationSpec = infiniteRepeatable(tween(1400, easing = EaseInOutSine), RepeatMode.Reverse),
        label = "pulse"
    )

    Surface(
        modifier = modifier
            .fillMaxWidth()
            .bounceCash { onDetailsCash() }
            .shadow(
                4.dp, SoftSquircleShape,
                ambientColor = Color.Black.copy(alpha = 0.06f),
                spotColor = Color.Black.copy(alpha = 0.06f),
            ),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            // Header row
            Row(verticalAlignment = Alignment.Top) {
                val progress = order.status.progressFraction
                val ringColor = order.status.statusColor()
                Box(
                    modifier = Modifier.size(44.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(
                        progress = { progress },
                        modifier = Modifier.size(44.dp).graphicsLayer {
                            scaleX = pulseScale
                            scaleY = pulseScale
                        },
                        color = ringColor,
                        trackColor = ringColor.copy(alpha = 0.15f),
                        strokeWidth = 6.dp,
                        strokeCap = StrokeCap.Round,
                    )
                    Text(
                        order.status.ringLabel,
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 8.sp, fontWeight = FontWeight.Bold),
                        color = ringColor,
                    )
                }
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    if (order.supplierName.isNotBlank()) {
                        Text(
                            order.supplierName,
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.SemiBold,
                            color = MaterialTheme.colorScheme.primary,
                        )
                    }
                    Text(
                        "Order #${order.id.takeLast(3)}",
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        "${order.itemCount} items · ${order.displayTotal}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
                OrderStatusBadge(order.status)
            }

            // Countdown
            if (order.estimatedDelivery != null) {
                Spacer(modifier = Modifier.height(12.dp))
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(SquircleShape)
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.2f))
                        .padding(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.Center,
                ) {
                    Text(
                        "Arriving in ",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                    CountdownTimer(
                        targetIso = order.estimatedDelivery,
                        style = MaterialTheme.typography.labelMedium.copy(
                            fontWeight = FontWeight.Bold,
                            letterSpacing = 1.sp,
                        ),
                        color = MaterialTheme.colorScheme.onSurface,
                    )
                }
            }

            Spacer(modifier = Modifier.height(12.dp))
            HorizontalDivider(
                thickness = 0.5.dp,
                color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f),
            )
            Spacer(modifier = Modifier.height(12.dp))

            // Action buttons
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    "Details",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    modifier = Modifier
                        .clip(PillShape)
                        .cashable { onDetailsCash() }
                        .background(
                            MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f),
                            PillShape,
                        )
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
                val qrUnlocked = isApproaching || order.status == OrderStatus.ARRIVED
                if (order.status.hasDeliveryToken && qrUnlocked) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier
                            .clip(PillShape)
                            .cashable { onQRCash() }
                            .background(MaterialTheme.colorScheme.primary, PillShape)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    ) {
                        Icon(
                            Icons.Outlined.QrCode2,
                            contentDescription = null,
                            modifier = Modifier.size(12.dp),
                            tint = MaterialTheme.colorScheme.onPrimary,
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text(
                            "Show QR",
                            style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                            color = MaterialTheme.colorScheme.onPrimary,
                        )
                    }
                } else if (order.status.hasDeliveryToken && !qrUnlocked) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier
                            .clip(PillShape)
                            .background(
                                MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f),
                                PillShape,
                            )
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    ) {
                        Icon(
                            Icons.Outlined.QrCode2,
                            contentDescription = null,
                            modifier = Modifier.size(12.dp),
                            tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text(
                            "Awaiting Driver",
                            style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
                        )
                    }
                } else {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier
                            .clip(PillShape)
                            .background(
                                MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f),
                                PillShape,
                            )
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    ) {
                        Icon(
                            Icons.Outlined.QrCode2,
                            contentDescription = null,
                            modifier = Modifier.size(12.dp),
                            tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text(
                            "Awaiting Dispatch",
                            style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
                        )
                    }
                }
            }
        }
    }
}


