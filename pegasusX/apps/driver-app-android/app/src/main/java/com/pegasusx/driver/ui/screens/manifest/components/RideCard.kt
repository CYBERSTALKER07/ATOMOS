package com.pegasusx.driver.ui.screens.manifest.components

import androidx.compose.foundation.background
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.components.StateBadge
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.ui.theme.pressable

@Composable
fun RideCard(order: Order, loadSeqLabel: String? = null, onClick: () -> Unit) {
    val lab = LocalPegasusColors.current
    val isDark = isSystemInDarkTheme()
    val colorScheme = MaterialTheme.colorScheme

    PegasusCard(
        modifier = Modifier
            .padding(horizontal = PegasusSpacing.s16, vertical = 7.dp)
            .pressable(onClick = onClick)
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.s20),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Loading sequence badge
            if (loadSeqLabel != null) {
                Box(
                    modifier = Modifier
                        .clip(RoundedCornerShape(4.dp))
                        .background(colorScheme.primaryContainer)
                        .padding(horizontal = 10.dp, vertical = 4.dp)
                ) {
                    Text(
                        text = loadSeqLabel,
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Bold,
                        fontFamily = FontFamily.Monospace,
                        color = colorScheme.onPrimaryContainer,
                    )
                }
            }

            // Top: order ID + status
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = order.id,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fg
                )
                StateBadge(state = order.state)
            }

            // Info chips
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                InfoChip(icon = Icons.Default.CreditCard, text = order.retailerName)
                InfoChip(icon = Icons.Default.LocalShipping, text = order.totalAmount.formattedAmount())
            }
            val intentBadges = buildList {
                order.preorderBadge?.takeIf { it.isNotBlank() }?.let { add(it) }
                if (order.orderSource == "MANUAL_PREORDER") add("Pre-order")
                order.deliverBefore?.takeIf { it.isNotBlank() }?.let { add("Deliver by") }
                if (order.deliveryPriority == "EXPRESS") add("Express")
            }
            if (intentBadges.isNotEmpty()) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    intentBadges.forEach { label ->
                        AssistChip(onClick = {}, label = { Text(label) })
                    }
                }
            }

            // Delivery target
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(
                    text = "DELIVERY TARGET",
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgTertiary
                )
                if (order.latitude != null && order.longitude != null) {
                    Text(
                        text = String.format("%.4f, %.4f", order.latitude, order.longitude),
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.SemiBold,
                        fontFamily = FontFamily.Monospace,
                        color = lab.fg
                    )
                } else {
                    Text(
                        text = order.deliveryAddress,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.SemiBold,
                        color = lab.fg
                    )
                }
            }

            // Bottom row: items count + arrow
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "${order.items.size} items",
                    style = MaterialTheme.typography.bodySmall,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgSecondary
                )
                Box(
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape)
                        .background(lab.fg.copy(alpha = 0.08f)),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                        contentDescription = null,
                        tint = lab.fg,
                        modifier = Modifier.size(18.dp)
                    )
                }
            }

            // Bottom accent bar
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(2.dp)
                    .clip(RoundedCornerShape(2.dp))
                    .background(lab.fg.copy(alpha = 0.12f))
            )
        }
    }
}
