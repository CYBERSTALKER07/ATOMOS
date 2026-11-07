package com.pegasusx.retailer.ui.screens.orders.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.QrCode2
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.OrderStatus
import com.pegasusx.retailer.data.model.RetailerAIPrediction
import com.pegasusx.retailer.ui.components.CountdownTimer
import com.pegasusx.retailer.ui.components.OrderStatusBadge
import com.pegasusx.retailer.ui.components.statusColor
import com.pegasusx.retailer.ui.theme.PillShape
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.ui.theme.StatusGreen
import com.pegasusx.retailer.ui.theme.StatusOrange
import com.pegasusx.retailer.ui.theme.StatusRed
import com.pegasusx.retailer.ui.theme.StatusTeal
import com.pegasusx.retailer.R

@Composable
fun ActiveOrderCard(
    order: Order,
    onDetailsCash: () -> Unit,
    onQRCash: () -> Unit,
) {
    val progress = order.status.progressFraction
    val ringColor = order.status.statusColor()

    Surface(
        modifier = Modifier.fillMaxWidth().shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.Top) {
                // 3-Step determinate progress ring
                Box(modifier = Modifier.size(44.dp), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(
                        progress = { progress },
                        modifier = Modifier.size(44.dp),
                        color = ringColor,
                        trackColor = ringColor.copy(alpha = 0.15f),
                        strokeWidth = 6.dp,
                        strokeCap = StrokeCap.Round,
                    )
                    Text(
                        order.status.ringLabel,
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.Bold),
                        color = ringColor,
                    )
                }
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(stringResource(R.string.mobile_retailer_ui_order_takelast, order.id.takeLast(3)), style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                        if (order.isAiGenerated) {
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                "AI",
                                style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.ExtraBold),
                                color = Color.White,
                                modifier = Modifier
                                    .background(MaterialTheme.colorScheme.primary, RoundedCornerShape(4.dp))
                                    .padding(horizontal = 5.dp, vertical = 1.dp),
                            )
                        }
                    }
                    Text(stringResource(R.string.mobile_retailer_ui_itemcount_items_displaytotal, order.itemCount, order.displayTotal), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                }
                OrderStatusBadge(order.status)
            }

            // ── Order Status Timeline ──
            Spacer(modifier = Modifier.height(14.dp))
            OrderStatusTimeline(currentStep = order.status.timelineStepIndex)

            // Countdown
            if (order.estimatedDelivery != null) {
                Spacer(modifier = Modifier.height(10.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    CountdownTimer(
                        targetIso = order.estimatedDelivery,
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold, fontSize = 12.sp),
                        color = StatusGreen,
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("until arrival", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
                }
            }

            // Tag pills
            if (order.items.isNotEmpty()) {
                Spacer(modifier = Modifier.height(10.dp))
                Row(
                    modifier = Modifier.horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    order.items.take(3).forEach { item ->
                        Text(
                            item.productName.split(" ").take(2).joinToString(" "),
                            style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp),
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                            modifier = Modifier
                                .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f), PillShape)
                                .padding(horizontal = 8.dp, vertical = 4.dp),
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f))
            Spacer(modifier = Modifier.height(12.dp))

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                // Details
                Text(
                    "Details",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onDetailsCash() }
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
                // QR
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onQRCash() }
                        .background(MaterialTheme.colorScheme.primary, PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                ) {
                    Icon(Icons.Outlined.QrCode2, contentDescription = null, modifier = Modifier.size(12.dp), tint = MaterialTheme.colorScheme.onPrimary)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Show QR", style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold), color = MaterialTheme.colorScheme.onPrimary)
                }
            }
        }
    }
}

@Composable
fun OrderedCard(
    order: Order,
    onDetailsCash: () -> Unit,
    onCancel: () -> Unit,
    onConfirmAi: () -> Unit = {},
    onRejectAi: () -> Unit = {},
    onConfirmPreorder: () -> Unit = {},
    onEditPreorder: () -> Unit = {},
    onAcceptDeliveryProposal: () -> Unit = {},
    onRejectDeliveryProposal: () -> Unit = {},
) {
    val ringColor = order.status.statusColor()

    Surface(
        modifier = Modifier.fillMaxWidth().shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                // Progress ring (1/3 for PENDING)
                Box(modifier = Modifier.size(44.dp), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(
                        progress = { order.status.progressFraction },
                        modifier = Modifier.size(44.dp),
                        color = ringColor,
                        trackColor = ringColor.copy(alpha = 0.15f),
                        strokeWidth = 6.dp,
                        strokeCap = StrokeCap.Round,
                    )
                    Text(
                        order.status.ringLabel,
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.Bold),
                        color = ringColor,
                    )
                }
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(stringResource(R.string.mobile_retailer_ui_order_takelast, order.id.takeLast(3)), style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                        if (order.isAiGenerated) {
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                "AI",
                                style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.ExtraBold),
                                color = Color.White,
                                modifier = Modifier
                                    .background(MaterialTheme.colorScheme.primary, RoundedCornerShape(4.dp))
                                    .padding(horizontal = 5.dp, vertical = 1.dp),
                            )
                        }
                    }
                    Text(stringResource(R.string.mobile_retailer_ui_itemcount_items_displaytotal, order.itemCount, order.displayTotal), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                }
                OrderStatusBadge(order.status)
            }

            if (order.needsDeliveryProposalReview) {
                Spacer(modifier = Modifier.height(10.dp))
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(StatusOrange.copy(alpha = 0.12f), SoftSquircleShape)
                        .padding(horizontal = 10.dp, vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        "Review Delivery",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold),
                        color = StatusOrange,
                    )
                    order.proposedDeliveryDate?.let { date ->
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            stringResource(R.string.mobile_retailer_ui_proposed_date_2, date),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
                        )
                    }
                }
                order.deliveryProposalReason?.takeIf { it.isNotBlank() }?.let { reason ->
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        reason,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    )
                }
            }

            // Tag pills
            if (order.items.isNotEmpty()) {
                Spacer(modifier = Modifier.height(10.dp))
                Row(
                    modifier = Modifier.horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    order.items.take(3).forEach { item ->
                        Text(
                            item.productName.split(" ").take(2).joinToString(" "),
                            style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp),
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                            modifier = Modifier
                                .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f), PillShape)
                                .padding(horizontal = 8.dp, vertical = 4.dp),
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f))
            Spacer(modifier = Modifier.height(12.dp))

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                if (order.needsDeliveryProposalReview) {
                    Text(
                        "Reject",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                        color = StatusRed,
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable { onRejectDeliveryProposal() }
                            .background(StatusRed.copy(alpha = 0.1f), PillShape)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                    Text(
                        "Accept Date",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                        color = Color.White,
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable { onAcceptDeliveryProposal() }
                            .background(MaterialTheme.colorScheme.primary, PillShape)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                } else if (order.needsAiConfirmation) {
                    Text(
                        "Reject",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                        color = StatusRed,
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable { onRejectAi() }
                            .background(StatusRed.copy(alpha = 0.1f), PillShape)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                    Text(
                        "Confirm",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                        color = Color.White,
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable { onConfirmAi() }
                            .background(MaterialTheme.colorScheme.primary, PillShape)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                } else if (order.needsPreorderAction) {
                    Text(
                        "Edit",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable { onEditPreorder() }
                            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f), PillShape)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                    Text(
                        "Confirm Preorder",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                        color = Color.White,
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable { onConfirmPreorder() }
                            .background(MaterialTheme.colorScheme.primary, PillShape)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                } else {
                    Text(
                        "Cancel",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                        color = StatusRed,
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable { onCancel() }
                            .background(StatusRed.copy(alpha = 0.1f), PillShape)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                }
                Text(
                    "Details",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onDetailsCash() }
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
        }
    }
}

@Composable
fun AiPlannedCard(
    item: RetailerAIPrediction,
    onConfirm: () -> Unit,
    onReject: () -> Unit,
) {
    val trackColor = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.3f)
    val statusShort = item.statusLabel.take(7)

    Surface(
        modifier = Modifier.fillMaxWidth().shadow(3.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    modifier = Modifier.size(40.dp).drawBehind {
                        val sw = 3.dp.toPx()
                        val arcSize = Size(size.width - sw, size.height - sw)
                        val tl = Offset(sw / 2, sw / 2)
                        drawArc(trackColor, 0f, 360f, false, topLeft = tl, size = arcSize, style = Stroke(sw))
                    },
                    contentAlignment = Alignment.Center,
                ) {
                    Text(statusShort, style = MaterialTheme.typography.labelSmall.copy(fontSize = 8.sp, fontWeight = FontWeight.Bold), color = StatusOrange)
                }
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        item.title,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        "${item.quantity} units · ${item.statusLabel}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
                Spacer(modifier = Modifier.width(8.dp))
                Column(horizontalAlignment = Alignment.End) {
                    Text(
                        item.deliveryLabel,
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Medium),
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        item.formattedTotal,
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 11.sp, fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.onSurface,
                    )
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.2f), SoftSquircleShape)
                    .padding(horizontal = 10.dp, vertical = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Rounded.AutoAwesome, contentDescription = null, modifier = Modifier.size(12.dp), tint = MaterialTheme.colorScheme.primary)
                Spacer(modifier = Modifier.width(6.dp))
                Text(
                    item.orderId,
                    style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                )
            }
            Spacer(modifier = Modifier.height(8.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    "Confirm",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onConfirm() }
                        .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.12f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
                Text(
                    "Reject",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    color = StatusRed,
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onReject() }
                        .background(StatusRed.copy(alpha = 0.1f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
        }
    }
}

@Composable
fun OrderStatusTimeline(currentStep: Int) {
    val steps = OrderStatus.timelineSteps
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.15f),
                RoundedCornerShape(10.dp),
            )
            .padding(horizontal = 10.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        steps.forEachIndexed { index, (label, _) ->
            val isCompleted = index < currentStep
            val isActive = index == currentStep
            val dotColor = when {
                isCompleted -> StatusGreen
                isActive -> StatusTeal
                else -> MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.4f)
            }
            val labelColor = when {
                isCompleted -> MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                isActive -> StatusTeal
                else -> MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f)
            }
            val labelWeight = if (isActive) FontWeight.Bold else FontWeight.Medium

            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier.weight(1f),
            ) {
                Box(
                    modifier = Modifier
                        .size(if (isActive) 10.dp else 8.dp)
                        .background(dotColor, CircleShape),
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = label,
                    style = MaterialTheme.typography.labelSmall.copy(
                        fontSize = 8.sp,
                        fontWeight = labelWeight,
                        lineHeight = 10.sp,
                    ),
                    color = labelColor,
                    maxLines = 1,
                )
            }
        }
    }
}
